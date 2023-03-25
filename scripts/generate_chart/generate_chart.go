// Package main implements generate chart application.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cli"
)

func main() {
	// Path relative to the Makefile where this is invoked.
	chartPath := filepath.Join("charts", "ecnet")
	source, err := cli.GetChartSource(chartPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error getting chart source:", err)
		os.Exit(1)
	}
	fmt.Print(string(source))
}
