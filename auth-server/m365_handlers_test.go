package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestM365SessionsCreatesEnterpriseOneClickURL(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		Store:              NewTokenStore(DefaultTTL),
		GoogleClientID:     "google-client",
		GoogleClientSecret: "google-secret",
		GoogleRedirectURL:  "https://auth.hv.example/callback",
		M365Enabled:        true,
		M365ClientID:       "m365-client",
		M365TenantID:       "hapvida-tenant",
		M365AdminToken:     "admin-token",
		PublicBaseURL:      "https://auth.hv.example",
	})

	req := httptest.NewRequest(http.MethodPost, "/m365/sessions", strings.NewReader(`{"expected_email":"Bernardo@Hapvida.com.br"}`))
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	loginURL, _ := body["login_url"].(string)
	if !strings.HasPrefix(loginURL, "https://auth.hv.example/m365/start/") {
		t.Fatalf("login_url = %q", loginURL)
	}
	if body["expected_email"] != "bernardo@hapvida.com.br" {
		t.Fatalf("expected_email = %#v", body["expected_email"])
	}
	if _, exists := body["code_verifier"]; exists {
		t.Fatalf("code_verifier leaked in response: %s", rec.Body.String())
	}
}

func TestM365SessionsRequireAdminBearerToken(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		Store:              NewTokenStore(DefaultTTL),
		GoogleClientID:     "google-client",
		GoogleClientSecret: "google-secret",
		GoogleRedirectURL:  "https://auth.hv.example/callback",
		M365Enabled:        true,
		M365ClientID:       "m365-client",
		M365TenantID:       "hapvida-tenant",
		M365AdminToken:     "admin-token",
		PublicBaseURL:      "https://auth.hv.example",
	})

	req := httptest.NewRequest(http.MethodPost, "/m365/sessions", strings.NewReader(`{"expected_email":"bernardo@hapvida.com.br"}`))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestM365StartRedirectsToMicrosoftAuthorize(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		Store:              NewTokenStore(DefaultTTL),
		GoogleClientID:     "google-client",
		GoogleClientSecret: "google-secret",
		GoogleRedirectURL:  "https://auth.hv.example/callback",
		M365Enabled:        true,
		M365ClientID:       "m365-client",
		M365TenantID:       "hapvida-tenant",
		M365AdminToken:     "admin-token",
		PublicBaseURL:      "https://auth.hv.example",
	})

	createReq := httptest.NewRequest(http.MethodPost, "/m365/sessions", strings.NewReader(`{"expected_email":"bernardo@hapvida.com.br"}`))
	createReq.Header.Set("Authorization", "Bearer admin-token")
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, createReq)

	var body map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	startPath := strings.TrimPrefix(body["login_url"].(string), "https://auth.hv.example")
	startReq := httptest.NewRequest(http.MethodGet, startPath, nil)
	startRec := httptest.NewRecorder()
	server.ServeHTTP(startRec, startReq)

	if startRec.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", startRec.Code, startRec.Body.String())
	}

	location := startRec.Header().Get("Location")
	for _, want := range []string{"https://login.microsoftonline.com/hapvida-tenant/oauth2/v2.0/authorize", "client_id=m365-client", "redirect_uri=https%3A%2F%2Fauth.hv.example%2Fm365%2Fcallback", "code_challenge_method=S256"} {
		if !strings.Contains(location, want) {
			t.Fatalf("redirect missing %s: %s", want, location)
		}
	}
	for _, forbidden := range []string{"Mail.Send", "Calendars.ReadWrite"} {
		if strings.Contains(location, forbidden) {
			t.Fatalf("redirect contains forbidden scope %s: %s", forbidden, location)
		}
	}
}

