package googleauth

import (
	"errors"
	"fmt"
	"testing"
)

var errOtherPoll = errors.New("something else")

func TestIsPollTimeout(t *testing.T) {
	// Direct poll timeout error.
	if !IsPollTimeout(errPollTimeout) {
		t.Fatal("expected IsPollTimeout(errPollTimeout) = true")
	}

	// Wrapped poll timeout error.
	wrapped := fmt.Errorf("outer: %w", errPollTimeout)
	if !IsPollTimeout(wrapped) {
		t.Fatal("expected IsPollTimeout(wrapped) = true")
	}

	// Unrelated error.
	if IsPollTimeout(errOtherPoll) {
		t.Fatal("expected IsPollTimeout(other) = false")
	}

	// Nil error.
	if IsPollTimeout(nil) {
		t.Fatal("expected IsPollTimeout(nil) = false")
	}
}
