package main

import (
	"bytes"
	"context"
	_ "embed" // required to embed resources
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cli"
)

const installDesc = `
This command installs an ecnet control plane on the Kubernetes cluster.

Example:
  $ ecnet install --ecnet-namespace hello-world
`
const (
	defaultChartPath          = ""
	defaultEcnetName          = "ecnet"
	defaultEnforceSingleEcnet = true
)

// chartTGZSource is the `helm package`d representation of the default Helm chart.
// Its value is embedded at build time.
//
//go:embed chart.tgz
var chartTGZSource []byte

type installCmd struct {
	out            io.Writer
	chartPath      string
	ecnetName      string
	timeout        time.Duration
	clientSet      kubernetes.Interface
	chartRequested *chart.Chart
	setOptions     []string
	atomic         bool
	// Toggle this to enforce only one ecnet in this cluster
	enforceSingleEcnet bool
	disableSpinner     bool
}

func newInstallCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	inst := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install ecnet control plane",
		Long:  installDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			inst.clientSet = clientset
			return inst.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.chartPath, "ecnet-chart-path", defaultChartPath, "path to ecnet chart to override default chart")
	f.StringVar(&inst.ecnetName, "ecnet-name", defaultEcnetName, "name for the new control plane instance")
	f.BoolVar(&inst.enforceSingleEcnet, "enforce-single-ecnet", defaultEnforceSingleEcnet, "Enforce only deploying one ecnet in the cluster")
	f.DurationVar(&inst.timeout, "timeout", 5*time.Minute, "Time to wait for installation and resources in a ready state, zero means no timeout")
	f.StringArrayVar(&inst.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	f.BoolVar(&inst.atomic, "atomic", false, "Automatically clean up resources if installation fails")

	return cmd
}

func (i *installCmd) run(config *helm.Configuration) error {
	if err := i.loadECNETChart(); err != nil {
		return err
	}

	// values represents the overrides for the ECNET chart's values.yaml file
	values, err := i.resolveValues()
	if err != nil {
		return err
	}

	installClient := helm.NewInstall(config)
	installClient.ReleaseName = i.ecnetName
	installClient.Namespace = settings.Namespace()
	installClient.CreateNamespace = true
	installClient.Wait = true
	installClient.Atomic = i.atomic
	installClient.Timeout = i.timeout

	debug("Beginning ECNET installation")
	if i.disableSpinner || settings.Verbose() {
		if _, err = installClient.Run(i.chartRequested, values); err != nil {
			if !settings.Verbose() {
				return err
			}

			pods, _ := i.clientSet.CoreV1().Pods(settings.Namespace()).List(context.Background(), metav1.ListOptions{})

			for _, pod := range pods.Items {
				fmt.Fprintf(i.out, "Status for pod %s in namespace %s:\n %v\n\n", pod.Name, pod.Namespace, pod.Status)
			}
			return err
		}
	} else {
		spinner := new(cli.Spinner)
		spinner.Init(i.clientSet, settings.Namespace(), values)
		err = spinner.Run(func() error {
			_, installErr := installClient.Run(i.chartRequested, values)
			return installErr
		})
		if err != nil {
			if !settings.Verbose() {
				return err
			}
		}
	}
	fmt.Fprintf(i.out, "ECNET installed successfully in namespace [%s] with ecnet name [%s]\n", settings.Namespace(), i.ecnetName)
	return nil
}

func (i *installCmd) loadECNETChart() error {
	debug("Loading ECNET helm chart")
	var err error
	if i.chartPath != "" {
		i.chartRequested, err = loader.Load(i.chartPath)
	} else {
		i.chartRequested, err = loader.LoadArchive(bytes.NewReader(chartTGZSource))
	}

	if err != nil {
		return fmt.Errorf("error loading chart for installation: %w", err)
	}

	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	if err := parseVal(i.setOptions, finalValues); err != nil {
		return nil, fmt.Errorf("invalid format for --set: %w", err)
	}

	valuesConfig := []string{
		fmt.Sprintf("ecnet.ecnetName=%s", i.ecnetName),
		fmt.Sprintf("ecnet.enforceSingleEcnet=%t", i.enforceSingleEcnet),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
	}

	return finalValues, nil
}

// parses Helm strvals line and merges into a map
func parseVal(vals []string, parsedVals map[string]interface{}) error {
	for _, v := range vals {
		if err := strvals.ParseInto(v, parsedVals); err != nil {
			return err
		}
	}
	return nil
}
