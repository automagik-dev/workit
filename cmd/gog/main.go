package main

import (
	"os"

	"github.com/steipete/gogcli/internal/cmd"
)

func main() {
	if err := cmd.Execute(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
