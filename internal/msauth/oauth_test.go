package msauth

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/config"
)

func TestManualAuthURLRequiresClientIDFailClosed(t *testing.T) {
	origClientID := config.DefaultM365ClientID
	origTenantID := config.DefaultM365TenantID

	t.Cleanup(func() {
		config.DefaultM365ClientID = origClientID
		config.DefaultM365TenantID = origTenantID
	})

	config.DefaultM365ClientID = ""
	config.DefaultM365TenantID = ""

	t.Setenv("WK_M365_CLIENT_ID", "")

	_, err := ManualAuthURL(context.Background(), ManualAuthURLOptions{Readonly: true})
	if err == nil || !strings.Contains(err.Error(), "client id") {
		t.Fatalf("expected missing client id error, got: %v", err)
	}
}

func TestManualAuthURLUsesMicrosoftOAuthWithOnlyReadPilotScopes(t *testing.T) {
	origClientID := config.DefaultM365ClientID
	origTenantID := config.DefaultM365TenantID
	origRandom := randomStateFn

	t.Cleanup(func() {
		config.DefaultM365ClientID = origClientID
		config.DefaultM365TenantID = origTenantID
		randomStateFn = origRandom
	})

	config.DefaultM365ClientID = "test-client-id"
	config.DefaultM365TenantID = "organizations"
	_ = os.Unsetenv("WK_M365_CLIENT_ID")
	randomStateFn = func() (string, error) { return "state-for-test", nil }

	result, err := ManualAuthURL(context.Background(), ManualAuthURLOptions{Readonly: true})
	if err != nil {
		t.Fatalf("ManualAuthURL: %v", err)
	}

	parsed, err := url.Parse(result.URL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	if parsed.Host != "login.microsoftonline.com" || !strings.Contains(parsed.Path, "/organizations/oauth2/v2.0/authorize") {
		t.Fatalf("unexpected auth endpoint: %s", result.URL)
	}

	q := parsed.Query()
	if q.Get("client_id") != "test-client-id" {
		t.Fatalf("client_id = %q", q.Get("client_id"))
	}

	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
		t.Fatalf("missing PKCE params: %s", result.URL)
	}

	scope := q.Get("scope")
	for _, want := range []string{"offline_access", "User.Read", "Mail.Read", "Calendars.Read"} {
		if !strings.Contains(scope, want) {
			t.Fatalf("scope %q missing %s", scope, want)
		}
	}

	for _, forbidden := range []string{"Mail.Send", "Calendars.ReadWrite"} {
		if strings.Contains(scope, forbidden) {
			t.Fatalf("scope %q contains forbidden %s", scope, forbidden)
		}
	}
}

func TestOAuthScopesRejectsNonReadonly(t *testing.T) {
	_, err := OAuthScopes(false)
	if err == nil {
		t.Fatal("expected non-readonly scopes to fail closed")
	}
}
