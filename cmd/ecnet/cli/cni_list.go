package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
)

const cniListDescription = `
This command will list all the ecnet control planes running in a Kubernetes cluster and controller pods.`

type cniListCmd struct {
	out       io.Writer
	config    *rest.Config
	clientSet kubernetes.Interface
	localPort uint16
}

type ecnetInfo struct {
	name                string
	namespace           string
	version             string
	monitoredNamespaces []string
}

type cniInfo struct {
	name      string
	namespace string
}

func newCniList(out io.Writer) *cobra.Command {
	listCmd := &cniListCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list control planes in k8s cluster",
		Long:  cniListDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			listCmd.config = config
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			listCmd.clientSet = clientset
			return listCmd.run()
		},
	}

	f := cmd.Flags()
	f.Uint16VarP(&listCmd.localPort, "local-port", "p", constants.ECNETHTTPServerPort, "Local port to use for port forwarding")

	return cmd
}

func (l *cniListCmd) run() error {
	ecnetInfoList, err := getEcnetInfoList(l.config, l.clientSet)
	if err != nil {
		fmt.Fprintf(l.out, "Unable to list ecnets within the cluster.\n")
		return err
	}
	if len(ecnetInfoList) == 0 {
		fmt.Fprintf(l.out, "No ecnet control planes found\n")
		return nil
	}

	w := newTabWriter(l.out)
	fmt.Fprint(w, getPrettyPrintedEcnetInfoList(ecnetInfoList))
	_ = w.Flush()

	cniInfoList := getSupportedCniInfoForEcnetList(ecnetInfoList, l.clientSet, l.config, l.localPort)
	fmt.Fprint(w, getPrettyPrintedCniInfoList(cniInfoList))
	_ = w.Flush()

	fmt.Fprintf(l.out, "\nTo list the ECNET controller pods, please run the following command passing in the ecnet's namespace\n")
	fmt.Fprintf(l.out, "\tkubectl get pods -n <ecnet-ecnet-namespace> -l app=ecnet-bridge\n")

	return nil
}
