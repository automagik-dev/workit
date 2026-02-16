package outfmt

import (
	"bytes"
	"strings"
	"testing"
)

// --- DiscoverFields tests ---

type sampleFlat struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func TestFieldDiscovery_FlatStruct(t *testing.T) {
	got := DiscoverFields(sampleFlat{})
	want := []string{"id", "name", "size"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}

	for i, f := range want {
		if got[i] != f {
			t.Errorf("field[%d]: expected %q, got %q", i, f, got[i])
		}
	}
}

type sampleNested struct {
	ID    string      `json:"id"`
	Owner sampleOwner `json:"owner"`
	Tags  []string    `json:"tags"`
}

type sampleOwner struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func TestFieldDiscovery_NestedStruct(t *testing.T) {
	got := DiscoverFields(sampleNested{})
	want := []string{"id", "owner", "tags"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}

	for i, f := range want {
		if got[i] != f {
			t.Errorf("field[%d]: expected %q, got %q", i, f, got[i])
		}
	}
}

type sampleWithSkip struct {
	ID       string `json:"id"`
	Internal string `json:"-"`
	Name     string `json:"name"`
}

func TestFieldDiscovery_SkipDashTag(t *testing.T) {
	got := DiscoverFields(sampleWithSkip{})
	want := []string{"id", "name"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}

	for i, f := range want {
		if got[i] != f {
			t.Errorf("field[%d]: expected %q, got %q", i, f, got[i])
		}
	}
}

type sampleWithOmitempty struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Count int    `json:"count"`
}

func TestFieldDiscovery_OmitemptyTag(t *testing.T) {
	got := DiscoverFields(sampleWithOmitempty{})
	want := []string{"id", "name", "count"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}

	for i, f := range want {
		if got[i] != f {
			t.Errorf("field[%d]: expected %q, got %q", i, f, got[i])
		}
	}
}

type sampleNoTag struct {
	ID      string `json:"id"`
	NoTag   string
	Labeled string `json:"labeled"`
}

func TestFieldDiscovery_FieldsWithoutJSONTag(t *testing.T) {
	// Fields without json tags use Go field name (matching encoding/json default).
	got := DiscoverFields(sampleNoTag{})
	want := []string{"id", "NoTag", "labeled"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}

	for i, f := range want {
		if got[i] != f {
			t.Errorf("field[%d]: expected %q, got %q", i, f, got[i])
		}
	}
}

func TestFieldDiscovery_Pointer(t *testing.T) {
	got := DiscoverFields(&sampleFlat{})
	want := []string{"id", "name", "size"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}
}

func TestFieldDiscovery_NonStruct(t *testing.T) {
	got := DiscoverFields("not a struct")
	if len(got) != 0 {
		t.Fatalf("expected 0 fields for non-struct, got %d: %v", len(got), got)
	}

	got = DiscoverFields(42)
	if len(got) != 0 {
		t.Fatalf("expected 0 fields for int, got %d: %v", len(got), got)
	}

	got = DiscoverFields(nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 fields for nil, got %d: %v", len(got), got)
	}
}

func TestFieldDiscovery_MapStringAny(t *testing.T) {
	m := map[string]any{
		"id":   "abc",
		"name": "test",
		"size": 42,
	}
	got := DiscoverFields(m)
	want := []string{"id", "name", "size"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}

	for i, f := range want {
		if got[i] != f {
			t.Errorf("field[%d]: expected %q, got %q", i, f, got[i])
		}
	}
}

func TestFieldDiscovery_EmptyMap(t *testing.T) {
	m := map[string]any{}
	got := DiscoverFields(m)
	if len(got) != 0 {
		t.Fatalf("expected 0 fields for empty map, got %d: %v", len(got), got)
	}
}

type sampleSliceResult struct {
	Files         []sampleFlat `json:"files"`
	NextPageToken string       `json:"nextPageToken"`
}

func TestFieldDiscovery_EnvelopeStruct(t *testing.T) {
	got := DiscoverFields(sampleSliceResult{})
	want := []string{"files", "nextPageToken"}

	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(got), got)
	}

	for i, f := range want {
		if got[i] != f {
			t.Errorf("field[%d]: expected %q, got %q", i, f, got[i])
		}
	}
}

// --- IsFieldDiscovery tests ---

