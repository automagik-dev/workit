package googleauth

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestHandleAccountsPage(t *testing.T) {
	ms := &ManageServer{csrfToken: "csrf123"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ms.handleAccountsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "csrf123") {
		t.Fatalf("expected csrf token in page")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/nope", nil)
	ms.handleAccountsPage(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for bad path")
	}
}

func TestFetchUserEmailDefault(t *testing.T) {
	if _, err := fetchUserEmailDefault(context.TODO(), nil); err == nil {
		t.Fatalf("expected missing token error")
	}

	if _, err := fetchUserEmailDefault(context.TODO(), &oauth2.Token{}); err == nil {
		t.Fatalf("expected missing access token error")
	}

	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"a@b.com"}`))
	idToken := "x." + payload + ".y"
	tok := &oauth2.Token{AccessToken: "access"}
	tok = tok.WithExtra(map[string]any{"id_token": idToken})

	email, err := fetchUserEmailDefault(context.TODO(), tok)
	if err != nil {
		t.Fatalf("fetchUserEmailDefault: %v", err)
	}
	if email != "a@b.com" {
		t.Fatalf("unexpected email: %q", email)
	}
}

func TestReadHTTPBodySnippet(t *testing.T) {
	out := readHTTPBodySnippet(strings.NewReader(""), 10)
	if out != "" {
		t.Fatalf("expected empty snippet")
	}

	out = readHTTPBodySnippet(strings.NewReader("access_token=secret"), 100)
	if !strings.Contains(out, "response_sha256=") {
		t.Fatalf("expected redacted hash, got: %q", out)
	}
}

func TestRenderSuccessPageWithDetails_More(t *testing.T) {
	rec := httptest.NewRecorder()
	renderSuccessPageWithDetails(rec, "a@b.com", []string{"gmail"})

	if !strings.Contains(rec.Body.String(), "a@b.com") {
		t.Fatalf("expected email in success page")
	}
}
