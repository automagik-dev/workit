package msauth

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/automagik-dev/workit/internal/config"
)

func TestCreateBrokerSessionBuildsOneClickLoginLinkWithoutWriteScopes(t *testing.T) {
	origClientID := config.DefaultM365ClientID
	origTenantID := config.DefaultM365TenantID
	origRandom := randomStateFn

	t.Cleanup(func() {
		config.DefaultM365ClientID = origClientID
		config.DefaultM365TenantID = origTenantID
		randomStateFn = origRandom
	})

	config.DefaultM365ClientID = "client-id"
	config.DefaultM365TenantID = "organizations"
	randomStateFn = func() (string, error) { return "opaque-state", nil }

	session, err := CreateBrokerSession(context.Background(), BrokerSessionOptions{
		ExpectedEmail: "Bernardo@Hapvida.com.br",
		BaseURL:       "https://login.workit.ai",
		CallbackURL:   "https://login.workit.ai/m365/callback",
		Readonly:      true,
		TTL:           10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateBrokerSession: %v", err)
	}

	if session.LoginURL != "https://login.workit.ai/m365/start/opaque-state" {
		t.Fatalf("login url = %q", session.LoginURL)
	}

	if session.ExpectedEmail != "bernardo@hapvida.com.br" {
		t.Fatalf("expected email = %q", session.ExpectedEmail)
	}

	if session.CodeVerifier == "" || session.CodeChallenge == "" {
		t.Fatalf("expected PKCE material: %#v", session)
	}

	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	q := authURL.Query()
	if got := q.Get("redirect_uri"); got != "https://login.workit.ai/m365/callback" {
		t.Fatalf("redirect_uri = %q", got)
	}

	if got := q.Get("code_challenge_method"); got != "S256" {
		t.Fatalf("code_challenge_method = %q", got)
	}

	scope := q.Get("scope")
	for _, want := range []string{"offline_access", "User.Read", "Mail.Read", "Calendars.Read"} {
		if !strings.Contains(scope, want) {
			t.Fatalf("scope missing %s: %s", want, scope)
		}
	}

	for _, forbidden := range []string{"Mail.Send", "Calendars.ReadWrite"} {
		if strings.Contains(scope, forbidden) {
			t.Fatalf("scope contains forbidden %s: %s", forbidden, scope)
		}
	}
}

func TestCreateBrokerSessionFailsClosedForUnsafeInputs(t *testing.T) {
	_, err := CreateBrokerSession(context.Background(), BrokerSessionOptions{
		ExpectedEmail: "pilot@example.com",
		BaseURL:       "http://login.workit.ai",
		CallbackURL:   "https://login.workit.ai/m365/callback",
		Readonly:      true,
	})
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https base URL failure, got %v", err)
	}

	_, err = CreateBrokerSession(context.Background(), BrokerSessionOptions{
		ExpectedEmail: "pilot@example.com",
		BaseURL:       "https://login.workit.ai",
		CallbackURL:   "https://login.workit.ai/m365/callback",
	})
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("expected read-only failure, got %v", err)
	}
}
