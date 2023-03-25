package registry

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy"
)

// ExplicitProxyServiceMapper is a custom ProxyServiceMapper implementation.
type ExplicitProxyServiceMapper func(*pipy.Proxy) ([]service.MeshService, error)

// ListProxyServices executes the given mapping.
func (e ExplicitProxyServiceMapper) ListProxyServices(p *pipy.Proxy) ([]service.MeshService, error) {
	return e(p)
}
