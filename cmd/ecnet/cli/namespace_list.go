package main

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
)

const namespaceListDescription = `
This command will list namespace information for all meshes. It is possible to filter by a given mesh.
`

type namespaceListCmd struct {
	out       io.Writer
	ecnetName string
	clientSet kubernetes.Interface
}

func newNamespaceList(out io.Writer) *cobra.Command {
	namespaceList := &namespaceListCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list namespaces enlisted in meshes",
		Long:  namespaceListDescription,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				namespaceList.ecnetName = args[0]
			}

			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			namespaceList.clientSet = clientset
			return namespaceList.run()
		},
	}

	//add ecnet name flag
	f := cmd.Flags()
	f.StringVar(&namespaceList.ecnetName, "ecnet-name", "", "Name of service mesh to list namespaces")

	return cmd
}

func (l *namespaceListCmd) run() error {
	namespaces, err := selectNamespacesMonitoredByMesh(l.ecnetName, l.clientSet)
	if err != nil {
		return fmt.Errorf("Could not list namespaces related to ecnet [%s]: %w", l.ecnetName, err)
	}

	if len(namespaces.Items) == 0 {
		if l.ecnetName != "" {
			fmt.Fprintf(l.out, "No namespaces in mesh [%s]\n", l.ecnetName)
			return nil
		}

		fmt.Fprintf(l.out, "No namespaces in any mesh\n")
		return nil
	}

	w := newTabWriter(l.out)
	fmt.Fprintln(w, "NAMESPACE\tMESH\tSIDECAR-INJECTION")
	for _, ns := range namespaces.Items {
		ecnetName := ns.ObjectMeta.Labels[constants.ECNETKubeResourceMonitorAnnotation]
		sidecarInjectionEnabled, ok := ns.ObjectMeta.Annotations[constants.SidecarInjectionAnnotation]
		if !ok {
			sidecarInjectionEnabled = "-" // not set
		}
		if _, ignored := ns.Labels[constants.IgnoreLabel]; ignored {
			sidecarInjectionEnabled = "disabled (ignored)"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", ns.Name, ecnetName, sidecarInjectionEnabled)
	}
	_ = w.Flush()

	return nil
}

func selectNamespacesMonitoredByMesh(ecnetName string, clientSet kubernetes.Interface) (*v1.NamespaceList, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector := constants.ECNETKubeResourceMonitorAnnotation
	if ecnetName != "" {
		selector = fmt.Sprintf("%s=%s", selector, ecnetName)
	}

	return clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
}