func TestM365StartUnknownStateRendersHTML(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		Store:              NewTokenStore(DefaultTTL),
		GoogleClientID:     "google-client",
		GoogleClientSecret: "google-secret",
		GoogleRedirectURL:  "https://auth.hv.example/callback",
		M365Enabled:        true,
		M365ClientID:       "m365-client",
		M365TenantID:       "hapvida-tenant",
		M365AdminToken:     "admin-token",
		PublicBaseURL:      "https://auth.hv.example",
	})

	req := httptest.NewRequest(http.MethodGet, "/m365/start/missing", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestValidateM365EmailDoesNotExposePII(t *testing.T) {
	err := validateM365Email("bernardo@hapvida.com.br", "other@hapvida.com.br")
	if err == nil {
		t.Fatal("expected mismatch")
	}
	if strings.Contains(err.Error(), "bernardo") || strings.Contains(err.Error(), "other") {
		t.Fatalf("PII leaked in error: %v", err)
	}
}

func TestM365SessionStorePrunesExpiredOnSave(t *testing.T) {
	store := newM365SessionStore()
	store.save(m365Session{State: "expired", ExpectedEmail: "pilot@example.com", ExpiresAt: time.Now().Add(-time.Minute)})
	store.save(m365Session{State: "fresh", ExpectedEmail: "pilot@example.com", ExpiresAt: time.Now().Add(time.Minute)})

	if _, err := store.get("expired"); err == nil {
		t.Fatal("expected expired session to be pruned")
	}
	if _, err := store.get("fresh"); err != nil {
		t.Fatalf("fresh session: %v", err)
	}
}

func TestM365SessionsFailClosedWhenServerConfigMissing(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		Store:              NewTokenStore(DefaultTTL),
		GoogleClientID:     "google-client",
		GoogleClientSecret: "google-secret",
		GoogleRedirectURL:  "https://auth.hv.example/callback",
		M365Enabled:        true,
		PublicBaseURL:      "https://auth.hv.example",
	})

	req := httptest.NewRequest(http.MethodPost, "/m365/sessions", strings.NewReader(`{"expected_email":"bernardo@hapvida.com.br"}`))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "token") {
		t.Fatalf("unexpected sensitive body: %s", rec.Body.String())
	}
}

const testM365TenantID = "11111111-2222-3333-4444-555555555555"

// m365TestBroker runs the real handlers against a stub Microsoft login + Graph server.
type m365TestBroker struct {
	server   *Server
	audit    *bytes.Buffer
	meEmail  string
	orgID    string
	verifier string
}

func newM365TestBroker(t *testing.T) *m365TestBroker {
	t.Helper()

	b := &m365TestBroker{audit: &bytes.Buffer{}, meEmail: "Bernardo@Hapvida.com.br", orgID: testM365TenantID}
	ms := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/"+testM365TenantID+"/oauth2/v2.0/token":
			_ = r.ParseForm()
			b.verifier = r.PostForm.Get("code_verifier")
			_, _ = io.WriteString(w, `{"access_token":"at-secret","refresh_token":"rt-secret","token_type":"Bearer","expires_in":3600}`)
		case r.URL.Path == "/me":
			_ = json.NewEncoder(w).Encode(map[string]string{"mail": b.meEmail})
		case r.URL.Path == "/organization":
			_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{map[string]any{
				"id":              b.orgID,
				"verifiedDomains": []any{map[string]string{"name": "hapvida.com.br"}},
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ms.Close)

	origLogin, origGraph, origAudit := m365LoginBaseURL, m365GraphBaseURL, m365Audit
	t.Cleanup(func() { m365LoginBaseURL, m365GraphBaseURL, m365Audit = origLogin, origGraph, origAudit })
	m365LoginBaseURL, m365GraphBaseURL = ms.URL, ms.URL
	m365Audit = slog.New(slog.NewJSONHandler(b.audit, nil))

	b.server = NewServerWithOptions(ServerOptions{
		Store:          NewTokenStore(DefaultTTL),
		M365Enabled:    true,
		M365ClientID:   "m365-client",
		M365TenantID:   testM365TenantID,
		M365AdminToken: "admin-token",
		PublicBaseURL:  "https://auth.hv.example",
	})

	return b
}

func (b *m365TestBroker) do(method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	b.server.ServeHTTP(rec, req)

	return rec
}

// newSession creates a login link and follows it, returning the state, poll key and Microsoft authorize URL.
func (b *m365TestBroker) newSession(t *testing.T) (string, string, *url.URL) {
	t.Helper()

	rec := b.do(http.MethodPost, "/m365/sessions", `{"expected_email":"bernardo@hapvida.com.br"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create session status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		State    string `json:"state"`
		PollKey  string `json:"poll_key"`
		LoginURL string `json:"login_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if body.PollKey == "" || body.PollKey == body.State || strings.Contains(body.LoginURL, body.PollKey) {
		t.Fatalf("poll key must be separate from the public link: %s", rec.Body.String())
	}

	start := b.do(http.MethodGet, strings.TrimPrefix(body.LoginURL, "https://auth.hv.example"), "")
	if start.Code != http.StatusFound {
		t.Fatalf("start status = %d", start.Code)
	}
	authURL, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}

	return body.State, body.PollKey, authURL
}

