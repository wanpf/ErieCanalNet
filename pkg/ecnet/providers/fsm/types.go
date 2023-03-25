package fsm

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/multicluster"
)

const (
	// ProviderName is the name of the Flomesh client that implements service.Provider and endpoint.Provider interfaces
	ProviderName = "flomesh"
)

// client is the type used to represent the k8s client for endpoints and service provider
type client struct {
	multiclusterController multicluster.Controller
	meshConfigurator       configurator.Configurator
}
