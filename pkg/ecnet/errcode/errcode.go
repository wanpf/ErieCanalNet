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

// Range 4150-4200 reserved for errors related to config.flomesh.io resources
const (
	// ErrEcnetConfigFetchFromCache indicates failed to fetch EcnetConfig from cache with specific key
	ErrEcnetConfigFetchFromCache ErrCode = iota + 4150

	// ErrEcnetConfigMarshaling indicates failed to marshal EcnetConfig into other format like JSON
	ErrEcnetConfigMarshaling
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
