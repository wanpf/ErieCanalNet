package podwatcher

import (
	"fmt"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller/helpers"
)

func runLocalPodController(client kubernetes.Interface, stop chan struct{}) error {
	var err error

	w := newWatcher(createLocalPodController(client))

	if err = w.start(); err != nil {
		return fmt.Errorf("start watcher failed: %v", err)
	}

	log.Info().Msg("Pod watcher Ready")
	if err = helpers.AttachProgs(); err != nil {
		return fmt.Errorf("failed to attach ebpf programs: %v", err)
	}
	<-stop
	w.shutdown()

	if err = helpers.UnLoadProgs(); err != nil {
		return fmt.Errorf("unload failed: %v", err)
	}
	log.Info().Msg("Pod watcher Down")
	return nil
}

func createLocalPodController(client kubernetes.Interface) watcher {
	return watcher{
		Client: client,
	}
}
