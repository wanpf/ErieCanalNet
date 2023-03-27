package server

import (
	"encoding/json"
	"fmt"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/util"
	"sync"
	"time"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/catalog"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/repo/client"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/repo/codebase"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver"
)

// PipyConfGeneratorJob is the job to generate pipy policy json
type PipyConfGeneratorJob struct {
	proxy      *proxyserver.Proxy
	repoServer *Server

	// Optional waiter
	done chan struct{}
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *PipyConfGeneratorJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *PipyConfGeneratorJob) Run() {
	defer close(job.done)
	if job.proxy == nil {
		return
	}

	s := job.repoServer
	proxy := job.proxy

	proxy.Mutex.Lock()
	defer proxy.Mutex.Unlock()

	cataloger := s.catalog
	pipyConf := new(PipyConf)

	features(s, proxy, pipyConf)
	pluginSetV := plugin(s, pipyConf)
	outbound(cataloger, s, pipyConf, proxy)
	balance(pipyConf)
	reorder(pipyConf)
	job.publishSidecarConf(s.repoClient, proxy, pipyConf, pluginSetV)
}

func balance(pipyConf *PipyConf) {
	pipyConf.rebalancedOutboundClusters()
}

func reorder(pipyConf *PipyConf) {
	if pipyConf.Outbound != nil && pipyConf.Outbound.TrafficMatches != nil {
		for _, trafficMatches := range pipyConf.Outbound.TrafficMatches {
			for _, trafficMatch := range trafficMatches {
				for _, routeRules := range trafficMatch.HTTPServiceRouteRules {
					routeRules.RouteRules.sort()
				}
			}
		}
	}
}

func outbound(cataloger catalog.MeshCataloger, s *Server, pipyConf *PipyConf, proxy *proxyserver.Proxy) bool {
	outboundTrafficPolicy := cataloger.GetOutboundMeshTrafficPolicy()
	if len(outboundTrafficPolicy.ServicesResolvableSet) > 0 {
		pipyConf.DNSResolveDB = outboundTrafficPolicy.ServicesResolvableSet
		//pipyConf.DNSResolveDB = make(map[string][]string)
		//for k := range outboundTrafficPolicy.ServicesResolvableSet {
		//	pipyConf.DNSResolveDB[k] = []string{"10.244.2.1"}
		//}
	}
	outboundDependClusters := generatePipyOutboundTrafficRoutePolicy(pipyConf, outboundTrafficPolicy)
	if len(outboundDependClusters) > 0 {
		if ready := generatePipyOutboundTrafficBalancePolicy(cataloger, pipyConf, outboundTrafficPolicy, outboundDependClusters); !ready {
			if s.retryProxiesJob != nil {
				s.retryProxiesJob()
			}
			return false
		}
	}
	return true
}

func plugin(s *Server, pipyConf *PipyConf) (pluginSetVersion string) {
	pipyConf.Chains = nil
	setSidecarChain(s.cfg, pipyConf)
	return
}

func features(s *Server, proxy *proxyserver.Proxy, pipyConf *PipyConf) {
	if mc, ok := s.catalog.(*catalog.MeshCatalog); ok {
		meshConf := mc.GetConfigurator()
		proxy.MeshConf = meshConf
		pipyConf.setSidecarLogLevel((*meshConf).GetMeshConfig().Spec.Sidecar.LogLevel)
		pipyConf.setLocalDNSProxy((*meshConf).LocalDNSProxyEnabled(), (*meshConf).GetLocalDNSProxyPrimaryUpstream(), (*meshConf).GetLocalDNSProxySecondaryUpstream())
	}
}

var (
	repoLock     sync.RWMutex
	latestConfig = PipyConf{}
)

func (job *PipyConfGeneratorJob) publishSidecarConf(repoClient *client.PipyRepoClient, proxy *proxyserver.Proxy, pipyConf *PipyConf, pluginSetV string) {
	repoLock.Lock()
	defer func() {
		repoLock.Unlock()
	}()
	pipyConf.Ts = nil
	pipyConf.Version = nil
	bytes, jsonErr := json.Marshal(pipyConf)

	if jsonErr == nil {
		codebasePreV := proxy.ETag
		bytes = append(bytes, []byte(pluginSetV)...)
		codebaseCurV := util.Hash(bytes)
		if codebaseCurV != codebasePreV {
			log.Log().Str("uid", proxy.UUID.String()).
				Str("codebasePreV", fmt.Sprintf("%d", codebasePreV)).
				Str("codebaseCurV", fmt.Sprintf("%d", codebaseCurV)).
				Msg("config.json")
			proxyCodebase := fmt.Sprintf("%s/proxy.bridge.ecnet", ecnetProxyCodebase)
			success, err := repoClient.DeriveCodebase(proxyCodebase, ecnetCodebaseRepo, codebaseCurV-2)
			if success {
				ts := time.Now()
				pipyConf.Ts = &ts
				version := fmt.Sprintf("%d", codebaseCurV)
				pipyConf.Version = &version
				bytes, _ = json.MarshalIndent(pipyConf, "", " ")
				_, err = repoClient.Batch(fmt.Sprintf("%d", codebaseCurV-1), []client.Batch{
					{
						Basepath: proxyCodebase,
						Items: []client.BatchItem{
							{
								Filename: codebase.EcnetCodebaseConfig,
								Content:  bytes,
							},
						},
					},
				})
			}
			if err != nil {
				log.Error().Err(err)
				_, _ = repoClient.Delete(proxyCodebase)
			} else {
				proxy.ETag = codebaseCurV
			}
		}
	}
}

// JobName implementation for this job, for logging purposes
func (job *PipyConfGeneratorJob) JobName() string {
	return fmt.Sprintf("pipyJob-%s", job.proxy.GetName())
}
