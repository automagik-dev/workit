package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/msauth"
)

func TestAuthManageM365PrintURLPropagatesForceConsent(t *testing.T) {
	origURL := m365ManualAuthURL
	t.Cleanup(func() { m365ManualAuthURL = origURL })

	var got msauth.ManualAuthURLOptions
	m365ManualAuthURL = func(_ context.Context, opts msauth.ManualAuthURLOptions) (msauth.ManualAuthURLResult, error) {
		got = opts
		return msauth.ManualAuthURLResult{URL: "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize", State: "state"}, nil
	}

	_ = captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "manage", "--services", "m365", "--force-consent", "--print-url"}); err != nil {
				t.Fatalf("auth manage m365: %v", err)
			}
		})
	})

	if !got.Readonly || !got.ForceConsent {
		t.Fatalf("options = %#v", got)
	}
}

func TestAuthManageM365RequiresPrintURLForTextMode(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"auth", "manage", "--services", "m365"})
		if err == nil {
			t.Fatal("expected m365 manage text mode to fail closed")
		}
		if !strings.Contains(err.Error(), "requires --print-url") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
