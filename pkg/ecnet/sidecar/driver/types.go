package driver

import (
	"context"
	"net/http"

	"k8s.io/client-go/rest"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/catalog"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/health"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
)

// Driver is an interface that must be implemented by a sidecar driver.
// Patch method is invoked by ecnet-injector and Start method is invoked by ecnet-controller
type Driver interface {
	Start(ctx context.Context) (health.Probes, error)
}

// ControllerCtxKey the pointer is the key that a ControllerContext returns itself for.
var ControllerCtxKey int

// ControllerContext carries the arguments for invoking ControllerDriver.Start
type ControllerContext struct {
	context.Context

	ProxyServerPort uint32
	EcnetNamespace  string
	KubeConfig      *rest.Config
	Configurator    configurator.Configurator
	MeshCatalog     catalog.MeshCataloger
	MsgBroker       *messaging.Broker
	DebugHandlers   map[string]http.Handler
	CancelFunc      func()
	Stop            chan struct {
	}
}
