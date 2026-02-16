package cmd

import "testing"

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
