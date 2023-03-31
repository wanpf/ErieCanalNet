package controller

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/kube"
)

var (
	disableWatch = false
)

// Run start to run controller to watch
func Run(disableWatcher, skip bool, cniReady chan struct{}, stop chan struct{}) error {
	var err error
	var client kubernetes.Interface

	// get default kubernetes client
	client, err = kube.GetKubernetesClientWithFile(config.KubeConfig, config.Context)
	if err != nil {
		return fmt.Errorf("create client error: %v", err)
	}

	disableWatch = disableWatcher

	// run local ip controller
	if err = runLocalPodController(skip, client, stop); err != nil {
		return fmt.Errorf("run local ip controller error: %v", err)
	}

	return nil
}
