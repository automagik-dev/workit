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
		"respond": true, "focus-time": true, "out-of-office": true,
		"working-location": true,
	},
	"docs": {
		"create": true, "update": true, "write": true, "insert": true,
		"delete": true, "copy": true, "find-replace": true,
		"generate": true,
	},
	"slides": {
		"create": true, "copy": true,
		"add-slide": true, "delete-slide": true,
		"update-notes": true, "replace-slide": true,
		"create-from-markdown": true,
	},
	"sheets": {
		"update": true, "append": true, "create": true, "clear": true,
		"format": true, "copy": true,
		"add-tab": true, "batch-update": true,
	},
	"contacts": {
		"create": true, "update": true, "delete": true, "batch": true,
	},
	"tasks": {
		"add": true, "update": true, "done": true, "undo": true,
		"delete": true, "clear": true,
	},
	"forms": {
		"create": true, "publish": true,
	},
	"keep": {
		"create": true, "delete": true,
	},
	"appscript": {
		"run": true, "create": true,
	},
}

// writeDesirePaths are top-level commands that are write operations.
// "update" replaces the wk binary itself.
var writeDesirePaths = map[string]bool{
	"send":   true,
	"upload": true,
	"update": true,
}

// readOnlyAllowedLeaves lists services where every subcommand is a write
// except the named leaves. `wk sync` runs a two-way daemon that uploads local
// changes and edits local sync state, so under --read-only only its
// inspection leaves stay available (including "sync service status").
var readOnlyAllowedLeaves = map[string]map[string]bool{
	"sync": {"status": true, "list": true},
}

// writeCommandPaths are full command paths that write although no token in
// them is a write verb.
var writeCommandPaths = map[string]bool{
	"chat dm space":                     true, // finds or creates a DM space
	"classroom announcements assignees": true,
	"classroom coursework assignees":    true,
	"gmail track setup":                 true, // stores tracking keys; --deploy provisions a worker
}

// readOnlyWriteFlags maps commands that read by default to the flags that make
// them write. Under --read-only the command stays available without them.
var readOnlyWriteFlags = map[string][]string{
	"calendar propose-time": {"decline", "comment"},
	"docs footer":           {"set", "clear"},
	"docs header":           {"set", "clear"},
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
	"turn-in": true, "return": true, "reclaim": true, "grade": true,
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
	if err := readOnlyCheck(cmd); err != nil {
		return err
	}
	return readOnlyFlagCheck(kctx, cmd)
}

// commandNames drops positional placeholders ("<fileId>") from a command path.
func commandNames(cmd []string) []string {
	names := make([]string, 0, len(cmd))
	for _, token := range cmd {
		if strings.HasPrefix(token, "<") {
			continue
		}
		names = append(names, strings.ToLower(token))
	}
	return names
}

// readOnlyFlagCheck blocks read commands that were given a flag that turns
// them into a write (see readOnlyWriteFlags).
func readOnlyFlagCheck(kctx *kong.Context, cmd []string) error {
	path := strings.Join(commandNames(cmd), " ")
	flags := readOnlyWriteFlags[path]
	if len(flags) == 0 {
		return nil
	}
	for _, el := range kctx.Path {
		if el.Flag == nil {
			continue
		}
		for _, name := range flags {
			if el.Flag.Name == name {
				return usagef("command %q with --%s is unavailable in read-only mode", path, name)
			}
		}
	}
	return nil
}

// readOnlyCheck reports whether the command path cmd (as returned by
// kong.Context.Command, split on whitespace) is a write that --read-only must
// block.
func readOnlyCheck(cmd []string) error {
	cmd = commandNames(cmd)
	if len(cmd) == 0 {
		return nil
	}

	topCmd := cmd[0]
	path := strings.Join(cmd, " ")

	if writeCommandPaths[path] {
		return usagef("command %q is unavailable in read-only mode", path)
	}

	if allowed, ok := readOnlyAllowedLeaves[topCmd]; ok {
		if len(cmd) < 2 || !allowed[cmd[len(cmd)-1]] {
			return usagef("command %q is unavailable in read-only mode", path)
		}
		return nil
	}

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

		// Check all remaining command path tokens for write verbs.
		// This catches nested writes at any depth (e.g., "chat messages send",
		// "gmail settings delegates add", "classroom courses archive").
		for i := 2; i < len(cmd); i++ {
			token := strings.ToLower(cmd[i])
			if nestedWriteVerbs[token] {
				return usagef("command %q is unavailable in read-only mode", strings.Join(cmd, " "))
			}
		}
	}

	return nil
}
