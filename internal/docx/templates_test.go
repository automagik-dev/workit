package docx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

func TestListTemplates_Empty(t *testing.T) {
	dir := t.TempDir()
	templatesDir := filepath.Join(dir, "templates")
	// Directory doesn't exist yet — should return nil, nil.
	names, err := docx.ListTemplates(templatesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(names) != 0 {
		t.Fatalf("expected 0 templates, got %d", len(names))
	}
}

func TestAddAndListTemplates(t *testing.T) {
	dir := t.TempDir()
	templatesDir := filepath.Join(dir, "templates")

	// Create a fake DOCX source file.
	srcPath := filepath.Join(dir, "source.docx")
	if err := os.WriteFile(srcPath, []byte("fake docx content"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Add a template.
	if err := docx.AddTemplate(templatesDir, "corporate", srcPath); err != nil {
		t.Fatalf("add template: %v", err)
	}

	// Add another template.
	if err := docx.AddTemplate(templatesDir, "invoice", srcPath); err != nil {
		t.Fatalf("add template: %v", err)
	}

	// List templates.
	names, err := docx.ListTemplates(templatesDir)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 templates, got %d: %v", len(names), names)
	}

	// Should be sorted.
	if names[0] != "corporate" {
		t.Errorf("expected first template 'corporate', got %q", names[0])
	}

	if names[1] != "invoice" {
		t.Errorf("expected second template 'invoice', got %q", names[1])
	}

	// Verify the file was actually copied.
	destPath := filepath.Join(templatesDir, "corporate.docx")

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read copied template: %v", err)
	}

	if string(data) != "fake docx content" {
		t.Errorf("template content mismatch: %q", data)
	}
}

func TestGetTemplatePath_Name(t *testing.T) {
	templatesDir := "/home/user/.config/workit/templates"

	got := docx.GetTemplatePath(templatesDir, "corporate")

	want := filepath.Join(templatesDir, "corporate.docx")
	if got != want {
		t.Errorf("GetTemplatePath('corporate') = %q, want %q", got, want)
	}
}

func TestGetTemplatePath_FilePath(t *testing.T) {
	templatesDir := "/home/user/.config/workit/templates"

	// Paths with slashes should be returned as-is.
	got := docx.GetTemplatePath(templatesDir, "/tmp/my-template.docx")

	want := "/tmp/my-template.docx"
	if got != want {
		t.Errorf("GetTemplatePath('/tmp/my-template.docx') = %q, want %q", got, want)
	}
}

func TestGetTemplatePath_DocxSuffix(t *testing.T) {
	templatesDir := "/home/user/.config/workit/templates"

	// Names ending in .docx should be returned as-is.
	got := docx.GetTemplatePath(templatesDir, "template.docx")

	want := "template.docx"
	if got != want {
		t.Errorf("GetTemplatePath('template.docx') = %q, want %q", got, want)
	}
}

func TestTemplatesDir(t *testing.T) {
	got := docx.TemplatesDir("/home/user/.config/workit")

	want := filepath.Join("/home/user/.config/workit", "templates")
	if got != want {
		t.Errorf("TemplatesDir = %q, want %q", got, want)
	}
}

func TestEnsureTemplatesDir(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "workit")

	got, err := docx.EnsureTemplatesDir(configDir)
	if err != nil {
		t.Fatalf("EnsureTemplatesDir: %v", err)
	}

	want := filepath.Join(configDir, "templates")
	if got != want {
		t.Errorf("EnsureTemplatesDir = %q, want %q", got, want)
	}

	// Directory should exist.
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat templates dir: %v", err)
	}

	if !info.IsDir() {
		t.Error("templates path is not a directory")
	}
}

func TestInspectTemplateByName(t *testing.T) {
	dir := t.TempDir()
	templatesDir := filepath.Join(dir, "templates")

	// Create a DOCX with {{RECIPIENT}} and {{DATE}} placeholders.
	srcPath := filepath.Join(dir, "src_tmpl.docx")
	buildLetterTemplateDOCX(t, srcPath)

	if err := docx.AddTemplate(templatesDir, "letter", srcPath); err != nil {
		t.Fatalf("add template: %v", err)
	}

	names, err := docx.InspectTemplateByName(templatesDir, "letter")
	if err != nil {
		t.Fatalf("inspect template: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 placeholders, got %d: %v", len(names), names)
	}

	if names[0] != "RECIPIENT" {
		t.Errorf("expected first placeholder 'RECIPIENT', got %q", names[0])
	}

	if names[1] != "DATE" {
		t.Errorf("expected second placeholder 'DATE', got %q", names[1])
	}
}
