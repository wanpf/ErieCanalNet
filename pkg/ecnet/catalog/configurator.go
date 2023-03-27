package catalog

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
)

// GetConfigurator converts private variable to public
func (mc *MeshCatalog) GetConfigurator() *configurator.Configurator {
	return &mc.configurator
}
