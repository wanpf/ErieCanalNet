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
	"os"
	"os/exec"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// LoadProgs load ebpf progs
func LoadProgs(debug bool) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("root user in required for this process or container")
	}
	cmd := exec.Command("make", "load")
	cmd.Env = os.Environ()
	if debug {
		cmd.Env = append(cmd.Env, "DEBUG=1")
	}

	if _, bridgeIP := GetBridgeIP(); bridgeIP > 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("BRIDGE_IP=%d", bridgeIP))
	} else {
		return fmt.Errorf("unexpected exit err: retrieves cni bridge veth's ipv4 addr")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if code := cmd.ProcessState.ExitCode(); code != 0 || err != nil {
		return fmt.Errorf("unexpected exit code: %d, err: %v", code, err)
	}
	return nil
}

// AttachProgs attach ebpf progs
func AttachProgs(skip bool) error {
	if skip {
		return nil
	}
	if os.Getuid() != 0 {
		return fmt.Errorf("root user in required for this process or container")
	}
	cmd := exec.Command("make", "attach")
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if code := cmd.ProcessState.ExitCode(); code != 0 || err != nil {
		return fmt.Errorf("unexpected exit code: %d, err: %v", code, err)
	}
	return nil
}

// UnLoadProgs unload ebpf progs
func UnLoadProgs(skip bool) error {
	if skip {
		return nil
	}
	cmd := exec.Command("make", "-k", "clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if code := cmd.ProcessState.ExitCode(); code != 0 || err != nil {
		return fmt.Errorf("unload unexpected exit code: %d, err: %v", code, err)
	}
	return nil
}

var (
	ingress *ebpf.Program
	egress  *ebpf.Program
)

// GetTrafficControlIngressProg returns tc ingress ebpf prog
func GetTrafficControlIngressProg() *ebpf.Program {
	if ingress == nil {
		err := initTrafficControlProgs()
		if err != nil {
			log.Errorf("init tc prog filed: %v", err)
		}
	}
	return ingress
}

// GetTrafficControlEgressProg returns tc egress ebpf prog
func GetTrafficControlEgressProg() *ebpf.Program {
	if egress == nil {
		err := initTrafficControlProgs()
		if err != nil {
			log.Errorf("init tc prog filed: %v", err)
		}
	}
	return egress
}

func initTrafficControlProgs() error {
	coll, err := ebpf.LoadCollectionSpec("bpf/ecnet_cni_tc.o")
	if err != nil {
		return err
	}
	type progs struct {
		Ingress *ebpf.Program `ebpf:"ecnet_cni_tc_ingress"`
		Egress  *ebpf.Program `ebpf:"ecnet_cni_tc_egress"`
	}
	ps := progs{}
	err = coll.LoadAndAssign(&ps, &ebpf.CollectionOptions{
		MapReplacements: map[string]*ebpf.Map{
			"ecnet_dns_nat": GetMcsDNSNatMap(),
			"ecnet_svc_nat": GetMcsSvcNatMap(),
		},
	})
	if err != nil {
		return err
	}
	ingress = ps.Ingress
	egress = ps.Egress
	return nil
}
