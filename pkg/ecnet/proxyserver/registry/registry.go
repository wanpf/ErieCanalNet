package registry

import (
	"sync"

	"github.com/google/uuid"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver"
)

// NewProxyRegistry initializes a new empty *ProxyRegistry.
func NewProxyRegistry(msgBroker *messaging.Broker) *ProxyRegistry {
	return &ProxyRegistry{
		msgBroker: msgBroker,
	}
}

// RegisterProxy registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy() *proxyserver.Proxy {
	lock.Lock()
	defer lock.Unlock()
	if pr.cacheProxy == nil {
		pr.cacheProxy = &proxyserver.Proxy{Mutex: new(sync.RWMutex)}
		pr.cacheProxy.UUID, _ = uuid.NewUUID()
		pr.cacheProxy.Quit = make(chan bool)
	}
	return pr.cacheProxy
}

// GetConnectedProxy loads a connected proxy from the registry.
func (pr *ProxyRegistry) GetConnectedProxy() *proxyserver.Proxy {
	lock.Lock()
	defer lock.Unlock()
	return pr.cacheProxy
}
