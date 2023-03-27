package catalog

import (
	"time"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/endpoint"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/multicluster"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/ticker"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(kubeController k8s.Controller,
	multiclusterController multicluster.Controller,
	stop <-chan struct{},
	cfg configurator.Configurator,
	serviceProviders []service.Provider,
	endpointsProviders []endpoint.Provider,
	msgBroker *messaging.Broker) *MeshCatalog {
	mc := &MeshCatalog{
		serviceProviders:       serviceProviders,
		endpointsProviders:     endpointsProviders,
		multiclusterController: multiclusterController,
		configurator:           cfg,
		kubeController:         kubeController,
	}

	// Start the Resync ticker to tick based on the resync interval.
	// Starting the resync ticker only starts the ticker config watcher which
	// internally manages the lifecycle of the ticker routine.
	resyncTicker := ticker.NewResyncTicker(msgBroker, 30*time.Second /* min resync interval */)
	resyncTicker.Start(stop, cfg.GetConfigResyncInterval())

	return mc
}

// GetKubeController returns the kube controller instance handling the current cluster
func (mc *MeshCatalog) GetKubeController() k8s.Controller {
	return mc.kubeController
}

// GetTrustDomain returns the currently configured trust domain, ie: cluster.local
func (mc *MeshCatalog) GetTrustDomain() string {
	// TODO benne
	return "cluster.local"
}