func (b *m365TestBroker) callback(state string) *httptest.ResponseRecorder {
	return b.do(http.MethodGet, "/m365/callback?code=auth-code&state="+url.QueryEscape(state), "")
}

func TestM365CallbackStoresTokenUnderPollKeyOnly(t *testing.T) {
	b := newM365TestBroker(t)
	state, pollKey, authURL := b.newSession(t)

	rec := b.callback(state)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Microsoft 365 connected read-only") {
		t.Fatalf("callback status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), state) {
		t.Fatal("success page must not echo the state")
	}

	sum := sha256.Sum256([]byte(b.verifier))
	if got := base64.RawURLEncoding.EncodeToString(sum[:]); b.verifier == "" || got != authURL.Query().Get("code_challenge") {
		t.Fatalf("token exchange did not use the PKCE verifier bound to this session")
	}

	// The public state (it travels in the chat link) must never unlock the token.
	if rec := b.do(http.MethodGet, "/token/"+state, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("token by public state status = %d body = %s", rec.Code, rec.Body.String())
	}
	tokRec := b.do(http.MethodGet, "/token/"+pollKey, "")
	if tokRec.Code != http.StatusOK || !strings.Contains(tokRec.Body.String(), `"refresh_token":"rt-secret"`) {
		t.Fatalf("token by poll key status = %d body = %s", tokRec.Code, tokRec.Body.String())
	}

	audit := b.audit.String()
	for _, want := range []string{`"msg":"m365.link_created"`, `"msg":"m365.callback_ok"`, `"email":"b***@hapvida.com.br"`, `"tenant":"` + testM365TenantID + `"`} {
		if !strings.Contains(audit, want) {
			t.Fatalf("audit missing %s:\n%s", want, audit)
		}
	}
	for _, secret := range []string{"rt-secret", "at-secret", "auth-code", pollKey, state, b.verifier, "bernardo@"} {
		if strings.Contains(strings.ToLower(audit), strings.ToLower(secret)) {
			t.Fatalf("audit leaked %q:\n%s", secret, audit)
		}
	}
}

func TestM365AuthorizeURLRequestsOnlyReadOnlyScopes(t *testing.T) {
	b := newM365TestBroker(t)
	_, _, authURL := b.newSession(t)

	got := strings.Fields(authURL.Query().Get("scope"))
	want := []string{"offline_access", "User.Read", "Mail.Read", "Calendars.Read"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("scope = %v, want %v", got, want)
	}
	for _, scope := range got {
		if strings.Contains(scope, "Write") || strings.Contains(scope, "Send") || strings.HasSuffix(scope, ".All") {
			t.Fatalf("write-capable scope requested: %s", scope)
		}
	}
}

