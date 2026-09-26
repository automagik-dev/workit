package googleauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/automagik-dev/workit/internal/config"
)

// fakeGrantedScopesEndpoint installs a fake token endpoint (and tokeninfo
// endpoint) and restores the package globals when the test ends.
func fakeGrantedScopesEndpoint(t *testing.T, token http.HandlerFunc, tokenInfo http.HandlerFunc) {
	t.Helper()

	origRead := readClientCredentials
	origEndpoint := oauthEndpoint
	origTokenInfo := scopeInfoURL

	t.Cleanup(func() {
		readClientCredentials = origRead
		oauthEndpoint = origEndpoint
		scopeInfoURL = origTokenInfo
	})

	readClientCredentials = func(string) (config.ClientCredentials, error) {
		return config.ClientCredentials{ClientID: "id", ClientSecret: "secret"}, nil
	}

	tokenSrv := httptest.NewServer(token)
	t.Cleanup(tokenSrv.Close)

	oauthEndpoint = oauth2.Endpoint{AuthURL: tokenSrv.URL, TokenURL: tokenSrv.URL}

	if tokenInfo != nil {
		infoSrv := httptest.NewServer(tokenInfo)
		t.Cleanup(infoSrv.Close)

		scopeInfoURL = infoSrv.URL
	} else {
		scopeInfoURL = "http://127.0.0.1:1/tokeninfo-must-not-be-called"
	}
}

func grantedScopeTokenHandler(t *testing.T, scope string) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}

		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "rt" {
			http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
			return
		}

		payload := map[string]any{
			"access_token": "access",
			"token_type":   "Bearer",
			"expires_in":   3599,
		}
		if scope != "" {
			payload["scope"] = scope
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func TestInspectGrantedScopes_ReadOnlyGrant(t *testing.T) {
	fakeGrantedScopesEndpoint(t, grantedScopeTokenHandler(t,
		"https://www.googleapis.com/auth/spreadsheets.readonly openid https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/userinfo.email",
	), nil)

	report, err := InspectGrantedScopes(context.Background(), "brain-ro", "rt", time.Second)
	if err != nil {
		t.Fatalf("InspectGrantedScopes: %v", err)
	}

	wantGranted := []string{
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/spreadsheets.readonly",
		"https://www.googleapis.com/auth/userinfo.email",
		"openid",
	}
	if !reflect.DeepEqual(report.Granted, wantGranted) {
		t.Fatalf("granted = %v, want %v", report.Granted, wantGranted)
	}

	if !report.ReadOnly || len(report.WriteScopes) != 0 {
		t.Fatalf("expected read-only report, got %#v", report)
	}

	if report.Source != GrantedScopesSourceTokenEndpoint {
		t.Fatalf("source = %q", report.Source)
	}
}

func TestInspectGrantedScopes_FullDriveIsNotReadOnly(t *testing.T) {
	// A --readonly request that Google answered with previously granted full
	// drive access (include_granted_scopes) must be reported as write-capable.
	fakeGrantedScopesEndpoint(t, grantedScopeTokenHandler(t,
		"openid https://www.googleapis.com/auth/drive https://www.googleapis.com/auth/spreadsheets.readonly",
	), nil)

	report, err := InspectGrantedScopes(context.Background(), "default", "rt", time.Second)
	if err != nil {
		t.Fatalf("InspectGrantedScopes: %v", err)
	}

	if report.ReadOnly {
		t.Fatalf("expected read_only=false, got %#v", report)
	}

	if want := []string{"https://www.googleapis.com/auth/drive"}; !reflect.DeepEqual(report.WriteScopes, want) {
		t.Fatalf("write scopes = %v, want %v", report.WriteScopes, want)
	}
}

func TestInspectGrantedScopes_TokenInfoFallback(t *testing.T) {
	var infoCalls int

	fakeGrantedScopesEndpoint(t, grantedScopeTokenHandler(t, ""), func(w http.ResponseWriter, r *http.Request) {
		infoCalls++

		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}

		if r.Method != http.MethodPost || r.URL.Query().Get("access_token") != "" || r.PostForm.Get("access_token") != "access" {
			http.Error(w, "access token must be sent in the POST body", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"scope": "https://www.googleapis.com/auth/gmail.modify openid",
		})
	})

	report, err := InspectGrantedScopes(context.Background(), "default", "rt", time.Second)
	if err != nil {
		t.Fatalf("InspectGrantedScopes: %v", err)
	}

	if infoCalls != 1 {
		t.Fatalf("tokeninfo calls = %d, want 1", infoCalls)
	}

	if report.Source != GrantedScopesSourceTokenInfo || report.ReadOnly {
		t.Fatalf("unexpected report: %#v", report)
	}

	if want := []string{"https://www.googleapis.com/auth/gmail.modify"}; !reflect.DeepEqual(report.WriteScopes, want) {
		t.Fatalf("write scopes = %v, want %v", report.WriteScopes, want)
	}
}

func TestInspectGrantedScopes_InvalidGrant(t *testing.T) {
	fakeGrantedScopesEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "Token has been expired or revoked.",
		})
	}, nil)

	_, err := InspectGrantedScopes(context.Background(), "default", "rt", time.Second)
	if !errors.Is(err, ErrRefreshTokenRejected) {
		t.Fatalf("expected ErrRefreshTokenRejected, got %v", err)
	}
}

func TestInspectGrantedScopes_OtherTokenErrorIsNotAuth(t *testing.T) {
	fakeGrantedScopesEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_client"})
	}, nil)

	_, err := InspectGrantedScopes(context.Background(), "default", "rt", time.Second)
	if err == nil || errors.Is(err, ErrRefreshTokenRejected) {
		t.Fatalf("expected a non-auth refresh error, got %v", err)
	}
}

func TestInspectGrantedScopes_MissingRefreshToken(t *testing.T) {
	t.Parallel()

	if _, err := InspectGrantedScopes(context.Background(), "default", "  ", time.Second); err == nil {
		t.Fatalf("expected error for empty refresh token")
	}
}

func TestWriteScopes(t *testing.T) {
	t.Parallel()

	got := WriteScopes([]string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/spreadsheets.readonly",
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/calendar",
	})

	want := []string{
		"https://www.googleapis.com/auth/calendar",
		"https://www.googleapis.com/auth/drive",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WriteScopes = %v, want %v", got, want)
	}

	if empty := WriteScopes([]string{"openid", "https://www.googleapis.com/auth/drive.readonly"}); empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-nil empty slice, got %#v", empty)
	}
}
