package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/automagik-dev/workit/internal/msauth"
	"github.com/automagik-dev/workit/internal/secrets"
)

func TestAuthM365LoginLinkPrintsServerCreatedOneClickURL(t *testing.T) {
	origCreate := createM365BrokerSessionOnServer
	t.Cleanup(func() { createM365BrokerSessionOnServer = origCreate })

	var gotBaseURL string
	var gotToken string
	var gotPayload remoteM365BrokerSessionRequest
	createM365BrokerSessionOnServer = func(_ context.Context, baseURL string, brokerToken string, payload remoteM365BrokerSessionRequest) (msauth.BrokerSession, error) {
		gotBaseURL = baseURL
		gotToken = brokerToken
		gotPayload = payload
		return msauth.BrokerSession{
			State:         "state",
			ExpectedEmail: "bernardo@hapvida.com.br",
			LoginURL:      "https://login.workit.ai/m365/start/state",
			ExpiresAt:     time.Unix(1893456000, 0).UTC(),
		}, nil
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "m365", "login-link", "bernardo@hapvida.com.br", "--base-url", "https://login.workit.ai", "--callback-url", "https://login.workit.ai/m365/callback", "--broker-token", "secret-token", "--no-poll"}); err != nil {
				t.Fatalf("login-link: %v", err)
			}
		})
	})

	if gotPayload.ExpectedEmail != "bernardo@hapvida.com.br" || gotPayload.ForceConsent {
		t.Fatalf("payload = %#v", gotPayload)
	}
	if gotBaseURL != "https://login.workit.ai" || gotToken != "secret-token" {
		t.Fatalf("remote args = base %q token %q", gotBaseURL, gotToken)
	}
	if !strings.Contains(out, "https://login.workit.ai/m365/start/state") {
		t.Fatalf("missing login link: %s", out)
	}
}

func TestAuthM365LoginLinkUsesCallbackServerDefaultWhenURLsOmitted(t *testing.T) {
	origCreate := createM365BrokerSessionOnServer
	t.Cleanup(func() { createM365BrokerSessionOnServer = origCreate })
	t.Setenv("WK_CALLBACK_SERVER", "https://auth.hv.example")
	t.Setenv("WK_M365_BROKER_TOKEN", "env-token")

	var gotBaseURL string
	createM365BrokerSessionOnServer = func(_ context.Context, baseURL string, brokerToken string, payload remoteM365BrokerSessionRequest) (msauth.BrokerSession, error) {
		gotBaseURL = baseURL
		if brokerToken != "env-token" {
			t.Fatalf("broker token = %q", brokerToken)
		}
		return msauth.BrokerSession{
			State:         "state",
			ExpectedEmail: "bernardo@hapvida.com.br",
			LoginURL:      "https://auth.hv.example/m365/start/state",
			ExpiresAt:     time.Unix(1893456000, 0).UTC(),
		}, nil
	}

	_ = captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "m365", "login-link", "bernardo@hapvida.com.br", "--no-poll"}); err != nil {
				t.Fatalf("login-link: %v", err)
			}
		})
	})

	if gotBaseURL != "https://auth.hv.example" {
		t.Fatalf("base url = %q", gotBaseURL)
	}
}

func TestAuthM365LoginLinkRejectsInvalidCallbackURL(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"auth", "m365", "login-link", "bernardo@hapvida.com.br", "--base-url", "https://auth.hv.example", "--callback-url", "not-a-url", "--broker-token", "token"})
		if err == nil {
			t.Fatal("expected invalid callback URL failure")
		}
		if !strings.Contains(err.Error(), "callback-url") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAuthM365LoginLinkRequiresBrokerToken(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"auth", "m365", "login-link", "bernardo@hapvida.com.br", "--base-url", "https://auth.hv.example"})
		if err == nil {
			t.Fatal("expected missing broker token failure")
		}
		if !strings.Contains(err.Error(), "broker-token") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCreateRemoteM365BrokerSessionPostsToBrokerWithBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/m365/sessions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		var req remoteM365BrokerSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ExpectedEmail != "bernardo@hapvida.com.br" {
			t.Fatalf("payload = %#v", req)
		}
		_ = json.NewEncoder(w).Encode(msauth.BrokerSession{
			State:         "state",
			ExpectedEmail: "bernardo@hapvida.com.br",
			LoginURL:      "https://auth.hv.example/m365/start/state",
			ExpiresAt:     time.Unix(1893456000, 0).UTC(),
		})
	}))
	defer server.Close()

	session, err := createRemoteM365BrokerSession(context.Background(), server.URL, "secret-token", remoteM365BrokerSessionRequest{ExpectedEmail: "bernardo@hapvida.com.br"})
	if err != nil {
		t.Fatalf("create remote session: %v", err)
	}
	if session.LoginURL != "https://auth.hv.example/m365/start/state" {
		t.Fatalf("login url = %q", session.LoginURL)
	}
}

