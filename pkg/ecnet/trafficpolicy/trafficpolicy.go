package trafficpolicy

import (
	"fmt"
	"reflect"
	"sort"

	mapset "github.com/deckarep/golang-set"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
)

// WildCardRouteMatch represents a wildcard HTTP route match condition
var WildCardRouteMatch = HTTPRouteMatch{
	Path:          constants.RegexMatchAll,
	PathMatchType: PathMatchRegex,
	Methods:       []string{constants.WildcardHTTPMethod},
}

// NewRouteWeightedCluster takes a route, weighted cluster, UpstreamTrafficSetting and returns a *RouteWeightedCluster
func NewRouteWeightedCluster(route HTTPRouteMatch, weightedClusters []service.WeightedCluster) *RouteWeightedClusters {
	weightedClusterSet := mapset.NewSet()
	for _, wc := range weightedClusters {
		weightedClusterSet.Add(wc)
	}
	routeWC := &RouteWeightedClusters{
		HTTPRouteMatch:   route,
		WeightedClusters: weightedClusterSet,
	}
	return routeWC
}

// NewInboundTrafficPolicy takes a name, list of hostnames, UpstreamTrafficSetting, and returns an *InboundTrafficPolicy
func NewInboundTrafficPolicy(name string, hostnames []string) *InboundTrafficPolicy {
	policy := &InboundTrafficPolicy{
		Name:      name,
		Hostnames: hostnames,
	}
	return policy
}

// NewOutboundTrafficPolicy takes a name and list of hostnames and returns an *OutboundTrafficPolicy
func NewOutboundTrafficPolicy(name string, hostnames []string) *OutboundTrafficPolicy {
	return &OutboundTrafficPolicy{
		Name:      name,
		Hostnames: hostnames,
	}
}

// TotalClustersWeight returns total weight of the WeightedClusters in RouteWeightedClusters
func (rwc *RouteWeightedClusters) TotalClustersWeight() int {
	var totalWeight int
	for clusterInterface := range rwc.WeightedClusters.Iter() { // iterate
		cluster := clusterInterface.(service.WeightedCluster)
		totalWeight += cluster.Weight
	}
	return totalWeight
}

// AddRoute adds a route to an OutboundTrafficPolicy given an HTTP route match and weighted cluster. If a Route with the given HTTP route match
//
// already exists, an error will be returned. If a Route with the given HTTP route match does not exist,
// a Route with the given HTTP route match and weighted clusters will be added to the Routes on the OutboundTrafficPolicy
func (out *OutboundTrafficPolicy) AddRoute(httpRouteMatch HTTPRouteMatch, weightedClusters ...service.WeightedCluster) error {
	wc := mapset.NewSet()
	for _, c := range weightedClusters {
		wc.Add(c)
	}

	for _, existingRoute := range out.Routes {
		if reflect.DeepEqual(existingRoute.HTTPRouteMatch, httpRouteMatch) {
			if existingRoute.WeightedClusters.Equal(wc) {
				return nil
			}
			return fmt.Errorf("Route for HTTP Route Match: %v already exists: %v for outbound traffic policy: %s", existingRoute.HTTPRouteMatch, existingRoute, out.Name)
		}
	}

	out.Routes = append(out.Routes, &RouteWeightedClusters{
		HTTPRouteMatch:   httpRouteMatch,
		WeightedClusters: wc,
	})

	return nil
}

// MergeInboundPolicies merges latest InboundTrafficPolicies into a slice of InboundTrafficPolicies that already exists (original)
// allowPartialHostnamesMatch when set to true merges inbound policies by partially comparing (subset of one another) the hostnames of the original traffic policy to the latest traffic policy
// A partial match on hostnames should be allowed for the following scenarios :
// 1. when an ingress policy is being merged with other ingress traffic policies or
// 2. when a policy having its hostnames from a host header needs to be merged with other inbound traffic policies
// in either of these cases the will be only a single hostname and there is a possibility that this hostname is part of an existing traffic policy
// hence the rules need to be merged
func MergeInboundPolicies(original []*InboundTrafficPolicy, latest ...*InboundTrafficPolicy) []*InboundTrafficPolicy {
	for _, l := range latest {
		foundHostnames := false
		for _, or := range original {
			// If l.Hostnames is a subset of or.Hostnames or vice versa then we need to get a union of the two
			if hostsUnion := slicesUnionIfSubset(or.Hostnames, l.Hostnames); len(hostsUnion) > 0 {
				or.Hostnames = hostsUnion
				foundHostnames = true
				or.Rules = MergeRules(or.Rules, l.Rules)
			}
		}
		if !foundHostnames {
			original = append(original, l)
		}
	}
	return original
}

// MergeRules merges the give slices of rules such that there is one Rule for a Route with all allowed service accounts listed in the
//
// returned slice of rules
func MergeRules(originalRules, latestRules []*Rule) []*Rule {
	for _, latest := range latestRules {
		foundRoute := false
		for _, original := range originalRules {
			if reflect.DeepEqual(latest.Route, original.Route) {
				foundRoute = true
				original.AllowedPrincipals = original.AllowedPrincipals.Union(latest.AllowedPrincipals)
				break
			}
		}
		if !foundRoute {
			originalRules = append(originalRules, latest)
		}
	}
	return originalRules
}

// slicesUnionIfSubset returns the union of the two slices if either slices is a subset of the other
func slicesUnionIfSubset(first, second []string) []string {
	areSubsets := false
	var unionSlice []string
	firstIntf := convertToInterface(first)
	secondIntf := convertToInterface(second)

	firstSet := mapset.NewSetFromSlice(firstIntf)
	secondSet := mapset.NewSetFromSlice(secondIntf)

	if firstSet.IsSubset(secondSet) || secondSet.IsSubset(firstSet) {
		areSubsets = true
	}

	if areSubsets {
		union := firstSet.Union(secondSet)
		for intf := range union.Iter() {
			unionSlice = append(unionSlice, intf.(string))
		}
		sort.Strings(unionSlice)
		return unionSlice
	}
	return unionSlice
}

func convertToInterface(slice []string) []interface{} {
	sliceInterface := make([]interface{}, len(slice))
	for i := range slice {
		sliceInterface[i] = slice[i]
	}
	return sliceInterface
}
