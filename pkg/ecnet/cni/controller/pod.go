// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
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

	log.Info("Pod watcher Ready")
	if err = helpers.AttachProgs(skip); err != nil {
		return fmt.Errorf("failed to attach ebpf programs: %v", err)
	}
	if config.EnableCNI {
		<-stop
	} else {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
		<-ch
	}
	w.shutdown()

	if err = helpers.UnLoadProgs(skip); err != nil {
		return fmt.Errorf("unload failed: %v", err)
	}
	log.Info("Pod watcher Down")
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

func isInjectedSidecar(pod *v1.Pod) bool {
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
	log.Debugf("got pod updated %s/%s", pod.Namespace, pod.Name)
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
		log.Debugf("got pod delete %s/%s", pod.Namespace, pod.Name)
	}
}
