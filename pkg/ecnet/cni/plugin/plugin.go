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

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	cniv1 "github.com/containernetworking/cni/pkg/types/100"
	log "github.com/sirupsen/logrus"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
)

// K8sArgs is the valid CNI_ARGS used for Kubernetes
// The field names need to match exact keys in kubelet args for unmarshalling
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString // nolint: revive, stylecheck
	K8S_POD_NAMESPACE          types.UnmarshallableString // nolint: revive, stylecheck
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString // nolint: revive, stylecheck
}

func ignore(_ *Config, _ *K8sArgs) bool {
	return false
}

// CmdAdd is the implementation of the cmdAdd interface of CNI plugin
func CmdAdd(args *skel.CmdArgs) (err error) {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		log.Errorf("ecnet-cni cmdAdd failed to parse config %v %v", string(args.StdinData), err)
		return err
	}
	k8sArgs := K8sArgs{}
	if err := types.LoadArgs(args.Args, &k8sArgs); err != nil {
		return err
	}

	if !ignore(conf, &k8sArgs) {
		httpc := http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", "/var/run/ecnet-cni.sock")
				},
			},
		}
		bs, _ := json.Marshal(args)
		body := bytes.NewReader(bs)
		_, err = httpc.Post("http://ecnet-cni"+config.CNICreatePodURL, "application/json", body)
		if err != nil {
			return err
		}
	}

	var result *cniv1.Result
	if conf.PrevResult == nil {
		result = &cniv1.Result{
			CNIVersion: cniv1.ImplementedSpecVersion,
		}
	} else {
		// Pass through the result for the next plugin
		result = conf.PrevResult
	}
	return types.PrintResult(result, conf.CNIVersion)
}

// CmdCheck is the implementation of the cmdCheck interface of CNI plugin
func CmdCheck(*skel.CmdArgs) (err error) {
	return err
}

// CmdDelete is the implementation of the cmdDelete interface of CNI plugin
func CmdDelete(args *skel.CmdArgs) (err error) {
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/ecnet-cni.sock")
			},
		},
	}
	bs, _ := json.Marshal(args)
	body := bytes.NewReader(bs)
	_, err = httpc.Post("http://ecnet-cni"+config.CNIDeletePodURL, "application/json", body)
	return err
}
