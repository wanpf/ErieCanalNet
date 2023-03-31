// Package main implements ecnet cni plugin.
package main

import (
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/plugin"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	ec "github.com/flomesh-io/ErieCanal/pkg/ecnet/version"
)

func init() {
	_ = logger.SetLogLevel("warn")
}

func main() {
	skel.PluginMain(plugin.CmdAdd, plugin.CmdCheck, plugin.CmdDelete, version.All,
		fmt.Sprintf("ErieCanal-Bridge-CNI %v", ec.Version))
}
