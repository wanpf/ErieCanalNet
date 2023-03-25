package catalog

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/identity"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/providers/fsm"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
)

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (mc *MeshCatalog) ListServiceIdentitiesForService(svc service.MeshService) []identity.ServiceIdentity {
	// Currently ECNET uses kubernetes service accounts as service identities
	var serviceIdentities []identity.ServiceIdentity
	for _, provider := range mc.serviceProviders {
		serviceIDs := provider.ListServiceIdentitiesForService(svc)
		serviceIdentities = append(serviceIdentities, serviceIDs...)
	}

	return serviceIdentities
}

// listMeshServices returns all services in the mesh
func (mc *MeshCatalog) listMeshServices() []service.MeshService {
	var services []service.MeshService
	var mcServices []service.MeshService
	var otherProviders []service.Provider

	for _, provider := range mc.serviceProviders {
		if provider.GetID() == fsm.ProviderName {
			svcs := provider.ListServices()
			services = append(services, svcs...)
			mcServices = append(mcServices, svcs...)
		} else {
			duplicate := provider
			otherProviders = append(otherProviders, duplicate)
		}
	}

	if len(mcServices) > 0 {
		for _, provider := range otherProviders {
			svcs := provider.ListServices()
			for _, svc := range svcs {
				for _, mcSvc := range mcServices {
					if svc.Name == mcSvc.Name && svc.Namespace == mcSvc.Namespace {
						services = append(services, svc)
						break
					}
				}
			}
		}
	}

	return services
}
