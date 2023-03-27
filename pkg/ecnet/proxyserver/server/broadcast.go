// Package server implements broadcast's methods.
package server

import (
	"sync"
	"time"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/announcements"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver"
)

// Routine which fulfills listening to proxy broadcasts
func (s *Server) broadcastListener() {
	// Register for proxy config updates broadcast by the message broker
	proxyUpdatePubSub := s.msgBroker.GetProxyUpdatePubSub()
	proxyUpdateChan := proxyUpdatePubSub.Sub(announcements.ProxyUpdate.String())
	defer s.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

	// Wait for two informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 20)
	defer slidingTimer.Stop()

	slidingTimerReset := func() {
		slidingTimer.Reset(time.Second * 5)
	}

	s.retryProxiesJob = slidingTimerReset
	s.proxyRegistry.UpdateProxies = slidingTimerReset

	reconfirm := true

	for {
		select {
		case <-proxyUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 5)
			// Avoid data omission
			reconfirm = true

		case <-slidingTimer.C:
			connectedProxies := s.fireExistProxies()
			if len(connectedProxies) > 0 {
				for _, proxy := range connectedProxies {
					newJob := func() *PipyConfGeneratorJob {
						return &PipyConfGeneratorJob{
							proxy:      proxy,
							repoServer: s,
							done:       make(chan struct{}),
						}
					}
					<-s.workQueues.AddJob(newJob())
				}
			}
			if reconfirm {
				reconfirm = false
				slidingTimer.Reset(time.Second * 10)
			}
		}
	}
}

func (s *Server) fireExistProxies() []*proxyserver.Proxy {
	var allProxies []*proxyserver.Proxy
	connectedProxy := s.proxyRegistry.GetConnectedProxy()
	if connectedProxy == nil {
		connectedProxy = s.proxyRegistry.RegisterProxy()
		s.informProxy(connectedProxy)
	}
	allProxies = append(allProxies, connectedProxy)
	return allProxies
}

func (s *Server) informProxy(proxy *proxyserver.Proxy) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if aggregatedErr := s.informTrafficPolicies(proxy, &wg); aggregatedErr != nil {
			log.Error().Err(aggregatedErr).Msgf("Pipy Aggregated Traffic Policies Error.")
		}
	}()
	wg.Wait()
}
