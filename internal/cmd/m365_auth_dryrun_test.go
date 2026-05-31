package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/config"
)

func TestAuthAddM365DryRunReportsPilotScopes(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "--dry-run", "auth", "add", "pilot@example.com", "--services", "m365", "--readonly"}); err != nil {
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
	origClientID := config.DefaultM365ClientID
	origEnv, hadEnv := os.LookupEnv("WK_M365_CLIENT_ID")
	t.Cleanup(func() {
		config.DefaultM365ClientID = origClientID
		if hadEnv {
			_ = os.Setenv("WK_M365_CLIENT_ID", origEnv)
		} else {
			_ = os.Unsetenv("WK_M365_CLIENT_ID")
		}
	})
	config.DefaultM365ClientID = ""
	_ = os.Unsetenv("WK_M365_CLIENT_ID")

	_ = captureStderr(t, func() {
		err := Execute([]string{"--json", "auth", "add", "pilot@example.com", "--services", "m365", "--readonly"})
		if err == nil {
			t.Fatal("expected missing m365 client id")
		}
		if !strings.Contains(err.Error(), "client id") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
