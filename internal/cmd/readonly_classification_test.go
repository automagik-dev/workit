package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// readOnlyClass is the expected --read-only verdict for one leaf command.
type readOnlyClass int

const (
	// roRead reads Workspace/M365 data or wk state and stays available.
	roRead readOnlyClass = iota + 1
	// roWrite changes remote data, the wk binary, or runs the sync daemon: blocked.
	roWrite
	// roLocal only changes local wk state or local files (credentials, config,
	// .docx files on disk, templates). --read-only guards account data, and
	// `auth add` must stay usable to log in with read-only scopes, so these
	// stay available.
	roLocal
	// roFlagGated reads by default and writes only with the flags listed in
	// readOnlyWriteFlags; those flags are blocked.
	roFlagGated
)

// readOnlyClassification classifies every leaf command in the CLI. The walk in
// TestReadOnly_EveryCommandClassified fails when a command is missing here, so
// a new command cannot ship without an explicit read-only decision.
var readOnlyClassification = map[string]readOnlyClass{
	"__complete":                            roRead,
	"agent exit-codes":                      roRead,
	"agent help":                            roRead,
	"appscript content":                     roRead,
	"appscript create":                      roWrite,
	"appscript get":                         roRead,
	"appscript run":                         roWrite,
	"auth add":                              roLocal,
	"auth alias list":                       roRead,
	"auth alias set":                        roWrite, // local credential state; blocked by the nested-verb rule
	"auth alias unset":                      roWrite, // local credential state; blocked by the nested-verb rule
	"auth credentials list":                 roRead,
	"auth credentials set":                  roWrite, // local credential state; blocked by the nested-verb rule
	"auth keep":                             roLocal,
	"auth keyring":                          roLocal,
	"auth list":                             roRead,
	"auth m365 login-link":                  roRead,
	"auth manage":                           roLocal,
	"auth poll":                             roLocal,
	"auth remove":                           roLocal,
	"auth scopes":                           roRead,  // reads the stored token and asks Google for its granted scopes; writes nothing
	"auth service-account set":              roWrite, // local credential state; blocked by the nested-verb rule
	"auth service-account status":           roRead,
	"auth service-account unset":            roWrite, // local credential state; blocked by the nested-verb rule
	"auth services":                         roRead,
	"auth status":                           roRead,
	"auth tokens delete":                    roWrite, // local credential state; blocked by the nested-verb rule
	"auth tokens export":                    roLocal,
	"auth tokens import":                    roWrite, // local credential state; blocked by the nested-verb rule
	"auth tokens list":                      roRead,
	"calendar acl":                          roRead,
	"calendar calendars":                    roRead,
	"calendar colors":                       roRead,
	"calendar conflicts":                    roRead,
	"calendar create":                       roWrite,
	"calendar delete":                       roWrite,
	"calendar event":                        roRead,
	"calendar events":                       roRead,
	"calendar focus-time":                   roWrite,
	"calendar freebusy":                     roRead,
	"calendar out-of-office":                roWrite,
	"calendar propose-time":                 roFlagGated,
	"calendar respond":                      roWrite,
	"calendar search":                       roRead,
	"calendar team":                         roRead,
	"calendar time":                         roRead,
	"calendar update":                       roWrite,
	"calendar users":                        roRead,
	"calendar working-location":             roWrite,
	"chat dm send":                          roWrite,
	"chat dm space":                         roWrite,
	"chat messages list":                    roRead,
	"chat messages send":                    roWrite,
	"chat spaces create":                    roWrite,
	"chat spaces find":                      roRead,
	"chat spaces list":                      roRead,
	"chat threads list":                     roRead,
	"classroom announcements assignees":     roWrite,
	"classroom announcements create":        roWrite,
	"classroom announcements delete":        roWrite,
	"classroom announcements get":           roRead,
	"classroom announcements list":          roRead,
	"classroom announcements update":        roWrite,
	"classroom courses archive":             roWrite,
	"classroom courses create":              roWrite,
	"classroom courses delete":              roWrite,
	"classroom courses get":                 roRead,
	"classroom courses join":                roWrite,
	"classroom courses leave":               roWrite,
	"classroom courses list":                roRead,
	"classroom courses unarchive":           roWrite,
	"classroom courses update":              roWrite,
	"classroom courses url":                 roRead,
	"classroom coursework assignees":        roWrite,
	"classroom coursework create":           roWrite,
	"classroom coursework delete":           roWrite,
	"classroom coursework get":              roRead,
	"classroom coursework list":             roRead,
	"classroom coursework update":           roWrite,
	"classroom guardian-invitations create": roWrite,
	"classroom guardian-invitations get":    roRead,
	"classroom guardian-invitations list":   roRead,
	"classroom guardians delete":            roWrite,
	"classroom guardians get":               roRead,
	"classroom guardians list":              roRead,
	"classroom invitations accept":          roWrite,
	"classroom invitations create":          roWrite,
	"classroom invitations delete":          roWrite,
	"classroom invitations get":             roRead,
	"classroom invitations list":            roRead,
	"classroom materials create":            roWrite,
	"classroom materials delete":            roWrite,
	"classroom materials get":               roRead,
	"classroom materials list":              roRead,
	"classroom materials update":            roWrite,
	"classroom profile get":                 roRead,
	"classroom roster":                      roRead,
	"classroom students add":                roWrite,
	"classroom students get":                roRead,
	"classroom students list":               roRead,
	"classroom students remove":             roWrite,
	"classroom submissions get":             roRead,
	"classroom submissions grade":           roWrite,
	"classroom submissions list":            roRead,
	"classroom submissions reclaim":         roWrite,
	"classroom submissions return":          roWrite,
	"classroom submissions turn-in":         roWrite,
	"classroom teachers add":                roWrite,
	"classroom teachers get":                roRead,
	"classroom teachers list":               roRead,
	"classroom teachers remove":             roWrite,
	"classroom topics create":               roWrite,
	"classroom topics delete":               roWrite,
	"classroom topics get":                  roRead,
	"classroom topics list":                 roRead,
	"classroom topics update":               roWrite,
	"completion":                            roRead,
	"config get":                            roRead,
	"config keys":                           roRead,
	"config list":                           roRead,
	"config path":                           roRead,
	"config set":                            roLocal,
	"config unset":                          roLocal,
	"contacts batch create":                 roWrite,
	"contacts batch delete":                 roWrite,
	"contacts create":                       roWrite,
	"contacts delete":                       roWrite,
	"contacts directory list":               roRead,
	"contacts directory search":             roRead,
	"contacts get":                          roRead,
	"contacts list":                         roRead,
	"contacts other delete":                 roWrite,
	"contacts other list":                   roRead,
	"contacts other search":                 roRead,
	"contacts search":                       roRead,
	"contacts update":                       roWrite,
	"docs cat":                              roRead,
	"docs comments add":                     roWrite,
	"docs comments delete":                  roWrite,
	"docs comments get":                     roRead,
	"docs comments list":                    roRead,
	"docs comments reply":                   roWrite,
	"docs comments resolve":                 roWrite,
	"docs copy":                             roWrite,
	"docs create":                           roWrite,
	"docs delete":                           roWrite,
	"docs export":                           roRead,
	"docs find-replace":                     roWrite,
	"docs footer":                           roFlagGated,
	"docs generate":                         roWrite,
	"docs header":                           roFlagGated,
	"docs info":                             roRead,
	"docs insert":                           roWrite,
	"docs list-tabs":                        roRead,
	"docs structure":                        roRead,
	"docs update":                           roWrite,
	"docs write":                            roWrite,
	"docx accept-changes":                   roLocal,
	"docx cat":                              roRead,
	"docx comment":                          roLocal,
	"docx create":                           roLocal,
	"docx delete":                           roLocal,
	"docx info":                             roRead,
	"docx insert":                           roLocal,
	"docx inspect":                          roRead,
	"docx list-comments":                    roRead,
	"docx reject-changes":                   roLocal,
	"docx replace":                          roLocal,
	"docx rewrite":                          roLocal,
	"docx style":                            roLocal,
	"docx table":                            roLocal,
	"docx to-pdf":                           roLocal,
	"docx track":                            roLocal,
	"download":                              roRead,
	"drive cat":                             roRead,
	"drive check-public":                    roRead,
	"drive comments create":                 roWrite,
	"drive comments delete":                 roWrite,
	"drive comments get":                    roRead,
	"drive comments list":                   roRead,
	"drive comments reply":                  roWrite,
	"drive comments update":                 roWrite,
	"drive copy":                            roWrite,
	"drive delete":                          roWrite,
	"drive download":                        roRead,
	"drive drives":                          roRead,
	"drive get":                             roRead,
	"drive ls":                              roRead,
	"drive mkdir":                           roWrite,
	"drive move":                            roWrite,
	"drive permissions":                     roRead,
	"drive rename":                          roWrite,
	"drive search":                          roRead,
	"drive share":                           roWrite,
	"drive unshare":                         roWrite,
	"drive upload":                          roWrite,
	"drive url":                             roRead,
	"exit-codes":                            roRead,
	"forms create":                          roWrite,
	"forms get":                             roRead,
	"forms publish":                         roWrite,
	"forms responses get":                   roRead,
	"forms responses list":                  roRead,
	"gmail attachment":                      roRead,
	"gmail autoforward get":                 roRead,
	"gmail autoforward update":              roWrite,
	"gmail batch delete":                    roWrite,
	"gmail batch modify":                    roWrite,
	"gmail delegates add":                   roWrite,
	"gmail delegates get":                   roRead,
	"gmail delegates list":                  roRead,
	"gmail delegates remove":                roWrite,
	"gmail drafts create":                   roWrite,
	"gmail drafts delete":                   roWrite,
	"gmail drafts get":                      roRead,
	"gmail drafts list":                     roRead,
	"gmail drafts send":                     roWrite,
	"gmail drafts update":                   roWrite,
	"gmail filters create":                  roWrite,
	"gmail filters delete":                  roWrite,
	"gmail filters get":                     roRead,
	"gmail filters list":                    roRead,
	"gmail forwarding create":               roWrite,
	"gmail forwarding delete":               roWrite,
	"gmail forwarding get":                  roRead,
	"gmail forwarding list":                 roRead,
	"gmail get":                             roRead,
	"gmail history":                         roRead,
	"gmail labels create":                   roWrite,
	"gmail labels delete":                   roWrite,
	"gmail labels get":                      roRead,
	"gmail labels list":                     roRead,
	"gmail labels modify":                   roWrite,
	"gmail messages search":                 roRead,
	"gmail search":                          roRead,
	"gmail send":                            roWrite,
	"gmail sendas create":                   roWrite,
	"gmail sendas delete":                   roWrite,
	"gmail sendas get":                      roRead,
	"gmail sendas list":                     roRead,
	"gmail sendas update":                   roWrite,
	"gmail sendas verify":                   roWrite,
	"gmail settings autoforward get":        roRead,
	"gmail settings autoforward update":     roWrite,
	"gmail settings delegates add":          roWrite,
	"gmail settings delegates get":          roRead,
	"gmail settings delegates list":         roRead,
	"gmail settings delegates remove":       roWrite,
	"gmail settings filters create":         roWrite,
	"gmail settings filters delete":         roWrite,
	"gmail settings filters get":            roRead,
	"gmail settings filters list":           roRead,
	"gmail settings forwarding create":      roWrite,
	"gmail settings forwarding delete":      roWrite,
	"gmail settings forwarding get":         roRead,
	"gmail settings forwarding list":        roRead,
	"gmail settings sendas create":          roWrite,
	"gmail settings sendas delete":          roWrite,
	"gmail settings sendas get":             roRead,
	"gmail settings sendas list":            roRead,
	"gmail settings sendas update":          roWrite,
	"gmail settings sendas verify":          roWrite,
	"gmail settings vacation get":           roRead,
	"gmail settings vacation update":        roWrite,
	"gmail settings watch renew":            roWrite,
	"gmail settings watch serve":            roWrite,
	"gmail settings watch start":            roWrite,
	"gmail settings watch status":           roRead,
	"gmail settings watch stop":             roWrite,
	"gmail thread attachments":              roRead,
	"gmail thread get":                      roRead,
	"gmail thread modify":                   roWrite,
	"gmail track opens":                     roRead,
	"gmail track setup":                     roWrite,
	"gmail track status":                    roRead,
	"gmail url":                             roRead,
	"gmail vacation get":                    roRead,
	"gmail vacation update":                 roWrite,
	"gmail watch renew":                     roWrite,
	"gmail watch serve":                     roWrite,
	"gmail watch start":                     roWrite,
	"gmail watch status":                    roRead,
	"gmail watch stop":                      roWrite,
	"groups list":                           roRead,
	"groups members":                        roRead,
	"keep attachment":                       roRead,
	"keep create":                           roWrite,
	"keep delete":                           roWrite,
	"keep get":                              roRead,
	"keep list":                             roRead,
	"keep permissions add":                  roWrite,
	"keep permissions remove":               roWrite,
	"keep search":                           roRead,
	"login":                                 roLocal,
	"logout":                                roLocal,
	"ls":                                    roRead,
	"m365 calendar events":                  roRead,
	"m365 calendar freebusy":                roRead,
	"m365 outlook message get":              roRead,
	"m365 outlook search":                   roRead,
	"me":                                    roRead,
	"open":                                  roRead,
	"people get":                            roRead,
	"people me":                             roRead,
	"people relations":                      roRead,
	"people search":                         roRead,
	"schema":                                roRead,
	"search":                                roRead,
	"send":                                  roWrite,
	"setup docx":                            roLocal,
	"sheets add-tab":                        roWrite,
	"sheets append":                         roWrite,
	"sheets batch-update":                   roWrite,
	"sheets clear":                          roWrite,
	"sheets copy":                           roWrite,
	"sheets create":                         roWrite,
	"sheets export":                         roRead,
	"sheets format":                         roWrite,
	"sheets get":                            roRead,
	"sheets metadata":                       roRead,
	"sheets notes":                          roRead,
	"sheets update":                         roWrite,
	"slides add-slide":                      roWrite,
	"slides copy":                           roWrite,
	"slides create":                         roWrite,
	"slides create-from-markdown":           roWrite,
	"slides delete-slide":                   roWrite,
	"slides export":                         roRead,
	"slides info":                           roRead,
	"slides list-slides":                    roRead,
	"slides read-slide":                     roRead,
	"slides replace-slide":                  roWrite,
	"slides update-notes":                   roWrite,
	"status":                                roRead,
	"sync init":                             roWrite,
	"sync list":                             roRead,
	"sync remove":                           roWrite,
	"sync repair":                           roWrite,
	"sync service install":                  roWrite,
	"sync service status":                   roRead,
	"sync service uninstall":                roWrite,
	"sync start":                            roWrite,
	"sync status":                           roRead,
	"sync stop":                             roWrite,
	"tasks add":                             roWrite,
	"tasks clear":                           roWrite,
	"tasks delete":                          roWrite,
	"tasks done":                            roWrite,
	"tasks get":                             roRead,
	"tasks list":                            roRead,
	"tasks lists create":                    roWrite,
	"tasks lists list":                      roRead,
	"tasks undo":                            roWrite,
	"tasks update":                          roWrite,
	"templates add":                         roLocal,
	"templates inspect":                     roRead,
	"templates list":                        roRead,
	"time now":                              roRead,
	"update":                                roWrite,
	"upload":                                roWrite,
	"version":                               roRead,
	"whoami":                                roRead,
}

