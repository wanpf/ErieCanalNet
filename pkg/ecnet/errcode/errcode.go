// Package errcode defines the error codes for error messages and an explanation
// of what the error signifies.
package errcode

import (
	"fmt"
)

// ErrCode defines the type to represent error codes
type ErrCode int

const (
	// Kind defines the kind for the error code constants
	Kind = "error_code"
)

// Range 1000-1050 is reserved for errors related to application startup or bootstrapping
const (
	// ErrInvalidCLIArgument indicates an invalid CLI argument
	ErrInvalidCLIArgument ErrCode = iota + 1000

	// ErrFetchingControllerPod indicates the ecnet-controller pod resource could not be fetched
	ErrFetchingControllerPod
)

// Range 2000-2500 is reserved for errors related to traffic policies
const (
	// ErrAddingRouteToOutboundTrafficPolicy indicates there was an error adding a route to an outbound traffic policy
	ErrAddingRouteToOutboundTrafficPolicy ErrCode = iota + 2000
)

// Range 4150-4200 reserved for errors related to config.openservicemesh.io resources
const (
	// ErrMeshConfigFetchFromCache indicates failed to fetch MeshConfig from cache with specific key
	ErrMeshConfigFetchFromCache ErrCode = iota + 4150

	// ErrMeshConfigMarshaling indicates failed to marshal MeshConfig into other format like JSON
	ErrMeshConfigMarshaling
)

// Range 5000-5500 reserved for errors related to Sidecar XDS control plane
const (
	// ErrFetchingPodFromCert indicates the proxy UUID obtained from a certificate's common name metadata was not
	// found as a ecnet-proxy-uuid label value for any pod
	ErrFetchingPodFromCert ErrCode = iota + 5000

	// ErrPodBelongsToMultipleServices indicates a pod in the mesh belongs to more than one service
	ErrPodBelongsToMultipleServices

	// ErrMismatchedServiceAccount inicates the ServiceAccount referenced in the NodeID does not match the
	// ServiceAccount specified in the proxy certificate
	ErrMismatchedServiceAccount
)

// String returns the error code as a string, ex. E1000
func (e ErrCode) String() string {
	return fmt.Sprintf("E%d", e)
}

// GetErrCodeWithMetric increments the ErrCodeCounter metric for the given error code
// Returns the error code as a string
func GetErrCodeWithMetric(e ErrCode) string {
	return e.String()
}
