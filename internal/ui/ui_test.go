package ui

import (
	"bytes"
	"testing"
)

func TestUIColorFlagValidation(t *testing.T) {
	_, err := New(Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Color: "nope"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
