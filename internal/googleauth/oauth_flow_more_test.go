package googleauth

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/automagik-dev/workit/internal/config"
)

func TestAuthURLParams(t *testing.T) {
	t.Parallel()

	cfg := oauth2.Config{
		ClientID:    "id",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://example.com/auth"},
		RedirectURL: "http://localhost",
		Scopes:      []string{"s1"},
	}

	u1 := cfg.AuthCodeURL("state", authURLParams(false, false)...)
	var parsed1 *url.URL

	if p, err := url.Parse(u1); err != nil {
		t.Fatalf("parse: %v", err)
	} else {
		parsed1 = p
	}

	if accessType := parsed1.Query().Get("access_type"); accessType != "offline" {
		t.Fatalf("expected offline, got: %q", accessType)
	}

	if includeScopes := parsed1.Query().Get("include_granted_scopes"); includeScopes != "true" {
		t.Fatalf("expected include_granted_scopes=true, got: %q", includeScopes)
	}

	if prompt := parsed1.Query().Get("prompt"); prompt != "" {
		t.Fatalf("expected no prompt, got: %q", prompt)
	}

	u2 := cfg.AuthCodeURL("state", authURLParams(true, false)...)
	var parsed2 *url.URL

	if p, err := url.Parse(u2); err != nil {
		t.Fatalf("parse: %v", err)
	} else {
		parsed2 = p
	}

	if parsed2.Query().Get("prompt") != "consent" {
		t.Fatalf("expected consent prompt, got: %q", parsed2.Query().Get("prompt"))
	}
}

func TestAuthURLParams_ReadonlyOmitsIncludeGrantedScopes(t *testing.T) {
	t.Parallel()

	cfg := oauth2.Config{
		ClientID:    "id",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://example.com/auth"},
		RedirectURL: "http://localhost",
		Scopes:      []string{"https://www.googleapis.com/auth/drive.readonly"},
	}

	for _, forceConsent := range []bool{false, true} {
		u, err := url.Parse(cfg.AuthCodeURL("state", authURLParams(forceConsent, true)...))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		q := u.Query()
		if _, present := q["include_granted_scopes"]; present {
			t.Fatalf("read-only consent URL must not carry include_granted_scopes (forceConsent=%v): %s", forceConsent, u)
		}

		if q.Get("access_type") != "offline" {
			t.Fatalf("expected access_type=offline, got %q", q.Get("access_type"))
		}

		if wantPrompt := map[bool]string{true: "consent", false: ""}[forceConsent]; q.Get("prompt") != wantPrompt {
			t.Fatalf("prompt = %q, want %q", q.Get("prompt"), wantPrompt)
		}
	}
}

func TestHeadlessAuthorize_ReadonlyOmitsIncludeGrantedScopes(t *testing.T) {
	origRead := readClientCredentials

	t.Cleanup(func() { readClientCredentials = origRead })

	readClientCredentials = func(string) (config.ClientCredentials, error) {
		return config.ClientCredentials{ClientID: "id", ClientSecret: "secret"}, nil
	}

	for _, readonly := range []bool{false, true} {
		info, err := HeadlessAuthorize(context.Background(), HeadlessOptions{
			Scopes:         []string{"https://www.googleapis.com/auth/drive.readonly"},
			Readonly:       readonly,
			CallbackServer: "https://relay.example.com",
		})
		if err != nil {
			t.Fatalf("HeadlessAuthorize: %v", err)
		}

		u, err := url.Parse(info.AuthURL)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		_, present := u.Query()["include_granted_scopes"]
		if present == readonly {
			t.Fatalf("readonly=%v: include_granted_scopes present=%v in %s", readonly, present, u)
		}
	}
}

func TestRandomState(t *testing.T) {
	t.Parallel()

	var s1 string

	if state, err := randomState(); err != nil {
		t.Fatalf("randomState: %v", err)
	} else {
		s1 = state
	}

	var s2 string

	if state, err := randomState(); err != nil {
		t.Fatalf("randomState: %v", err)
	} else {
		s2 = state
	}

	if s1 == "" || s2 == "" || s1 == s2 {
		t.Fatalf("expected two non-empty distinct states")
	}
	// base64 RawURLEncoding charset should not include '+' or '/' or '='.
	if strings.ContainsAny(s1, "+/=") || strings.ContainsAny(s2, "+/=") {
		t.Fatalf("unexpected charset: %q %q", s1, s2)
	}
}
