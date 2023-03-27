package server

import (
	"sync"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/announcements"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver"
)

func (s *Server) informTrafficPolicies(proxy *proxyserver.Proxy, wg *sync.WaitGroup) error {
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
