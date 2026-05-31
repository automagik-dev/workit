package msauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/automagik-dev/workit/internal/config"
)

func TestFetchEmailUsesMailThenUserPrincipalName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}

		_, _ = w.Write([]byte(`{"mail":"","userPrincipalName":"bernardo@hapvida.com.br"}`))
	}))
	defer server.Close()

	origURL := graphMeURL

	t.Cleanup(func() { graphMeURL = origURL })
	graphMeURL = server.URL

	email, err := FetchEmail(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("FetchEmail: %v", err)
	}

	if email != "bernardo@hapvida.com.br" {
		t.Fatalf("email = %q", email)
	}
}

func TestFetchEmailFailsClosedOnBadProfileStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	origURL := graphMeURL

	t.Cleanup(func() { graphMeURL = origURL })
	graphMeURL = server.URL

	_, err := FetchEmail(context.Background(), "access-token")
	if !errors.Is(err, ErrProfileStatus) {
		t.Fatalf("expected ErrProfileStatus, got: %v", err)
	}
}

func TestHandleM365OAuthCallbackValidatesStateAndCode(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	req := httptest.NewRequest(http.MethodGet, "/oauth2/callback?state=good&code=abc", nil)
	rec := httptest.NewRecorder()
	handleM365OAuthCallback(rec, req, "good", codeCh, errCh)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	if got := <-codeCh; got != "abc" {
		t.Fatalf("code = %q", got)
	}

	badReq := httptest.NewRequest(http.MethodGet, "/oauth2/callback?state=bad&code=abc", nil)
	badRec := httptest.NewRecorder()
	handleM365OAuthCallback(badRec, badReq, "good", codeCh, errCh)

	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("bad status = %d", badRec.Code)
	}

	if err := <-errCh; !errors.Is(err, ErrStateMismatch) {
		t.Fatalf("expected state mismatch, got: %v", err)
	}
}

func TestHandleM365OAuthCallbackReportsProviderErrorAndMissingCode(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 2)

	errorReq := httptest.NewRequest(http.MethodGet, "/oauth2/callback?error=access_denied", nil)
	errorRec := httptest.NewRecorder()
	handleM365OAuthCallback(errorRec, errorReq, "state", codeCh, errCh)

	if err := <-errCh; !errors.Is(err, ErrAuthorization) {
		t.Fatalf("expected ErrAuthorization, got: %v", err)
	}

	missingCodeReq := httptest.NewRequest(http.MethodGet, "/oauth2/callback?state=state", nil)
	missingCodeRec := httptest.NewRecorder()
	handleM365OAuthCallback(missingCodeRec, missingCodeReq, "state", codeCh, errCh)

	if missingCodeRec.Code != http.StatusBadRequest {
		t.Fatalf("missing-code status = %d", missingCodeRec.Code)
	}

	if err := <-errCh; !errors.Is(err, ErrMissingCode) {
		t.Fatalf("expected ErrMissingCode, got: %v", err)
	}
}

func TestExchangeCodeAndProfileRequiresRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"access-token","token_type":"Bearer"}`))

			return
		}

		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	cfg := oauth2.Config{
		ClientID:    "client-id",
		Endpoint:    oauth2.Endpoint{TokenURL: server.URL + "/token"},
		RedirectURL: "http://localhost:8085/oauth2/callback",
		Scopes:      []string{"User.Read"},
	}

	_, err := exchangeCodeAndProfile(context.Background(), &http.Server{}, cfg, "code", "verifier")
	if !errors.Is(err, ErrNoRefreshToken) {
		t.Fatalf("expected ErrNoRefreshToken, got: %v", err)
	}
}

func TestExchangeCodeAndProfileStoresSuccessfulProfileEmail(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/token") {
			t.Fatalf("unexpected token path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","token_type":"Bearer"}`))
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mail":"bernardo@hapvida.com.br"}`))
	}))
	defer graphServer.Close()

	origURL := graphMeURL

	t.Cleanup(func() { graphMeURL = origURL })
	graphMeURL = graphServer.URL

	cfg := oauth2.Config{
		ClientID:    "client-id",
		Endpoint:    oauth2.Endpoint{TokenURL: tokenServer.URL + "/token"},
		RedirectURL: "http://localhost:8085/oauth2/callback",
		Scopes:      []string{"User.Read"},
	}

	result, err := exchangeCodeAndProfile(context.Background(), &http.Server{}, cfg, "code", "verifier")
	if err != nil {
		t.Fatalf("exchangeCodeAndProfile: %v", err)
	}

	if result.Email != "bernardo@hapvida.com.br" || result.RefreshToken != "refresh-token" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAuthorizeCompletesBrowserOAuthWithPKCE(t *testing.T) {
	origClientID := config.DefaultM365ClientID
	origTenantID := config.DefaultM365TenantID
	origRandom := randomStateFn
	origOpen := openBrowserFn
	origOAuthConfig := oauthConfigFn
	origGraphURL := graphMeURL

	t.Cleanup(func() {
		config.DefaultM365ClientID = origClientID
		config.DefaultM365TenantID = origTenantID
		randomStateFn = origRandom
		openBrowserFn = origOpen
		oauthConfigFn = origOAuthConfig
		graphMeURL = origGraphURL
	})

	config.DefaultM365ClientID = "client-id"
	config.DefaultM365TenantID = "organizations"
	randomStateFn = func() (string, error) { return "fixed-state", nil }

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","token_type":"Bearer"}`))
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mail":"bernardo@hapvida.com.br"}`))
	}))
	defer graphServer.Close()
	graphMeURL = graphServer.URL

	oauthConfigFn = func(settings oauthSettings, redirectURI string, scopes []string) oauth2.Config {
		return oauth2.Config{
			ClientID:    settings.ClientID,
			Endpoint:    oauth2.Endpoint{AuthURL: tokenServer.URL + "/authorize", TokenURL: tokenServer.URL + "/token"},
			RedirectURL: redirectURI,
			Scopes:      scopes,
		}
	}
	openBrowserFn = func(_ string) error {
		go func() {
			time.Sleep(25 * time.Millisecond)
			callbackURL := "http://localhost:8085/oauth2/callback?state=fixed-state&" + "code=ok"

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, callbackURL, nil)
			if err != nil {
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()

		return nil
	}

	result, err := Authorize(context.Background(), AuthorizeOptions{Readonly: true})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if result.Email != "bernardo@hapvida.com.br" || result.RefreshToken != "refresh-token" {
		t.Fatalf("result = %#v", result)
	}
}

func TestServicesInfoReturnsM365ReadOnlyMetadata(t *testing.T) {
	infos := ServicesInfo()
	if len(infos) != 1 || infos[0].Service != "m365" {
		t.Fatalf("unexpected infos: %#v", infos)
	}

	for _, forbidden := range []string{"Mail.Send", "Calendars.ReadWrite"} {
		for _, scope := range infos[0].Scopes {
			if scope == forbidden {
				t.Fatalf("forbidden scope exposed: %s", forbidden)
			}
		}
	}
}
