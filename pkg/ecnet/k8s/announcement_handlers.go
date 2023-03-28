package k8s

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/announcements"
	configv1alpha1 "github.com/flomesh-io/ErieCanal/pkg/ecnet/apis/config/v1alpha1"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s/events"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
)

// WatchAndUpdateLogLevel watches for log level changes and updates the global log level
func WatchAndUpdateLogLevel(msgBroker *messaging.Broker, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	meshCfgUpdateChan := kubePubSub.Sub(announcements.EcnetConfigUpdated.String())
	defer msgBroker.Unsub(kubePubSub, meshCfgUpdateChan)

	for {
		select {
		case <-stop:
			log.Info().Msg("Received stop signal, exiting log level update routine")
			return

		case event := <-meshCfgUpdateChan:
			msg, ok := event.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Error casting to PubSubMessage, got type %T", msg)
				continue
			}

			prevObj, prevOk := msg.OldObj.(*configv1alpha1.EcnetConfig)
			newObj, newOk := msg.NewObj.(*configv1alpha1.EcnetConfig)
			if !prevOk || !newOk {
				log.Error().Msgf("Error casting to *EcnetConfig, got type prev=%T, new=%T", prevObj, newObj)
			}
		}
	}
}
