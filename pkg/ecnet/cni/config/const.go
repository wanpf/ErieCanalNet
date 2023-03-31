// Package config defines the constants that are used by multiple other packages within ECNET.
package config

const (
	// CNISock defines the sock file
	CNISock = "/var/run/osm-cni.sock"

	// CNICreatePodURL is the route for cni plugin for creating pod
	CNICreatePodURL = "/v1/cni/create-pod"
	// CNIDeletePodURL is the route for cni plugin for deleting pod
	CNIDeletePodURL = "/v1/cni/delete-pod"

	// ECNetDNSNatEbpfMap is the mount point of ecnet_dns_nat map
	ECNetDNSNatEbpfMap = "/sys/fs/bpf/tc/globals/ecnet_dns_nat"
	// ECNetSVCNatEbpfMap is the mount point of ecnet_svc_nat map
	ECNetSVCNatEbpfMap = "/sys/fs/bpf/tc/globals/ecnet_svc_nat"
)
