//go:build !debug

package main

import "github.com/spf13/cobra"

func registerPprofFlag(_ *cobra.Command, _ *bool) {
	// pprof is not available in production builds (requires -tags debug).
}

func startPprofIfEnabled(_ bool) {
	// pprof is not available in production builds (requires -tags debug).
}
