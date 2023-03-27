// Package service implements types and utility routines related to the identity of a workload, as used within ECNET.
package service

import (
	"fmt"
)

const (
	// namespaceNameSeparator used for marshalling/unmarshalling MeshService to a string or vice versa
	namespaceNameSeparator = "/"
)

// K8sServiceAccount is a type for a namespaced service account
type K8sServiceAccount struct {
	Namespace string
	Name      string
}

// String returns the string representation of the service account object
func (sa K8sServiceAccount) String() string {
	return fmt.Sprintf("%s%s%s", sa.Namespace, namespaceNameSeparator, sa.Name)
}
