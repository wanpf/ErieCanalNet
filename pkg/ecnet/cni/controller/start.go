package controller

import (
	"fmt"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/kube"
)

// Run start to run controller to watch
func Run(cniReady chan struct{}, stop chan struct{}) error {
	// get default kubernetes client
	client, err := kube.GetKubernetesClientWithFile(config.KubeConfig, config.Context)
	if err != nil {
		return fmt.Errorf("create client error: %v", err)
	}

	// run local ip controller
	if err = runLocalPodController(client, stop); err != nil {
		return fmt.Errorf("run local ip controller error: %v", err)
	}

	return nil
}
