package podwatcher

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type watcher struct {
	Client kubernetes.Interface
	Stop   chan struct{}
}

func (w *watcher) start() error {
	kubeInformerFactory := kubeinformer.NewFilteredSharedInformerFactory(
		w.Client, 30*time.Second, metav1.NamespaceAll,
		func(o *metav1.ListOptions) {
		},
	)

	_, _ = kubeInformerFactory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{})
	kubeInformerFactory.Start(w.Stop)
	return nil
}

func (w *watcher) shutdown() {
	close(w.Stop)
}

func newWatcher(watch watcher) *watcher {
	return &watcher{
		Client: watch.Client,
		Stop:   make(chan struct{}),
	}
}
