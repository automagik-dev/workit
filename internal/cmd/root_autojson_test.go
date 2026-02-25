package cmd

import (
	"strings"
	"testing"
)

func TestAutoJSON_Version_DefaultsToJSONWhenEnabled(t *testing.T) {
	t.Setenv("WK_AUTO_JSON", "1")

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"version"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("expected json output, got: %q", out)
	}
	if !strings.Contains(out, "\"version\"") {
		t.Fatalf("expected version field in json output, got: %q", out)
	}
}

func TestJQ_AutoEnablesJSONMode(t *testing.T) {
	// When --jq is provided without --json, JSON mode should be auto-enabled.
	// Previously, --jq was silently ignored in default (text) mode because
	// IsJSON(ctx) returned false and commands would skip JSON output entirely.
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--jq", ".version", "version"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		t.Fatalf("expected non-empty output")
	}
	// `wk version --json` produces {"version":"0.12.0-dev","commit":"","date":""}
	// After jq ".version" the output should be a quoted JSON string: "0.12.0-dev"
	// Without auto-JSON, the version command produces plain "0.12.0-dev" (unquoted).
	// We verify jq was active by checking the output is the JSON-quoted string.
	if !strings.HasPrefix(trimmed, "\"") {
		t.Fatalf("expected jq-filtered JSON string (quoted), got: %q -- --jq may have been silently bypassed", trimmed)
	}
}

func TestJQ_RejectsWithPlain(t *testing.T) {
	// --jq combined with --plain should be rejected.
	var execErr error
	_ = captureStdout(t, func() {
		_ = captureStderr(t, func() {
			execErr = Execute([]string{"--jq", ".version", "--plain", "version"})
		})
	})

	if execErr == nil {
		t.Fatalf("expected error when combining --jq with --plain")
	}
	if !strings.Contains(execErr.Error(), "--jq requires --json") {
		t.Fatalf("unexpected error: %v", execErr)
	}
}

func TestAutoJSON_Version_RespectsExplicitPlainFlag(t *testing.T) {
	t.Setenv("WK_AUTO_JSON", "1")

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--plain", "version"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	if strings.HasPrefix(strings.TrimSpace(out), "{") || strings.Contains(out, "\"version\"") {
		t.Fatalf("expected text output (not json), got: %q", out)
	}
}
