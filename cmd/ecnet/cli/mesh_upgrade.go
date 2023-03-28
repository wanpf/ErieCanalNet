package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
)

const upgradeDesc = `
This command upgrades an ECNET control plane by upgrading the
underlying Helm release.
`

const meshUpgradeExample = `
# Upgrade the mesh with the default name in the ecnet-system namespace, setting
# the image registry and tag to the defaults, and leaving all other values unchanged.
ecnet mesh upgrade --ecnet-namespace ecnet-system
`

type meshUpgradeCmd struct {
	out io.Writer

	ecnetName string
	chart     *chart.Chart

	setOptions []string
}

func newMeshUpgradeCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	upg := &meshUpgradeCmd{
		out: out,
	}
	var chartPath string

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "upgrade ecnet control plane",
		Long:    upgradeDesc,
		Example: meshUpgradeExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if chartPath != "" {
				var err error
				upg.chart, err = loader.Load(chartPath)
				if err != nil {
					return err
				}
			}

			return upg.run(config)
		},
	}

	f := cmd.Flags()

	f.StringVar(&upg.ecnetName, "ecnet-name", defaultEcnetName, "Name of the mesh to upgrade")
	f.StringVar(&chartPath, "ecnet-chart-path", "", "path to ecnet chart to override default chart")
	f.StringArrayVar(&upg.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	return cmd
}

func (u *meshUpgradeCmd) run(config *helm.Configuration) error {
	if u.chart == nil {
		var err error
		u.chart, err = loader.LoadArchive(bytes.NewReader(chartTGZSource))
		if err != nil {
			return err
		}
	}

	// Add the overlay values to be updated to the current release's values map
	values, err := u.resolveValues()
	if err != nil {
		return err
	}

	upgradeClient := helm.NewUpgrade(config)
	upgradeClient.Wait = true
	upgradeClient.Timeout = 5 * time.Minute
	upgradeClient.ResetValues = true
	if _, err = upgradeClient.Run(u.ecnetName, u.chart, values); err != nil {
		return err
	}

	fmt.Fprintf(u.out, "ECNET successfully upgraded mesh [%s] in namespace [%s]\n", u.ecnetName, settings.Namespace())
	return nil
}

func (u *meshUpgradeCmd) resolveValues() (map[string]interface{}, error) {
	vals := make(map[string]interface{})
	for _, val := range u.setOptions {
		if err := strvals.ParseInto(val, vals); err != nil {
			return nil, fmt.Errorf("invalid format for --set: %w", err)
		}
	}
	return vals, nil
}
