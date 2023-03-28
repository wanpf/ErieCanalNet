// Package informers centralize informers by creating a single object that
// runs a set of informers, instead of creating different objects
// that each manage their own informer collections.
// A pointer to this object is then shared with all objects that need it.
package informers

import (
	"errors"
	"testing"

	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
	configClientset "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/client/config/clientset/versioned"
	configInformers "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/client/config/informers/externalversions"
	multiclusterClientset "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/client/multicluster/clientset/versioned"
	multiclusterInformers "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/client/multicluster/informers/externalversions"
)

// InformerCollectionOption is a function that modifies an informer collection
type InformerCollectionOption func(*InformerCollection)

// NewInformerCollection creates a new InformerCollection
func NewInformerCollection(ecnetName string, stop <-chan struct{}, opts ...InformerCollectionOption) (*InformerCollection, error) {
	ic := &InformerCollection{
		ecnetName: ecnetName,
		informers: map[InformerKey]cache.SharedIndexInformer{},
	}

	// Execute all of the given options (e.g. set clients, set custom stores, etc.)
	for _, opt := range opts {
		if opt != nil {
			opt(ic)
		}
	}

	if err := ic.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start informer collection")
		return nil, err
	}

	return ic, nil
}

// WithKubeClient sets the kubeClient for the InformerCollection
func WithKubeClient(kubeClient kubernetes.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		// initialize informers
		monitorNamespaceLabel := map[string]string{constants.ECNETKubeResourceMonitorAnnotation: ic.ecnetName}

		labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
		option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.LabelSelector = labelSelector
		})

		nsInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, DefaultKubeEventResyncInterval, option)
		informerFactory := informers.NewSharedInformerFactory(kubeClient, DefaultKubeEventResyncInterval)
		v1api := informerFactory.Core().V1()
		ic.informers[InformerKeyNamespace] = nsInformerFactory.Core().V1().Namespaces().Informer()
		ic.informers[InformerKeyService] = v1api.Services().Informer()
		ic.informers[InformerKeyServiceAccount] = v1api.ServiceAccounts().Informer()
		ic.informers[InformerKeyPod] = v1api.Pods().Informer()
		ic.informers[InformerKeyEndpoints] = v1api.Endpoints().Informer()
	}
}

// WithConfigClient sets the config client for the InformerCollection
func WithConfigClient(configClient configClientset.Interface, ecnetConfigName, ecnetNamespace string) InformerCollectionOption {
	return func(ic *InformerCollection) {
		listOption := configInformers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, ecnetConfigName).String()
		})
		ecnetConfiginformerFactory := configInformers.NewSharedInformerFactoryWithOptions(configClient, DefaultKubeEventResyncInterval, configInformers.WithNamespace(ecnetNamespace), listOption)
		ic.informers[InformerKeyEcnetConfig] = ecnetConfiginformerFactory.Config().V1alpha1().EcnetConfigs().Informer()
	}
}

// WithMultiClusterClient sets the multicluster client for the InformerCollection
func WithMultiClusterClient(multiclusterClient multiclusterClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := multiclusterInformers.NewSharedInformerFactory(multiclusterClient, DefaultKubeEventResyncInterval)
		ic.informers[InformerKeyServiceImport] = informerFactory.Flomesh().V1alpha1().ServiceImports().Informer()
		ic.informers[InformerKeyGlobalTrafficPolicy] = informerFactory.Flomesh().V1alpha1().GlobalTrafficPolicies().Informer()
	}
}

func (ic *InformerCollection) run(stop <-chan struct{}) error {
	log.Info().Msg("InformerCollection started")
	var hasSynced []cache.InformerSynced
	var names []string

	if ic.informers == nil {
		return errInitInformers
	}

	for name, informer := range ic.informers {
		if informer == nil {
			continue
		}

		go informer.Run(stop)
		names = append(names, string(name))
		log.Info().Msgf("Waiting for %s informer cache sync...", name)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	log.Info().Msgf("Caches for %v synced successfully", names)

	return nil
}

// Add is only exported for the sake of tests and requires a testing.T to ensure it's
// never used in production. This functionality was added for the express purpose of testing
// flexibility since alternatives can often lead to flaky tests and race conditions
// between the time an object is added to a fake clientset and when that object
// is actually added to the informer `cache.Store`
func (ic *InformerCollection) Add(key InformerKey, obj interface{}, t *testing.T) error {
	if t == nil {
		return errors.New("this method should only be used in tests")
	}

	i, ok := ic.informers[key]
	if !ok {
		t.Errorf("tried to add to nil store with key %s", key)
	}

	return i.GetStore().Add(obj)
}

// Update is only exported for the sake of tests and requires a testing.T to ensure it's
// never used in production. This functionality was added for the express purpose of testing
// flexibility since the alternatives can often lead to flaky tests and race conditions
// between the time an object is added to a fake clientset and when that object
// is actually added to the informer `cache.Store`
func (ic *InformerCollection) Update(key InformerKey, obj interface{}, t *testing.T) error {
	if t == nil {
		return errors.New("this method should only be used in tests")
	}

	i, ok := ic.informers[key]
	if !ok {
		t.Errorf("tried to update to nil store with key %s", key)
	}

	return i.GetStore().Update(obj)
}

// AddEventHandler adds an handler to the informer indexed by the given InformerKey
func (ic *InformerCollection) AddEventHandler(informerKey InformerKey, handler cache.ResourceEventHandler) {
	i, ok := ic.informers[informerKey]
	if !ok {
		log.Info().Msgf("attempted to add event handler for nil informer %s", informerKey)
		return
	}

	_, _ = i.AddEventHandler(handler)
}

// GetByKey retrieves an item (based on the given index) from the store of the informer indexed by the given InformerKey
func (ic *InformerCollection) GetByKey(informerKey InformerKey, objectKey string) (interface{}, bool, error) {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil, false, nil
	}

	return informer.GetStore().GetByKey(objectKey)
}

// List returns the contents of the store of the informer indexed by the given InformerKey
func (ic *InformerCollection) List(informerKey InformerKey) []interface{} {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil
	}

	return informer.GetStore().List()
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (ic InformerCollection) IsMonitoredNamespace(namespace string) bool {
	return true
}
