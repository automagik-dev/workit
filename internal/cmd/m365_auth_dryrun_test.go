package cmd

import (
	"strings"
	"testing"
)

func TestAuthAddM365DryRunReportsPilotScopes(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "--dry-run", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365", "--readonly"}); err != nil {
				t.Fatalf("dry-run m365 auth: %v", err)
			}
		})
	})

	for _, want := range []string{"auth.add.m365", "microsoft_graph", "User.Read", "Mail.Read", "Calendars.Read"} {
		if !strings.Contains(out, want) {
			t.Fatalf("dry-run output missing %s: %s", want, out)
		}
	}
}

func TestAuthAddM365RealFlowFailsClosedWithoutClientID(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365", "--readonly"})
		if err == nil {
			t.Fatal("expected missing m365 client id")
		}
		if !strings.Contains(err.Error(), "client id") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
