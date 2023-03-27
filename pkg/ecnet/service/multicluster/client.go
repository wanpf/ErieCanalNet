package multicluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/announcements"
	multiclusterv1alpha1 "github.com/flomesh-io/ErieCanal/pkg/ecnet/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s/informers"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
)

// NewMultiClusterController returns a multicluster.Controller interface related to functionality provided by the resources in the flomesh.io API group
func NewMultiClusterController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := &Client{
		informers:      informerCollection,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		if _, object := obj.(metav1.Object); !object {
			return false
		}
		if _, serviceImport := obj.(*multiclusterv1alpha1.ServiceImport); serviceImport {
			return true
		}
		if _, gblTrafficPolicy := obj.(*multiclusterv1alpha1.GlobalTrafficPolicy); gblTrafficPolicy {
			return true
		}
		return false
	}

	svcImportEventTypes := k8s.EventTypes{
		Add:    announcements.ServiceImportAdded,
		Update: announcements.ServiceImportUpdated,
		Delete: announcements.ServiceImportDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyServiceImport, k8s.GetEventHandlerFuncs(shouldObserve, svcImportEventTypes, msgBroker))

	glbTrafficPolicyTypes := k8s.EventTypes{
		Add:    announcements.GlobalTrafficPolicyAdded,
		Update: announcements.GlobalTrafficPolicyUpdated,
		Delete: announcements.GlobalTrafficPolicyDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyGlobalTrafficPolicy, k8s.GetEventHandlerFuncs(shouldObserve, glbTrafficPolicyTypes, msgBroker))

	return client
}
