package proxyserver

import (
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
)

// Proxy is a representation of an Sidecar proxy .
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {
	// UUID of the proxy
	uuid.UUID

	Mutex    *sync.RWMutex
	MeshConf *configurator.Configurator
	ETag     uint64
	Quit     chan bool
}

func (p *Proxy) String() string {
	return fmt.Sprintf("[ProxyUUID=%s]", p.UUID)
}

// GetName returns a unique name for this proxy.
func (p *Proxy) GetName() string {
	return p.UUID.String()
}
