// Package plugin implements ecnet cni plugin.
package plugin

import (
	"os"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
)

var (
	log = logger.New("ecnet-cni-plugin")
)

func init() {
	if logfile, err := os.OpenFile("/tmp/ecnet-cni.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
		log = log.Output(logfile)
	}
}
