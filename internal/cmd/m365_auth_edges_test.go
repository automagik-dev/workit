package cmd

import (
	"strings"
	"testing"
)

func TestAuthAddM365RejectsUnsupportedModes(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--json", "auth", "add", "pilot@example.com", "--services", "m365", "--readonly", "--headless"}, "headless callback-server mode is not supported yet"},
		{[]string{"--json", "auth", "add", "pilot@example.com", "--services", "m365", "--readonly", "--auth-code", "raw"}, "does not accept raw --auth-code"},
		{[]string{"--json", "auth", "add", "pilot@example.com", "--services", "m365", "--readonly", "--remote", "--step", "2"}, "remote auth is not supported yet"},
	}

	for _, tc := range tests {
		_ = captureStderr(t, func() {
			err := Execute(tc.args)
			if err == nil {
				t.Fatalf("expected error for %#v", tc.args)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q for %#v, got %v", tc.want, tc.args, err)
			}
		})
	}
}

func TestAuthAddMixedM365AndGoogleFailsClosed(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"--json", "auth", "add", "pilot@example.com", "--services", "m365,gmail", "--readonly"})
		if err == nil {
			t.Fatal("expected mixed m365/google services to fail closed")
		}
		if !strings.Contains(err.Error(), "unknown service") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
