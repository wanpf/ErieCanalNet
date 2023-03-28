package main

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
)

const trueValue = "true"

const namespaceAddDescription = `
This command adds a namespace or set of namespaces to the mesh so that the ecnet
control plane with the given ecnet name can observe resources within that namespace
or set of namespaces.
`

type namespaceAddCmd struct {
	out                     io.Writer
	namespaces              []string
	ecnetName               string
	disableSidecarInjection bool
	clientSet               kubernetes.Interface
}

func newNamespaceAdd(out io.Writer) *cobra.Command {
	namespaceAdd := &namespaceAddCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "add NAMESPACE ...",
		Short: "add namespace to ecnet",
		Long:  namespaceAddDescription,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			namespaceAdd.namespaces = args
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			namespaceAdd.clientSet = clientset
			return namespaceAdd.run()
		},
	}

	//add ecnet name flag
	f := cmd.Flags()
	f.StringVar(&namespaceAdd.ecnetName, "ecnet-name", "ecnet", "Name of the ecnet")

	//add sidecar injection flag
	f.BoolVar(&namespaceAdd.disableSidecarInjection, "disable-sidecar-injection", false, "Disable automatic sidecar injection")

	return cmd
}

func (a *namespaceAddCmd) run() error {
	for _, ns := range a.namespaces {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		exists, err := meshExists(a.clientSet, a.ecnetName)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("mesh [%s] does not exist, please specify another mesh using --ecnet-name or create a new mesh", a.ecnetName)
		}

		deploymentsClient := a.clientSet.AppsV1().Deployments(ns)
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.ECNETControllerName}}

		listOptions := metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		}
		list, _ := deploymentsClient.List(context.TODO(), listOptions)

		// if ecnet-controller is installed in this namespace then don't add that to mesh
		if len(list.Items) != 0 {
			_, _ = fmt.Fprintf(a.out, "Namespace [%s] already has [%s] installed and cannot be added to mesh [%s]\n", ns, constants.ECNETControllerName, a.ecnetName)
			continue
		}

		// if the namespace is already a part of the mesh then don't add it again
		namespace, err := a.clientSet.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Could not add namespace [%s] to mesh [%s]: %w", ns, a.ecnetName, err)
		}
		ecnetName := namespace.Labels[constants.ECNETKubeResourceMonitorAnnotation]
		if a.ecnetName == ecnetName {
			_, _ = fmt.Fprintf(a.out, "Namespace [%s] has already been added to mesh [%s]\n", ns, a.ecnetName)
			continue
		}

		// if ignore label exits don`t add namespace
		if val, ok := namespace.ObjectMeta.Labels[constants.IgnoreLabel]; ok && val == trueValue {
			return fmt.Errorf("Cannot add ignored namespace")
		}

		var patch string
		if a.disableSidecarInjection {
			// Patch the namespace with monitoring label.
			// Disable sidecar injection.
			patch = fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": "%s"
		},
		"annotations": {
			"%s": "disabled"
		}
	}
}`, constants.ECNETKubeResourceMonitorAnnotation, a.ecnetName, constants.SidecarInjectionAnnotation)
		} else {
			// Patch the namespace with the monitoring label.
			// Enable sidecar injection.
			patch = fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": "%s"
		},
		"annotations": {
			"%s": "enabled"
		}
	}
}`, constants.ECNETKubeResourceMonitorAnnotation, a.ecnetName, constants.SidecarInjectionAnnotation)
		}

		_, err = a.clientSet.CoreV1().Namespaces().Patch(ctx, ns, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return fmt.Errorf("Could not add namespace [%s] to mesh [%s]: %w", ns, a.ecnetName, err)
		}

		_, _ = fmt.Fprintf(a.out, "Namespace [%s] successfully added to mesh [%s]\n", ns, a.ecnetName)
	}

	return nil
}

// meshExists determines if a mesh with ecnetName exists within the cluster
func meshExists(clientSet kubernetes.Interface, ecnetName string) (bool, error) {
	// search for the mesh across all namespaces
	deploymentsClient := clientSet.AppsV1().Deployments("")
	// search and match using the ecnet name provided
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"ecnetName": ecnetName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	ecnetControllerDeployments, err := deploymentsClient.List(context.TODO(), listOptions)
	if err != nil {
		return false, fmt.Errorf("Cannot obtain information about the mesh [%s]: [%w]", ecnetName, err)
	}
	// the mesh is present if there are ecnet controllers for the mesh
	return len(ecnetControllerDeployments.Items) != 0, nil
}
