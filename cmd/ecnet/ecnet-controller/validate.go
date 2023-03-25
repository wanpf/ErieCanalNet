package main

import (
	"fmt"
)

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return fmt.Errorf("Please specify the mesh name using --mesh-name")
	}

	if ecnetNamespace == "" {
		return fmt.Errorf("Please specify the ECNET namespace using --ecnet-namespace")
	}

	return nil
}
