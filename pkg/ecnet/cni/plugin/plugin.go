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

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/util"
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
func CmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		log.Error().Msgf("ecnet-cni cmdAdd failed to parse config %v %v", string(args.StdinData), err)
	} else {
		k8sArgs := K8sArgs{}
		if err = types.LoadArgs(args.Args, &k8sArgs); err != nil {
			log.Error().Msgf("ecnet-cni cmdAdd failed to load args %v %v", string(args.StdinData), err)
		} else {
			if !ignore(conf, &k8sArgs) {
				if util.Exists(config.CNISock) {
					httpc := http.Client{
						Transport: &http.Transport{
							DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
								return net.Dial("unix", config.CNISock)
							},
						},
					}
					bs, _ := json.Marshal(args)
					body := bytes.NewReader(bs)
					if _, err = httpc.Post("http://ecnet-cni"+config.CNICreatePodURL, "application/json", body); err != nil {
						log.Error().Msgf("ecnet-cni cmdAdd failed to post args %v %v", string(args.StdinData), err)
					}
				}
			}
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
func CmdCheck(*skel.CmdArgs) error {
	return nil
}

// CmdDelete is the implementation of the cmdDelete interface of CNI plugin
func CmdDelete(args *skel.CmdArgs) error {
	if !util.Exists(config.CNISock) {
		return nil
	}

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", config.CNISock)
			},
		},
	}
	bs, _ := json.Marshal(args)
	body := bytes.NewReader(bs)
	_, err := httpc.Post("http://ecnet-cni"+config.CNIDeletePodURL, "application/json", body)
	log.Error().Msgf("ecnet-cni cmdDelete failed to parse config %v %v", string(args.StdinData), err)
	return nil
}