func TestAuthM365LoginLinkWaitsAndStoresTokenUnderM365Client(t *testing.T) {
	origCreate, origPoll, origOpen, origKeychain := createM365BrokerSessionOnServer, pollForToken, openSecretsStore, ensureKeychainAccess
	t.Cleanup(func() {
		createM365BrokerSessionOnServer, pollForToken, openSecretsStore, ensureKeychainAccess = origCreate, origPoll, origOpen, origKeychain
	})

	store := newMemSecretsStore()
	openSecretsStore = func() (secrets.Store, error) { return store, nil }
	ensureKeychainAccess = func() error { return nil }
	createM365BrokerSessionOnServer = func(context.Context, string, string, remoteM365BrokerSessionRequest) (msauth.BrokerSession, error) {
		return msauth.BrokerSession{
			State:         "state",
			PollKey:       "poll-secret",
			ExpectedEmail: "bernardo@hapvida.com.br",
			LoginURL:      "https://auth.hv.example/m365/start/state",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		}, nil
	}
	var gotServer, gotKey string
	var gotTimeout time.Duration
	pollForToken = func(_ context.Context, server, key string, timeout time.Duration) (string, error) {
		gotServer, gotKey, gotTimeout = server, key, timeout
		return "m365-refresh-token", nil
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "m365", "login-link", "Bernardo@Hapvida.com.br", "--base-url", "https://auth.hv.example", "--broker-token", "admin"}); err != nil {
				t.Fatalf("login-link: %v", err)
			}
		})
	})

	if !strings.Contains(out, "https://auth.hv.example/m365/start/state") || !strings.Contains(out, `"stored": true`) {
		t.Fatalf("output = %s", out)
	}
	if strings.Contains(out, "poll-secret") {
		t.Fatalf("poll key leaked to output: %s", out)
	}
	if gotServer != "https://auth.hv.example" || gotKey != "poll-secret" || gotTimeout < 10*time.Minute {
		t.Fatalf("poll args = %q %q %s", gotServer, gotKey, gotTimeout)
	}
	tok, err := store.GetToken(msauth.ClientName, "bernardo@hapvida.com.br")
	if err != nil {
		t.Fatalf("stored m365 token: %v", err)
	}
	if tok.RefreshToken != "m365-refresh-token" || tok.Client != msauth.ClientName || strings.Join(tok.Services, ",") != "m365" {
		t.Fatalf("token = %#v", tok)
	}
}

func TestAuthM365LoginLinkWithoutPollKeyPrintsNoLink(t *testing.T) {
	origCreate, origKeychain := createM365BrokerSessionOnServer, ensureKeychainAccess
	t.Cleanup(func() { createM365BrokerSessionOnServer, ensureKeychainAccess = origCreate, origKeychain })

	ensureKeychainAccess = func() error { return nil }
	createM365BrokerSessionOnServer = func(context.Context, string, string, remoteM365BrokerSessionRequest) (msauth.BrokerSession, error) {
		return msauth.BrokerSession{ExpectedEmail: "bernardo@hapvida.com.br", LoginURL: "https://auth.hv.example/m365/start/state"}, nil
	}

	var err error
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			err = Execute([]string{"auth", "m365", "login-link", "bernardo@hapvida.com.br", "--base-url", "https://auth.hv.example", "--broker-token", "admin"})
		})
	})
	if err == nil || !strings.Contains(err.Error(), "poll_key") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(out, "m365/start") {
		t.Fatalf("uncollectable link was printed: %s", out)
	}
}
