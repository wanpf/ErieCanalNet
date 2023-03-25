// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package helpers implements ebpf helpers.
package helpers

import (
	"fmt"

	"github.com/cilium/ebpf"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
)

var (
	mcsDNSNatMap *ebpf.Map
	mcsSvcNatMap *ebpf.Map
)

// InitLoadPinnedMap init, load and pinned mapsß
func InitLoadPinnedMap() error {
	var err error
	mcsDNSNatMap, err = ebpf.LoadPinnedMap(config.ECNetDNSNatEbpfMap, &ebpf.LoadPinOptions{})
	if err != nil {
		return fmt.Errorf("load map[%s] error: %v", config.ECNetDNSNatEbpfMap, err)
	}
	mcsSvcNatMap, err = ebpf.LoadPinnedMap(config.ECNetSVCNatEbpfMap, &ebpf.LoadPinOptions{})
	if err != nil {
		return fmt.Errorf("load map[%s] error: %v", err, config.ECNetSVCNatEbpfMap)
	}
	return nil
}

// GetMcsDNSNatMap returns pod fib map
func GetMcsDNSNatMap() *ebpf.Map {
	if mcsDNSNatMap == nil {
		_ = InitLoadPinnedMap()
	}
	return mcsDNSNatMap
}

// GetMcsSvcNatMap returns nat fib map
func GetMcsSvcNatMap() *ebpf.Map {
	if mcsSvcNatMap == nil {
		_ = InitLoadPinnedMap()
	}
	return mcsSvcNatMap
}
