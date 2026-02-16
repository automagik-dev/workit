package cmd

import (
	"strings"

	"github.com/alecthomas/kong"
)

// writeCommands maps service -> subcommand names that are write operations.
var writeCommands = map[string]map[string]bool{
	"gmail": {
		"send": true, "delete": true, "trash": true, "untrash": true,
		"modify": true, "batch": true,
	},
	"drive": {
		"upload": true, "mkdir": true, "mv": true, "rm": true,
		"cp": true, "share": true,
	},
	"calendar": {
		"create": true, "update": true, "delete": true,
	},
	"docs": {
		"create": true, "update": true, "write": true, "insert": true,
	},
	"slides": {
		"create": true,
	},
	"sheets": {
		"update": true, "append": true, "create": true, "clear": true,
		"format": true, "copy": true,
	},
	"contacts": {
		"create": true, "update": true, "delete": true, "batch": true,
	},
	"tasks": {
		"add": true, "update": true, "done": true, "undo": true,
		"delete": true, "clear": true,
	},
	"forms": {
		"create": true,
	},
	"appscript": {
		"run": true, "create": true,
	},
}

// writeDesirePaths are top-level desire paths that are write operations.
var writeDesirePaths = map[string]bool{
	"send":   true,
	"upload": true,
}

func enforceReadOnly(kctx *kong.Context, readOnly bool) error {
	if !readOnly {
		return nil
	}

	cmd := strings.Fields(kctx.Command())
	if len(cmd) == 0 {
		return nil
	}

	topCmd := strings.ToLower(cmd[0])

	// Check top-level desire paths.
	if writeDesirePaths[topCmd] {
		return usagef("command %q is unavailable in read-only mode", topCmd)
	}

	// Check service subcommands.
	if len(cmd) >= 2 {
		subCmd := strings.ToLower(cmd[1])
		if writes, ok := writeCommands[topCmd]; ok {
			if writes[subCmd] {
				return usagef("command %q %q is unavailable in read-only mode", topCmd, subCmd)
			}
		}

		// Handle nested write commands (e.g., chat messages send).
		if len(cmd) >= 3 {
			nestedCmd := strings.ToLower(cmd[2])
			if nestedCmd == "send" || nestedCmd == "create" || nestedCmd == "delete" || nestedCmd == "update" || nestedCmd == "post" {
				return usagef("command %q %q %q is unavailable in read-only mode", topCmd, subCmd, nestedCmd)
			}
		}
	}

	return nil
}
