// Package cniserver implements ECNET CNI control server.
package cniserver

import "github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"

var (
	log = logger.New("bridge-ctrl-server")
)

// Server CNI Server.
type Server interface {
	Start() error
	Stop()
}
