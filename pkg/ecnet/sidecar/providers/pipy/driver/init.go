// Package driver implements pipy sidecar driver.
package driver

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar"
)

const (
	driverName = `pipy`
)

func init() {
	sidecar.Register(driverName, new(PipySidecarDriver))
}
