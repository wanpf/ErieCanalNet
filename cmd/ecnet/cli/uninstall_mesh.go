package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	helmStorage "helm.sh/helm/v3/pkg/storage/driver"
	extensionsClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8sApiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
)

const uninstallMeshDescription = `
This command will uninstall an instance of the ecnet control plane
given the ecnet name and namespace.
`

type uninstallMeshCmd struct {
	out                        io.Writer
	in                         io.Reader
	config                     *rest.Config
	ecnetName                  string
	ecnetNamespace             string
	caBundleSecretName         string
	force                      bool
	deleteNamespace            bool
	client                     *action.Uninstall
	clientSet                  kubernetes.Interface
	localPort                  uint16
	deleteClusterWideResources bool
	extensionsClientset        extensionsClientset.Interface
	actionConfig               *action.Configuration
}

func newUninstallMeshCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	uninstall := &uninstallMeshCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "uninstall ecnet control plane instance",
		Long:  uninstallMeshDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			uninstall.actionConfig = config
			uninstall.client = action.NewUninstall(config)

			// get kubeconfig and initialize k8s client
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			uninstall.config = kubeconfig

			uninstall.clientSet, err = kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}

			uninstall.extensionsClientset, err = extensionsClientset.NewForConfig(kubeconfig)
			if err != nil {
				return fmt.Errorf("Could not access extension client set: %w", err)
			}

			uninstall.ecnetNamespace = settings.Namespace()
			return uninstall.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&uninstall.ecnetName, "ecnet-name", "", "Name of the service mesh")
	f.BoolVarP(&uninstall.force, "force", "f", false, "Attempt to uninstall the ecnet control plane instance without prompting for confirmation.")
	f.BoolVarP(&uninstall.deleteClusterWideResources, "delete-cluster-wide-resources", "a", false, "Cluster wide resources (such as ecnet CRDs, mutating webhook configurations, validating webhook configurations and ecnet secrets) are fully deleted from the cluster after control plane components are deleted.")
	f.BoolVar(&uninstall.deleteNamespace, "delete-namespace", false, "Attempt to delete the namespace after control plane components are deleted")
	f.Uint16VarP(&uninstall.localPort, "local-port", "p", constants.ECNETHTTPServerPort, "Local port to use for port forwarding")
	f.StringVar(&uninstall.caBundleSecretName, "ca-bundle-secret-name", constants.DefaultCABundleSecretName, "Name of the secret for the ECNET CA bundle")

	return cmd
}

