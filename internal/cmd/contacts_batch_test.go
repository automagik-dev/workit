package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/outfmt"
)

func TestContactsBatchCreate_ParsesJSONInput(t *testing.T) {
	// Test that parseContactInputs correctly parses JSON input.
	input := `[
		{"givenName": "Alice", "familyName": "Smith", "email": "alice@example.com"},
		{"givenName": "Bob", "familyName": "Jones", "phone": "+1555000"}
	]`
	contacts, err := parseContactInputs(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts))
	}
	if contacts[0].GivenName != "Alice" {
		t.Fatalf("expected GivenName=Alice, got %q", contacts[0].GivenName)
	}
	if contacts[0].Email != "alice@example.com" {
		t.Fatalf("expected Email=alice@example.com, got %q", contacts[0].Email)
	}
	if contacts[1].Phone != "+1555000" {
		t.Fatalf("expected Phone=+1555000, got %q", contacts[1].Phone)
	}
}

func TestContactsBatchCreate_LimitedReader(t *testing.T) {
	// Test that parseContactInputs enforces the size limit.
	// Create input larger than maxBatchInputSize.
	big := strings.Repeat(`[{"givenName":"A","familyName":"B"}]`, maxBatchInputSize/35+1)
	_, err := parseContactInputs(strings.NewReader(big))
	if err == nil {
		t.Fatal("expected error for oversized input")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected 'too large' error, got: %v", err)
	}
}

func TestContactsBatchCreate_EmptyInputError(t *testing.T) {
	_, err := parseContactInputs(bytes.NewReader([]byte("[]")))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "no contacts") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsBatchCreate_InvalidJSONError(t *testing.T) {
	_, err := parseContactInputs(bytes.NewReader([]byte("not json")))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestContactsBatchCreate_DryRun(t *testing.T) {
	ctx := context.Background()
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})
	ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{})

	input := `[{"givenName": "Alice", "familyName": "Smith"}]`

	out := captureStdout(t, func() {
		cmd := &ContactsBatchCreateCmd{File: "-"}
		withStdin(t, input, func() {
			err := cmd.Run(ctx, &RootFlags{DryRun: true, Account: "test@example.com"})
			var exitErr *ExitError
			if !errors.As(err, &exitErr) || exitErr.Code != 0 {
				t.Fatalf("expected exit code 0 (dry run), got: %v", err)
			}
		})
	})

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput=%q", err, out)
	}
	if got["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got=%v", got["dry_run"])
	}
}

func TestContactsBatchCreate_ChunkSize(t *testing.T) {
	// Verify the batch chunk size constant is set to 200 (API limit).
	if batchCreateChunkSize != 200 {
		t.Fatalf("expected batchCreateChunkSize=200, got %d", batchCreateChunkSize)
	}
}

func TestContactsBatchDelete_EmptyNamesError(t *testing.T) {
	ctx := context.Background()
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})
	ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{})

	cmd := &ContactsBatchDeleteCmd{}
	err := cmd.Run(ctx, &RootFlags{Account: "test@example.com"})
	if err == nil {
		t.Fatal("expected error for empty resource names")
	}
	if !strings.Contains(err.Error(), "resource names") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsBatchDelete_DryRun(t *testing.T) {
	ctx := context.Background()
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})
	ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{})

	out := captureStdout(t, func() {
		cmd := &ContactsBatchDeleteCmd{
			ResourceNames: []string{"people/123", "people/456"},
		}
		err := cmd.Run(ctx, &RootFlags{DryRun: true, Account: "test@example.com"})
		var exitErr *ExitError
		if !errors.As(err, &exitErr) || exitErr.Code != 0 {
			t.Fatalf("expected exit code 0 (dry run), got: %v", err)
		}
	})

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput=%q", err, out)
	}
	if got["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got=%v", got["dry_run"])
	}
}

func TestContactsBatchDelete_ParsesFileInput(t *testing.T) {
	// parseResourceNames should parse JSON array of strings from reader.
	input := `["people/111", "people/222", "people/333"]`
	names, err := parseResourceNames(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "people/111" {
		t.Fatalf("expected people/111, got %q", names[0])
	}
}

func TestContactsBatchDelete_InvalidJSONFile(t *testing.T) {
	_, err := parseResourceNames(bytes.NewReader([]byte("not json")))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
