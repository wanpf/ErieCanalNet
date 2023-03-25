package driver

import (
	"context"

	"github.com/pkg/errors"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/health"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/driver"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy/registry"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy/repo"
)

// PipySidecarDriver is the pipy sidecar driver
type PipySidecarDriver struct {
}

// Start is the implement for ControllerDriver.Start
func (sd PipySidecarDriver) Start(ctx context.Context) (health.Probes, error) {
	parentCtx := ctx.Value(&driver.ControllerCtxKey)
	if parentCtx == nil {
		return nil, errors.New("missing Controller Context")
	}
	ctrlCtx := parentCtx.(*driver.ControllerContext)
	cfg := ctrlCtx.Configurator
	k8sClient := ctrlCtx.MeshCatalog.GetKubeController()
	proxyServerPort := ctrlCtx.ProxyServerPort

	proxyRegistry := registry.NewProxyRegistry(ctrlCtx.MsgBroker)
	// Create and start the pipy repo http service
	repoServer := repo.NewRepoServer(ctrlCtx.MeshCatalog, proxyRegistry, ctrlCtx.EcnetNamespace, cfg, k8sClient, ctrlCtx.MsgBroker)

	return repoServer, repoServer.Start(proxyServerPort)
}
