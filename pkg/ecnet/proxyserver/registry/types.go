// Package registry implements handler's methods.
package registry

import (
	"sync"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver"
)

var (
	lock sync.Mutex
)

// ProxyRegistry keeps track of Sidecar proxies
// from the control plane.
type ProxyRegistry struct {
	msgBroker  *messaging.Broker
	cacheProxy *proxyserver.Proxy

	// Fire a inform to update proxies
	UpdateProxies func()
}
