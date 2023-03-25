package trafficpolicy

// Plugin defines plugin
type Plugin struct {
	// Name defines the Name of the plugin.
	Name string

	// priority defines the priority of the plugin.
	Priority float32

	// Script defines pipy script used by the PlugIn.
	Script string

	// BuildIn indicates PlugIn type.
	BuildIn bool
}