func TestIsFieldDiscovery_EmptyStringAndFlagSet(t *testing.T) {
	if !IsFieldDiscovery("", true) {
		t.Fatal("expected true for empty string with flag set")
	}
}

func TestIsFieldDiscovery_NonEmptyString(t *testing.T) {
	if IsFieldDiscovery("name,id", true) {
		t.Fatal("expected false for non-empty select value")
	}
}

func TestIsFieldDiscovery_FlagNotSet(t *testing.T) {
	if IsFieldDiscovery("", false) {
		t.Fatal("expected false when flag is not set")
	}
}

// --- SelectFlagExplicitlySet tests ---

func TestSelectFlagExplicitlySet_NotPresent(t *testing.T) {
	args := []string{"drive", "ls", "--json"}
	if SelectFlagExplicitlySet(args) {
		t.Fatal("expected false when --select not in args")
	}
}

func TestSelectFlagExplicitlySet_WithValue(t *testing.T) {
	args := []string{"drive", "ls", "--json", "--select", "name,id"}
	if !SelectFlagExplicitlySet(args) {
		t.Fatal("expected true when --select is present with value")
	}
}

func TestSelectFlagExplicitlySet_EmptyString(t *testing.T) {
	args := []string{"drive", "ls", "--json", "--select", ""}
	if !SelectFlagExplicitlySet(args) {
		t.Fatal("expected true when --select is present with empty string")
	}
}

func TestSelectFlagExplicitlySet_EqualsEmpty(t *testing.T) {
	args := []string{"drive", "ls", "--json", "--select="}
	if !SelectFlagExplicitlySet(args) {
		t.Fatal("expected true when --select= is present")
	}
}

func TestSelectFlagExplicitlySet_EqualsValue(t *testing.T) {
	args := []string{"drive", "ls", "--json", "--select=name"}
	if !SelectFlagExplicitlySet(args) {
		t.Fatal("expected true when --select=name is present")
	}
}

func TestSelectFlagExplicitlySet_FieldsAlias(t *testing.T) {
	// After rewriteDesirePathArgs, --fields becomes --select,
	// but we also check --pick and --project aliases.
	args := []string{"drive", "ls", "--json", "--pick", ""}
	if !SelectFlagExplicitlySet(args) {
		t.Fatal("expected true when --pick alias is present")
	}
}

func TestSelectFlagExplicitlySet_ProjectAlias(t *testing.T) {
	args := []string{"drive", "ls", "--json", "--project", ""}
	if !SelectFlagExplicitlySet(args) {
		t.Fatal("expected true when --project alias is present")
	}
}

func TestSelectFlagExplicitlySet_AfterDoubleDash(t *testing.T) {
	// --select after -- should not be considered as the flag.
	args := []string{"drive", "ls", "--", "--select", ""}
	if SelectFlagExplicitlySet(args) {
		t.Fatal("expected false when --select appears after --")
	}
}

// --- PrintFieldDiscovery tests ---

func TestPrintFieldDiscovery_Output(t *testing.T) {
	var buf bytes.Buffer
	fields := []string{"id", "name", "size", "mimeType"}
	PrintFieldDiscovery(&buf, fields, "gog drive ls")

	out := buf.String()

	if !strings.Contains(out, "Available fields:") {
		t.Error("expected 'Available fields:' header")
	}

	for _, f := range fields {
		if !strings.Contains(out, "  "+f) {
			t.Errorf("expected field %q in output", f)
		}
	}

	// Hint should use first 3 fields.
	if !strings.Contains(out, `Usage: gog drive ls --json --select "id,name,size"`) {
		t.Errorf("expected usage hint with first 3 fields, got:\n%s", out)
	}
}

func TestPrintFieldDiscovery_FewFields(t *testing.T) {
	var buf bytes.Buffer
	fields := []string{"id", "name"}
	PrintFieldDiscovery(&buf, fields, "gog drive ls")

	out := buf.String()
	if !strings.Contains(out, `Usage: gog drive ls --json --select "id,name"`) {
		t.Errorf("expected usage hint with all fields, got:\n%s", out)
	}
}

func TestPrintFieldDiscovery_EmptyCommand(t *testing.T) {
	var buf bytes.Buffer
	PrintFieldDiscovery(&buf, []string{}, "")

	out := buf.String()
	if !strings.Contains(out, `Usage: gog <command> --json --select "field1,field2"`) {
		t.Errorf("expected generic usage hint, got:\n%s", out)
	}
}
