package conf

import (
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/policy"
	v1 "k8s.io/api/core/v1"
	"time"
)

// Protocol is a string wrapper type
type Protocol string

// Address is a string wrapper type
type Address string

// Port is a uint16 wrapper type
type Port uint16

// Weight is a uint32 wrapper type
type Weight uint32

// ClusterName is a string wrapper type
type ClusterName string

// WeightedEndpoint is a wrapper type of map[HTTPHostPort]Weight
type WeightedEndpoint map[HTTPHostPort]Weight

// Header is a string wrapper type
type Header string

// HeaderRegexp is a string wrapper type
type HeaderRegexp string

// Headers is a wrapper type of map[Header]HeaderRegexp
type Headers map[Header]HeaderRegexp

// Method is a string wrapper type
type Method string

// Methods is a wrapper type of []Method
type Methods []Method

// WeightedClusters is a wrapper type of map[ClusterName]Weight
type WeightedClusters map[ClusterName]Weight

// URIPathValue is a uri value wrapper
type URIPathValue string

// URIMatchType is a match type wrapper
type URIMatchType string

// PluginSlice plugin array
type PluginSlice []policy.Plugin

// URIPath is a uri wrapper type
type URIPath struct {
	Value URIPathValue
	Type  URIMatchType
}

// ServiceName is a string wrapper type
type ServiceName string

// Services is a wrapper type of []ServiceName
type Services []ServiceName

// HTTPMatchRule http match rule
type HTTPMatchRule struct {
	Path             URIPathValue
	Type             URIMatchType
	Headers          Headers `json:"Headers"`
	Methods          Methods `json:"Methods"`
	allowedAnyMethod bool
}

// HTTPRouteRule http route rule
type HTTPRouteRule struct {
	HTTPMatchRule
	TargetClusters WeightedClusters `json:"TargetClusters"`
}

// HTTPRouteRuleName is a string wrapper type
type HTTPRouteRuleName string

// HTTPHostPort is a string wrapper type
type HTTPHostPort string

// HTTPHostPort2Service is a wrapper type of map[HTTPHostPort]HTTPRouteRuleName
type HTTPHostPort2Service map[HTTPHostPort]HTTPRouteRuleName

// AllowedEndpoints is a wrapper type of map[Address]ServiceName
type AllowedEndpoints map[Address]ServiceName

// UpstreamDNSServers defines upstream DNS servers for local DNS Proxy.
type UpstreamDNSServers struct {
	// Primary defines a primary upstream DNS server for local DNS Proxy.
	Primary *string `json:"Primary,omitempty"`
	// Secondary defines a secondary upstream DNS server for local DNS Proxy.
	Secondary *string `json:"Secondary,omitempty"`
}

// LocalDNSProxy is the type to represent ECNET's local DNS proxy configuration.
type LocalDNSProxy struct {
	// UpstreamDNSServers defines upstream DNS servers for local DNS Proxy.
	UpstreamDNSServers *UpstreamDNSServers `json:"UpstreamDNSServers,omitempty"`
}

// MeshConfigSpec represents the spec of mesh config
type MeshConfigSpec struct {
	SidecarLogLevel string
	Probes          struct {
		ReadinessProbes []v1.Probe `json:"ReadinessProbes,omitempty"`
		LivenessProbes  []v1.Probe `json:"LivenessProbes,omitempty"`
		StartupProbes   []v1.Probe `json:"StartupProbes,omitempty"`
	}
	LocalDNSProxy *LocalDNSProxy `json:"LocalDNSProxy,omitempty"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	service.WeightedCluster
}

// OutboundHTTPRouteRule http route rule
type OutboundHTTPRouteRule struct {
	HTTPRouteRule
}

// OutboundHTTPRouteRuleSlice http route rule array
type OutboundHTTPRouteRuleSlice []*OutboundHTTPRouteRule

// OutboundHTTPRouteRules is a wrapper type
type OutboundHTTPRouteRules struct {
	RouteRules OutboundHTTPRouteRuleSlice `json:"RouteRules"`
}

// OutboundHTTPServiceRouteRules is a wrapper type of map[HTTPRouteRuleName]*HTTPRouteRules
type OutboundHTTPServiceRouteRules map[HTTPRouteRuleName]*OutboundHTTPRouteRules

// OutboundTCPServiceRouteRules is a wrapper type
type OutboundTCPServiceRouteRules struct {
	TargetClusters WeightedClusters `json:"TargetClusters"`
}

// OutboundTrafficMatch represents the match of OutboundTraffic
type OutboundTrafficMatch struct {
	Port                  Port                          `json:"Port"`
	Protocol              Protocol                      `json:"Protocol"`
	HTTPHostPort2Service  HTTPHostPort2Service          `json:"HttpHostPort2Service"`
	HTTPServiceRouteRules OutboundHTTPServiceRouteRules `json:"HttpServiceRouteRules"`
	TCPServiceRouteRules  *OutboundTCPServiceRouteRules `json:"TcpServiceRouteRules"`
}

// OutboundTrafficMatchSlice is a wrapper type of []*OutboundTrafficMatch
type OutboundTrafficMatchSlice []*OutboundTrafficMatch

// OutboundTrafficMatches is a wrapper type of map[Port][]*OutboundTrafficMatch
type OutboundTrafficMatches map[Port]OutboundTrafficMatchSlice

// WeightedZoneEndpoint represents the endpoint with zone and weight
type WeightedZoneEndpoint struct {
	Weight      Weight `json:"Weight"`
	Cluster     string `json:"Key,omitempty"`
	LBType      string `json:"-"`
	ContextPath string `json:"Path,omitempty"`
}

// WeightedEndpoints is a wrapper type of map[HTTPHostPort]WeightedZoneEndpoint
type WeightedEndpoints map[HTTPHostPort]*WeightedZoneEndpoint

// ClusterConfigs represents the configs of Cluster
type ClusterConfigs struct {
	Endpoints *WeightedEndpoints `json:"Endpoints"`
}

// OutboundTrafficPolicy represents the policy of OutboundTraffic
type OutboundTrafficPolicy struct {
	TrafficMatches  OutboundTrafficMatches          `json:"TrafficMatches"`
	ClustersConfigs map[ClusterName]*ClusterConfigs `json:"ClustersConfigs"`
}

// ProxyConf is a policy used by pipy proxy
type ProxyConf struct {
	Ts           *time.Time
	Version      *string
	Spec         MeshConfigSpec
	Outbound     *OutboundTrafficPolicy `json:"Outbound"`
	Chains       map[string][]string    `json:"Chains,omitempty"`
	DNSResolveDB map[string][]string    `json:"DNSResolveDB,omitempty"`
	BridgeV4Addr string
}
