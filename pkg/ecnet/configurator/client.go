package configurator

import (
	"fmt"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/announcements"
	configv1alpha1 "github.com/flomesh-io/ErieCanal/pkg/ecnet/apis/config/v1alpha1"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/errcode"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s/informers"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(informerCollection *informers.InformerCollection, ecnetNamespace, meshConfigName string, msgBroker *messaging.Broker) *Client {
	c := &Client{
		informers:      informerCollection,
		ecnetNamespace: ecnetNamespace,
		meshConfigName: meshConfigName,
	}

	// configure listener
	meshConfigEventTypes := k8s.EventTypes{
		Add:    announcements.MeshConfigAdded,
		Update: announcements.MeshConfigUpdated,
		Delete: announcements.MeshConfigDeleted,
	}

	informerCollection.AddEventHandler(informers.InformerKeyMeshConfig, k8s.GetEventHandlerFuncs(nil, meshConfigEventTypes, msgBroker))

	return c
}

func (c *Client) getMeshConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.ecnetNamespace, c.meshConfigName)
}

// Returns the current MeshConfig
func (c *Client) getMeshConfig() configv1alpha1.MeshConfig {
	var meshConfig configv1alpha1.MeshConfig

	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.informers.GetByKey(informers.InformerKeyMeshConfig, meshConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigFetchFromCache)).Msgf("Error getting MeshConfig from cache with key %s", meshConfigCacheKey)
		return meshConfig
	}

	if !exists {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", meshConfigCacheKey)
		return meshConfig
	}

	meshConfig = *item.(*configv1alpha1.MeshConfig)
	return meshConfig
}
