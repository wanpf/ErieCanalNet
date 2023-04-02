package config

var (
	// KernelTracing indicates debug feature of/off
	KernelTracing = false
	// IsKind indicates Kubernetes running in Docker
	IsKind = false
	// BridgeEth indicates cni bridge dev
	BridgeEth string
	// HostProc defines HostProc volume
	HostProc string
	// CNIBinDir defines CNIBIN volume
	CNIBinDir string
	// CNIConfigDir defines CNIConfig volume
	CNIConfigDir string
	// HostVarRun defines HostVar volume
	HostVarRun string
)