func (d *uninstallMeshCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	meshesToUninstall := []meshInfo{}

	if !settings.IsManaged() {
		meshInfoList, err := getMeshInfoList(d.config, d.clientSet)
		if err != nil {
			return fmt.Errorf("unable to list meshes within the cluster: %w", err)
		}
		if len(meshInfoList) == 0 {
			fmt.Fprintf(d.out, "No ECNET control planes found\n")
			return nil
		}

		if d.meshSpecified() {
			// Searches for the mesh specified by the ecnet-name flag if specified
			specifiedMeshFound := d.findSpecifiedMesh(meshInfoList)
			if !specifiedMeshFound {
				return nil
			}
		}

		// Adds the mesh to be force uninstalled
		if d.force {
			// For force uninstall, if single mesh in cluster, set default to that mesh
			if len(meshInfoList) == 1 {
				d.ecnetName = meshInfoList[0].name
				d.ecnetNamespace = meshInfoList[0].namespace
			}
			forceMesh := meshInfo{name: d.ecnetName, namespace: d.ecnetNamespace}
			meshesToUninstall = append(meshesToUninstall, forceMesh)
		} else {
			// print a list of meshes within the cluster for a better user experience
			err := d.printMeshes()
			if err != nil {
				return err
			}
			// Prompts user on whether to uninstall each ECNET mesh in the cluster
			uninstallMeshes, err := d.promptMeshUninstall(meshInfoList, meshesToUninstall)
			if err != nil {
				return err
			}
			meshesToUninstall = append(meshesToUninstall, uninstallMeshes...)
		}

		for _, m := range meshesToUninstall {
			// Re-initializes uninstall config with the namespace of the mesh to be uninstalled
			err := d.actionConfig.Init(settings.RESTClientGetter(), m.namespace, "secret", debug)
			if err != nil {
				return err
			}

			_, err = d.client.Run(m.name)
			if err != nil {
				if errors.Is(err, helmStorage.ErrReleaseNotFound) {
					fmt.Fprintf(d.out, "No ECNET control plane with ecnet name [%s] found in namespace [%s]\n", m.name, m.namespace)
				}

				if !d.deleteClusterWideResources && !d.deleteNamespace {
					return err
				}

				fmt.Fprintf(d.out, "Could not uninstall ecnet name [%s] in namespace [%s]- %v - continuing to deleteClusterWideResources and/or deleteNamespace\n", m.name, m.namespace, err)
			}

			if err == nil {
				fmt.Fprintf(d.out, "ECNET [ecnet name: %s] in namespace [%s] uninstalled\n", m.name, m.namespace)
			}

			err = d.deleteNs(ctx, m.namespace)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintf(d.out, "ECNET CANNOT be uninstalled in a managed environment\n")
		if d.deleteNamespace {
			fmt.Fprintf(d.out, "ECNET namespace CANNOT be deleted in a managed environment\n")
		}
	}

	err := d.deleteClusterResources()
	return err
}

func (d *uninstallMeshCmd) meshSpecified() bool {
	return d.ecnetName != ""
}

func (d *uninstallMeshCmd) findSpecifiedMesh(meshInfoList []meshInfo) bool {
	specifiedMeshFound := d.findMesh(meshInfoList)
	if !specifiedMeshFound {
		fmt.Fprintf(d.out, "Did not find mesh [%s] in namespace [%s]\n", d.ecnetName, d.ecnetNamespace)
		// print a list of meshes within the cluster for a better user experience
		if err := d.printMeshes(); err != nil {
			fmt.Fprintf(d.out, "Unable to list meshes in the cluster - [%v]", err)
		}
	}

	return specifiedMeshFound
}

func (d *uninstallMeshCmd) promptMeshUninstall(meshInfoList, meshesToUninstall []meshInfo) ([]meshInfo, error) {
	for _, mesh := range meshInfoList {
		// Only prompt for specified mesh if `ecnet-name` is specified
		if d.meshSpecified() && mesh.name != d.ecnetName {
			continue
		}
		confirm, err := confirm(d.in, d.out, fmt.Sprintf("\nUninstall ECNET [ecnet name: %s] in namespace [%s] and/or ECNET resources?", mesh.name, mesh.namespace), 3)
		if err != nil {
			return nil, err
		}
		if confirm {
			meshesToUninstall = append(meshesToUninstall, mesh)
		}
	}
	return meshesToUninstall, nil
}

func (d *uninstallMeshCmd) deleteNs(ctx context.Context, ns string) error {
	if !d.deleteNamespace {
		return nil
	}
	if err := d.clientSet.CoreV1().Namespaces().Delete(ctx, ns, v1.DeleteOptions{}); err != nil {
		if k8sApiErrors.IsNotFound(err) {
			fmt.Fprintf(d.out, "ECNET namespace [%s] not found\n", ns)
			return nil
		}
		return fmt.Errorf("Could not delete ECNET namespace [%s] - %v", ns, err)
	}
	fmt.Fprintf(d.out, "ECNET namespace [%s] deleted successfully\n", ns)
	return nil
}

func (d *uninstallMeshCmd) deleteClusterResources() error {
	if d.deleteClusterWideResources {
		meshInfoList, err := getMeshInfoList(d.config, d.clientSet)
		if err != nil {
			return fmt.Errorf("unable to list meshes within the cluster: %w", err)
		}
		if len(meshInfoList) != 0 {
			fmt.Fprintf(d.out, "Deleting cluster resources will affect current mesh(es) in cluster:\n")
			for _, m := range meshInfoList {
				fmt.Fprintf(d.out, "[%s] mesh in namespace [%s]\n", m.name, m.namespace)
			}
		}

		failedDeletions := d.uninstallClusterResources()
		if len(failedDeletions) != 0 {
			return fmt.Errorf("Failed to completely delete the following ECNET resource types: %+v", failedDeletions)
		}
	}
	return nil
}

// uninstallClusterResources uninstalls all ecnet and smi-related cluster resources
func (d *uninstallMeshCmd) uninstallClusterResources() []string {
	var failedDeletions []string
	err := d.uninstallCustomResourceDefinitions()
	if err != nil {
		failedDeletions = append(failedDeletions, "CustomResourceDefinitions")
	}

	err = d.uninstallSecrets()
	if err != nil {
		failedDeletions = append(failedDeletions, "Secrets")
	}
	return failedDeletions
}

// uninstallCustomResourceDefinitions uninstalls ecnet and smi-related crds from the cluster.
func (d *uninstallMeshCmd) uninstallCustomResourceDefinitions() error {
	crds := []string{
		"ecnetconfigs.config.flomesh.io",
	}

	var failedDeletions []string
	for _, crd := range crds {
		err := d.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), crd, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted ECNET CRD: %s\n", crd)
			continue
		}

		if k8sApiErrors.IsNotFound(err) {
			fmt.Fprintf(d.out, "Ignoring - did not find ECNET CRD: %s\n", crd)
		} else {
			fmt.Fprintf(d.out, "Failed to delete ECNET CRD %s: %s\n", crd, err.Error())
			failedDeletions = append(failedDeletions, crd)
		}
	}

	if len(failedDeletions) != 0 {
		return fmt.Errorf("Failed to delete the following ECNET CRDs: %+v", failedDeletions)
	}

	return nil
}