func TestM365CallbackStateIsOneTime(t *testing.T) {
	b := newM365TestBroker(t)
	state, _, _ := b.newSession(t)

	if rec := b.callback(state); rec.Code != http.StatusOK {
		t.Fatalf("first callback status = %d", rec.Code)
	}
	if rec := b.callback(state); rec.Code != http.StatusBadRequest {
		t.Fatalf("replayed callback status = %d", rec.Code)
	}
	if !strings.Contains(b.audit.String(), `"reason":"state_not_found_or_reused"`) {
		t.Fatalf("audit missing replay failure:\n%s", b.audit.String())
	}
}

func TestM365CallbackExpiredStateFailsClosed(t *testing.T) {
	b := newM365TestBroker(t)
	state, pollKey, _ := b.newSession(t)

	session := b.server.m365Sessions.sessions[state]
	session.ExpiresAt = time.Now().Add(-time.Second)
	b.server.m365Sessions.sessions[state] = session

	if rec := b.callback(state); rec.Code != http.StatusBadRequest {
		t.Fatalf("expired callback status = %d", rec.Code)
	}
	if b.verifier != "" {
		t.Fatal("expired state must not reach the token exchange")
	}
	if rec := b.do(http.MethodGet, "/token/"+pollKey, ""); rec.Code != http.StatusAccepted {
		t.Fatalf("token after expired callback status = %d", rec.Code)
	}
	if !strings.Contains(b.audit.String(), `"msg":"m365.state_expired"`) {
		t.Fatalf("audit missing state_expired:\n%s", b.audit.String())
	}
}

func TestM365CallbackIdentityMismatchFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*m365TestBroker)
		reason string
	}{
		{"email", func(b *m365TestBroker) { b.meEmail = "someone.else@hapvida.com.br" }, "email_mismatch"},
		{"tenant", func(b *m365TestBroker) { b.orgID = "99999999-0000-0000-0000-000000000000" }, "tenant_mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newM365TestBroker(t)
			tc.mutate(b)
			state, pollKey, _ := b.newSession(t)

			if rec := b.callback(state); rec.Code != http.StatusForbidden {
				t.Fatalf("callback status = %d body = %s", rec.Code, rec.Body.String())
			}
			if rec := b.do(http.MethodGet, "/token/"+pollKey, ""); rec.Code != http.StatusAccepted {
				t.Fatalf("token must not be stored on mismatch, status = %d", rec.Code)
			}
			if !strings.Contains(b.audit.String(), `"reason":"`+tc.reason+`"`) || strings.Contains(b.audit.String(), "rt-secret") {
				t.Fatalf("audit = %s", b.audit.String())
			}
		})
	}
}

func TestM365SessionsRefuseUnpinnedTenant(t *testing.T) {
	for _, tenant := range []string{"", "organizations", "common"} {
		server := NewServerWithOptions(ServerOptions{
			Store:          NewTokenStore(DefaultTTL),
			M365Enabled:    true,
			M365ClientID:   "m365-client",
			M365TenantID:   tenant,
			M365AdminToken: "admin-token",
			PublicBaseURL:  "https://auth.hv.example",
		})
		req := httptest.NewRequest(http.MethodPost, "/m365/sessions", strings.NewReader(`{"expected_email":"bernardo@hapvida.com.br"}`))
		req.Header.Set("Authorization", "Bearer admin-token")
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "tenant") {
			t.Fatalf("tenant %q: status = %d body = %s", tenant, rec.Code, rec.Body.String())
		}
	}
}

func TestValidateM365TenantAcceptsIDOrVerifiedDomain(t *testing.T) {
	if err := validateM365Tenant(testM365TenantID, strings.ToUpper(testM365TenantID), nil); err != nil {
		t.Fatalf("tenant id: %v", err)
	}
	if err := validateM365Tenant("hapvida.com.br", testM365TenantID, []string{"Hapvida.com.br"}); err != nil {
		t.Fatalf("verified domain: %v", err)
	}
	if err := validateM365Tenant("hapvida.com.br", testM365TenantID, []string{"contoso.com"}); err == nil {
		t.Fatal("expected mismatch")
	}
}
