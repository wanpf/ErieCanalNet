package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/identity"
)

// Proxy is an interface providing adaptiving proxies of multiple sidecars
type Proxy interface {
	GetUUID() uuid.UUID
	GetIdentity() identity.ServiceIdentity
	GetConnectedAt() time.Time
}
