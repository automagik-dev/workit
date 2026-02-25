package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("X_TEST", "")
	if got := envOr("X_TEST", "fallback"); got != "fallback" {
		t.Fatalf("unexpected: %q", got)
	}
	t.Setenv("X_TEST", "value")
	if got := envOr("X_TEST", "fallback"); got != "value" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestExecute_Help(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--help"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	if !strings.Contains(out, "Google CLI") && !strings.Contains(out, "Usage:") {
		t.Fatalf("unexpected help output: %q", out)
	}
	if !strings.Contains(out, "config.json") || !strings.Contains(out, "keyring backend") {
		t.Fatalf("expected config info in help output: %q", out)
	}
	if strings.Contains(out, "gmail (mail,email) thread get") {
		t.Fatalf("expected collapsed help (no expanded subcommands), got: %q", out)
	}
}

func TestExecute_Help_GmailHasGroupsAndRelativeCommands(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"gmail", "--help"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	if !strings.Contains(out, "\nRead\n") || !strings.Contains(out, "\nWrite\n") || !strings.Contains(out, "\nAdmin\n") {
		t.Fatalf("expected command groups in gmail help, got: %q", out)
	}
	if !strings.Contains(out, "\n  search") || !strings.Contains(out, "Search threads using Gmail query syntax") {
		t.Fatalf("expected relative command summaries in gmail help, got: %q", out)
	}
	if strings.Contains(out, "\n  gmail (mail,email) search <query>") {
		t.Fatalf("unexpected full command prefix in gmail help, got: %q", out)
	}
	if strings.Contains(out, "\n  watch <command>") {
		t.Fatalf("expected watch to be under gmail settings (not top-level gmail help), got: %q", out)
	}
	if !strings.Contains(out, "\n  settings <command>") {
		t.Fatalf("expected settings subgroup in gmail help, got: %q", out)
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	errText := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			if err := Execute([]string{"no_such_cmd"}); err == nil {
				t.Fatalf("expected error")
			}
		})
	})
	if errText == "" {
		t.Fatalf("expected stderr output")
	}
}

func TestExecute_UnknownFlag(t *testing.T) {
	errText := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			if err := Execute([]string{"--definitely-nope"}); err == nil {
				t.Fatalf("expected error")
			}
		})
	})
	if errText == "" {
		t.Fatalf("expected stderr output")
	}
}

