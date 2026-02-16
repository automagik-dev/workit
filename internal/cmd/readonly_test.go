package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// readonlyTestCLI is a minimal CLI struct for testing read-only enforcement.
type readonlyTestCLI struct {
	Gmail struct {
		Search struct{} `cmd:"" name:"search"`
		Send   struct{} `cmd:"" name:"send"`
		Delete struct{} `cmd:"" name:"delete"`
		Trash  struct{} `cmd:"" name:"trash"`
		Get    struct{} `cmd:"" name:"get"`
	} `cmd:"" name:"gmail"`
	Drive struct {
		Ls       struct{} `cmd:"" name:"ls"`
		Upload   struct{} `cmd:"" name:"upload"`
		Mkdir    struct{} `cmd:"" name:"mkdir"`
		Rm       struct{} `cmd:"" name:"rm"`
		Delete   struct{} `cmd:"" name:"delete"`
		Move     struct{} `cmd:"" name:"move"`
		Copy     struct{} `cmd:"" name:"copy"`
		Rename   struct{} `cmd:"" name:"rename"`
		Share    struct{} `cmd:"" name:"share"`
		Unshare  struct{} `cmd:"" name:"unshare"`
		Search   struct{} `cmd:"" name:"search"`
		Download struct{} `cmd:"" name:"download"`
	} `cmd:"" name:"drive"`
	Calendar struct {
		Ls     struct{} `cmd:"" name:"ls"`
		Create struct{} `cmd:"" name:"create"`
		Update struct{} `cmd:"" name:"update"`
		Delete struct{} `cmd:"" name:"delete"`
	} `cmd:"" name:"calendar"`
	Contacts struct {
		List   struct{} `cmd:"" name:"list"`
		Search struct{} `cmd:"" name:"search"`
		Create struct{} `cmd:"" name:"create"`
		Delete struct{} `cmd:"" name:"delete"`
		Batch  struct{} `cmd:"" name:"batch"`
	} `cmd:"" name:"contacts"`
	Tasks struct {
		List   struct{} `cmd:"" name:"list"`
		Add    struct{} `cmd:"" name:"add"`
		Delete struct{} `cmd:"" name:"delete"`
		Done   struct{} `cmd:"" name:"done"`
	} `cmd:"" name:"tasks"`
	Chat struct {
		Spaces struct {
			List   struct{} `cmd:"" name:"list"`
			Create struct{} `cmd:"" name:"create"`
		} `cmd:"" name:"spaces"`
		Messages struct {
			List struct{} `cmd:"" name:"list"`
			Send struct{} `cmd:"" name:"send"`
		} `cmd:"" name:"messages"`
		DM struct {
			Send struct{} `cmd:"" name:"send"`
		} `cmd:"" name:"dm"`
	} `cmd:"" name:"chat"`
	Docs struct {
		Get         struct{} `cmd:"" name:"get"`
		Create      struct{} `cmd:"" name:"create"`
		Write       struct{} `cmd:"" name:"write"`
		Insert      struct{} `cmd:"" name:"insert"`
		Delete      struct{} `cmd:"" name:"delete"`
		Update      struct{} `cmd:"" name:"update"`
		Copy        struct{} `cmd:"" name:"copy"`
		FindReplace struct{} `cmd:"" name:"find-replace"`
	} `cmd:"" name:"docs"`
	Slides struct {
		Get         struct{} `cmd:"" name:"get"`
		Create      struct{} `cmd:"" name:"create"`
		Copy        struct{} `cmd:"" name:"copy"`
		AddSlide    struct{} `cmd:"" name:"add-slide"`
		DeleteSlide struct{} `cmd:"" name:"delete-slide"`
	} `cmd:"" name:"slides"`
	Sheets struct {
		Get    struct{} `cmd:"" name:"get"`
		Update struct{} `cmd:"" name:"update"`
		Append struct{} `cmd:"" name:"append"`
		Create struct{} `cmd:"" name:"create"`
		Clear  struct{} `cmd:"" name:"clear"`
		Format struct{} `cmd:"" name:"format"`
		Copy   struct{} `cmd:"" name:"copy"`
	} `cmd:"" name:"sheets"`
	Send   struct{} `cmd:"" name:"send"`
	Search struct{} `cmd:"" name:"search"`
	Upload struct{} `cmd:"" name:"upload"`
}

