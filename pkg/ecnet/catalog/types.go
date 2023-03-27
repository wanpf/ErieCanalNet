// Package catalog implements the MeshCataloger interface, which forms the central component in ECNET that transforms
// outputs from all other components (SMI policies, Kubernetes services, endpoints etc.) into configuration that is
// consumed by the the proxy control plane component to program sidecar proxies.
// Reference: https://github.com/flomesh-io/ErieCanal/blob/main/DESIGN.md#5-mesh-catalog
package catalog

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/endpoint"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/multicluster"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/policy"
)

var (
	log = logger.New("mesh-catalog")
)

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	endpointsProviders []endpoint.Provider
	serviceProviders   []service.Provider
	configurator       configurator.Configurator

	// This is the kubernetes client that operates async caches to avoid issuing synchronous
	// calls through kubeClient and instead relies on background cache synchronization and local
	// lookups
	kubeController k8s.Controller

	// multiclusterController implements the functionality related to the resources part of the flomesh.io
	// API group, such a serviceimport.
	multiclusterController multicluster.Controller
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all sidecar proxies connected to the catalog.
type MeshCataloger interface {
	// ListOutboundServices list the services the given service identity is allowed to initiate outbound connections to
	ListOutboundServices() []service.MeshService

	// ListUpstreamEndpointsForService returns the list of endpoints over which the downstream client identity
	// is allowed access the upstream service
	ListUpstreamEndpointsForService(service.MeshService) []endpoint.Endpoint

	// GetKubeController returns the kube controller instance handling the current cluster
	GetKubeController() k8s.Controller

	// GetOutboundMeshTrafficPolicy returns the outbound mesh traffic policy for the given downstream identity
	GetOutboundMeshTrafficPolicy() *policy.OutboundMeshTrafficPolicy
}
