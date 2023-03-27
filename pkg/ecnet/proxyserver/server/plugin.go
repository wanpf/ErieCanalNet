package server

import (
	"fmt"
	"sort"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
)

// getPluginURI return the URI of the plugin.
func getPluginURI(name string) string {
	return fmt.Sprintf("plugins/%s.js", name)
}

func setSidecarChain(cfg configurator.Configurator, pipyConf *PipyConf) {
	pluginChains := cfg.GetGlobalPluginChains()
	pipyConf.Chains = make(map[string][]string)
	for mountPoint, pluginItems := range pluginChains {
		pluginSlice := PluginSlice(pluginItems)
		if len(pluginSlice) > 0 {
			var pluginURIs []string
			sort.Sort(&pluginSlice)
			for _, pluginItem := range pluginItems {
				if pluginItem.BuildIn {
					pluginURIs = append(pluginURIs, fmt.Sprintf("%s.js", pluginItem.Name))
				} else {
					pluginURIs = append(pluginURIs, getPluginURI(pluginItem.Name))
				}
			}
			pipyConf.Chains[mountPoint] = pluginURIs
		}
	}
}
