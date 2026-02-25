package cmd

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

func TestVersionStringVariants(t *testing.T) {
	origVersion, origBranch, origCommit, origDate := version, branch, commit, date
	t.Cleanup(func() { version, branch, commit, date = origVersion, origBranch, origCommit, origDate })

	version, branch, commit, date = "v1", "", "", ""
	if got := VersionString(); got != "Workit v1" {
		t.Fatalf("unexpected: %q", got)
	}
	version, branch, commit, date = "v1", "main", "", ""
	if got := VersionString(); got != "Workit v1 (main)" {
		t.Fatalf("unexpected: %q", got)
	}
	version, branch, commit, date = "v1", "", "abc", ""
	if got := VersionString(); got != "Workit v1 (abc)" {
		t.Fatalf("unexpected: %q", got)
	}
	version, branch, commit, date = "v1", "", "", "2025-01-01"
	if got := VersionString(); got != "Workit v1 (2025-01-01)" {
		t.Fatalf("unexpected: %q", got)
	}
	version, branch, commit, date = "v1", "main", "abc", "2025-01-01"
	if got := VersionString(); got != "Workit v1 (main abc 2025-01-01)" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestVersionCmd_JSON(t *testing.T) {
	origVersion, origBranch, origCommit, origDate := version, branch, commit, date
	t.Cleanup(func() { version, branch, commit, date = origVersion, origBranch, origCommit, origDate })
	version, branch, commit, date = "v2", "dev", "c1", "d1"

	u, err := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	jsonOut := captureStdout(t, func() {
		if err := runKong(t, &VersionCmd{}, []string{}, ctx, nil); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	var parsed struct {
		Version string `json:"version"`
		Branch  string `json:"branch"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if parsed.Version != "v2" || parsed.Branch != "dev" || parsed.Commit != "c1" || parsed.Date != "d1" {
		t.Fatalf("unexpected json: %#v", parsed)
	}
}