func TestExtractCommandTokens(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "simple subcommand",
			args: []string{"drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "global flag with value",
			args: []string{"--account", "me@example.com", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "global flag with equals",
			args: []string{"--account=me@example.com", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "unknown command-specific flag with value leaks value as token",
			args: []string{"gmail", "search", "--query", "is:unread", "--json"},
			// "is:unread" is not consumed because --query is not a known global flag.
			// findCommandNode will reject the unknown token cleanly.
			want: []string{"gmail", "search", "is:unread"},
		},
		{
			name: "unknown flag with equals",
			args: []string{"gmail", "search", "--query=is:unread"},
			want: []string{"gmail", "search"},
		},
		{
			name: "boolean flag no value",
			args: []string{"drive", "ls", "--json"},
			want: []string{"drive", "ls"},
		},
		{
			name: "double dash stops parsing",
			args: []string{"drive", "--", "ls"},
			want: []string{"drive"},
		},
		{
			name: "mixed flags and subcommands",
			// Boolean flags like --verbose do not consume the next token.
			// --query is unknown so "foo" leaks as a token, but the
			// command tokens "drive" and "ls" are correctly extracted.
			args: []string{"drive", "--query", "foo", "--verbose", "ls"},
			want: []string{"drive", "foo", "ls"},
		},
		{
			name: "short flag",
			args: []string{"-a", "me@example.com", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "flag followed by another flag",
			args: []string{"drive", "ls", "--json", "--verbose"},
			want: []string{"drive", "ls"},
		},
		{
			name: "boolean --json before subcommand",
			args: []string{"--json", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "boolean --verbose before subcommand",
			args: []string{"--verbose", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "boolean --force before subcommand",
			args: []string{"--force", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "short boolean -j before subcommand",
			args: []string{"-j", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "multiple booleans before subcommand",
			args: []string{"--json", "--verbose", "--force", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "boolean mixed with value flag",
			args: []string{"--json", "--account", "me@example.com", "--verbose", "drive", "ls"},
			want: []string{"drive", "ls"},
		},
		{
			name: "value flag with equals and booleans",
			args: []string{"--account=me@example.com", "--json", "gmail", "send"},
			want: []string{"gmail", "send"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommandTokens(tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("extractCommandTokens(%v) = %v, want %v", tt.args, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("extractCommandTokens(%v)[%d] = %q, want %q", tt.args, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractEnableCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  string
		want string
	}{
		{
			name: "flag with equals",
			args: []string{"--enable-commands=gmail,drive", "gmail", "--generate-input"},
			want: "gmail,drive",
		},
		{
			name: "flag with space",
			args: []string{"--enable-commands", "calendar,tasks", "calendar", "--generate-input"},
			want: "calendar,tasks",
		},
		{
			name: "env fallback",
			args: []string{"gmail", "--generate-input"},
			env:  "gmail,drive",
			want: "gmail,drive",
		},
		{
			name: "no flag no env",
			args: []string{"gmail", "--generate-input"},
			want: "",
		},
		{
			name: "stops at double dash",
			args: []string{"--", "--enable-commands=gmail", "gmail"},
			want: "",
		},
		{
			name: "flag overrides env",
			args: []string{"--enable-commands=calendar", "calendar"},
			env:  "gmail",
			want: "calendar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("WK_ENABLE_COMMANDS", tt.env)
			} else {
				t.Setenv("WK_ENABLE_COMMANDS", "")
			}
			got := extractEnableCommands(tt.args)
			if got != tt.want {
				t.Fatalf("extractEnableCommands(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestGenerateInput_RespectsEnableCommands(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "blocked command errors",
			args:    []string{"--enable-commands=gmail", "--generate-input", "drive", "ls"},
			wantErr: true,
		},
		{
			name:    "allowed command succeeds",
			args:    []string{"--enable-commands=drive", "--generate-input", "drive", "ls"},
			wantErr: false,
		},
		{
			name:    "wildcard allows all",
			args:    []string{"--enable-commands=*", "--generate-input", "drive", "ls"},
			wantErr: false,
		},
		{
			name:    "no restriction allows all",
			args:    []string{"--generate-input", "drive", "ls"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any env-level enable-commands to isolate flag behavior.
			t.Setenv("WK_ENABLE_COMMANDS", "")

			err := captureStderr(t, func() {
				_ = captureStdout(t, func() {
					execErr := Execute(tt.args)
					if tt.wantErr && execErr == nil {
						t.Fatalf("expected error for args %v, got nil", tt.args)
					}
					if !tt.wantErr && execErr != nil {
						t.Fatalf("unexpected error for args %v: %v", tt.args, execErr)
					}
				})
			})
			_ = err // stderr text; not relevant to pass/fail
		})
	}
}

func TestHasGenerateInput_EqualsForm(t *testing.T) {
	if !hasGenerateInput([]string{"drive", "ls", "--generate-input=true"}) {
		t.Fatal("expected --generate-input=true to be recognized")
	}
	if !hasGenerateInput([]string{"--gen-input=1", "gmail", "send"}) {
		t.Fatal("expected --gen-input=1 to be recognized")
	}
	// The bare forms should still work.
	if !hasGenerateInput([]string{"drive", "--generate-input"}) {
		t.Fatal("expected bare --generate-input to still be recognized")
	}
	// After --, the flag should be ignored.
	if hasGenerateInput([]string{"drive", "--", "--generate-input=true"}) {
		t.Fatal("expected --generate-input=true after -- to be ignored")
	}
}

func TestStripGenerateInputFlag_EqualsForm(t *testing.T) {
	got := stripGenerateInputFlag([]string{"drive", "--generate-input=true", "ls"})
	want := []string{"drive", "ls"}
	if len(got) != len(want) {
		t.Fatalf("stripGenerateInputFlag equals form: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("stripGenerateInputFlag equals form[%d]: got %q, want %q", i, got[i], want[i])
		}
	}

	// Also verify the bare form still works.
	got2 := stripGenerateInputFlag([]string{"drive", "--gen-input", "ls"})
	if len(got2) != len(want) {
		t.Fatalf("stripGenerateInputFlag bare form: got %v, want %v", got2, want)
	}
}

func TestNewUsageError(t *testing.T) {
	if newUsageError(nil) != nil {
		t.Fatalf("expected nil for nil error")
	}

	err := errors.New("bad")
	wrapped := newUsageError(err)
	if wrapped == nil {
		t.Fatalf("expected wrapped error")
	}
	var exitErr *ExitError
	if !errors.As(wrapped, &exitErr) || exitErr.Code != 2 || !errors.Is(exitErr.Err, err) {
		t.Fatalf("unexpected wrapped error: %#v", wrapped)
	}
}
