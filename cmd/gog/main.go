// Package main provides a backward-compatible "gog" alias binary.
// It prints a deprecation warning and then delegates to the same
// underlying CLI as the "wk" binary.
package main

import (
	"fmt"
	"os"

	"github.com/namastexlabs/workit/internal/cmd"
	"github.com/namastexlabs/workit/internal/config"
)

func main() {
	fmt.Fprintln(os.Stderr, "WARNING: 'gog' is deprecated. Use 'wk' instead.")

	// Migrate legacy config dir if needed (same as wk entrypoint).
	_ = config.MigrateConfigDir()

	if err := cmd.Execute(os.Args[1:]); err != nil {
		os.Exit(cmd.ExitCode(err))
	}
}
