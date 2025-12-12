package secrets

import "testing"

func TestTokenKey(t *testing.T) {
	if got := tokenKey("a@b.com"); got != "token:a@b.com" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestParseTokenKey(t *testing.T) {
	email, ok := ParseTokenKey("token:a@b.com")
	if !ok {
		t.Fatalf("expected ok")
	}
	if email != "a@b.com" {
		t.Fatalf("unexpected: %q", email)
	}
	if _, ok := ParseTokenKey("nope"); ok {
		t.Fatalf("expected not ok")
	}
}
