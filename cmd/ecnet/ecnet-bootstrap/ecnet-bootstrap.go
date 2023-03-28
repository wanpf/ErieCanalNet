// Package main implements the main entrypoint for ecnet-bootstrap and utility routines to
// bootstrap the various internal components of ecnet-bootstrap.
// ecnet-bootstrap provides crd conversion capability in ECNET.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/util"

	configv1alpha1 "github.com/flomesh-io/ErieCanal/pkg/ecnet/apis/config/v1alpha1"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
	configClientset "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/client/config/clientset/versioned"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/health"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/httpserver"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s/events"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/signals"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/version"
)

const (
	configName               = "ecnet-config"
	presetEcnetConfigName    = "preset-ecnet-config"
	presetEcnetConfigJSONKey = "preset-ecnet-config.json"
)

var (
	verbosity       string
	ecnetNamespace  string
	ecnetConfigName string
	meshName        string
	ecnetVersion    string
	trustDomain     string

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`ecnet-bootstrap`, pflag.ExitOnError)
	log   = logger.New(constants.ECNETBootstrapName)
)

type bootstrap struct {
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface
	namespace    string
}

func init() {
	flags.StringVar(&meshName, "mesh-name", "", "ECNET mesh name")
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&ecnetNamespace, "ecnet-namespace", "", "Namespace to which ECNET belongs to.")
	flags.StringVar(&ecnetConfigName, "ecnet-config-name", "ecnet-config", "Name of the ECNET EcnetConfig")
	flags.StringVar(&ecnetVersion, "ecnet-version", "", "Version of ECNET")

	// TODO (#4502): Remove when we add full MRC support
	flags.StringVar(&trustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting ecnet-bootstrap %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating kube configs using in-cluster config")
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	crdClient := apiclient.NewForConfigOrDie(kubeConfig)
	configClient, err := configClientset.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not access Kubernetes cluster, check kubeconfig.")
		return
	}

	k8s.SetTrustDomain(trustDomain)

	bootstrap := bootstrap{
		kubeClient:   kubeClient,
		configClient: configClient,
		namespace:    ecnetNamespace,
	}

	applyOrUpdateCRDs(crdClient)

	err = bootstrap.ensureEcnetConfig()
	if err != nil {
		log.Fatal().Err(err).Msgf("Error setting up default EcnetConfig %s from ConfigMap %s", configName, presetEcnetConfigName)
		return
	}

	err = bootstrap.initiatilizeKubernetesEventsRecorder()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing Kubernetes events recorder")
	}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	/*
	 * Initialize ecnet-bootstrap's HTTP server
	 */
	httpServer := httpserver.NewHTTPServer(constants.ECNETHTTPServerPort)
	// Version
	httpServer.AddHandler(constants.VersionPath, version.GetVersionHandler())

	httpServer.AddHandler(constants.WebhookHealthPath, http.HandlerFunc(health.SimpleHandler))

	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start ECNET metrics/probes HTTP server")
	}

	<-stop
	cancel()
	log.Info().Msgf("Stopping ecnet-bootstrap %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func applyOrUpdateCRDs(crdClient *apiclient.ApiextensionsV1Client) {
	crdFiles, err := filepath.Glob("/ecnet-crds/*.yaml")

	if err != nil {
		log.Fatal().Err(err).Msgf("error reading files from /ecnet-crds")
	}

	scheme = runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	decode := codecs.UniversalDeserializer().Decode

	for _, file := range crdFiles {
		yaml, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			log.Fatal().Err(err).Msgf("Error reading CRD file %s", file)
		}

		crd := &apiv1.CustomResourceDefinition{}
		_, _, err = decode(yaml, nil, crd)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error decoding CRD file %s", file)
		}

		if crd.Labels == nil {
			crd.Labels = make(map[string]string)
		}

		crdExisting, err := crdClient.CustomResourceDefinitions().Get(context.Background(), crd.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			log.Fatal().Err(err).Msgf("error getting CRD %s", crd.Name)
		}

		if apierrors.IsNotFound(err) {
			log.Info().Msgf("crds %s not found, creating CRD", crd.Name)
			if err := util.CreateApplyAnnotation(crd, unstructured.UnstructuredJSONScheme); err != nil {
				log.Fatal().Err(err).Msgf("Error applying annotation to CRD %s", crd.Name)
			}
			if _, err = crdClient.CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{}); err != nil {
				log.Fatal().Err(err).Msgf("Error creating crd : %s", crd.Name)
			}
			log.Info().Msgf("Successfully created crd: %s", crd.Name)
		} else {
			log.Info().Msgf("Patching conversion webhook configuration for crd: %s, setting to \"None\"", crd.Name)
			crdExisting.Spec = crd.Spec
			crdExisting.Spec.Conversion = &apiv1.CustomResourceConversion{
				Strategy: apiv1.NoneConverter,
			}
			if _, err = crdClient.CustomResourceDefinitions().Update(context.Background(), crdExisting, metav1.UpdateOptions{}); err != nil {
				log.Fatal().Err(err).Msgf("Error updating conversion webhook configuration for crd : %s", crd.Name)
			}
			log.Info().Msgf("successfully set conversion webhook configuration for crd : %s to \"None\"", crd.Name)
		}
	}
}

