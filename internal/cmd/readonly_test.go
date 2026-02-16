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
			List struct{} `cmd:"" name:"list"`
		} `cmd:"" name:"spaces"`
		Messages struct {
			List struct{} `cmd:"" name:"list"`
			Send struct{} `cmd:"" name:"send"`
		} `cmd:"" name:"messages"`
	} `cmd:"" name:"chat"`
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
		{"drive upload", true},
		{"drive ls", false},
		{"drive mkdir", true},
		{"drive rm", true},
		{"drive search", false},
		{"drive download", false},
		{"calendar create", true},
		{"calendar ls", false},
		{"calendar update", true},
		{"calendar delete", true},
		{"contacts create", true},
		{"contacts list", false},
		{"contacts search", false},
		{"contacts delete", true},
		{"contacts batch", true},
		{"tasks add", true},
		{"tasks list", false},
		{"tasks done", true},
		{"tasks delete", true},
		{"send", true},    // top-level desire path
		{"search", false}, // top-level read desire path
		{"upload", true},  // top-level desire path
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
