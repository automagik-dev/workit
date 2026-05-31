package cmd

import (
	"strings"
	"testing"
)

func TestAuthAddM365RejectsUnsupportedModes(t *testing.T) {
	tests := [][]string{
		{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365", "--readonly", "--headless"},
		{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365", "--readonly", "--auth-code", "raw"},
		{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365", "--readonly", "--remote", "--step", "2"},
	}

	for _, args := range tests {
		_ = captureStderr(t, func() {
			err := Execute(args)
			if err == nil {
				t.Fatalf("expected error for %#v", args)
			}
		})
	}
}

func TestAuthAddMixedM365AndGoogleFailsClosed(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365,gmail", "--readonly"})
		if err == nil {
			t.Fatal("expected mixed m365/google services to fail closed")
		}
		if !strings.Contains(err.Error(), "unknown service") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
