package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const ecnetDescription = `
This command consists of multiple subcommands related to managing instances of
ecnet installations. Each installation receives a unique ecnet name.
`

func newCniCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cni",
		Short: "manage ecnet cni installations",
		Long:  ecnetDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newCniList(out))

	if !settings.IsManaged() {
		cmd.AddCommand(newCniUpgradeCmd(config, out))
	}

	return cmd
}
