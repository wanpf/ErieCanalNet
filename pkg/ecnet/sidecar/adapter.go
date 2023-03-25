// Package sidecar implements adapter's methods.
package sidecar

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/health"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/models"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/driver"
)

var (
	driversMutex sync.RWMutex
	drivers      = make(map[string]driver.Driver)
	engineDriver driver.Driver
)

// GetCertCNPrefix returns a newly generated CommonName : proxy.<identity>
// where identity itself is of the form <name>.<namespace>
func GetCertCNPrefix(proxy models.Proxy) string {
	return fmt.Sprintf("proxy.%s", proxy.GetIdentity())
}

// InstallDriver is to serve as an indication of the using sidecar driver
func InstallDriver(driverName string) error {
	driversMutex.Lock()
	defer driversMutex.Unlock()
	registeredDriver, ok := drivers[driverName]
	if !ok {
		return fmt.Errorf("sidecar: unknown driver %q (forgot to import?)", driverName)
	}
	engineDriver = registeredDriver
	return nil
}

// Register makes a sidecar driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.Driver) {
	driversMutex.Lock()
	defer driversMutex.Unlock()
	if driver == nil {
		panic("sidecar: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("sidecar: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Start is an adapter method for ControllerDriver.Start
func Start(ctx context.Context) (health.Probes, error) {
	driversMutex.RLock()
	defer driversMutex.RUnlock()
	if engineDriver == nil {
		return nil, errors.New("sidecar: unknown driver (forgot to init?)")
	}
	return engineDriver.Start(ctx)
}
