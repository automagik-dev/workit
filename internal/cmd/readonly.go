package cmd

import (
	"strings"

	"github.com/alecthomas/kong"
)

// writeCommands maps service -> subcommand names that are write operations.
// Both canonical names (e.g. "move", "copy", "delete") and aliases
// (e.g. "mv", "cp", "rm") are listed so that enforceReadOnly blocks
// whichever name Kong resolves.
var writeCommands = map[string]map[string]bool{
	"gmail": {
		"send": true, "delete": true, "trash": true, "untrash": true,
		"modify": true, "batch": true,
	},
	"drive": {
		"upload": true, "mkdir": true,
		"mv": true, "move": true, "rename": true,
		"rm": true, "delete": true,
		"cp": true, "copy": true,
		"share": true, "unshare": true,
	},
	"calendar": {
		"create": true, "update": true, "delete": true,
	},
	"docs": {
		"create": true, "update": true, "write": true, "insert": true,
		"delete": true, "copy": true, "find-replace": true,
	},
	"slides": {
		"create": true, "copy": true,
		"add-slide": true, "delete-slide": true,
		"update-notes": true, "replace-slide": true,
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

// nestedWriteVerbs is the comprehensive set of verbs that indicate a write
// operation when they appear as a nested (depth >= 3) subcommand.
// This covers verbs found in actual subcommands (e.g. chat messages send,
// docs comments reply, gmail delegates remove) plus defensive entries for
// plausible future write verbs.
var nestedWriteVerbs = map[string]bool{
	// Core CRUD verbs
	"send": true, "create": true, "delete": true, "update": true,
	"post": true, "add": true, "remove": true,
	// Mutation verbs found in the codebase
	"modify": true, "reply": true, "resolve": true, "verify": true,
	"set": true, "unset": true,
	"start": true, "stop": true, "renew": true, "serve": true,
	"turn-in": true,
	// Movement / structural verbs
	"move": true, "copy": true, "rename": true, "transfer": true,
	// Sharing / access verbs
	"share": true, "unshare": true,
	// State-change verbs
	"archive": true, "unarchive": true,
	"publish": true, "unpublish": true,
	"submit": true, "accept": true, "decline": true,
	"join": true, "leave": true,
	// Data manipulation verbs
	"clear": true, "append": true, "prepend": true,
	"write": true, "insert": true,
	"format": true, "find-replace": true,
	// Task-specific verbs
	"done": true, "undo": true,
	// Drive verbs
	"upload": true, "mkdir": true, "trash": true, "untrash": true,
	// Other mutating verbs
	"batch": true, "run": true, "import": true,
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
			if nestedWriteVerbs[nestedCmd] {
				return usagef("command %q %q %q is unavailable in read-only mode", topCmd, subCmd, nestedCmd)
			}
		}
	}

	return nil
}
