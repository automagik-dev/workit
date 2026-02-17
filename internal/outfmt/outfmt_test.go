package outfmt

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestFromFlags(t *testing.T) {
	if _, err := FromFlags(true, true); err == nil {
		t.Fatalf("expected error when combining --json and --plain")
	}

	got, err := FromFlags(true, false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if !got.JSON || got.Plain {
		t.Fatalf("unexpected mode: %#v", got)
	}
}

func TestContextMode(t *testing.T) {
	ctx := context.Background()

	if IsJSON(ctx) || IsPlain(ctx) {
		t.Fatalf("expected default text")
	}
	ctx = WithMode(ctx, Mode{JSON: true})

	if !IsJSON(ctx) || IsPlain(ctx) {
		t.Fatalf("expected json-only")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(context.Background(), &buf, map[string]any{"ok": true}); err != nil {
		t.Fatalf("err: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatalf("expected output")
	}
}

func TestWriteJSON_ResultsOnlyAndSelect(t *testing.T) {
	ctx := WithJSONTransform(context.Background(), JSONTransform{
		ResultsOnly: true,
		Select:      []string{"id"},
	})

	var buf bytes.Buffer
	if err := WriteJSON(ctx, &buf, map[string]any{
		"files": []map[string]any{
			{"id": "1", "name": "one"},
			{"id": "2", "name": "two"},
		},
		"nextPageToken": "tok",
	}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (out=%q)", err, buf.String())
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}

	if got[0]["id"] != "1" || got[1]["id"] != "2" {
		t.Fatalf("unexpected ids: %#v", got)
	}

	if _, ok := got[0]["name"]; ok {
		t.Fatalf("expected name to be stripped, got %#v", got[0])
	}
}

func TestWriteJSON_JQFilter(t *testing.T) {
	ctx := WithJSONTransform(context.Background(), JSONTransform{
		JQ: ".[].name",
	})

	var buf bytes.Buffer
	if err := WriteJSON(ctx, &buf, []map[string]any{
		{"name": "alice", "age": 25},
		{"name": "bob", "age": 35},
	}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	want := "\"alice\"\n\"bob\"\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteJSON_ResultsOnlyThenJQ(t *testing.T) {
	// Pipeline: --results-only strips the envelope, then --jq extracts fields.
	ctx := WithJSONTransform(context.Background(), JSONTransform{
		ResultsOnly: true,
		JQ:          "length",
	})

	var buf bytes.Buffer
	if err := WriteJSON(ctx, &buf, map[string]any{
		"files": []map[string]any{
			{"id": "1", "name": "one"},
			{"id": "2", "name": "two"},
			{"id": "3", "name": "three"},
		},
		"nextPageToken": "tok",
	}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	want := "3\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}

func TestFromEnvAndParseError(t *testing.T) {
	t.Setenv("GOG_JSON", "yes")
	t.Setenv("GOG_PLAIN", "0")
	mode := FromEnv()

	if !mode.JSON || mode.Plain {
		t.Fatalf("unexpected env mode: %#v", mode)
	}

	if err := (&ParseError{msg: "boom"}).Error(); err != "boom" {
		t.Fatalf("unexpected parse error: %q", err)
	}
}

func TestFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey{}, "nope")
	if got := FromContext(ctx); got != (Mode{}) {
		t.Fatalf("expected zero mode, got %#v", got)
	}
}

type fieldDiscoverySample struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func TestWriteJSON_FieldDiscovery(t *testing.T) {
	// When SelectExplicit is true and Select is empty, WriteJSON should write
	// field names to the FieldDiscoveryWriter and return immediately without
	// encoding the payload to stdout.
	var stderrBuf bytes.Buffer
	ctx := WithJSONTransform(context.Background(), JSONTransform{
		SelectExplicit:       true,
		Select:               nil,
		FieldDiscoveryWriter: &stderrBuf,
	})

	var stdoutBuf bytes.Buffer

	err := WriteJSON(ctx, &stdoutBuf, fieldDiscoverySample{ID: "1", Name: "test", Size: 42})
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	// Field discovery output should go to stderr (the FieldDiscoveryWriter).
	output := stderrBuf.String()

	if !bytes.Contains(stderrBuf.Bytes(), []byte("Available fields:")) {
		t.Fatalf("expected field discovery output on FieldDiscoveryWriter, got %q", output)
	}

	if !bytes.Contains(stderrBuf.Bytes(), []byte("id")) {
		t.Fatalf("expected 'id' in discovered fields, got %q", output)
	}

	if !bytes.Contains(stderrBuf.Bytes(), []byte("name")) {
		t.Fatalf("expected 'name' in discovered fields, got %q", output)
	}

	// Stdout must be empty -- field discovery should not write the JSON payload.
	if stdoutBuf.Len() > 0 {
		t.Fatalf("expected no stdout output during field discovery, got %q", stdoutBuf.String())
	}
}

func TestWriteJSON_FieldDiscovery_NotTriggeredWhenSelectHasValues(t *testing.T) {
	// When Select has values, field discovery should NOT be triggered even if SelectExplicit is true.
	var stderrBuf bytes.Buffer
	ctx := WithJSONTransform(context.Background(), JSONTransform{
		SelectExplicit:       true,
		Select:               []string{"id"},
		FieldDiscoveryWriter: &stderrBuf,
	})

	var stdoutBuf bytes.Buffer

	err := WriteJSON(ctx, &stdoutBuf, fieldDiscoverySample{ID: "1", Name: "test", Size: 42})
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	// Should NOT have field discovery output; should have normal JSON select output.
	if stderrBuf.Len() > 0 {
		t.Fatalf("expected no field discovery output when Select has values, got %q", stderrBuf.String())
	}
}
