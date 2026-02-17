package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// tierTestCLI is a minimal CLI struct for testing tier enforcement.
type tierTestCLI struct {
	Gmail struct {
		Search struct{} `cmd:"" name:"search"`
		Send   struct{} `cmd:"" name:"send"`
		Get    struct{} `cmd:"" name:"get"`
		Labels struct{} `cmd:"" name:"labels"`
		Delete struct{} `cmd:"" name:"delete"`
	} `cmd:"" name:"gmail"`
	Drive struct {
		Ls          struct{} `cmd:"" name:"ls"`
		Search      struct{} `cmd:"" name:"search"`
		Download    struct{} `cmd:"" name:"download"`
		Mkdir       struct{} `cmd:"" name:"mkdir"`
		Permissions struct{} `cmd:"" name:"permissions"`
	} `cmd:"" name:"drive"`
	Calendar struct {
		Ls     struct{} `cmd:"" name:"ls"`
		Get    struct{} `cmd:"" name:"get"`
		Create struct{} `cmd:"" name:"create"`
		Delete struct{} `cmd:"" name:"delete"`
	} `cmd:"" name:"calendar"`
	Auth   struct{} `cmd:"" name:"auth"`
	Config struct{} `cmd:"" name:"config"`
}

func parseTierKong(t *testing.T, args []string) *kong.Context {
	t.Helper()
	cli := &tierTestCLI{}
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

func TestCommandTierCoreBlocksExtended(t *testing.T) {
	// gmail labels is "extended" - should be blocked by "core" tier
	kctx := parseTierKong(t, []string{"gmail", "labels"})
	err := enforceCommandTier(kctx, "core")
	if err == nil {
		t.Fatal("expected error for extended command under core tier")
	}
	if !strings.Contains(err.Error(), "requires tier") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandTierCoreBlocksComplete(t *testing.T) {
	// gmail delete is "complete" - should be blocked by "core" tier
	kctx := parseTierKong(t, []string{"gmail", "delete"})
	err := enforceCommandTier(kctx, "core")
	if err == nil {
		t.Fatal("expected error for complete command under core tier")
	}
	if !strings.Contains(err.Error(), "requires tier") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandTierCoreAllowsCore(t *testing.T) {
	// gmail search is "core" - should pass with "core" tier
	kctx := parseTierKong(t, []string{"gmail", "search"})
	err := enforceCommandTier(kctx, "core")
	if err != nil {
		t.Fatalf("expected no error for core command under core tier, got: %v", err)
	}
}

func TestCommandTierExtendedAllowsCoreAndExtended(t *testing.T) {
	// gmail search is "core" - should pass with "extended" tier
	kctx := parseTierKong(t, []string{"gmail", "search"})
	err := enforceCommandTier(kctx, "extended")
	if err != nil {
		t.Fatalf("expected no error for core command under extended tier, got: %v", err)
	}

	// gmail labels is "extended" - should pass with "extended" tier
	kctx = parseTierKong(t, []string{"gmail", "labels"})
	err = enforceCommandTier(kctx, "extended")
	if err != nil {
		t.Fatalf("expected no error for extended command under extended tier, got: %v", err)
	}
}

func TestCommandTierExtendedBlocksComplete(t *testing.T) {
	// gmail delete is "complete" - should be blocked by "extended" tier
	kctx := parseTierKong(t, []string{"gmail", "delete"})
	err := enforceCommandTier(kctx, "extended")
	if err == nil {
		t.Fatal("expected error for complete command under extended tier")
	}
}

func TestCommandTierCompleteAllowsAll(t *testing.T) {
	// complete tier allows everything
	tests := [][]string{
		{"gmail", "search"},
		{"gmail", "labels"},
		{"gmail", "delete"},
		{"drive", "ls"},
		{"drive", "mkdir"},
		{"drive", "permissions"},
	}
	for _, args := range tests {
		kctx := parseTierKong(t, args)
		err := enforceCommandTier(kctx, "complete")
		if err != nil {
			t.Fatalf("expected no error for %v under complete tier, got: %v", args, err)
		}
	}
}

func TestCommandTierEmptyMeansComplete(t *testing.T) {
	// empty tier string should allow everything (default)
	kctx := parseTierKong(t, []string{"gmail", "delete"})
	err := enforceCommandTier(kctx, "")
	if err != nil {
		t.Fatalf("expected no error for empty tier, got: %v", err)
	}
}

func TestCommandTierAlwaysVisibleBypassCheck(t *testing.T) {
	// auth and config are always-visible utility commands
	kctx := parseTierKong(t, []string{"auth"})
	err := enforceCommandTier(kctx, "core")
	if err != nil {
		t.Fatalf("expected no error for always-visible 'auth' under core tier, got: %v", err)
	}

	kctx = parseTierKong(t, []string{"config"})
	err = enforceCommandTier(kctx, "core")
	if err != nil {
		t.Fatalf("expected no error for always-visible 'config' under core tier, got: %v", err)
	}
}

func TestCommandTierInvalidTierValue(t *testing.T) {
	kctx := parseTierKong(t, []string{"gmail", "search"})
	err := enforceCommandTier(kctx, "ultra")
	if err == nil {
		t.Fatal("expected error for invalid tier value")
	}
	if !strings.Contains(err.Error(), "invalid command tier") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCommandTierComposabilityWithEnabledCommands(t *testing.T) {
	// Both enforceEnabledCommands and enforceCommandTier can be used together.
	// If enable-commands allows "gmail", and tier is "core", then:
	// - gmail search (core) should pass both
	// - gmail labels (extended) should pass enabled but fail tier
	kctx := parseTierKong(t, []string{"gmail", "search"})

	// Check enabledCommands passes
	err := enforceEnabledCommands(kctx, "gmail")
	if err != nil {
		t.Fatalf("expected enabledCommands to pass for gmail: %v", err)
	}
	// Check tier passes
	err = enforceCommandTier(kctx, "core")
	if err != nil {
		t.Fatalf("expected tier core to pass for gmail search: %v", err)
	}

	kctx = parseTierKong(t, []string{"gmail", "labels"})
	// enabledCommands should still pass (gmail is enabled)
	err = enforceEnabledCommands(kctx, "gmail")
	if err != nil {
		t.Fatalf("expected enabledCommands to pass for gmail labels: %v", err)
	}
	// But tier check should fail (labels is extended, tier is core)
	err = enforceCommandTier(kctx, "core")
	if err == nil {
		t.Fatal("expected tier check to fail for gmail labels under core tier")
	}
}

func TestCommandTierDriveSubcommands(t *testing.T) {
	// drive ls is core, drive mkdir is extended, drive permissions is complete
	tests := []struct {
		args    []string
		tier    string
		wantErr bool
	}{
		{[]string{"drive", "ls"}, "core", false},
		{[]string{"drive", "search"}, "core", false},
		{[]string{"drive", "download"}, "core", false},
		{[]string{"drive", "mkdir"}, "core", true},
		{[]string{"drive", "permissions"}, "core", true},
		{[]string{"drive", "mkdir"}, "extended", false},
		{[]string{"drive", "permissions"}, "extended", true},
		{[]string{"drive", "permissions"}, "complete", false},
	}
	for _, tt := range tests {
		kctx := parseTierKong(t, tt.args)
		err := enforceCommandTier(kctx, tt.tier)
		if (err != nil) != tt.wantErr {
			t.Errorf("args=%v tier=%q: wantErr=%v got=%v", tt.args, tt.tier, tt.wantErr, err)
		}
	}
}

func TestCommandTierYAMLParsesSuccessfully(t *testing.T) {
	// Verify the embedded YAML is valid and non-empty
	if len(commandTiersYAML) == 0 {
		t.Fatal("commandTiersYAML is empty")
	}
	tiers, err := parseCommandTiers()
	if err != nil {
		t.Fatalf("failed to parse command tiers YAML: %v", err)
	}
	if len(tiers) == 0 {
		t.Fatal("parsed tiers map is empty")
	}
	// Spot-check some entries
	if tiers["gmail"]["search"] != "core" {
		t.Errorf("expected gmail.search=core, got %q", tiers["gmail"]["search"])
	}
	if tiers["drive"]["permissions"] != "complete" {
		t.Errorf("expected drive.permissions=complete, got %q", tiers["drive"]["permissions"])
	}
}
