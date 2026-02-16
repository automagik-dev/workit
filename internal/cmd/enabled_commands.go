package cmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v3"
)

//go:embed command_tiers.yaml
var commandTiersYAML []byte

// tierLevel maps tier names to numeric levels for comparison.
var tierLevel = map[string]int{
	"core":     1,
	"extended": 2,
	"complete": 3,
}

// alwaysVisibleCommands are utility commands not subject to tier filtering.
var alwaysVisibleCommands = map[string]bool{
	"auth": true, "config": true, "time": true, "agent": true,
	"schema": true, "sync": true, "version": true, "completion": true,
	"__complete": true, "exit-codes": true, "open": true,
	// Top-level desire paths (aliases) are always visible.
	"send": true, "ls": true, "search": true, "download": true,
	"upload": true, "login": true, "logout": true, "status": true,
	"me": true, "whoami": true,
}

func enforceEnabledCommands(kctx *kong.Context, enabled string) error {
	enabled = strings.TrimSpace(enabled)
	if enabled == "" {
		return nil
	}
	allow := parseEnabledCommands(enabled)
	if len(allow) == 0 {
		return nil
	}
	if allow["*"] || allow["all"] {
		return nil
	}
	cmd := strings.Fields(kctx.Command())
	if len(cmd) == 0 {
		return nil
	}
	top := strings.ToLower(cmd[0])
	if !allow[top] {
		return usagef("command %q is not enabled (set --enable-commands to allow it)", top)
	}
	return nil
}

func parseEnabledCommands(value string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		out[part] = true
	}
	return out
}

// parseCommandTiers parses the embedded YAML into the tiers map.
func parseCommandTiers() (map[string]map[string]string, error) {
	var tiers map[string]map[string]string
	if err := yaml.Unmarshal(commandTiersYAML, &tiers); err != nil {
		return nil, fmt.Errorf("parse command tiers: %w", err)
	}
	return tiers, nil
}

func enforceCommandTier(kctx *kong.Context, tier string) error {
	tier = strings.TrimSpace(strings.ToLower(tier))
	if tier == "" || tier == "complete" {
		return nil
	}

	requestedLevel, ok := tierLevel[tier]
	if !ok {
		return usagef("invalid command tier %q (expected core|extended|complete)", tier)
	}

	tiers, err := parseCommandTiers()
	if err != nil {
		return err
	}

	// Get the command path.
	cmd := strings.Fields(kctx.Command())
	if len(cmd) == 0 {
		return nil
	}

	topCmd := strings.ToLower(cmd[0])

	// Always-visible commands bypass tier check.
	if alwaysVisibleCommands[topCmd] {
		return nil
	}

	// If we have a subcommand, check its tier.
	if len(cmd) >= 2 {
		subCmd := strings.ToLower(cmd[1])
		if serviceTiers, ok := tiers[topCmd]; ok {
			if cmdTier, ok := serviceTiers[subCmd]; ok {
				cmdLevel := tierLevel[cmdTier]
				if cmdLevel > requestedLevel {
					return usagef("command %q %q requires tier %q (current: %q)", topCmd, subCmd, cmdTier, tier)
				}
				return nil
			}
			// Commands not in YAML default to "complete" tier,
			// so they are hidden in core/extended mode.
			if requestedLevel < tierLevel["complete"] {
				return usagef("command %q %q requires tier %q (current: %q)", topCmd, subCmd, "complete", tier)
			}
		}
	}

	return nil
}
