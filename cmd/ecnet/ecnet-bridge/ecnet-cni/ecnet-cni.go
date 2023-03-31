/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
