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
func NewConfigurator(informerCollection *informers.InformerCollection, ecnetNamespace, ecnetConfigName string, msgBroker *messaging.Broker) *Client {
	c := &Client{
		informers:       informerCollection,
		ecnetNamespace:  ecnetNamespace,
		ecnetConfigName: ecnetConfigName,
	}

	// configure listener
	ecnetConfigEventTypes := k8s.EventTypes{
		Add:    announcements.EcnetConfigAdded,
		Update: announcements.EcnetConfigUpdated,
		Delete: announcements.EcnetConfigDeleted,
	}

	informerCollection.AddEventHandler(informers.InformerKeyEcnetConfig, k8s.GetEventHandlerFuncs(nil, ecnetConfigEventTypes, msgBroker))

	return c
}

func (c *Client) getEcnetConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.ecnetNamespace, c.ecnetConfigName)
}

// Returns the current EcnetConfig
func (c *Client) getEcnetConfig() configv1alpha1.EcnetConfig {
	var ecnetConfig configv1alpha1.EcnetConfig

	ecnetConfigCacheKey := c.getEcnetConfigCacheKey()
	item, exists, err := c.informers.GetByKey(informers.InformerKeyEcnetConfig, ecnetConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEcnetConfigFetchFromCache)).Msgf("Error getting EcnetConfig from cache with key %s", ecnetConfigCacheKey)
		return ecnetConfig
	}

	if !exists {
		log.Warn().Msgf("EcnetConfig %s does not exist. Default config values will be used.", ecnetConfigCacheKey)
		return ecnetConfig
	}

	ecnetConfig = *item.(*configv1alpha1.EcnetConfig)
	return ecnetConfig
}
