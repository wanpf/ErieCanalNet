// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config defines the constants that are used by multiple other packages within ECNET.
package config

const (
	// CNICreatePodURL is the route for cni plugin for creating pod
	CNICreatePodURL = "/v1/cni/create-pod"
	// CNIDeletePodURL is the route for cni plugin for deleting pod
	CNIDeletePodURL = "/v1/cni/delete-pod"
	// CNITransferFdStartURL is the route for cni plugin for transfer fd
	CNITransferFdStartURL = "/v1/cni/transfer-fd"

	// ECNetDNSNatEbpfMap is the mount point of ecnet_dns_nat map
	ECNetDNSNatEbpfMap = "/sys/fs/bpf/tc/globals/ecnet_dns_nat"
	// ECNetSVCNatEbpfMap is the mount point of ecnet_ssvc_nat map
	ECNetSVCNatEbpfMap = "/sys/fs/bpf/tc/globals/ecnet_ssvc_nat"
)