func parseReadonlyKong(t *testing.T, args []string) *kong.Context {
	t.Helper()
	cli := &readonlyTestCLI{}
	parser, err := kong.New(cli, kong.Writers(io.Discard, io.Discard))
	if err != nil {
		t.Fatalf("kong new: %v", err)
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		t.Fatalf("kong parse %v: %v", args, err)
	}
	return kctx
}

func TestEnforceReadOnly_BlocksWriteCommands(t *testing.T) {
	tests := []struct {
		command string
		blocked bool
	}{
		{"gmail send", true},
		{"gmail search", false},
		{"gmail delete", true},
		{"gmail trash", true},
		{"gmail get", false},
		// Drive: canonical names
		{"drive upload", true},
		{"drive ls", false},
		{"drive mkdir", true},
		{"drive rm", true},
		{"drive delete", true},
		{"drive move", true},
		{"drive copy", true},
		{"drive rename", true},
		{"drive share", true},
		{"drive unshare", true},
		{"drive search", false},
		{"drive download", false},
		// Calendar
		{"calendar create", true},
		{"calendar ls", false},
		{"calendar update", true},
		{"calendar delete", true},
		// Contacts
		{"contacts create", true},
		{"contacts list", false},
		{"contacts search", false},
		{"contacts delete", true},
		{"contacts batch", true},
		// Tasks
		{"tasks add", true},
		{"tasks list", false},
		{"tasks done", true},
		{"tasks delete", true},
		// Docs
		{"docs create", true},
		{"docs write", true},
		{"docs insert", true},
		{"docs delete", true},
		{"docs update", true},
		{"docs copy", true},
		{"docs find-replace", true},
		{"docs get", false},
		// Slides
		{"slides create", true},
		{"slides copy", true},
		{"slides add-slide", true},
		{"slides delete-slide", true},
		{"slides get", false},
		// Sheets
		{"sheets update", true},
		{"sheets append", true},
		{"sheets create", true},
		{"sheets clear", true},
		{"sheets format", true},
		{"sheets copy", true},
		{"sheets get", false},
		// Top-level desire paths
		{"send", true},
		{"search", false},
		{"upload", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			kctx := parseReadonlyKong(t, strings.Fields(tt.command))
			err := enforceReadOnly(kctx, true)
			if tt.blocked {
				if err == nil {
					t.Fatalf("expected %q to be blocked in read-only mode", tt.command)
				}
				if !strings.Contains(err.Error(), "read-only mode") {
					t.Fatalf("unexpected error message: %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected %q to be allowed in read-only mode, got: %v", tt.command, err)
			}
		})
	}
}

func TestEnforceReadOnly_DefaultAllowsEverything(t *testing.T) {
	// readOnly=false should allow all commands
	cmds := []string{
		"gmail send",
		"drive upload",
		"calendar create",
		"contacts delete",
		"tasks add",
		"send",
		"upload",
	}
	for _, cmd := range cmds {
		t.Run(cmd, func(t *testing.T) {
			kctx := parseReadonlyKong(t, strings.Fields(cmd))
			err := enforceReadOnly(kctx, false)
			if err != nil {
				t.Fatalf("expected command %q to be allowed when readOnly=false, got: %v", cmd, err)
			}
		})
	}
}

func TestEnforceReadOnly_NestedWriteCommands(t *testing.T) {
	tests := []struct {
		command string
		blocked bool
	}{
		{"chat messages send", true},
		{"chat messages list", false},
		{"chat spaces list", false},
		{"chat spaces create", true},
		{"chat dm send", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			kctx := parseReadonlyKong(t, strings.Fields(tt.command))
			err := enforceReadOnly(kctx, true)
			if tt.blocked {
				if err == nil {
					t.Fatalf("expected %q to be blocked in read-only mode", tt.command)
				}
				if !strings.Contains(err.Error(), "read-only mode") {
					t.Fatalf("unexpected error message: %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected %q to be allowed in read-only mode, got: %v", tt.command, err)
			}
		})
	}
}

func TestEnforceReadOnly_ErrorIsExitCode2(t *testing.T) {
	kctx := parseReadonlyKong(t, []string{"gmail", "send"})
	err := enforceReadOnly(kctx, true)
	if err == nil {
		t.Fatal("expected error")
	}
	code := ExitCode(err)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
