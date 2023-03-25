package repo

import (
	"sync"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/announcements"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/errcode"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/identity"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy"
)

func (s *Server) informTrafficPolicies(proxyPtr **pipy.Proxy, wg *sync.WaitGroup, callback func(**pipy.Proxy)) error {
	proxy := *proxyPtr
	if initError := s.recordPodMetadata(proxy); initError == errServiceAccountMismatch {
		// Service Account mismatch
		log.Error().Err(initError).Str("proxy", proxy.String()).Msg("Mismatched service account for proxy")
		return initError
	}

	proxy = s.proxyRegistry.RegisterProxy(proxy)
	if callback != nil {
		callback(&proxy)
	}

	defer s.proxyRegistry.UnregisterProxy(proxy)

	proxy.Quit = make(chan bool)
	// Subscribe to both broadcast and proxy UUID specific events
	proxyUpdatePubSub := s.msgBroker.GetProxyUpdatePubSub()
	proxyUpdateChan := proxyUpdatePubSub.Sub(announcements.ProxyUpdate.String(), messaging.GetPubSubTopicForProxyUUID(proxy.UUID.String()))
	defer s.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

	newJob := func() *PipyConfGeneratorJob {
		return &PipyConfGeneratorJob{
			proxy:      proxy,
			repoServer: s,
			done:       make(chan struct{}),
		}
	}

	wg.Done()

	for {
		select {
		case <-proxy.Quit:
			log.Info().Str("proxy", proxy.String()).Msgf("Pipy Restful session closed")
			return nil

		case <-proxyUpdateChan:
			log.Info().Str("proxy", proxy.String()).Msg("Broadcast update received")
			// Queue a full configuration update
			// Do not send SDS, let sidecar figure out what certs does it want.
			<-s.workQueues.AddJob(newJob())
		}
	}
}

// recordPodMetadata records pod metadata and verifies the certificate issued for this pod
// is for the same service account as seen on the pod's service account
func (s *Server) recordPodMetadata(p *pipy.Proxy) error {
	if p.PodMetadata == nil {
		pod, err := s.kubeController.GetPodForProxy(p)
		if err != nil {
			log.Warn().Str("proxy", p.String()).Msg("Could not find pod for connecting proxy. No metadata was recorded.")
			return nil
		}

		workloadKind := ""
		workloadName := ""
		for _, ref := range pod.GetOwnerReferences() {
			if ref.Controller != nil && *ref.Controller {
				workloadKind = ref.Kind
				workloadName = ref.Name
				break
			}
		}

		p.PodMetadata = &pipy.PodMetadata{
			UID:       string(pod.UID),
			Name:      pod.Name,
			Namespace: pod.Namespace,
			ServiceAccount: identity.K8sServiceAccount{
				Namespace: pod.Namespace,
				Name:      pod.Spec.ServiceAccountName,
			},
			CreationTime: pod.GetCreationTimestamp().Time,
			WorkloadKind: workloadKind,
			WorkloadName: workloadName,
		}

		for idx := range pod.Spec.Containers {
			if pod.Spec.Containers[idx].ReadinessProbe != nil {
				p.PodMetadata.ReadinessProbes = append(p.PodMetadata.ReadinessProbes, pod.Spec.Containers[idx].ReadinessProbe)
			}
			if pod.Spec.Containers[idx].LivenessProbe != nil {
				p.PodMetadata.LivenessProbes = append(p.PodMetadata.LivenessProbes, pod.Spec.Containers[idx].LivenessProbe)
			}
			if pod.Spec.Containers[idx].StartupProbe != nil {
				p.PodMetadata.StartupProbes = append(p.PodMetadata.StartupProbes, pod.Spec.Containers[idx].StartupProbe)
			}
		}

		if len(pod.Status.PodIP) > 0 {
			p.Addr = pipy.NewNetAddress(pod.Status.PodIP)
		}
	}

	// Verify Service account matches (cert to pod Service Account)
	if p.Identity.ToK8sServiceAccount() != p.PodMetadata.ServiceAccount {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMismatchedServiceAccount)).Str("proxy", p.String()).
			Msgf("Service Account referenced in NodeID (%s) does not match Service Account in Certificate (%s). This proxy is not allowed to join the mesh.", p.PodMetadata.ServiceAccount, p.Identity.ToK8sServiceAccount())
		return errServiceAccountMismatch
	}

	return nil
}
