// Package kube implements KubeClient's methods.
package kube

import (
	"net"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/endpoint"
)

// Ensure interface compliance
var _ endpoint.Provider = (*client)(nil)
var _ service.Provider = (*client)(nil)

// NewClient returns a client that has all components necessary to connect to and maintain state of a Kubernetes cluster.
func NewClient(kubeController k8s.Controller, cfg configurator.Configurator) *client { //nolint: revive // unexported-return
	return &client{
		kubeController:   kubeController,
		meshConfigurator: cfg,
	}
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interfaces: EndpointsProvider, ServiceProvider
func (c *client) GetID() string {
	return providerName
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c *client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	log.Trace().Msgf("Getting Endpoints for MeshService %s on Kubernetes", svc)

	kubernetesEndpoints, err := c.kubeController.GetEndpoints(svc)
	if err != nil || kubernetesEndpoints == nil {
		log.Info().Msgf("No k8s endpoints found for MeshService %s", svc)
		return nil
	}

	var endpoints []endpoint.Endpoint
	for _, kubernetesEndpoint := range kubernetesEndpoints.Subsets {
		for _, port := range kubernetesEndpoint.Ports {
			// If a TargetPort is specified for the service, filter the endpoint by this port.
			// This is required to ensure we do not attempt to filter the endpoints when the endpoints
			// are being listed for a MeshService whose TargetPort is not known.
			if svc.TargetPort != 0 && port.Port != int32(svc.TargetPort) {
				// k8s service's port does not match MeshService port, ignore this port
				continue
			}
			for _, address := range kubernetesEndpoint.Addresses {
				if svc.Subdomain() != "" && svc.Subdomain() != address.Hostname {
					// if there's a subdomain on this meshservice, make sure it matches the endpoint's hostname
					continue
				}
				ip := net.ParseIP(address.IP)
				if ip == nil {
					log.Error().Msgf("Error parsing endpoint IP address %s for MeshService %s", address.IP, svc)
					continue
				}
				ept := endpoint.Endpoint{
					IP:   ip,
					Port: endpoint.Port(port.Port),
				}
				endpoints = append(endpoints, ept)
			}
		}
	}

	log.Trace().Msgf("Endpoints for MeshService %s: %v", svc, endpoints)

	return endpoints
}

// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service
// FQDN is resolved
func (c *client) GetResolvableEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	// Check if the service has been given Cluster IP
	kubeService := c.kubeController.GetService(svc)
	if kubeService == nil {
		log.Info().Msgf("No k8s services found for MeshService %s", svc)
		return nil
	}

	if len(kubeService.Spec.ClusterIP) == 0 || kubeService.Spec.ClusterIP == corev1.ClusterIPNone {
		// If service has no cluster IP or cluster IP is <none>, use final endpoint as resolvable destinations
		return c.ListEndpointsForService(svc)
	}

	// Cluster IP is present
	ip := net.ParseIP(kubeService.Spec.ClusterIP)
	if ip == nil {
		log.Error().Msgf("[%s] Could not parse Cluster IP %s", c.GetID(), kubeService.Spec.ClusterIP)
		return nil
	}

	for _, svcPort := range kubeService.Spec.Ports {
		endpoints = append(endpoints, endpoint.Endpoint{
			IP:   ip,
			Port: endpoint.Port(svcPort.Port),
		})
	}

	return endpoints
}

// ListServices returns a list of services that are part of monitored namespaces
func (c *client) ListServices() []service.MeshService {
	var services []service.MeshService
	for _, svc := range c.kubeController.ListServices() {
		services = append(services, k8s.ServiceToMeshServices(c.kubeController, *svc)...)
	}
	return services
}
