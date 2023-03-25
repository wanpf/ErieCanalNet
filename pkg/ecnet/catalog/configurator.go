package catalog

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/endpoint"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/identity"
)

// GetConfigurator converts private variable to public
func (mc *MeshCatalog) GetConfigurator() *configurator.Configurator {
	return &mc.configurator
}

// ListEndpointsForServiceIdentity converts private method to public
func (mc *MeshCatalog) ListEndpointsForServiceIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	return mc.listEndpointsForServiceIdentity(serviceIdentity)
}
