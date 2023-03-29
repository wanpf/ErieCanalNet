// Package main implements the previous install methods.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/version"
)

var log = logger.New("ecnet-preinstall")

func main() {
	log.Info().Msgf("Starting ecnet-preinstall %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)

	var verbosity string
	var enforceSingleEcnet bool
	var namespace string

	flags := pflag.NewFlagSet("ecnet-preinstall", pflag.ExitOnError)
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.BoolVar(&enforceSingleEcnet, "enforce-single-ecnet", true, "Enforce only deploying one ecnet in the cluster")
	flags.StringVar(&namespace, "namespace", "", "The namespace where the new ecnet is to be installed")

	err := flags.Parse(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("parsing flags")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("getting kube client config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("creating kube client")
	}

	checks := []func() error{
		singleEcnetOK(clientset, enforceSingleEcnet),
		namespaceHasNoEcnet(clientset, namespace),
	}

	ok := true
	for _, check := range checks {
		if err := check(); err != nil {
			ok = false
			log.Error().Err(err).Msg("check failed")
		}
	}
	if !ok {
		log.Fatal().Msg("checks failed")
	}
	log.Info().Msg("checks OK")
}

func singleEcnetOK(clientset kubernetes.Interface, enforceSingleEcnet bool) func() error {
	return func() error {
		deps, err := clientset.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				constants.AppLabel: constants.ECNETControllerName,
			}).String(),
		})
		if err != nil {
			return fmt.Errorf("listing ECNET deployments: %w", err)
		}

		var existingEcnets []string
		var existingSingleEcnets []string
		for _, dep := range deps.Items {
			ecn := fmt.Sprintf("namespace: %s, name: %s", dep.Namespace, dep.Labels["ecnetName"])
			existingEcnets = append(existingEcnets, ecn)
			if dep.Labels["enforceSingleEcnet"] == "true" {
				existingSingleEcnets = append(existingSingleEcnets, ecn)
			}
		}

		if len(existingSingleEcnets) > 0 {
			return fmt.Errorf("Ecnet(s) %s already enforce it is the only ecnet in the cluster, cannot install new ecnets", strings.Join(existingSingleEcnets, ", "))
		}

		if enforceSingleEcnet && len(existingEcnets) > 0 {
			return fmt.Errorf("Ecnet(s) %s already exist so a new ecnet enforcing it is the only one cannot be installed", strings.Join(existingEcnets, ", "))
		}

		return nil
	}
}

func namespaceHasNoEcnet(clientset kubernetes.Interface, namespace string) func() error {
	return func() error {
		deps, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				constants.AppLabel: constants.ECNETControllerName,
			}).String(),
		})
		if err != nil {
			return fmt.Errorf("listing ecnet-controller deployments in namespace %s: %w", namespace, err)
		}
		var ecnetNames []string
		for _, dep := range deps.Items {
			ecnetNames = append(ecnetNames, dep.Labels["ecnetName"])
		}
		if len(ecnetNames) > 0 {
			return fmt.Errorf("Namespace %s already contains ecnet %v", namespace, ecnetNames)
		}
		return nil
	}
}
