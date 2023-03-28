// Package fsm implements MulticlusterClient's methods.
package fsm

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/rs/zerolog/log"
	"k8s.io/utils/pointer"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/endpoint"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/multicluster"
)

// Ensure interface compliance
var _ endpoint.Provider = (*client)(nil)
var _ service.Provider = (*client)(nil)

// NewClient returns a client that has all components necessary to connect to and maintain state of a multi cluster.
func NewClient(multiclusterController multicluster.Controller, cfg configurator.Configurator) *client { //nolint: revive // unexported-return
	return &client{
		multiclusterController: multiclusterController,
		ecnetConfigurator:      cfg,
	}
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interfaces: EndpointsProvider, ServiceProvider
func (c *client) GetID() string {
	return ProviderName
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c *client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	kubernetesEndpoints, err := c.multiclusterController.GetEndpoints(svc)
	if err != nil || kubernetesEndpoints == nil {
		log.Info().Msgf("No mcs endpoints found for MeshService %s", svc)
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
				weight, _ := strconv.ParseUint(kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportLBWeightAnnotation, address.IP, port.Port)], 10, 32)
				ept := endpoint.Endpoint{
					IP:         ip,
					Port:       endpoint.Port(port.Port),
					ClusterKey: kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportClusterKeyAnnotation, address.IP, port.Port)],
					LBType:     kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportLBTypeAnnotation, address.IP, port.Port)],
					Weight:     endpoint.Weight(weight),
					Path:       kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportContextPathAnnotation, address.IP, port.Port)],
				}
				endpoints = append(endpoints, ept)
			}
		}
	}
	return endpoints
}

// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service
// FQDN is resolved
func (c *client) GetResolvableEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	// Check if the service has been given Cluster IP
	kubeService := c.multiclusterController.GetService(svc)
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
			IP:         ip,
			Port:       endpoint.Port(svcPort.Port),
			ClusterKey: c.GetID(),
		})
	}

	return endpoints
}

// ListServices returns a list of services that are part of monitored namespaces
func (c *client) ListServices() []service.MeshService {
	var services []service.MeshService
	for _, svc := range c.multiclusterController.ListServices() {
		services = append(services, ServiceToMeshServices(c.multiclusterController, *svc)...)
	}
	return services
}

// ServiceToMeshServices translates a k8s service with one or more ports to one or more
// MeshService objects per port.
func ServiceToMeshServices(c multicluster.Controller, svc corev1.Service) []service.MeshService {
	var meshServices []service.MeshService

	for _, portSpec := range svc.Spec.Ports {
		meshSvc := service.MeshService{
			Namespace:        svc.Namespace,
			Name:             svc.Name,
			Port:             uint16(portSpec.Port),
			ServiceImportUID: svc.UID,
		}

		// attempt to parse protocol from port name
		// Order of Preference is:
		// 1. port.appProtocol field
		// 2. protocol prefixed to port name (e.g. tcp-my-port)
		// 3. default to http
		protocol := constants.ProtocolHTTP
		for _, p := range constants.SupportedProtocolsInMesh {
			if strings.HasPrefix(portSpec.Name, p+"-") {
				protocol = p
				break
			}
		}

		// use port.appProtocol if specified, else use port protocol
		meshSvc.Protocol = pointer.StringDeref(portSpec.AppProtocol, protocol)

		// The endpoints for the kubernetes service carry information that allows
		// us to retrieve the TargetPort for the MeshService.
		endpoints, _ := c.GetEndpoints(meshSvc)
		if endpoints == nil {
			continue
		}

		for _, subset := range endpoints.Subsets {
			for _, port := range subset.Ports {
				meshServices = append(meshServices, service.MeshService{
					Namespace:        meshSvc.Namespace,
					Name:             meshSvc.Name,
					Port:             meshSvc.Port,
					TargetPort:       uint16(port.Port),
					Protocol:         meshSvc.Protocol,
					ServiceImportUID: meshSvc.ServiceImportUID,
				})
			}
		}
	}

	return meshServices
}
