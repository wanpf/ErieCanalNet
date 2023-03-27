package server

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	multiclusterv1alpha1 "github.com/flomesh-io/ErieCanal/pkg/ecnet/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
)

var (
	addrWithPort, _ = regexp.Compile(`:\d+$`)
)

func (p *PipyConf) setSidecarLogLevel(sidecarLogLevel string) (update bool) {
	if update = !strings.EqualFold(p.Spec.SidecarLogLevel, sidecarLogLevel); update {
		p.Spec.SidecarLogLevel = sidecarLogLevel
	}
	return
}

func (p *PipyConf) setLocalDNSProxy(enable bool, primary, secondary string) {
	if enable {
		p.Spec.LocalDNSProxy = new(LocalDNSProxy)
		if len(primary) > 0 || len(secondary) > 0 {
			p.Spec.LocalDNSProxy.UpstreamDNSServers = new(UpstreamDNSServers)
			if len(primary) > 0 {
				p.Spec.LocalDNSProxy.UpstreamDNSServers.Primary = &primary
			}
			if len(secondary) > 0 {
				p.Spec.LocalDNSProxy.UpstreamDNSServers.Secondary = &secondary
			}
		}
	} else {
		p.Spec.LocalDNSProxy = nil
	}
}

func (p *PipyConf) newOutboundTrafficPolicy() *OutboundTrafficPolicy {
	if p.Outbound == nil {
		p.Outbound = new(OutboundTrafficPolicy)
	}
	return p.Outbound
}

func (p *PipyConf) rebalancedOutboundClusters() {
	if p.Outbound == nil {
		return
	}
	if p.Outbound.ClustersConfigs == nil || len(p.Outbound.ClustersConfigs) == 0 {
		return
	}
	for _, clusterConfigs := range p.Outbound.ClustersConfigs {
		weightedEndpoints := clusterConfigs.Endpoints
		if weightedEndpoints == nil || len(*weightedEndpoints) == 0 {
			continue
		}
		hasLocalEndpoints := false
		for _, wze := range *weightedEndpoints {
			if len(wze.Cluster) == 0 {
				hasLocalEndpoints = true
				break
			}
		}
		for _, wze := range *weightedEndpoints {
			if len(wze.Cluster) > 0 {
				if multiclusterv1alpha1.FailOverLbType == multiclusterv1alpha1.LoadBalancerType(wze.LBType) {
					if hasLocalEndpoints {
						wze.Weight = constants.ClusterWeightFailOver
					} else {
						wze.Weight = constants.ClusterWeightAcceptAll
					}
				} else if multiclusterv1alpha1.ActiveActiveLbType == multiclusterv1alpha1.LoadBalancerType(wze.LBType) {
					if wze.Weight == 0 {
						wze.Weight = constants.ClusterWeightAcceptAll
					}
				}
			} else {
				if wze.Weight == 0 {
					wze.Weight = constants.ClusterWeightAcceptAll
				}
			}
		}
	}
}

func (otm *OutboundTrafficMatch) setPort(port Port) {
	otm.Port = port
}

func (otm *OutboundTrafficMatch) setProtocol(protocol Protocol) {
	protocol = Protocol(strings.ToLower(string(protocol)))
	if constants.ProtocolTCPServerFirst == protocol {
		otm.Protocol = constants.ProtocolTCP
	} else {
		otm.Protocol = protocol
	}
}

func (otm *OutboundTrafficMatch) newTCPServiceRouteRules() *OutboundTCPServiceRouteRules {
	if otm.TCPServiceRouteRules == nil {
		otm.TCPServiceRouteRules = new(OutboundTCPServiceRouteRules)
	}
	return otm.TCPServiceRouteRules
}

func (srr *OutboundTCPServiceRouteRules) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if srr.TargetClusters == nil {
		srr.TargetClusters = make(WeightedClusters)
	}
	srr.TargetClusters[clusterName] = weight
}

func (otm *OutboundTrafficMatch) addHTTPHostPort2Service(hostPort HTTPHostPort, ruleName HTTPRouteRuleName) {
	if otm.HTTPHostPort2Service == nil {
		otm.HTTPHostPort2Service = make(HTTPHostPort2Service)
	}
	otm.HTTPHostPort2Service[hostPort] = ruleName
}

func (otm *OutboundTrafficMatch) newHTTPServiceRouteRules(httpRouteRuleName HTTPRouteRuleName) *OutboundHTTPRouteRules {
	if otm.HTTPServiceRouteRules == nil {
		otm.HTTPServiceRouteRules = make(OutboundHTTPServiceRouteRules)
	}
	if len(httpRouteRuleName) == 0 {
		return nil
	}
	rules, exist := otm.HTTPServiceRouteRules[httpRouteRuleName]
	if !exist || rules == nil {
		newCluster := new(OutboundHTTPRouteRules)
		otm.HTTPServiceRouteRules[httpRouteRuleName] = newCluster
		return newCluster
	}
	return rules
}

