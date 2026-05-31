package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/automagik-dev/workit/internal/msauth"
	"github.com/automagik-dev/workit/internal/secrets"
)

func TestAuthAddM365UsesOAuthAndStoresToken(t *testing.T) {
	origOpen := openSecretsStore
	origAuth := authorizeM365
	origKeychain := ensureKeychainAccess
	t.Cleanup(func() {
		openSecretsStore = origOpen
		authorizeM365 = origAuth
		ensureKeychainAccess = origKeychain
	})

	store := newMemSecretsStore()
	openSecretsStore = func() (secrets.Store, error) { return store, nil }
	ensureKeychainAccess = func() error { return nil }
	authorizeM365 = func(context.Context, msauth.AuthorizeOptions) (msauth.AuthorizeResult, error) {
		return msauth.AuthorizeResult{Email: "bernardo@hapvida.com.br", RefreshToken: "m365-refresh-token"}, nil
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365", "--readonly"}); err != nil {
				t.Fatalf("auth add m365: %v", err)
			}
		})
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json output: %v\n%s", err, out)
	}
	if payload["provider"] != "microsoft_graph" || payload["stored"] != true {
		t.Fatalf("unexpected output: %#v", payload)
	}
	tok, err := store.GetToken(msauth.ClientName, "bernardo@hapvida.com.br")
	if err != nil {
		t.Fatalf("stored m365 token: %v", err)
	}
	if tok.RefreshToken != "m365-refresh-token" {
		t.Fatalf("refresh token = %q", tok.RefreshToken)
	}
	if !stringSliceContainsForM365AuthTest(tok.Services, "m365") {
		t.Fatalf("services = %#v", tok.Services)
	}
}

func TestAuthAddM365RequiresReadonly(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365"})
		if err == nil {
			t.Fatal("expected missing --readonly to fail closed")
		}
		if !strings.Contains(err.Error(), "--readonly") {
			t.Fatalf("expected --readonly error, got: %v", err)
		}
	})
}

func TestAuthAddM365RemoteStepOnePrintsMicrosoftURL(t *testing.T) {
	origURL := m365ManualAuthURL
	t.Cleanup(func() { m365ManualAuthURL = origURL })
	m365ManualAuthURL = func(context.Context, msauth.ManualAuthURLOptions) (msauth.ManualAuthURLResult, error) {
		return msauth.ManualAuthURLResult{URL: "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize?client_id=test", State: "state"}, nil
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "add", "bernardo@hapvida.com.br", "--services", "m365", "--readonly", "--remote", "--step", "1"}); err != nil {
				t.Fatalf("remote step 1: %v", err)
			}
		})
	})
	if !strings.Contains(out, "login.microsoftonline.com") || !strings.Contains(out, "auth_url") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAuthManageM365PrintURLIsNonTechnicalOAuthHandoff(t *testing.T) {
	origURL := m365ManualAuthURL
	t.Cleanup(func() { m365ManualAuthURL = origURL })
	m365ManualAuthURL = func(context.Context, msauth.ManualAuthURLOptions) (msauth.ManualAuthURLResult, error) {
		return msauth.ManualAuthURLResult{URL: "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize?client_id=test", State: "state", ExpiresIn: int((5 * time.Minute).Seconds())}, nil
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "manage", "--services", "m365", "--print-url"}); err != nil {
				t.Fatalf("auth manage m365 print-url: %v", err)
			}
		})
	})
	if !strings.Contains(out, "login.microsoftonline.com") || !strings.Contains(out, "microsoft_graph") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func stringSliceContainsForM365AuthTest(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
