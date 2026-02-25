package docx_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

func TestValidateGoodFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "good.docx")
	buildSampleDOCX(t, tmp)

	result, err := docx.Validate(tmp)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestValidateBadZip(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.docx")
	if err := os.WriteFile(tmp, []byte("this is not a zip file"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := docx.Validate(tmp)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if result.Valid {
		t.Fatal("expected invalid for non-ZIP file")
	}

	if len(result.Errors) == 0 {
		t.Fatal("expected errors for non-ZIP file")
	}

	found := false

	for _, e := range result.Errors {
		if contains(e, "not a valid ZIP") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected 'not a valid ZIP' error, got: %v", result.Errors)
	}
}

func TestValidateMissingParts(t *testing.T) {
	// Create a ZIP with only [Content_Types].xml but missing other required parts.
	tmp := filepath.Join(t.TempDir(), "missing.docx")

	f, err := os.Create(tmp)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	zw := zip.NewWriter(f)

	w, err := zw.Create("[Content_Types].xml")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"></Types>`))

	zw.Close()
	f.Close()

	result, err := docx.Validate(tmp)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if result.Valid {
		t.Fatal("expected invalid for DOCX missing required parts")
	}

	// Should report missing word/document.xml and _rels/.rels.
	errStr := joinStrings(result.Errors)
	if !contains(errStr, "word/document.xml") {
		t.Errorf("expected error about missing word/document.xml, got: %v", result.Errors)
	}

	if !contains(errStr, "_rels/.rels") {
		t.Errorf("expected error about missing _rels/.rels, got: %v", result.Errors)
	}
}

func TestValidateBadXML(t *testing.T) {
	// Create a DOCX where document.xml is not valid XML.
	tmp := filepath.Join(t.TempDir(), "badxml.docx")

	f, err := os.Create(tmp)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	zw := zip.NewWriter(f)

	entries := map[string]string{
		"[Content_Types].xml": contentTypesXML,
		"_rels/.rels":         relsXML,
		"word/document.xml":   "<broken><xml",
	}

	for name, content := range entries {
		ew, createErr := zw.Create(name)
		if createErr != nil {
			t.Fatalf("zip create %s: %v", name, createErr)
		}
		_, _ = ew.Write([]byte(content))
	}

	zw.Close()
	f.Close()

	result, err := docx.Validate(tmp)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if result.Valid {
		t.Fatal("expected invalid for bad XML")
	}

	found := false

	for _, e := range result.Errors {
		if contains(e, "not valid XML") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected 'not valid XML' error, got: %v", result.Errors)
	}
}

func TestValidateDiff(t *testing.T) {
	// Create a valid original.
	origPath := filepath.Join(t.TempDir(), "orig.docx")
	buildSampleDOCX(t, origPath)

	// Create a valid edited copy (same file = no diff).
	editedPath := filepath.Join(t.TempDir(), "edited.docx")
	buildSampleDOCX(t, editedPath)

	diff, err := docx.ValidateDiff(origPath, editedPath)
	if err != nil {
		t.Fatalf("ValidateDiff: %v", err)
	}

	if !diff.Valid {
		t.Fatalf("expected no new errors, got: %v", diff.Errors)
	}

	if len(diff.Errors) != 0 {
		t.Errorf("expected empty errors, got: %v", diff.Errors)
	}
}

func TestValidateDiffNewErrors(t *testing.T) {
	// Create a valid original.
	origPath := filepath.Join(t.TempDir(), "orig.docx")
	buildSampleDOCX(t, origPath)

	// Create an invalid edited file (bad ZIP).
	editedPath := filepath.Join(t.TempDir(), "edited.docx")
	if err := os.WriteFile(editedPath, []byte("not a zip"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	diff, err := docx.ValidateDiff(origPath, editedPath)
	if err != nil {
		t.Fatalf("ValidateDiff: %v", err)
	}

	if diff.Valid {
		t.Fatal("expected new errors in diff")
	}

	if len(diff.Errors) == 0 {
		t.Fatal("expected at least one new error")
	}
}

func TestValidateNoBody(t *testing.T) {
	// Create a DOCX where document.xml is valid XML but has no w:body.
	tmp := filepath.Join(t.TempDir(), "nobody.docx")
	buildDOCXFromXML(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
</w:document>`)

	result, err := docx.Validate(tmp)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if result.Valid {
		t.Fatal("expected invalid for document without w:body")
	}

	found := false

	for _, e := range result.Errors {
		if contains(e, "w:body") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected error about missing w:body, got: %v", result.Errors)
	}
}

// helper: check if s contains substr (case-insensitive-ish via plain contains).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}

	return false
}

func joinStrings(ss []string) string {
	result := ""
	for _, s := range ss {
		result += s + "; "
	}

	return result
}