// uninstallSecrets uninstalls ecnet-related secrets from the cluster.
func (d *uninstallMeshCmd) uninstallSecrets() error {
	secrets := []string{
		d.caBundleSecretName,
	}

	var failedDeletions []string
	for _, secret := range secrets {
		err := d.clientSet.CoreV1().Secrets(d.ecnetNamespace).Delete(context.Background(), secret, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted ECNET secret %s in namespace %s\n", secret, d.ecnetNamespace)
			continue
		}

		if k8sApiErrors.IsNotFound(err) {
			if secret == d.caBundleSecretName {
				fmt.Fprintf(d.out, "Ignoring - did not find ECNET CA bundle secret %s in namespace %s. Use --ca-bundle-secret-name and --ecnet-namespace to delete a specific ecnet namespace's CA bundle secret if desired\n", secret, d.ecnetNamespace)
			} else {
				fmt.Fprintf(d.out, "Ignoring - did not find ECNET secret %s in namespace %s. Use --ecnet-namespace to delete a specific ecnet namespace's secret if desired\n", secret, d.ecnetNamespace)
			}
		} else {
			fmt.Fprintf(d.out, "Found but failed to delete the ECNET secret %s in namespace %s: %s\n", secret, d.ecnetNamespace, err.Error())
			failedDeletions = append(failedDeletions, secret)
		}
	}

	if len(failedDeletions) != 0 {
		return fmt.Errorf("Found but failed to delete the following ECNET secrets in namespace %s: %+v", d.ecnetNamespace, failedDeletions)
	}

	return nil
}

// findMesh looks for specified `ecnet-name` mesh from the meshes in the cluster
func (d *uninstallMeshCmd) findMesh(meshInfoList []meshInfo) bool {
	found := false
	for _, m := range meshInfoList {
		if m.name == d.ecnetName {
			found = true
			break
		}
	}
	return found
}

// printMeshes prints list of meshes within the cluster for a better user experience
func (d *uninstallMeshCmd) printMeshes() error {
	fmt.Fprintf(d.out, "List of meshes present in the cluster:\n")

	listCmd := &meshListCmd{
		out:       d.out,
		config:    d.config,
		clientSet: d.clientSet,
		localPort: d.localPort,
	}

	err := listCmd.run()
	// Unable to list meshes in the cluster
	if err != nil {
		return err
	}
	return nil
}
