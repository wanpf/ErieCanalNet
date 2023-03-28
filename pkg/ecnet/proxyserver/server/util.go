package server

import (
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/catalog"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/endpoint"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/policy"
)

func generatePipyOutboundTrafficRoutePolicy(pipyConf *PipyConf, outboundPolicy *policy.OutboundMeshTrafficPolicy) map[service.ClusterName]*WeightedCluster {
	if len(outboundPolicy.TrafficMatches) == 0 {
		return nil
	}

	otp := pipyConf.newOutboundTrafficPolicy()
	dependClusters := make(map[service.ClusterName]*WeightedCluster)

	for _, trafficMatch := range outboundPolicy.TrafficMatches {
		destinationProtocol := strings.ToLower(trafficMatch.DestinationProtocol)
		trafficMatchName := trafficMatch.Name
		if destinationProtocol == constants.ProtocolHTTP || destinationProtocol == constants.ProtocolGRPC {
			trafficMatchName = constants.ProtocolHTTP
		}
		tm, exist := otp.newTrafficMatch(Port(trafficMatch.DestinationPort), trafficMatchName)
		if !exist {
			tm.setProtocol(Protocol(destinationProtocol))
			tm.setPort(Port(trafficMatch.DestinationPort))
		}

		if destinationProtocol == constants.ProtocolHTTP ||
			destinationProtocol == constants.ProtocolGRPC {
			upstreamSvc := trafficMatchToMeshSvc(trafficMatch)
			upstreamSvcFQDN := upstreamSvc.FQDN()

			httpRouteConfigs := getOutboundHTTPRouteConfigs(outboundPolicy.HTTPRouteConfigsPerPort,
				int(upstreamSvc.TargetPort), upstreamSvcFQDN, trafficMatch.WeightedClusters)
			if len(httpRouteConfigs) == 0 {
				continue
			}

			for _, httpRouteConfig := range httpRouteConfigs {
				ruleName := HTTPRouteRuleName(httpRouteConfig.Name)
				hsrrs := tm.newHTTPServiceRouteRules(ruleName)
				for _, hostname := range httpRouteConfig.Hostnames {
					tm.addHTTPHostPort2Service(HTTPHostPort(hostname), ruleName)
				}

				for _, route := range httpRouteConfig.Routes {
					httpMatch := new(HTTPMatchRule)
					httpMatch.Path = URIPathValue(route.HTTPRouteMatch.Path)
					httpMatch.Type = matchType(route.HTTPRouteMatch.PathMatchType)
					if len(httpMatch.Type) == 0 {
						httpMatch.Type = PathMatchRegex
					}
					if len(httpMatch.Path) == 0 {
						httpMatch.Path = constants.RegexMatchAll
					}
					for k, v := range route.HTTPRouteMatch.Headers {
						httpMatch.addHeaderMatch(Header(k), HeaderRegexp(v))
					}
					if len(route.HTTPRouteMatch.Methods) == 0 {
						httpMatch.addMethodMatch("*")
					} else {
						for _, method := range route.HTTPRouteMatch.Methods {
							httpMatch.addMethodMatch(Method(method))
						}
					}

					hsrr, _ := hsrrs.newHTTPServiceRouteRule(httpMatch)
					for cluster := range route.WeightedClusters.Iter() {
						serviceCluster := cluster.(service.WeightedCluster)
						weightedCluster := new(WeightedCluster)
						weightedCluster.WeightedCluster = serviceCluster
						if _, exist := dependClusters[weightedCluster.ClusterName]; !exist {
							dependClusters[weightedCluster.ClusterName] = weightedCluster
						}
						hsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
					}
				}
			}
		} else if destinationProtocol == constants.ProtocolTCP ||
			destinationProtocol == constants.ProtocolTCPServerFirst {
			tsrr := tm.newTCPServiceRouteRules()
			for _, serviceCluster := range trafficMatch.WeightedClusters {
				weightedCluster := new(WeightedCluster)
				weightedCluster.WeightedCluster = serviceCluster
				if _, exist := dependClusters[weightedCluster.ClusterName]; !exist {
					dependClusters[weightedCluster.ClusterName] = weightedCluster
				}
				tsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
			}
		} else if destinationProtocol == constants.ProtocolHTTPS {
			upstreamSvc := trafficMatchToMeshSvc(trafficMatch)
			upstreamSvcFQDN := upstreamSvc.FQDN()

			httpRouteConfigs := getOutboundHTTPRouteConfigs(outboundPolicy.HTTPRouteConfigsPerPort,
				int(upstreamSvc.TargetPort), upstreamSvcFQDN, trafficMatch.WeightedClusters)
			if len(httpRouteConfigs) == 0 {
				continue
			}

			tsrr := tm.newTCPServiceRouteRules()
			for _, httpRouteConfig := range httpRouteConfigs {
				for _, route := range httpRouteConfig.Routes {
					for cluster := range route.WeightedClusters.Iter() {
						serviceCluster := cluster.(service.WeightedCluster)
						weightedCluster := new(WeightedCluster)
						weightedCluster.WeightedCluster = serviceCluster
						if _, exist := dependClusters[weightedCluster.ClusterName]; !exist {
							dependClusters[weightedCluster.ClusterName] = weightedCluster
						}
						tsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
					}
				}
			}
		}
	}

	return dependClusters
}

