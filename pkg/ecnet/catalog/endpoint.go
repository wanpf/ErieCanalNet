package catalog

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/endpoint"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/identity"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
)

// ListEndpointsForService returns the list of provider endpoints corresponding to a service
func (mc *MeshCatalog) listEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep := provider.ListEndpointsForService(svc)
		if len(ep) == 0 {
			log.Trace().Msgf("No endpoints found for service %s by endpoints provider %s", provider.GetID(), svc)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints
}

// getDNSResolvableServiceEndpoints returns the resolvable set of endpoint over which a service is accessible using its FQDN
func (mc *MeshCatalog) getDNSResolvableServiceEndpoints(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep := provider.GetResolvableEndpointsForService(svc)
		endpoints = append(endpoints, ep...)
	}
	return endpoints
}

// ListUpstreamEndpointsForService returns the list of endpoints over which the downstream client identity
// is allowed access the upstream service
func (mc *MeshCatalog) ListUpstreamEndpointsForService(upstreamSvc service.MeshService) []endpoint.Endpoint {
	outboundEndpoints := mc.listEndpointsForService(upstreamSvc)
	if len(outboundEndpoints) == 0 {
		return nil
	}
	return outboundEndpoints
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/flomesh-io/ErieCanal/issues/3188]
func (mc *MeshCatalog) listEndpointsForServiceIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep := provider.ListEndpointsForIdentity(serviceIdentity)
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service account=%s", provider.GetID(), serviceIdentity)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints
}
