package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
)

// confirm displays a prompt `s` to the user and returns a bool indicating yes / no
// If the lowercased, trimmed input begins with anything other than 'y', it returns false
// It accepts an int `tries` representing the number of attempts before returning false
func confirm(stdin io.Reader, stdout io.Writer, s string, tries int) (bool, error) {
	r := bufio.NewReader(stdin)

	for ; tries > 0; tries-- {
		fmt.Fprintf(stdout, "%s [y/n]: ", s)

		res, err := r.ReadString('\n')
		if err != nil {
			return false, err
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(res)) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			fmt.Fprintf(stdout, "Invalid input.\n")
			continue
		}
	}

	return false, nil
}

// getPrettyPrintedEcnetInfoList returns a pretty printed list of meshes.
func getPrettyPrintedEcnetInfoList(ecnetInfoList []ecnetInfo) string {
	s := "\nECNET NAME\tECNET NAMESPACE\tVERSION\tADDED NAMESPACES\n"

	for _, netInfo := range ecnetInfoList {
		m := fmt.Sprintf(
			"%s\t%s\t%s\t%s\n",
			netInfo.name,
			netInfo.namespace,
			netInfo.version,
			strings.Join(netInfo.monitoredNamespaces, ","),
		)
		s += m
	}

	return s
}

// getEcnetInfoList returns a list of meshes (including the info of each mesh) within the cluster
func getEcnetInfoList(restConfig *rest.Config, clientSet kubernetes.Interface) ([]ecnetInfo, error) {
	var ecnetInfoList []ecnetInfo

	ecnetControllerDeployments, err := getControllerDeployments(clientSet)
	if err != nil {
		return ecnetInfoList, fmt.Errorf("Could not list deployments %w", err)
	}
	if len(ecnetControllerDeployments.Items) == 0 {
		return ecnetInfoList, nil
	}

	for _, ecnetControllerDeployment := range ecnetControllerDeployments.Items {
		ecnetName := ecnetControllerDeployment.ObjectMeta.Labels["ecnetName"]
		ecnetNamespace := ecnetControllerDeployment.ObjectMeta.Namespace

		ecnetVersion := ecnetControllerDeployment.ObjectMeta.Labels[constants.ECNETAppVersionLabelKey]
		if ecnetVersion == "" {
			ecnetVersion = "Unknown"
		}

		var ecnetMonitoredNamespaces []string
		nsList, err := selectNamespacesMonitoredByEcnet(ecnetName, clientSet)
		if err == nil && len(nsList.Items) > 0 {
			for _, ns := range nsList.Items {
				ecnetMonitoredNamespaces = append(ecnetMonitoredNamespaces, ns.Name)
			}
		}

		ecnetInfoList = append(ecnetInfoList, ecnetInfo{
			name:                ecnetName,
			namespace:           ecnetNamespace,
			version:             ecnetVersion,
			monitoredNamespaces: ecnetMonitoredNamespaces,
		})
	}

	return ecnetInfoList, nil
}

// getControllerDeployments returns a list of Deployments corresponding to ecnet-controller
func getControllerDeployments(clientSet kubernetes.Interface) (*appsv1.DeploymentList, error) {
	deploymentsClient := clientSet.AppsV1().Deployments("") // Get deployments from all namespaces
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.ECNETControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return deploymentsClient.List(context.TODO(), listOptions)
}

// getControllerPods returns a list of ecnet-controller Pods in a specified namespace
func getControllerPods(clientSet kubernetes.Interface, namespace string) (*corev1.PodList, error) {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.ECNETControllerName}}
	podClient := clientSet.CoreV1().Pods(namespace)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return podClient.List(context.TODO(), metav1.ListOptions{LabelSelector: listOptions.LabelSelector})
}

// getPrettyPrintedCniInfoList returns a pretty printed list
// of meshes with supported smi versions
func getPrettyPrintedCniInfoList(cniInfoList []cniInfo) string {
	s := "\nECNET NAME\tECNET NAMESPACE\n"

	for _, cni := range cniInfoList {
		m := fmt.Sprintf(
			"%s\t%s\n",
			cni.name,
			cni.namespace,
		)
		s += m
	}

	return s
}

// getSupportedCniInfoForEcnetList returns a cniInfo list showing
// the supported smi versions for each ecnet mesh in the mesh list
func getSupportedCniInfoForEcnetList(ecnetInfoList []ecnetInfo, clientSet kubernetes.Interface, config *rest.Config, localPort uint16) []cniInfo {
	var cniInfoList []cniInfo

	for _, ecnet := range ecnetInfoList {
		cniInfoList = append(cniInfoList, cniInfo{
			name:      ecnet.name,
			namespace: ecnet.namespace,
		})
	}

	return cniInfoList
}
