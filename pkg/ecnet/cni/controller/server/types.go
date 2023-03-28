// Package server implements ECNET CNI Controller.
package server

// Server CNI Server.
type Server interface {
	Start() error
	Stop()
}