func (b *bootstrap) createDefaultEcnetConfig() error {
	// find presets config map to build the default EcnetConfig from that
	presetsConfigMap, err := b.kubeClient.CoreV1().ConfigMaps(b.namespace).Get(context.TODO(), presetEcnetConfigName, metav1.GetOptions{})

	// If the presets EcnetConfig could not be loaded return the error
	if err != nil {
		return err
	}

	// Create a default EcnetConfig
	defaultEcnetConfig, err := buildDefaultEcnetConfig(presetsConfigMap)
	if err != nil {
		return err
	}
	if _, err = b.configClient.ConfigV1alpha1().EcnetConfigs(b.namespace).Create(context.TODO(), defaultEcnetConfig, metav1.CreateOptions{}); err == nil {
		log.Info().Msgf("EcnetConfig (%s) created in namespace %s", configName, b.namespace)
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("EcnetConfig already exists in %s. Skip creating.", b.namespace)
		return nil
	}

	return err
}

func (b *bootstrap) ensureEcnetConfig() error {
	config, err := b.configClient.ConfigV1alpha1().EcnetConfigs(b.namespace).Get(context.TODO(), configName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// create a default mesh config since it was not found
		return b.createDefaultEcnetConfig()
	}
	if err != nil {
		return err
	}

	if _, exists := config.Annotations[corev1.LastAppliedConfigAnnotation]; !exists {
		// Mesh was found, but may not have the last applied annotation.
		if err := util.CreateApplyAnnotation(config, unstructured.UnstructuredJSONScheme); err != nil {
			return err
		}
		if _, err := b.configClient.ConfigV1alpha1().EcnetConfigs(b.namespace).Update(context.TODO(), config, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// initiatilizeKubernetesEventsRecorder initializes the generic Kubernetes event recorder and associates it with
//
//	the ecnet-bootstrap pod resource. The events recorder allows the ecnet-bootstap to publish Kubernets events to
//	report fatal errors with initializing this application. These events will show up in the output of `kubectl get events`
func (b *bootstrap) initiatilizeKubernetesEventsRecorder() error {
	bootstrapPod, err := b.getBootstrapPod()
	if err != nil {
		return fmt.Errorf("Error fetching ecnet-bootstrap pod: %w", err)
	}
	eventRecorder := events.GenericEventRecorder()
	return eventRecorder.Initialize(bootstrapPod, b.kubeClient, ecnetNamespace)
}

// getBootstrapPod returns the ecnet-bootstrap pod spec.
// The pod name is inferred from the 'BOOTSTRAP_POD_NAME' env variable which is set during deployment.
func (b *bootstrap) getBootstrapPod() (*corev1.Pod, error) {
	podName := os.Getenv("BOOTSTRAP_POD_NAME")
	if podName == "" {
		return nil, errors.New("BOOTSTRAP_POD_NAME env variable cannot be empty")
	}

	pod, err := b.kubeClient.CoreV1().Pods(b.namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving ecnet-bootstrap pod %s", podName)
		return nil, err
	}

	return pod, nil
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if ecnetNamespace == "" {
		return errors.New("Please specify the ECNET namespace using --ecnet-namespace")
	}

	return nil
}

func buildDefaultEcnetConfig(presetEcnetConfigMap *corev1.ConfigMap) (*configv1alpha1.EcnetConfig, error) {
	presetEcnetConfig := presetEcnetConfigMap.Data[presetEcnetConfigJSONKey]
	presetEcnetConfigSpec := configv1alpha1.EcnetConfigSpec{}
	err := json.Unmarshal([]byte(presetEcnetConfig), &presetEcnetConfigSpec)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error converting preset-ecnet-config json string to ecnetConfig object")
	}

	config := &configv1alpha1.EcnetConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EcnetConfig",
			APIVersion: "config.flomesh.io/configv1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
		Spec: presetEcnetConfigSpec,
	}

	return config, util.CreateApplyAnnotation(config, unstructured.UnstructuredJSONScheme)
}
