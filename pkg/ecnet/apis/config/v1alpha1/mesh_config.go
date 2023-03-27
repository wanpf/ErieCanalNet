package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MeshConfig is the type used to represent the mesh configuration.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfig struct {
	// Object's type metadata.
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// Object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec is the MeshConfig specification.
	// +optional
	Spec MeshConfigSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// MeshConfigSpec is the spec for ECNET's configuration.
type MeshConfigSpec struct {
	// Sidecar defines the configurations of the proxy sidecar in a mesh.
	Sidecar SidecarSpec `json:"sidecar,omitempty"`

	// RepoServer defines the configurations of pipy repo server.
	RepoServer RepoServerSpec `json:"repoServer,omitempty"`

	// PluginChains defines the default plugin chains.
	PluginChains PluginChainsSpec `json:"pluginChains,omitempty"`
}

// LocalDNSProxy is the type to represent ECNET's local DNS proxy configuration.
type LocalDNSProxy struct {
	// Enable defines a boolean indicating if the sidecars are enabled for local DNS Proxy.
	Enable bool `json:"enable"`

	// PrimaryUpstreamDNSServerIPAddr defines a primary upstream DNS server for local DNS Proxy.
	PrimaryUpstreamDNSServerIPAddr string `json:"primaryUpstreamDNSServerIPAddr,omitempty"`

	// SecondaryUpstreamDNSServerIPAddr defines a secondary upstream DNS server for local DNS Proxy.
	SecondaryUpstreamDNSServerIPAddr string `json:"secondaryUpstreamDNSServerIPAddr,omitempty"`
}

// SidecarSpec is the type used to represent the specifications for the proxy sidecar.
type SidecarSpec struct {
	// LogLevel defines the logging level for the sidecar's logs. Non developers should generally never set this value. In production environments the LogLevel should be set to error.
	LogLevel string `json:"logLevel,omitempty"`

	// ProxyServerPort is the port on which the Discovery Service listens for new connections from Sidecars
	ProxyServerPort uint32 `json:"proxyServerPort"`

	// ConfigResyncInterval defines the resync interval for regular proxy broadcast updates.
	ConfigResyncInterval string `json:"configResyncInterval,omitempty"`

	// LocalDNSProxy improves the performance of your computer by caching the responses coming from your DNS servers
	LocalDNSProxy LocalDNSProxy `json:"localDNSProxy,omitempty"`
}

// TracingSpec is the type to represent ECNET's tracing configuration.
type TracingSpec struct {
	// Enable defines a boolean indicating if the sidecars are enabled for tracing.
	Enable bool `json:"enable"`

	// Port defines the tracing collector's port.
	Port int16 `json:"port,omitempty"`

	// Address defines the tracing collectio's hostname.
	Address string `json:"address,omitempty"`

	// Endpoint defines the API endpoint for tracing requests sent to the collector.
	Endpoint string `json:"endpoint,omitempty"`

	// SampledFraction defines the sampled fraction.
	SampledFraction *string `json:"sampledFraction,omitempty"`
}

// RemoteLoggingSpec is the type to represent ECNET's remote logging configuration.
type RemoteLoggingSpec struct {
	// Enable defines a boolean indicating if the sidecars are enabled for remote logging.
	Enable bool `json:"enable"`

	// Port defines the remote logging's port.
	Port int16 `json:"port,omitempty"`

	// Address defines the remote logging's hostname.
	Address string `json:"address,omitempty"`

	// Endpoint defines the API endpoint for remote logging requests sent to the collector.
	Endpoint string `json:"endpoint,omitempty"`

	// Authorization defines the access entity that allows to authorize someone in remote logging service.
	Authorization string `json:"authorization,omitempty"`

	// SampledFraction defines the sampled fraction.
	SampledFraction *string `json:"sampledFraction,omitempty"`
}

// IngressGatewayCertSpec is the type to represent the certificate specification for an ingress gateway.
type IngressGatewayCertSpec struct {
	// SubjectAltNames defines the Subject Alternative Names (domain names and IP addresses) secured by the certificate.
	SubjectAltNames []string `json:"subjectAltNames"`

	// ValidityDuration defines the validity duration of the certificate.
	ValidityDuration string `json:"validityDuration"`

	// Secret defines the secret in which the certificate is stored.
	Secret corev1.SecretReference `json:"secret"`
}

// MeshConfigList lists the MeshConfig objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MeshConfig `json:"items"`
}

// RepoServerSpec is the type to represent repo server.
type RepoServerSpec struct {
	// IPAddr of the pipy repo server
	IPAddr string `json:"ipaddr"`

	// Codebase is the folder used by ecnetController
	Codebase string `json:"codebase"`
}

// PluginChainsSpec is the type to represent plugin chains.
type PluginChainsSpec struct {
	// InboundTCPChains defines inbound tcp chains
	InboundTCPChains []*PluginChainSpec `json:"inbound-tcp"`

	// InboundHTTPChains defines inbound http chains
	InboundHTTPChains []*PluginChainSpec `json:"inbound-http"`

	// OutboundTCPChains defines outbound tcp chains
	OutboundTCPChains []*PluginChainSpec `json:"outbound-tcp"`

	// OutboundHTTPChains defines outbound http chains
	OutboundHTTPChains []*PluginChainSpec `json:"outbound-http"`
}

// PluginChainSpec is the type to represent plugin chain.
type PluginChainSpec struct {
	// Plugin defines the name of plugin
	Plugin string `json:"plugin"`

	// Priority defines the priority of plugin
	Priority float32 `json:"priority"`

	// Disable defines the visibility of plugin
	Disable bool `json:"disable"`
}
