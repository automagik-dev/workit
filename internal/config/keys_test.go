package config

import (
	"testing"
)

func TestCallbackServerKey_ValidURLs(t *testing.T) {
	tests := []struct {
		value string
	}{
		{"https://auth.automagik.dev"},
		{"http://localhost:8089"},
		{"https://example.com/path"},
	}
	for _, tt := range tests {
		var cfg File
		if err := SetValue(&cfg, KeyCallbackServer, tt.value); err != nil {
			t.Errorf("SetValue(%q) unexpected error: %v", tt.value, err)
		}

		if got := GetValue(cfg, KeyCallbackServer); got != tt.value {
			t.Errorf("GetValue after Set(%q) = %q", tt.value, got)
		}
	}
}

func TestCallbackServerKey_InvalidURLs(t *testing.T) {
	tests := []struct {
		value string
	}{
		{"ftp://bad"},
		{"auth.automagik.dev"},
		{""},
	}
	for _, tt := range tests {
		var cfg File
		if err := SetValue(&cfg, KeyCallbackServer, tt.value); err == nil {
			t.Errorf("SetValue(%q) expected error, got nil", tt.value)
		}
	}
}

func TestCallbackServerKey_Unset(t *testing.T) {
	var cfg File

	_ = SetValue(&cfg, KeyCallbackServer, "https://example.com")

	if err := UnsetValue(&cfg, KeyCallbackServer); err != nil {
		t.Fatalf("UnsetValue: %v", err)
	}

	if got := GetValue(cfg, KeyCallbackServer); got != "" {
		t.Fatalf("expected empty after unset, got %q", got)
	}
}

func TestCallbackServerKey_EmptyHint(t *testing.T) {
	spec, err := KeySpecFor(KeyCallbackServer)
	if err != nil {
		t.Fatalf("KeySpecFor: %v", err)
	}

	hint := spec.EmptyHint()
	if hint != "(not set)" {
		t.Fatalf("unexpected hint: %q", hint)
	}
}

func TestAuthModeKey_ValidValues(t *testing.T) {
	for _, mode := range []string{"auto", "browser", "headless", "manual"} {
		var cfg File
		if err := SetValue(&cfg, KeyAuthMode, mode); err != nil {
			t.Errorf("SetValue(%q) unexpected error: %v", mode, err)
		}

		if got := GetValue(cfg, KeyAuthMode); got != mode {
			t.Errorf("GetValue after Set(%q) = %q", mode, got)
		}
	}
}

func TestAuthModeKey_InvalidValues(t *testing.T) {
	for _, mode := range []string{"invalid", "HEADLESS", "Browser", ""} {
		var cfg File
		if err := SetValue(&cfg, KeyAuthMode, mode); err == nil {
			t.Errorf("SetValue(%q) expected error, got nil", mode)
		}
	}
}

func TestAuthModeKey_Unset(t *testing.T) {
	var cfg File

	_ = SetValue(&cfg, KeyAuthMode, "headless")

	if err := UnsetValue(&cfg, KeyAuthMode); err != nil {
		t.Fatalf("UnsetValue: %v", err)
	}

	if got := GetValue(cfg, KeyAuthMode); got != "" {
		t.Fatalf("expected empty after unset, got %q", got)
	}
}

func TestAuthModeKey_EmptyHint(t *testing.T) {
	spec, err := KeySpecFor(KeyAuthMode)
	if err != nil {
		t.Fatalf("KeySpecFor: %v", err)
	}

	hint := spec.EmptyHint()
	if hint != "(not set, using auto)" {
		t.Fatalf("unexpected hint: %q", hint)
	}
}

func TestKeyOrder_IncludesNewKeys(t *testing.T) {
	keys := KeyList()

	found := map[Key]bool{}

	for _, k := range keys {
		found[k] = true
	}

	for _, want := range []Key{KeyCallbackServer, KeyAuthMode} {
		if !found[want] {
			t.Errorf("KeyList missing %s", want)
		}
	}
}

func TestParseKey_NewKeys(t *testing.T) {
	for _, raw := range []string{"callback_server", "auth_mode"} {
		if _, err := ParseKey(raw); err != nil {
			t.Errorf("ParseKey(%q) unexpected error: %v", raw, err)
		}
	}
}
