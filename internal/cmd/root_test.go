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
			name: "unknown command-specific flag with value",
			args: []string{"gmail", "search", "--query", "is:unread", "--json"},
			want: []string{"gmail", "search"},
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
			// Heuristic: --verbose (no =) followed by "drive" (no -)
			// causes "drive" to be skipped as a presumed flag value.
			// This is acceptable: the main use case is --generate-input
			// where flags like --verbose typically precede commands.
			args: []string{"drive", "--query", "foo", "--verbose", "ls"},
			want: []string{"drive"},
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
