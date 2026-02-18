package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExecute_VersionFlag(t *testing.T) {
	origV, origB, origC, origD := version, branch, commit, date
	t.Cleanup(func() {
		version = origV
		branch = origB
		commit = origC
		date = origD
	})
	version = "1.2.3"
	branch = "main"
	commit = "abc123"
	date = ""

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--version"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	if !strings.Contains(out, "1.2.3") || !strings.Contains(out, "main") {
		t.Fatalf("unexpected out=%q", out)
	}
}

func TestExecute_VersionCommand_JSON(t *testing.T) {
	origV, origB, origC, origD := version, branch, commit, date
	t.Cleanup(func() {
		version = origV
		branch = origB
		commit = origC
		date = origD
	})
	version = "1.2.3"
	branch = "dev"
	commit = "abc123"
	date = "2025-12-26T00:00:00Z"

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "version"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if parsed["version"] != "1.2.3" || parsed["branch"] != "dev" || parsed["commit"] != "abc123" || parsed["date"] != "2025-12-26T00:00:00Z" {
		t.Fatalf("unexpected json: %#v", parsed)
	}
}

func TestExecute_ExitCodes(t *testing.T) {
	err := Execute([]string{"--nope"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("expected exit code 2, got %d (err=%v)", ExitCode(err), err)
	}

	err = Execute([]string{"drive", "get"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("expected exit code 2, got %d (err=%v)", ExitCode(err), err)
	}
}
