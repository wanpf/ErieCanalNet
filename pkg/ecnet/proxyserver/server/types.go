package server

import (
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"

	mapset "github.com/deckarep/golang-set"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/catalog"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/pipy/repo/client"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver/registry"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/policy"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/workerpool"
)

var (
	log = logger.New("flomesh-pipy")
)

// Server implements the Aggregate Discovery Services
type Server struct {
	catalog        catalog.MeshCataloger
	proxyRegistry  *registry.ProxyRegistry
	ecnetNamespace string
	cfg            configurator.Configurator
	ready          bool
	workQueues     *workerpool.WorkerPool
	kubeController k8s.Controller

	// When snapshot cache is enabled, we (currently) don't keep track of proxy information, however different
	// config versions have to be provided to the cache as we keep adding snapshots. The following map
	// tracks at which version we are at given a proxy UUID
	configVerMutex sync.Mutex
	configVersion  map[string]uint64

	pluginSet mapset.Set

	msgBroker *messaging.Broker

	repoClient *client.PipyRepoClient

	retryProxiesJob func()
}

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

const (
	// PathMatchRegex is the type used to specify regex based path matching
	PathMatchRegex URIMatchType = "Regex"

	// PathMatchExact is the type used to specify exact path matching
	PathMatchExact URIMatchType = "Exact"

	// PathMatchPrefix is the type used to specify prefix based path matching
	PathMatchPrefix URIMatchType = "Prefix"
)

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

// namedOutboundTrafficMatches is a wrapper type of map[string]*OutboundTrafficMatch
type namedOutboundTrafficMatches map[string]*OutboundTrafficMatch

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
	namedTrafficMatches namedOutboundTrafficMatches
	TrafficMatches      OutboundTrafficMatches          `json:"TrafficMatches"`
	ClustersConfigs     map[ClusterName]*ClusterConfigs `json:"ClustersConfigs"`
}

// PipyConf is a policy used by pipy proxy
type PipyConf struct {
	Ts           *time.Time
	Version      *string
	Spec         MeshConfigSpec
	Outbound     *OutboundTrafficPolicy   `json:"Outbound"`
	Chains       map[string][]string      `json:"Chains,omitempty"`
	DNSResolveDB map[string][]interface{} `json:"DNSResolveDB,omitempty"`
}
