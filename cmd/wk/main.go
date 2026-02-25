package main

import (
	"os"

	"github.com/namastexlabs/workit/internal/cmd"
	"github.com/namastexlabs/workit/internal/config"
)

func main() {
	// Migrate legacy ~/.config/gogcli/ → ~/.config/workit/ on first run.
	_ = config.MigrateConfigDir()

	if err := cmd.Execute(os.Args[1:]); err != nil {
		os.Exit(cmd.ExitCode(err))
	}
}