func (otp *OutboundTrafficPolicy) newTrafficMatch(port Port, name string) (*OutboundTrafficMatch, bool) {
	namedPort := fmt.Sprintf(`%d=%s`, port, name)
	if otp.namedTrafficMatches == nil {
		otp.namedTrafficMatches = make(namedOutboundTrafficMatches)
	}
	trafficMatch, exists := otp.namedTrafficMatches[namedPort]
	if exists {
		return trafficMatch, true
	}

	trafficMatch = new(OutboundTrafficMatch)
	otp.namedTrafficMatches[namedPort] = trafficMatch

	if otp.TrafficMatches == nil {
		otp.TrafficMatches = make(OutboundTrafficMatches)
	}
	trafficMatches := otp.TrafficMatches[port]
	trafficMatches = append(trafficMatches, trafficMatch)
	otp.TrafficMatches[port] = trafficMatches
	return trafficMatch, false
}

func (hrrs *OutboundHTTPRouteRules) newHTTPServiceRouteRule(matchRule *HTTPMatchRule) (route *OutboundHTTPRouteRule, duplicate bool) {
	for _, routeRule := range hrrs.RouteRules {
		if reflect.DeepEqual(*matchRule, routeRule.HTTPMatchRule) {
			return routeRule, true
		}
	}

	routeRule := new(OutboundHTTPRouteRule)
	routeRule.HTTPMatchRule = *matchRule
	hrrs.RouteRules = append(hrrs.RouteRules, routeRule)
	return routeRule, false
}

func (hmr *HTTPMatchRule) addHeaderMatch(header Header, headerRegexp HeaderRegexp) {
	if hmr.Headers == nil {
		hmr.Headers = make(Headers)
	}
	hmr.Headers[header] = headerRegexp
}

func (hmr *HTTPMatchRule) addMethodMatch(method Method) {
	if hmr.allowedAnyMethod {
		return
	}
	if "*" == method {
		hmr.allowedAnyMethod = true
	}
	if hmr.allowedAnyMethod {
		hmr.Methods = nil
	} else {
		hmr.Methods = append(hmr.Methods, method)
	}
}

func (hrr *HTTPRouteRule) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if hrr.TargetClusters == nil {
		hrr.TargetClusters = make(WeightedClusters)
	}
	hrr.TargetClusters[clusterName] = weight
}

func (otp *OutboundTrafficPolicy) newClusterConfigs(clusterName ClusterName) *ClusterConfigs {
	if otp.ClustersConfigs == nil {
		otp.ClustersConfigs = make(map[ClusterName]*ClusterConfigs)
	}
	cluster, exist := otp.ClustersConfigs[clusterName]
	if !exist || cluster == nil {
		newCluster := new(ClusterConfigs)
		otp.ClustersConfigs[clusterName] = newCluster
		return newCluster
	}
	return cluster
}

func (otp *ClusterConfigs) addWeightedZoneEndpoint(address Address, port Port, weight Weight, cluster, lbType, contextPath string) {
	if otp.Endpoints == nil {
		weightedEndpoints := make(WeightedEndpoints)
		otp.Endpoints = &weightedEndpoints
	}
	otp.Endpoints.addWeightedZoneEndpoint(address, port, weight, cluster, lbType, contextPath)
}

func (wes *WeightedEndpoints) addWeightedZoneEndpoint(address Address, port Port, weight Weight, cluster, lbType, contextPath string) {
	if addrWithPort.MatchString(string(address)) {
		httpHostPort := HTTPHostPort(address)
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight:      weight,
			Cluster:     cluster,
			LBType:      lbType,
			ContextPath: contextPath,
		}
	} else {
		httpHostPort := HTTPHostPort(fmt.Sprintf("%s:%d", address, port))
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight:      weight,
			Cluster:     cluster,
			LBType:      lbType,
			ContextPath: contextPath,
		}
	}
}

func (hrrs *OutboundHTTPRouteRuleSlice) sort() {
	if len(*hrrs) > 1 {
		sort.Sort(hrrs)
	}
}

func (hrrs *OutboundHTTPRouteRuleSlice) Len() int {
	return len(*hrrs)
}

func (hrrs *OutboundHTTPRouteRuleSlice) Swap(i, j int) {
	(*hrrs)[j], (*hrrs)[i] = (*hrrs)[i], (*hrrs)[j]
}

func (hrrs *OutboundHTTPRouteRuleSlice) Less(i, j int) bool {
	a, b := (*hrrs)[i], (*hrrs)[j]
	if a.Path == constants.RegexMatchAll {
		return false
	}
	return strings.Compare(string(a.Path), string(b.Path)) == -1
}

func (ps *PluginSlice) Len() int {
	return len(*ps)
}

func (ps *PluginSlice) Swap(i, j int) {
	(*ps)[j], (*ps)[i] = (*ps)[i], (*ps)[j]
}

func (ps *PluginSlice) Less(i, j int) bool {
	a, b := (*ps)[i], (*ps)[j]
	return a.Priority > b.Priority
}
