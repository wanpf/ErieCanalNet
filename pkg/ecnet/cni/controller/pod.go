package controller

import (
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller/helpers"
)

func runLocalPodController(skip bool, client kubernetes.Interface, stop chan struct{}) error {
	var err error

	if err = helpers.InitLoadPinnedMap(); err != nil {
		return fmt.Errorf("failed to load ebpf maps: %v", err)
	}

	w := newWatcher(createLocalPodController(client))

	if err = w.start(); err != nil {
		return fmt.Errorf("start watcher failed: %v", err)
	}

	log.Info().Msg("Pod watcher Ready")
	if err = helpers.AttachProgs(skip); err != nil {
		return fmt.Errorf("failed to attach ebpf programs: %v", err)
	}
	<-stop
	w.shutdown()

	if err = helpers.UnLoadProgs(skip); err != nil {
		return fmt.Errorf("unload failed: %v", err)
	}
	log.Info().Msg("Pod watcher Down")
	return nil
}

func createLocalPodController(client kubernetes.Interface) watcher {
	localName, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return watcher{
		Client:          client,
		CurrentNodeName: localName,
		OnAddFunc:       addFunc,
		OnUpdateFunc:    updateFunc,
		OnDeleteFunc:    deleteFunc,
	}
}

func isInjectedSidecar(_ *v1.Pod) bool {
	return true
}

func addFunc(obj interface{}) {
	if disableWatch {
		return
	}
	pod, ok := obj.(*v1.Pod)
	if !ok || len(pod.Status.PodIP) == 0 {
		return
	}
	if !isInjectedSidecar(pod) {
		return
	}
	log.Debug().Msgf("got pod updated %s/%s", pod.Namespace, pod.Name)
}

func updateFunc(old, cur interface{}) {
	if disableWatch {
		return
	}
	oldPod, ok := old.(*v1.Pod)
	if !ok {
		return
	}
	curPod, ok := cur.(*v1.Pod)
	if !ok {
		return
	}
	if oldPod.Status.PodIP != curPod.Status.PodIP {
		// only care about ip changes
		addFunc(cur)
	}
}

func deleteFunc(obj interface{}) {
	if disableWatch {
		return
	}
	if pod, ok := obj.(*v1.Pod); ok {
		log.Debug().Msgf("got pod delete %s/%s", pod.Namespace, pod.Name)
	}
}
