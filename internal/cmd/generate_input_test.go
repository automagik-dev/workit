package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// TestGenerateInput_ProducesValidJSON verifies the template marshals to valid JSON.
func TestGenerateInput_ProducesValidJSON(t *testing.T) {
	type TestCmd struct {
		Name    string `arg:"" required:"" help:"The name"`
		Verbose bool   `help:"Be verbose" short:"v"`
		Count   int    `help:"Number of items" default:"10"`
		Format  string `help:"Output format" enum:"json,text,csv" default:"json"`
	}

	type TestCLI struct {
		Test TestCmd `cmd:"" name:"test" help:"A test command"`
	}

	var cli TestCLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}

	kctx, err := parser.Parse([]string{"test", "myname"})
	if err != nil {
		t.Fatal(err)
	}

	template, err := generateInputTemplate(kctx)
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("template not valid JSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty template")
	}
}

// TestGenerateInput_RequiredFields verifies required fields are marked.
func TestGenerateInput_RequiredFields(t *testing.T) {
	type TestCmd struct {
		Name string `arg:"" required:"" help:"The name"`
	}

	type TestCLI struct {
		Test TestCmd `cmd:"" name:"test" help:"A test command"`
	}

	var cli TestCLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}

	kctx, err := parser.Parse([]string{"test", "myname"})
	if err != nil {
		t.Fatal(err)
	}

	template, err := generateInputTemplate(kctx)
	if err != nil {
		t.Fatal(err)
	}

	val, ok := template["name"]
	if !ok {
		t.Fatal("expected 'name' key in template")
	}
	str, ok := val.(string)
	if !ok {
		t.Fatalf("expected string value for required arg, got %T", val)
	}
	if !strings.Contains(str, "(required)") {
		t.Errorf("expected required marker in %q", str)
	}
}

// TestGenerateInput_ExcludesHiddenAndBuiltins verifies hidden flags and builtins are excluded.
func TestGenerateInput_ExcludesHiddenAndBuiltins(t *testing.T) {
	type TestCmd struct {
		Visible bool `help:"Visible flag"`
		Secret  bool `help:"Secret flag" hidden:""`
	}

	type TestCLI struct {
		Test TestCmd `cmd:"" name:"test" help:"A test command"`
	}

	var cli TestCLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}

	kctx, err := parser.Parse([]string{"test"})
	if err != nil {
		t.Fatal(err)
	}

	template, err := generateInputTemplate(kctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := template["help"]; ok {
		t.Error("template should not contain 'help' builtin")
	}
	if _, ok := template["secret"]; ok {
		t.Error("template should not contain hidden 'secret' flag")
	}
	if _, ok := template["visible"]; !ok {
		t.Error("template should contain 'visible' flag")
	}
}

// TestGenerateInput_IncludesEnumValues verifies enum fields show their values.
func TestGenerateInput_IncludesEnumValues(t *testing.T) {
	type TestCmd struct {
		Format string `help:"Output format" enum:"json,text,csv" default:"json"`
	}

	type TestCLI struct {
		Test TestCmd `cmd:"" name:"test" help:"A test command"`
	}

	var cli TestCLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}

	kctx, err := parser.Parse([]string{"test"})
	if err != nil {
		t.Fatal(err)
	}

	template, err := generateInputTemplate(kctx)
	if err != nil {
		t.Fatal(err)
	}

	val, ok := template["format"]
	if !ok {
		t.Fatal("expected 'format' key in template")
	}
	str, ok := val.(string)
	if !ok {
		t.Fatalf("expected string value for enum field, got %T", val)
	}
	if !strings.Contains(str, "enum") {
		t.Errorf("expected enum marker in %q", str)
	}
	for _, v := range []string{"json", "text", "csv"} {
		if !strings.Contains(str, v) {
			t.Errorf("expected enum value %q in %q", v, str)
		}
	}
}

