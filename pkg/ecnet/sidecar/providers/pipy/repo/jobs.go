package repo

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/catalog"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/identity"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy/client"
)

// PipyConfGeneratorJob is the job to generate pipy policy json
type PipyConfGeneratorJob struct {
	proxy      *pipy.Proxy
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

	probes(proxy, pipyConf)
	features(s, proxy, pipyConf)
	pluginSetV := plugin(s, pipyConf)
	outbound(cataloger, proxy.Identity, s, pipyConf, proxy)
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

func outbound(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxy *pipy.Proxy) bool {
	outboundTrafficPolicy := cataloger.GetOutboundMeshTrafficPolicy(serviceIdentity)
	if len(outboundTrafficPolicy.ServicesResolvableSet) > 0 {
		pipyConf.DNSResolveDB = make(map[string][]string)
		for k := range outboundTrafficPolicy.ServicesResolvableSet {
			pipyConf.DNSResolveDB[k] = []string{"10.244.2.1"}
		}
	}
	outboundDependClusters := generatePipyOutboundTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf,
		outboundTrafficPolicy)
	if len(outboundDependClusters) > 0 {
		if ready := generatePipyOutboundTrafficBalancePolicy(cataloger, proxy, serviceIdentity, pipyConf,
			outboundTrafficPolicy, outboundDependClusters); !ready {
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

func features(s *Server, proxy *pipy.Proxy, pipyConf *PipyConf) {
	if mc, ok := s.catalog.(*catalog.MeshCatalog); ok {
		meshConf := mc.GetConfigurator()
		proxy.MeshConf = meshConf
		pipyConf.setSidecarLogLevel((*meshConf).GetMeshConfig().Spec.Sidecar.LogLevel)
		pipyConf.setLocalDNSProxy((*meshConf).LocalDNSProxyEnabled(), (*meshConf).GetLocalDNSProxyPrimaryUpstream(), (*meshConf).GetLocalDNSProxySecondaryUpstream())
	}
}

func probes(proxy *pipy.Proxy, pipyConf *PipyConf) {
	if proxy.PodMetadata != nil {
		if len(proxy.PodMetadata.StartupProbes) > 0 {
			for idx := range proxy.PodMetadata.StartupProbes {
				pipyConf.Spec.Probes.StartupProbes = append(pipyConf.Spec.Probes.StartupProbes, *proxy.PodMetadata.StartupProbes[idx])
			}
		}
		if len(proxy.PodMetadata.LivenessProbes) > 0 {
			for idx := range proxy.PodMetadata.LivenessProbes {
				pipyConf.Spec.Probes.LivenessProbes = append(pipyConf.Spec.Probes.LivenessProbes, *proxy.PodMetadata.LivenessProbes[idx])
			}
		}
		if len(proxy.PodMetadata.ReadinessProbes) > 0 {
			for idx := range proxy.PodMetadata.ReadinessProbes {
				pipyConf.Spec.Probes.ReadinessProbes = append(pipyConf.Spec.Probes.ReadinessProbes, *proxy.PodMetadata.ReadinessProbes[idx])
			}
		}
	}
}

var (
	repoLock sync.Mutex
)

func (job *PipyConfGeneratorJob) publishSidecarConf(repoClient *client.PipyRepoClient, proxy *pipy.Proxy, pipyConf *PipyConf, pluginSetV string) {
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
		codebaseCurV := hash(bytes)
		if codebaseCurV != codebasePreV {
			log.Log().Str("Proxy", proxy.GetCNPrefix()).
				Str("uid", proxy.GetUUID().String()).
				Str("id", fmt.Sprintf("%d", proxy.ID)).
				Str("codebasePreV", fmt.Sprintf("%d", codebasePreV)).
				Str("codebaseCurV", fmt.Sprintf("%d", codebaseCurV)).
				Msg("config.json")
			codebase := fmt.Sprintf("%s/%s", ecnetProxyCodebase, proxy.GetCNPrefix())
			success, err := repoClient.DeriveCodebase(codebase, ecnetCodebaseRepo, codebaseCurV-2)
			if success {
				ts := time.Now()
				pipyConf.Ts = &ts
				version := fmt.Sprintf("%d", codebaseCurV)
				pipyConf.Version = &version
				bytes, _ = json.MarshalIndent(pipyConf, "", " ")
				_, err = repoClient.Batch(fmt.Sprintf("%d", codebaseCurV-1), []client.Batch{
					{
						Basepath: codebase,
						Items: []client.BatchItem{
							{
								Filename: ecnetCodebaseConfig,
								Content:  bytes,
							},
						},
					},
				})
			}
			if err != nil {
				log.Error().Err(err)
				_, _ = repoClient.Delete(codebase)
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