// readOnlyLeafCommands returns the command-name path of every leaf command,
// hidden ones included (they are still runnable).
func readOnlyLeafCommands(root *kong.Node) [][]string {
	var leaves [][]string
	var walk func(n *kong.Node, path []string)
	walk = func(n *kong.Node, path []string) {
		children := 0
		for _, child := range n.Children {
			if child == nil || child.Type != kong.CommandNode {
				continue
			}
			children++
			walk(child, append(append([]string{}, path...), child.Name))
		}
		if children == 0 && len(path) > 0 {
			leaves = append(leaves, path)
		}
	}
	walk(root, nil)
	return leaves
}

func TestReadOnly_EveryCommandClassified(t *testing.T) {
	parser, _, err := newParser("test")
	if err != nil {
		t.Fatalf("newParser: %v", err)
	}

	leaves := readOnlyLeafCommands(parser.Model.Node)
	if len(leaves) < 100 {
		t.Fatalf("walked only %d leaf commands; the kong model walk is broken", len(leaves))
	}

	seen := make(map[string]bool, len(leaves))
	for _, path := range leaves {
		key := strings.Join(path, " ")
		seen[key] = true

		class, ok := readOnlyClassification[key]
		if !ok {
			t.Errorf("command %q is not classified for --read-only; add it to readOnlyClassification (and to readonly.go if it writes)", key)
			continue
		}

		blocked := readOnlyCheck(path) != nil
		switch class {
		case roWrite:
			if !blocked {
				t.Errorf("command %q is classified as a write but --read-only allows it", key)
			}
		case roRead, roLocal, roFlagGated:
			if blocked {
				t.Errorf("command %q is classified as available but --read-only blocks it", key)
			}
		default:
			t.Errorf("command %q has unknown class %d", key, class)
		}

		if gated := len(readOnlyWriteFlags[key]) > 0; gated != (class == roFlagGated) {
			t.Errorf("command %q: class %d disagrees with readOnlyWriteFlags (gated=%v)", key, class, gated)
		}
	}

	for key := range readOnlyClassification {
		if !seen[key] {
			t.Errorf("readOnlyClassification lists %q, which is not a command any more", key)
		}
	}
	for key := range readOnlyWriteFlags {
		if !seen[key] {
			t.Errorf("readOnlyWriteFlags lists %q, which is not a command", key)
		}
	}
	for key := range writeCommandPaths {
		if !seen[key] {
			t.Errorf("writeCommandPaths lists %q, which is not a command", key)
		}
	}
}

