//go:build debug

package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/spf13/cobra"
)

func registerPprofFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVar(target, "pprof", false, "Enable pprof profiling server on localhost:6060")
}

func startPprofIfEnabled(enabled bool) {
	if enabled {
		go http.ListenAndServe("localhost:6060", nil) //nolint:errcheck,gosec
	}
}
