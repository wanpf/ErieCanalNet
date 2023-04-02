// Package main implements ecnet bridge.
package main

import (
	"flag"
	"os"
	"path"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller/helpers"
	cniserver "github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller/server"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s/events"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/version"
)

var (
	verbosity      string
	kubeConfigFile string
	ecnetVersion   string

	scheme = runtime.NewScheme()

	flags = pflag.NewFlagSet(`ecnet-bridge`, pflag.ExitOnError)
	log   = logger.New("ecnet/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&ecnetVersion, "ecnet-version", "", "Version of ECNET")

	// Get some flags from commands
	flags.BoolVarP(&config.KernelTracing, "kernel-tracing", "d", false, "kernel tracing mode")
	flags.BoolVarP(&config.IsKind, "kind", "k", false, "Enable when Kubernetes is running in Kind")
	flags.StringVar(&config.BridgeEth, "bridge-eth", "cni0", "bridge veth created by CNI")
	flags.StringVar(&config.HostProc, "host-proc", "/host/proc", "/proc mount path")
	flags.StringVar(&config.CNIBinDir, "cni-bin-dir", "/host/opt/cni/bin", "/opt/cni/bin mount path")
	flags.StringVar(&config.CNIConfigDir, "cni-config-dir", "/host/etc/cni/net.d", "/etc/cni/net.d mount path")
	flags.StringVar(&config.HostVarRun, "host-var-run", "/host/var/run", "/var/run mount path")

	_ = clientgoscheme.AddToScheme(scheme)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	return nil
}

func main() {
	log.Info().Msgf("Starting ecnet-bridge %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", kubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	if err = helpers.LoadProgs(config.KernelTracing); err != nil {
		log.Fatal().Msgf("failed to load ebpf programs: %v", err)
	}

	stop := make(chan struct{}, 1)
	cniReady := make(chan struct{}, 1)
	s := cniserver.NewServer(path.Join("/host", config.CNISock), "/sys/fs/bpf", cniReady, stop)
	if err = s.Start(); err != nil {
		log.Fatal().Err(err)
	}
	if err = controller.Run(kubeClient, stop); err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Msgf("Stopping ecnet-bridge %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
