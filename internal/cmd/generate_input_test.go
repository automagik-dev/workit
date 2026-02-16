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

// TestGenerateInput_IncludesDefaults verifies default values appear in the template.
func TestGenerateInput_IncludesDefaults(t *testing.T) {
	type TestCmd struct {
		Count int `help:"Number of items" default:"10"`
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

	val, ok := template["count"]
	if !ok {
		t.Fatal("expected 'count' key in template")
	}
	str, ok := val.(string)
	if !ok {
		t.Fatalf("expected string value for default field, got %T", val)
	}
	if str != "10" {
		t.Errorf("expected default value '10', got %q", str)
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
