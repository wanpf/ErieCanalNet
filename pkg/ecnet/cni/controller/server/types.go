// Package server implements ECNET CNI control server.
package server

import "github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"

var (
	log = logger.New("bridge-helpers")
)

// Server CNI Server.
type Server interface {
	Start() error
	Stop()
}