func parseRealCLI(t *testing.T, args []string) *kong.Context {
	t.Helper()
	parser, _, err := newParser("test")
	if err != nil {
		t.Fatalf("newParser: %v", err)
	}
	parser.Stdout = io.Discard
	parser.Stderr = io.Discard
	kctx, err := parser.Parse(args)
	if err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return kctx
}

func TestEnforceReadOnly_NamedGaps(t *testing.T) {
	blocked := [][]string{
		{"sheets", "add-tab", "sid", "Tab"},
		{"sheets", "batch-update", "sid", "--file", "r.json"},
		{"docs", "generate", "--template", "tpl", "--data", "d.json"},
		{"sync", "init", "/tmp/x", "--drive-folder", "f"},
		{"sync", "start", "/tmp/x"},
		{"sync", "stop"},
		{"sync", "remove", "/tmp/x"},
		{"sync", "service", "install", "/tmp/x"},
		{"update"},
		{"keep", "delete", "notes/1"},
		{"calendar", "respond", "primary", "e1", "--status", "accepted"},
		{"docs", "header", "d1", "--set", "x"},
		{"docs", "footer", "d1", "--clear"},
		{"calendar", "propose-time", "primary", "e1", "--decline"},
		{"calendar", "propose-time", "primary", "e1", "--comment", "no"},
	}
	for _, args := range blocked {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			kctx := parseRealCLI(t, args)
			err := enforceReadOnly(kctx, true)
			if err == nil {
				t.Fatalf("expected %v to be blocked in read-only mode", args)
			}
			if ExitCode(err) != 2 || !strings.Contains(err.Error(), "read-only mode") {
				t.Fatalf("unexpected error: %v (exit %d)", err, ExitCode(err))
			}
		})
	}

	allowed := [][]string{
		{"sheets", "notes", "sid", "A1:B2"},
		{"sheets", "get", "sid", "A1:B2"},
		{"sync", "status"},
		{"sync", "list"},
		{"sync", "service", "status"},
		{"docs", "header", "d1"},
		{"docs", "footer", "d1"},
		{"calendar", "propose-time", "primary", "e1"},
		{"drive", "get", "f1"},
	}
	for _, args := range allowed {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			kctx := parseRealCLI(t, args)
			if err := enforceReadOnly(kctx, true); err != nil {
				t.Fatalf("expected %v to be allowed in read-only mode, got %v", args, err)
			}
		})
	}
}
