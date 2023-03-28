package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const ecnetDescription = `
This command consists of multiple subcommands related to managing instances of
ecnet installations. Each ecnet installation results in a mesh. Each installation
receives a unique ecnet name.

`

func newMeshCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ecnet",
		Short: "manage ecnet installations",
		Long:  ecnetDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newMeshList(out))

	if !settings.IsManaged() {
		cmd.AddCommand(newMeshUpgradeCmd(config, out))
	}

	return cmd
}
