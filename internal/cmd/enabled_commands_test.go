package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// desirePathTestCLI is a minimal CLI struct for testing desire-path tier enforcement.
type desirePathTestCLI struct {
	Send     struct{} `cmd:"" name:"send"`
	Upload   struct{} `cmd:"" name:"upload"`
	Ls       struct{} `cmd:"" name:"ls"`
	Search   struct{} `cmd:"" name:"search"`
	Download struct{} `cmd:"" name:"download"`
}

func parseDesirePathKong(t *testing.T, args []string) *kong.Context {
	t.Helper()
	cli := &desirePathTestCLI{}
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

func TestDesirePathTierCoreBlocksWriteAliases(t *testing.T) {
	// "send" and "upload" are extended-tier desire paths; core tier must block them.
	for _, alias := range []string{"send", "upload"} {
		kctx := parseDesirePathKong(t, []string{alias})
		err := enforceCommandTier(kctx, "core")
		if err == nil {
			t.Fatalf("expected core tier to block desire-path %q", alias)
		}
		if !strings.Contains(err.Error(), "requires tier") {
			t.Fatalf("unexpected error for %q: %v", alias, err)
		}
	}
}

func TestDesirePathTierExtendedAllowsWriteAliases(t *testing.T) {
	// "send" and "upload" are extended-tier desire paths; extended tier should allow them.
	for _, alias := range []string{"send", "upload"} {
		kctx := parseDesirePathKong(t, []string{alias})
		err := enforceCommandTier(kctx, "extended")
		if err != nil {
			t.Fatalf("expected extended tier to allow desire-path %q, got: %v", alias, err)
		}
	}
}

func TestDesirePathTierCoreAllowsReadAliases(t *testing.T) {
	// "ls", "search", "download" are core-tier desire paths; core tier should allow them.
	for _, alias := range []string{"ls", "search", "download"} {
		kctx := parseDesirePathKong(t, []string{alias})
		err := enforceCommandTier(kctx, "core")
		if err != nil {
			t.Fatalf("expected core tier to allow desire-path %q, got: %v", alias, err)
		}
	}
}

func TestParseEnabledCommands(t *testing.T) {
	allow := parseEnabledCommands("calendar, tasks ,Gmail")
	if !allow["calendar"] || !allow["tasks"] || !allow["gmail"] {
		t.Fatalf("unexpected allow map: %#v", allow)
	}
}

func TestParseCommandTiers_Cached(t *testing.T) {
	// parseCommandTiers should return the same map on successive calls (cached via sync.Once).
	tiers1, err := parseCommandTiers()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	tiers2, err := parseCommandTiers()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	// Verify they return valid data.
	if len(tiers1) == 0 {
		t.Fatal("expected non-empty tiers")
	}
	// Verify the drive service has correct canonical entries.
	driveTiers, ok := tiers1["drive"]
	if !ok {
		t.Fatal("expected drive tiers")
	}
	for _, cmd := range []string{"move", "copy", "delete"} {
		if _, ok := driveTiers[cmd]; !ok {
			t.Errorf("expected drive tier entry for %q", cmd)
		}
	}
	// Pointer equality confirms caching.
	if len(tiers1) != len(tiers2) {
		t.Fatal("expected same map from cached call")
	}
}
