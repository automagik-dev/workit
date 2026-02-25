package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/automagik-dev/workit/internal/outfmt"
)

var (
	version = "0.12.0-dev"
	branch  = ""
	commit  = ""
	date    = ""
)

func VersionString() string {
	v := strings.TrimSpace(version)
	if v == "" {
		v = buildVersionDev
	}

	metadata := make([]string, 0, 3)
	if b := strings.TrimSpace(branch); b != "" {
		metadata = append(metadata, b)
	}
	if c := strings.TrimSpace(commit); c != "" {
		metadata = append(metadata, c)
	}
	if d := strings.TrimSpace(date); d != "" {
		metadata = append(metadata, d)
	}

	if len(metadata) == 0 {
		return "Workit " + v
	}
	return fmt.Sprintf("Workit %s (%s)", v, strings.Join(metadata, " "))
}

type VersionCmd struct{}

func (c *VersionCmd) Run(ctx context.Context) error {
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"version": strings.TrimSpace(version),
			"branch":  strings.TrimSpace(branch),
			"commit":  strings.TrimSpace(commit),
			"date":    strings.TrimSpace(date),
		})
	}
	fmt.Fprintln(os.Stdout, VersionString())
	return nil
}
