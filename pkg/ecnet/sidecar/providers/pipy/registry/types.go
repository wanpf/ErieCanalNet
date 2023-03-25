// Package registry implements handler's methods.
package registry

import (
	"sync"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
)

var log = logger.New("proxy-registry")

var (
	lock             sync.Mutex
	connectedProxies sync.Map
)

// ProxyRegistry keeps track of Sidecar proxies as they connect and disconnect
// from the control plane.
type ProxyRegistry struct {
	msgBroker *messaging.Broker

	// Fire a inform to update proxies
	UpdateProxies func()
}
