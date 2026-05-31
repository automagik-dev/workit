package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/automagik-dev/workit/internal/msauth"
)

func TestAuthM365LoginLinkPrintsOneClickURL(t *testing.T) {
	origCreate := createM365BrokerSession
	t.Cleanup(func() { createM365BrokerSession = origCreate })

	var got msauth.BrokerSessionOptions
	createM365BrokerSession = func(_ context.Context, opts msauth.BrokerSessionOptions) (msauth.BrokerSession, error) {
		got = opts
		return msauth.BrokerSession{
			State:         "state",
			ExpectedEmail: "bernardo@hapvida.com.br",
			LoginURL:      "https://login.workit.ai/m365/start/state",
			ExpiresAt:     time.Unix(1893456000, 0).UTC(),
		}, nil
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "m365", "login-link", "bernardo@hapvida.com.br", "--base-url", "https://login.workit.ai", "--callback-url", "https://login.workit.ai/m365/callback"}); err != nil {
				t.Fatalf("login-link: %v", err)
			}
		})
	})

	if got.ExpectedEmail != "bernardo@hapvida.com.br" || !got.Readonly {
		t.Fatalf("options = %#v", got)
	}
	if got.BaseURL != "https://login.workit.ai" || got.CallbackURL != "https://login.workit.ai/m365/callback" {
		t.Fatalf("urls = %#v", got)
	}
	if !strings.Contains(out, "https://login.workit.ai/m365/start/state") {
		t.Fatalf("missing login link: %s", out)
	}
}

func TestAuthM365LoginLinkRequiresExplicitBrokerURLs(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"auth", "m365", "login-link", "bernardo@hapvida.com.br"})
		if err == nil {
			t.Fatal("expected missing broker URL failure")
		}
		if !strings.Contains(err.Error(), "base-url") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
