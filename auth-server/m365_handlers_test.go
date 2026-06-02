package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