// TestGenerateInput_TypedDefaults verifies default values are typed, not strings.
func TestGenerateInput_TypedDefaults(t *testing.T) {
	type TestCmd struct {
		Count   int    `help:"Number of items" default:"10"`
		Enabled bool   `help:"Enable feature" default:"true"`
		Label   string `help:"Label" default:"hello"`
		Limit   int64  `help:"Limit" default:"500"`
	}

	type TestCLI struct {
		Test TestCmd `cmd:"" name:"test" help:"A test command"`
	}

	var cli TestCLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}

	kctx, err := parser.Parse([]string{"test"})
	if err != nil {
		t.Fatal(err)
	}

	template, err := generateInputTemplate(kctx)
	if err != nil {
		t.Fatal(err)
	}

	// int default should be int, not string
	countVal, ok := template["count"]
	if !ok {
		t.Fatal("expected 'count' key in template")
	}
	// When marshaled to JSON and back, ints become float64 in Go's any.
	// But in the template map we expect the actual typed value.
	switch cv := countVal.(type) {
	case int:
		if cv != 10 {
			t.Errorf("expected count=10, got %d", cv)
		}
	case int64:
		if cv != 10 {
			t.Errorf("expected count=10, got %d", cv)
		}
	default:
		t.Fatalf("expected int for 'count' default, got %T (%v)", countVal, countVal)
	}

	// bool default should be bool, not string
	enabledVal, ok := template["enabled"]
	if !ok {
		t.Fatal("expected 'enabled' key in template")
	}
	boolVal, ok := enabledVal.(bool)
	if !ok {
		t.Fatalf("expected bool for 'enabled' default, got %T (%v)", enabledVal, enabledVal)
	}
	if !boolVal {
		t.Error("expected enabled=true")
	}

	// string default stays string
	labelVal, ok := template["label"]
	if !ok {
		t.Fatal("expected 'label' key in template")
	}
	strVal, ok := labelVal.(string)
	if !ok {
		t.Fatalf("expected string for 'label' default, got %T (%v)", labelVal, labelVal)
	}
	if strVal != "hello" {
		t.Errorf("expected label='hello', got %q", strVal)
	}

	// int64 default should be int64, not string
	limitVal, ok := template["limit"]
	if !ok {
		t.Fatal("expected 'limit' key in template")
	}
	switch lv := limitVal.(type) {
	case int64:
		if lv != 500 {
			t.Errorf("expected limit=500, got %d", lv)
		}
	case int:
		if lv != 500 {
			t.Errorf("expected limit=500, got %d", lv)
		}
	default:
		t.Fatalf("expected int64 for 'limit' default, got %T (%v)", limitVal, limitVal)
	}
}

// TestGenerateInput_NoCommand verifies error when no command is selected.
func TestGenerateInput_NoCommand(t *testing.T) {
	type TestCLI struct{}

	var cli TestCLI
	parser, err := kong.New(&cli, kong.Exit(func(int) {}))
	if err != nil {
		t.Fatal(err)
	}

	kctx, err := parser.Parse([]string{})
	if err != nil {
		// Expected: no command selected
		return
	}

	_, genErr := generateInputTemplate(kctx)
	if genErr == nil {
		t.Fatal("expected error when no command selected")
	}
	if !strings.Contains(genErr.Error(), "no command selected") {
		t.Errorf("unexpected error: %v", genErr)
	}
}

// TestGenerateInput_BoolDefaultsFalse verifies bool flags without defaults get false.
func TestGenerateInput_BoolDefaultsFalse(t *testing.T) {
	type TestCmd struct {
		Verbose bool `help:"Be verbose" short:"v"`
	}

	type TestCLI struct {
		Test TestCmd `cmd:"" name:"test" help:"A test command"`
	}

	var cli TestCLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}

	kctx, err := parser.Parse([]string{"test"})
	if err != nil {
		t.Fatal(err)
	}

	template, err := generateInputTemplate(kctx)
	if err != nil {
		t.Fatal(err)
	}

	val, ok := template["verbose"]
	if !ok {
		t.Fatal("expected 'verbose' key in template")
	}
	boolVal, ok := val.(bool)
	if !ok {
		t.Fatalf("expected bool value for bool flag, got %T (%v)", val, val)
	}
	if boolVal != false {
		t.Error("expected false for bool flag without default")
	}
}

// TestGenerateInput_ViaExecute tests the full Execute path with --generate-input.
func TestGenerateInput_ViaExecute(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--generate-input", "drive", "ls"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var template map[string]any
	if err := json.Unmarshal([]byte(out), &template); err != nil {
		t.Fatalf("output not valid JSON: %v (out=%q)", err, out)
	}

	if len(template) == 0 {
		t.Fatal("empty template from Execute")
	}
}
