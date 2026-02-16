package outfmt

import (
	"encoding/json"
	"testing"
)

func TestApplyJQ_SimpleFieldExtraction(t *testing.T) {
	input := `[{"name":"alice","age":25},{"name":"bob","age":35}]`

	got, err := ApplyJQ([]byte(input), ".[].name")
	if err != nil {
		t.Fatalf("ApplyJQ: %v", err)
	}

	want := "\"alice\"\n\"bob\""
	if string(got) != want {
		t.Fatalf("got %q, want %q", string(got), want)
	}
}

func TestApplyJQ_ComplexTransform(t *testing.T) {
	input := `[{"name":"alice","age":25},{"name":"bob","age":35}]`

	got, err := ApplyJQ([]byte(input), `[.[] | {a: .name}]`)
	if err != nil {
		t.Fatalf("ApplyJQ: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("unmarshal result: %v (raw=%q)", err, string(got))
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	if result[0]["a"] != "alice" || result[1]["a"] != "bob" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestApplyJQ_Length(t *testing.T) {
	input := `[1,2,3,4,5]`

	got, err := ApplyJQ([]byte(input), "length")
	if err != nil {
		t.Fatalf("ApplyJQ: %v", err)
	}

	if string(got) != "5" {
		t.Fatalf("got %q, want %q", string(got), "5")
	}
}

func TestApplyJQ_Identity(t *testing.T) {
	input := `{"key":"value","num":42}`

	got, err := ApplyJQ([]byte(input), ".")
	if err != nil {
		t.Fatalf("ApplyJQ: %v", err)
	}

	// The identity filter should produce equivalent JSON.
	var orig, result any
	if err := json.Unmarshal([]byte(input), &orig); err != nil {
		t.Fatalf("unmarshal original: %v", err)
	}

	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("unmarshal result: %v (raw=%q)", err, string(got))
	}

	origB, _ := json.Marshal(orig)

	resultB, _ := json.Marshal(result)
	if string(origB) != string(resultB) {
		t.Fatalf("identity mismatch: orig=%s result=%s", origB, resultB)
	}
}

func TestApplyJQ_ParseError(t *testing.T) {
	input := `{"a":1}`

	_, err := ApplyJQ([]byte(input), ".[invalid")
	if err == nil {
		t.Fatalf("expected error for invalid jq expression")
	}
}

func TestApplyJQ_EmptyInput(t *testing.T) {
	input := `null`

	got, err := ApplyJQ([]byte(input), ".")
	if err != nil {
		t.Fatalf("ApplyJQ: %v", err)
	}

	if string(got) != "null" {
		t.Fatalf("got %q, want %q", string(got), "null")
	}
}

func TestApplyJQ_Select(t *testing.T) {
	input := `[{"name":"alice","age":25},{"name":"bob","age":35},{"name":"carol","age":40}]`

	got, err := ApplyJQ([]byte(input), `[.[] | select(.age > 30)]`)
	if err != nil {
		t.Fatalf("ApplyJQ: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("unmarshal result: %v (raw=%q)", err, string(got))
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	if result[0]["name"] != "bob" || result[1]["name"] != "carol" {
		t.Fatalf("unexpected result: %v", result)
	}
}
