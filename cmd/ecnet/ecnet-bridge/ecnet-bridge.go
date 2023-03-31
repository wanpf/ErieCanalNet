// Package main implements ecnet bridge.
package main

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller/helpers"
	cniserver "github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/controller/server"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
)

var log = logger.New("ecent-bridge-cli")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "ecnet-bridge",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := helpers.LoadProgs(config.Debug); err != nil {
			return fmt.Errorf("failed to load ebpf programs: %v", err)
		}

		stop := make(chan struct{}, 1)
		cniReady := make(chan struct{}, 1)
		s := cniserver.NewServer(path.Join("/host", config.CNISock), "/sys/fs/bpf", cniReady, stop)
		if err := s.Start(); err != nil {
			log.Fatal().Err(err)
			return err
		}
		// todo: wait for stop
		if err := controller.Run(config.DisableWatcher, config.Skip, cniReady, stop); err != nil {
			log.Fatal().Err(err)
			return err
		}
		return nil
	},
}

func execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func main() {
	execute()
}

func init() {
	// Get some flags from commands
	rootCmd.PersistentFlags().BoolVarP(&config.Debug, "kernel-tracing", "d", false, "kernel tracing mode")
	rootCmd.PersistentFlags().BoolVarP(&config.Skip, "skip", "s", false, "Skip init bpf")
	rootCmd.PersistentFlags().BoolVarP(&config.DisableWatcher, "disableWatcher", "w", false, "disable Pod watcher")
	rootCmd.PersistentFlags().BoolVarP(&config.IsKind, "kind", "k", false, "Enable when Kubernetes is running in Kind")
	_ = rootCmd.PersistentFlags().MarkDeprecated("ips-file", "no need to collect node IPs")
	rootCmd.PersistentFlags().StringVar(&config.BridgeEth, "bridge-eth", "cni0", "bridge veth created by CNI")
	rootCmd.PersistentFlags().StringVar(&config.HostProc, "host-proc", "/host/proc", "/proc mount path")
	rootCmd.PersistentFlags().StringVar(&config.CNIBinDir, "cni-bin-dir", "/host/opt/cni/bin", "/opt/cni/bin mount path")
	rootCmd.PersistentFlags().StringVar(&config.CNIConfigDir, "cni-config-dir", "/host/etc/cni/net.d", "/etc/cni/net.d mount path")
	rootCmd.PersistentFlags().StringVar(&config.HostVarRun, "host-var-run", "/host/var/run", "/var/run mount path")
	rootCmd.PersistentFlags().StringVar(&config.KubeConfig, "kubeconfig", "", "Kubernetes configuration file")
	rootCmd.PersistentFlags().StringVar(&config.Context, "kubecontext", "", "The name of the kube config context to use")
}
