// Package constants defines the constants that are used by multiple other packages within ECNET.
package constants

const (
	// DefaultSidecarLogLevel is the default sidecar log level if not defined in the ecnet MeshConfig
	DefaultSidecarLogLevel = "error"

	// DefaultECNETLogLevel is the default ECNET log level if none is specified
	DefaultECNETLogLevel = "info"

	// ECNETHTTPServerPort is the port on which ecnet-controller and ecnet-injector serve HTTP requests for metrics, health probes etc.
	ECNETHTTPServerPort = 9091

	// ECNETControllerName is the name of the ECNET Controller (formerly ADS service).
	ECNETControllerName = "ecnet-controller"

	// ECNETBootstrapName is the name of the ECNET Bootstrap.
	ECNETBootstrapName = "ecnet-bootstrap"

	// DefaultCABundleSecretName is the default name of the secret for the ECNET CA bundle
	DefaultCABundleSecretName = "ecnet-ca-bundle" // #nosec G101: Potential hardcoded credentials

	// RegexMatchAll is a regex pattern match for all
	RegexMatchAll = ".*"

	// WildcardHTTPMethod is a wildcard for all HTTP methods
	WildcardHTTPMethod = "*"

	// ECNETKubeResourceMonitorAnnotation is the key of the annotation used to monitor a K8s resource
	ECNETKubeResourceMonitorAnnotation = "openservicemesh.io/monitored-by"

	// EnvVarHumanReadableLogMessages is an environment variable, which when set to "true" enables colorful human-readable log messages.
	EnvVarHumanReadableLogMessages = "ECNET_HUMAN_DEBUG_LOG"

	// ClusterWeightAcceptAll is the weight for a cluster that accepts 100 percent of traffic sent to it
	ClusterWeightAcceptAll = 100

	// ClusterWeightFailOver is the weight for a cluster that accepts 0 percent of traffic sent to it
	ClusterWeightFailOver = 0
)

// Annotations used by the control plane
const (
	// SidecarInjectionAnnotation is the annotation used for sidecar injection
	SidecarInjectionAnnotation = "openservicemesh.io/sidecar-injection"

	// MetricsAnnotation is the annotation used for enabling/disabling metrics
	MetricsAnnotation = "openservicemesh.io/metrics"
)

// Labels used by the control plane
const (
	// IgnoreLabel is the label used to ignore a resource
	IgnoreLabel = "openservicemesh.io/ignore"

	// AppLabel is the label used to identify the app
	AppLabel = "app"
)

// Annotations used for Metrics
const (
	// PrometheusScrapeAnnotation is the annotation used to configure prometheus scraping
	PrometheusScrapeAnnotation = "prometheus.io/scrape"
)

// App labels as defined in the "ecnet.labels" template in _helpers.tpl of the Helm chart.
const (
	ECNETAppInstanceLabelKey = "app.kubernetes.io/instance"
	ECNETAppVersionLabelKey  = "app.kubernetes.io/version"
)

// Application protocols
const (
	// HTTP protocol
	ProtocolHTTP = "http"

	// HTTPS protocol
	ProtocolHTTPS = "https"

	// TCP protocol
	ProtocolTCP = "tcp"

	// gRPC protocol
	ProtocolGRPC = "grpc"

	// ProtocolTCPServerFirst implies TCP based server first protocols
	// Ex. MySQL, SMTP, PostgreSQL etc. where the server initiates the first
	// byte in a TCP connection.
	ProtocolTCPServerFirst = "tcp-server-first"
)

// Control plane HTTP server paths
const (
	// ECNETControllerReadinessPath is the path at which ECNET controller serves readiness probes
	ECNETControllerReadinessPath = "/health/ready"

	// ECNETControllerLivenessPath is the path at which ECNET controller serves liveness probes
	ECNETControllerLivenessPath = "/health/alive"

	// VersionPath is the path at which ECNET controller serves version info
	VersionPath = "/version"

	// WebhookHealthPath is the path at which the webooks serve health probes
	WebhookHealthPath = "/healthz"
)

// ECNET HTTP Server Responses
const (
	// ServiceReadyResponse is the response returned by the server to indicate it is ready
	ServiceReadyResponse = "Service is ready"

	// ServiceAliveResponse is the response returned by the server to indicate it is alive
	ServiceAliveResponse = "Service is alive"
)

var (
	// SupportedProtocolsInMesh is a list of the protocols ECNET supports for in-mesh traffic
	SupportedProtocolsInMesh = []string{ProtocolTCPServerFirst, ProtocolHTTP, ProtocolTCP, ProtocolGRPC}
)

const (
	// SidecarClassPipy is the SidecarClass field value for context field.
	SidecarClassPipy = "pipy"
)
