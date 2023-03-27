package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller/helpers"
	pb "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/proxy"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/conf"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/repo/client"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/repo/codebase"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

var (
	ecnetCodebase = "BridgeProxy"
)

// DiscoveryServer implements the Config Discovery Server
type DiscoveryServer struct {
	ready        bool
	stop         chan struct{}
	tickInterval time.Duration
	etag         uint64
	repoClient   *client.PipyRepoClient
}

// NewDiscoveryServer creates a Config Discovery Server
func NewDiscoveryServer() *DiscoveryServer {
	server := DiscoveryServer{
		tickInterval: time.Second * 5,
		repoClient:   client.NewRepoClient("127.0.0.1", 6060),
	}
	return &server
}

// Start runs the ticker routine and ticks periodically at the given interval.
// It stops when 'stopTicker()' is invoked.
func (s *DiscoveryServer) Start() error {
	// wait until pipy repo is up
	err := wait.PollImmediate(5*time.Second, 90*time.Second, func() (bool, error) {
		success, err := s.repoClient.IsRepoUp()
		if success {
			log.Info().Msg("Repo is READY!")
			return success, nil
		}
		log.Error().Msg("Repo is not up, sleeping ...")
		return success, err
	})
	if err != nil {
		log.Error().Err(err)
		return err
	}

	proxyCfg := conf.ProxyConf{}
	ts := time.Now()
	proxyCfg.Ts = &ts
	bridgeIP, _, err := helpers.GetBridgeIP()
	if err != nil {
		log.Error().Msg(err.Error())
		return err
	}
	proxyCfg.BridgeV4Addr = bridgeIP.String()
	bytes, _ := json.MarshalIndent(proxyCfg, "", " ")
	codebaseCurV := util.Hash(bytes)
	version := fmt.Sprintf("%d", codebaseCurV)
	proxyCfg.Version = &version
	codebase.EcnetCodebaseItems = append(codebase.EcnetCodebaseItems, client.BatchItem{
		Filename: codebase.EcnetCodebaseConfig, Content: bytes,
	})

	_, err = s.repoClient.Batch(fmt.Sprintf("%d", 0), []client.Batch{
		{
			Basepath: ecnetCodebase,
			Items:    codebase.EcnetCodebaseItems,
		},
	})
	if err != nil {
		log.Error().Err(err)
		return err
	}
	s.etag = codebaseCurV

	// wait until base codebase is ready
	err = wait.PollImmediate(2*time.Second, 60*time.Second, func() (bool, error) {
		success, _, _ := s.repoClient.GetCodebase(ecnetCodebase)
		if success {
			log.Info().Msg("Codebase is READY!")
			return success, nil
		}
		log.Error().Msg("Codebase is NOT READY, sleeping ...")
		return success, err
	})
	if err != nil {
		log.Error().Err(err)
		return err
	}

	err = s.repoClient.RunCodebase(ecnetCodebase)
	if err != nil {
		log.Error().Err(err)
		return err
	}

	go func() {
		ticker := time.NewTicker(s.tickInterval)
		for {
			select {
			case <-s.stop:
				log.Info().Msgf("Received signal to stop ticker, exiting ticker routine")
				return

			case <-ticker.C:
				s.syncConfigJSON()
			}
		}
	}()

	s.ready = true

	return nil
}

func (s *DiscoveryServer) syncConfigJSON() {
	conn, err := grpc.Dial("ecnet-controller.ecnet-system.svc.cluster.local:6060", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error().Msgf("did not connect: %v", err)
		return
	}
	defer conn.Close()
	c := pb.NewConfigClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	bridgeIP, _, err := helpers.GetBridgeIP()
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	if len(bridgeIP) > 0 {
		if r, pollErr := c.Poll(ctx, &pb.ConfigRequest{Addr: bridgeIP.String()}); pollErr == nil {
			bytes := []byte(r.GetJson())
			if len(bytes) == 0 || bytes[0] != '{' || bytes[len(bytes)-1] != '}' {
				return
			}
			proxyCfg := conf.ProxyConf{}
			err = json.Unmarshal(bytes, &proxyCfg)
			if err != nil {
				log.Error().Msg(err.Error())
				return
			}

			codebaseCurV := util.Hash(bytes)
			codebasePreV := s.etag
			if codebaseCurV != codebasePreV {
				log.Log().Str("codebasePreV", fmt.Sprintf("%d", codebasePreV)).
					Str("codebaseCurV", fmt.Sprintf("%d", codebaseCurV)).
					Msg("config.json")

				ts := time.Now()
				proxyCfg.Ts = &ts
				version := fmt.Sprintf("%d", codebaseCurV)
				proxyCfg.Version = &version
				proxyCfg.BridgeV4Addr = bridgeIP.String()
				if len(proxyCfg.DNSResolveDB) > 0 {
					for k := range proxyCfg.DNSResolveDB {
						proxyCfg.DNSResolveDB[k] = []string{proxyCfg.BridgeV4Addr}
					}
				}
				bytes, _ = json.MarshalIndent(proxyCfg, "", " ")

				_, repoErr := s.repoClient.Batch(fmt.Sprintf("%d", codebaseCurV-1), []client.Batch{
					{
						Basepath: ecnetCodebase,
						Items: []client.BatchItem{
							{
								Filename: codebase.EcnetCodebaseConfig,
								Content:  bytes,
							},
						},
					},
				})
				if repoErr != nil {
					log.Error().Err(err)
				} else {
					s.etag = codebaseCurV
				}
			}
		} else {
			log.Error().Msgf("could not Poll: %v", pollErr)
		}
	}
}
