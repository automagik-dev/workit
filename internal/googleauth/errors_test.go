package googleauth

import (
	"errors"
	"strings"
	"testing"
)

var (
	errOAuthUnauthorizedClient = errors.New("oauth2: unauthorized_client: something went wrong")
	errOAuthInvalidGrant       = errors.New("oauth2: invalid_grant: Token has been revoked")
	errOAuthInvalidClient      = errors.New("oauth2: invalid_client: The OAuth client was not found")
	errOAuthUnknown            = errors.New("some totally unrelated error")
	errOAuthWrappedGrant       = errors.New("invalid_grant: bad token")
)

func TestWrapOAuthError_Nil(t *testing.T) {
	if err := WrapOAuthError(nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWrapOAuthError_UnauthorizedClient(t *testing.T) {
	orig := errOAuthUnauthorizedClient

	wrapped := WrapOAuthError(orig)
	if wrapped == nil {
		t.Fatal("expected non-nil error")
	}

	msg := wrapped.Error()

	if !strings.Contains(msg, "hint: refresh token expired") {
		t.Fatalf("expected hint about refresh token expired, got: %s", msg)
	}

	if !strings.Contains(msg, "unauthorized_client") {
		t.Fatalf("expected original error text preserved, got: %s", msg)
	}

	// Verify original error is preserved for unwrapping
	if !errors.Is(wrapped, orig) {
		t.Fatal("errors.Is should find the original error")
	}
}

func TestWrapOAuthError_InvalidGrant(t *testing.T) {
	orig := errOAuthInvalidGrant

	wrapped := WrapOAuthError(orig)
	if wrapped == nil {
		t.Fatal("expected non-nil error")
	}

	msg := wrapped.Error()

	if !strings.Contains(msg, "hint: token revoked or invalid") {
		t.Fatalf("expected hint about token revoked, got: %s", msg)
	}

	if !strings.Contains(msg, "invalid_grant") {
		t.Fatalf("expected original error text preserved, got: %s", msg)
	}

	if !errors.Is(wrapped, orig) {
		t.Fatal("errors.Is should find the original error")
	}
}

func TestWrapOAuthError_InvalidClient(t *testing.T) {
	orig := errOAuthInvalidClient

	wrapped := WrapOAuthError(orig)
	if wrapped == nil {
		t.Fatal("expected non-nil error")
	}

	msg := wrapped.Error()

	if !strings.Contains(msg, "hint: client_id/secret invalid") {
		t.Fatalf("expected hint about client_id/secret, got: %s", msg)
	}

	if !strings.Contains(msg, "invalid_client") {
		t.Fatalf("expected original error text preserved, got: %s", msg)
	}

	if !errors.Is(wrapped, orig) {
		t.Fatal("errors.Is should find the original error")
	}
}

func TestWrapOAuthError_UnknownPassthrough(t *testing.T) {
	orig := errOAuthUnknown
	wrapped := WrapOAuthError(orig)

	if !errors.Is(wrapped, orig) {
		t.Fatalf("unknown error should pass through unchanged and preserve error identity, got: %v", wrapped)
	}
}

func TestWrapOAuthError_WrappedOriginalPreserved(t *testing.T) {
	// Verify errors.Is works through the wrapping chain
	inner := errOAuthWrappedGrant
	outer := WrapOAuthError(inner)

	if !errors.Is(outer, inner) {
		t.Fatal("errors.Is should still find inner error after WrapOAuthError")
	}
}
