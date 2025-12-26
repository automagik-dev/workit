//go:build darwin

package secrets

import (
	"testing"
)

func TestShouldTrustKeychainApplication_DefaultsTrue(t *testing.T) {
	t.Setenv(keychainTrustApplicationEnv, "")
	if !shouldTrustKeychainApplication() {
		t.Fatalf("expected true")
	}
}

func TestShouldTrustKeychainApplication_CanDisable(t *testing.T) {
	t.Setenv(keychainTrustApplicationEnv, "0")
	if shouldTrustKeychainApplication() {
		t.Fatalf("expected false")
	}
}
