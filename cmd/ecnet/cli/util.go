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

// getPrettyPrintedMeshInfoList returns a pretty printed list of meshes.
func getPrettyPrintedMeshInfoList(meshInfoList []meshInfo) string {
	s := "\nMESH NAME\tMESH NAMESPACE\tVERSION\tADDED NAMESPACES\n"

	for _, meshInfo := range meshInfoList {
		m := fmt.Sprintf(
			"%s\t%s\t%s\t%s\n",
			meshInfo.name,
			meshInfo.namespace,
			meshInfo.version,
			strings.Join(meshInfo.monitoredNamespaces, ","),
		)
		s += m
	}

	return s
}

// getMeshInfoList returns a list of meshes (including the info of each mesh) within the cluster
func getMeshInfoList(restConfig *rest.Config, clientSet kubernetes.Interface) ([]meshInfo, error) {
	var meshInfoList []meshInfo

	ecnetControllerDeployments, err := getControllerDeployments(clientSet)
	if err != nil {
		return meshInfoList, fmt.Errorf("Could not list deployments %w", err)
	}
	if len(ecnetControllerDeployments.Items) == 0 {
		return meshInfoList, nil
	}

	for _, ecnetControllerDeployment := range ecnetControllerDeployments.Items {
		meshName := ecnetControllerDeployment.ObjectMeta.Labels["meshName"]
		meshNamespace := ecnetControllerDeployment.ObjectMeta.Namespace

		meshVersion := ecnetControllerDeployment.ObjectMeta.Labels[constants.ECNETAppVersionLabelKey]
		if meshVersion == "" {
			meshVersion = "Unknown"
		}

		var meshMonitoredNamespaces []string
		nsList, err := selectNamespacesMonitoredByMesh(meshName, clientSet)
		if err == nil && len(nsList.Items) > 0 {
			for _, ns := range nsList.Items {
				meshMonitoredNamespaces = append(meshMonitoredNamespaces, ns.Name)
			}
		}

		meshInfoList = append(meshInfoList, meshInfo{
			name:                meshName,
			namespace:           meshNamespace,
			version:             meshVersion,
			monitoredNamespaces: meshMonitoredNamespaces,
		})
	}

	return meshInfoList, nil
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

// getPrettyPrintedMeshSmiInfoList returns a pretty printed list
// of meshes with supported smi versions
func getPrettyPrintedMeshSmiInfoList(meshSmiInfoList []meshSmiInfo) string {
	s := "\nMESH NAME\tMESH NAMESPACE\n"

	for _, mesh := range meshSmiInfoList {
		m := fmt.Sprintf(
			"%s\t%s\n",
			mesh.name,
			mesh.namespace,
		)
		s += m
	}

	return s
}

// getSupportedSmiInfoForMeshList returns a meshSmiInfo list showing
// the supported smi versions for each ecnet mesh in the mesh list
func getSupportedSmiInfoForMeshList(meshInfoList []meshInfo, clientSet kubernetes.Interface, config *rest.Config, localPort uint16) []meshSmiInfo {
	var meshSmiInfoList []meshSmiInfo

	for _, mesh := range meshInfoList {
		meshSmiInfoList = append(meshSmiInfoList, meshSmiInfo{
			name:      mesh.name,
			namespace: mesh.namespace,
		})
	}

	return meshSmiInfoList
}

func annotateErrorMessageWithEcnetNamespace(errMsgFormat string, args ...interface{}) error {
	ecnetNamespaceErrorMsg := fmt.Sprintf(
		"Note: The command failed when run in the ECNET namespace [%s].\n"+
			"Use the global flag --ecnet-namespace if [%s] is not the intended ECNET namespace.",
		settings.Namespace(), settings.Namespace())

	return annotateErrorMessageWithActionableMessage(ecnetNamespaceErrorMsg, errMsgFormat, args...)
}

func annotateErrorMessageWithActionableMessage(actionableMessage string, errMsgFormat string, args ...interface{}) error {
	if !strings.HasSuffix(errMsgFormat, "\n") {
		errMsgFormat += "\n"
	}

	if !strings.HasSuffix(errMsgFormat, "\n\n") {
		errMsgFormat += "\n"
	}

	return fmt.Errorf(errMsgFormat+actionableMessage, args...)
}
