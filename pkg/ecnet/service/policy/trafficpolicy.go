package policy

import (
	"fmt"
	"reflect"

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