func generatePipyOutboundTrafficBalancePolicy(meshCatalog catalog.MeshCataloger,
	pipyConf *PipyConf, outboundPolicy *policy.OutboundMeshTrafficPolicy,
	dependClusters map[service.ClusterName]*WeightedCluster) bool {
	ready := true
	otp := pipyConf.newOutboundTrafficPolicy()
	for _, cluster := range dependClusters {
		clusterConfig := getMeshClusterConfigs(outboundPolicy.ClustersConfigs, cluster.ClusterName)
		if clusterConfig == nil {
			ready = false
			continue
		}
		clusterConfigs := otp.newClusterConfigs(ClusterName(cluster.ClusterName.String()))
		upstreamEndpoints := getUpstreamEndpoints(meshCatalog, cluster.ClusterName)
		if len(upstreamEndpoints) == 0 {
			ready = false
			continue
		}
		for _, upstreamEndpoint := range upstreamEndpoints {
			address := Address(upstreamEndpoint.IP.String())
			port := Port(clusterConfig.Service.Port)
			if len(upstreamEndpoint.ClusterKey) > 0 {
				if targetPort := Port(clusterConfig.Service.TargetPort); targetPort > 0 {
					port = targetPort
				}
			}
			weight := Weight(upstreamEndpoint.Weight)
			clusterConfigs.addWeightedZoneEndpoint(address, port, weight, upstreamEndpoint.ClusterKey, upstreamEndpoint.LBType, upstreamEndpoint.Path)
		}
	}
	return ready
}

func getOutboundHTTPRouteConfigs(httpRouteConfigsPerPort map[int][]*policy.OutboundTrafficPolicy,
	targetPort int, upstreamSvcFQDN string, weightedClusters []service.WeightedCluster) []*policy.OutboundTrafficPolicy {
	var outboundTrafficPolicies []*policy.OutboundTrafficPolicy
	if trafficPolicies, ok := httpRouteConfigsPerPort[targetPort]; ok {
		for _, trafficPolicy := range trafficPolicies {
			if trafficPolicy.Name == upstreamSvcFQDN {
				for _, route := range trafficPolicy.Routes {
					if arrayEqual(weightedClusters, route.WeightedClusters) {
						outboundTrafficPolicies = append(outboundTrafficPolicies, trafficPolicy)
						break
					}
				}
			}
		}
	}
	return outboundTrafficPolicies
}

func trafficMatchToMeshSvc(trafficMatch *policy.TrafficMatch) *service.MeshService {
	splitFunc := func(r rune) bool {
		return r == '_'
	}

	chunks := strings.FieldsFunc(trafficMatch.Name, splitFunc)
	if len(chunks) != 4 {
		log.Error().Msgf("Invalid traffic match name. Expected: xxx_<namespace>/<name>_<port>_<protocol>, got: %s",
			trafficMatch.Name)
		return nil
	}

	namespacedName, err := k8s.NamespacedNameFrom(chunks[1])
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving NamespacedName from TrafficMatch")
		return nil
	}
	return &service.MeshService{
		Namespace:  namespacedName.Namespace,
		Name:       namespacedName.Name,
		Protocol:   strings.ToLower(trafficMatch.DestinationProtocol),
		TargetPort: uint16(trafficMatch.DestinationPort),
	}
}

func getMeshClusterConfigs(clustersConfigs []*policy.MeshClusterConfig,
	clusterName service.ClusterName) *policy.MeshClusterConfig {
	if len(clustersConfigs) == 0 {
		return nil
	}

	for _, clustersConfig := range clustersConfigs {
		if clusterName.String() == clustersConfig.Name {
			return clustersConfig
		}
	}

	return nil
}

func getUpstreamEndpoints(meshCatalog catalog.MeshCataloger, clusterName service.ClusterName) []endpoint.Endpoint {
	if dstSvc, err := clusterToMeshSvc(clusterName.String()); err == nil {
		return meshCatalog.ListUpstreamEndpointsForService(dstSvc)
	}
	return nil
}

// clusterToMeshSvc returns the MeshService associated with the given cluster name
func clusterToMeshSvc(cluster string) (service.MeshService, error) {
	splitFunc := func(r rune) bool {
		return r == '/' || r == '|'
	}

	chunks := strings.FieldsFunc(cluster, splitFunc)
	if len(chunks) != 3 {
		return service.MeshService{},
			errors.Errorf("Invalid cluster name. Expected: <namespace>/<name>|<port>, got: %s", cluster)
	}

	port, err := strconv.ParseUint(chunks[2], 10, 16)
	if err != nil {
		return service.MeshService{}, errors.Errorf("Invalid cluster port %s, expected int value: %s", chunks[2], err)
	}

	return service.MeshService{
		Namespace: chunks[0],
		Name:      chunks[1],
		// The port always maps to MeshServer.TargetPort and not MeshService.Port because
		// endpoints of a service are derived from it's TargetPort and not Port.
		TargetPort: uint16(port),
	}, nil
}

func arrayEqual(a []service.WeightedCluster, set mapset.Set) bool {
	var b []service.WeightedCluster
	for e := range set.Iter() {
		if o, ok := e.(service.WeightedCluster); ok {
			b = append(b, o)
		}
	}
	if len(a) == len(b) {
		for _, ca := range a {
			caEqualb := false
			for _, cb := range b {
				if ca.ClusterName == cb.ClusterName && ca.Weight == cb.Weight {
					caEqualb = true
					break
				}
			}
			if !caEqualb {
				return false
			}
		}
		for _, cb := range b {
			cbEquala := false
			for _, ca := range a {
				if cb.ClusterName == ca.ClusterName && cb.Weight == ca.Weight {
					cbEquala = true
					break
				}
			}
			if !cbEquala {
				return false
			}
		}
		return true
	}
	return false
}

func matchType(matchType policy.PathMatchType) URIMatchType {
	switch matchType {
	case policy.PathMatchExact:
		return PathMatchExact
	case policy.PathMatchPrefix:
		return PathMatchPrefix
	default:
		return PathMatchRegex
	}
}
