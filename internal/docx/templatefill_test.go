package docx_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

// templateDocumentXML is a document with {{PLACEHOLDER}} patterns.
const templateDocumentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Title"/></w:pPr>
      <w:r><w:rPr><w:b/><w:sz w:val="48"/></w:rPr><w:t>{{TITLE}}</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Written by {{AUTHOR}}</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>{{BODY}}</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`

// templateFragmentedXML has a placeholder split across multiple runs.
const templateFragmentedXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:rPr><w:b/></w:rPr><w:t>{{</w:t></w:r>
      <w:r><w:t>TITLE</w:t></w:r>
      <w:r><w:t>}}</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Hello </w:t></w:r>
      <w:r><w:t>{{AUTH</w:t></w:r>
      <w:r><w:t>OR}}</w:t></w:r>
      <w:r><w:t> world</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`

// buildTemplateDOCX creates a DOCX with template placeholders.
func buildTemplateDOCX(t *testing.T, path string) {
	t.Helper()
	buildDOCXFromXML(t, path, templateDocumentXML)
}

// buildFragmentedTemplateDOCX creates a DOCX with placeholders split across runs.
func buildFragmentedTemplateDOCX(t *testing.T, path string) {
	t.Helper()
	buildDOCXFromXML(t, path, templateFragmentedXML)
}

// buildDOCXFromXML creates a minimal DOCX with custom document XML.
func buildDOCXFromXML(t *testing.T, path string, documentContent string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	entries := map[string]string{
		"[Content_Types].xml": contentTypesXML,
		"_rels/.rels":         relsXML,
		"word/document.xml":   documentContent,
	}

	for name, content := range entries {
		ew, createErr := zw.Create(name)
		if createErr != nil {
			t.Fatalf("create %s: %v", name, createErr)
		}

		if _, writeErr := ew.Write([]byte(content)); writeErr != nil {
			t.Fatalf("write %s: %v", name, writeErr)
		}
	}
}

func TestFillTemplate(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "template.docx")
	buildTemplateDOCX(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	values := map[string]string{
		"TITLE":  "Annual Report",
		"AUTHOR": "Jane Doe",
		"BODY":   "This is the main content of the report.",
	}

	n, err := docx.FillTemplate(session, values)
	if err != nil {
		t.Fatalf("FillTemplate: %v", err)
	}

	if n != 3 {
		t.Errorf("replacements = %d, want 3", n)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and verify.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "Annual Report") {
		t.Errorf("expected 'Annual Report' in output:\n%s", md)
	}

	if !strings.Contains(md, "Jane Doe") {
		t.Errorf("expected 'Jane Doe' in output:\n%s", md)
	}

	if !strings.Contains(md, "main content of the report") {
		t.Errorf("expected body content in output:\n%s", md)
	}

	// Placeholders should be gone.
	if strings.Contains(md, "{{") {
		t.Errorf("placeholder markers still present:\n%s", md)
	}
}

func TestFillTemplateFragmented(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "fragmented.docx")
	buildFragmentedTemplateDOCX(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	values := map[string]string{
		"TITLE":  "Quarterly Review",
		"AUTHOR": "Alice Smith",
	}

	n, err := docx.FillTemplate(session, values)
	if err != nil {
		t.Fatalf("FillTemplate: %v", err)
	}

	if n != 2 {
		t.Errorf("replacements = %d, want 2", n)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and verify.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "Quarterly Review") {
		t.Errorf("expected 'Quarterly Review' in output:\n%s", md)
	}

	if !strings.Contains(md, "Alice Smith") {
		t.Errorf("expected 'Alice Smith' in output:\n%s", md)
	}

	// Verify surrounding text is preserved.
	if !strings.Contains(md, "Hello") {
		t.Errorf("expected 'Hello' surrounding text:\n%s", md)
	}

	if !strings.Contains(md, "world") {
		t.Errorf("expected 'world' surrounding text:\n%s", md)
	}

	// No leftover placeholders.
	if strings.Contains(md, "{{") {
		t.Errorf("placeholder markers still present:\n%s", md)
	}
}

func TestInspectTemplate(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "inspect.docx")
	buildTemplateDOCX(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	names, err := docx.InspectTemplate(session)
	if err != nil {
		t.Fatalf("InspectTemplate: %v", err)
	}

	if len(names) != 3 {
		t.Fatalf("placeholder count = %d, want 3; got %v", len(names), names)
	}

	// Convert to set for checking.
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	for _, expected := range []string{"TITLE", "AUTHOR", "BODY"} {
		if !nameSet[expected] {
			t.Errorf("missing placeholder %q in %v", expected, names)
		}
	}
}

func TestInspectTemplateFragmented(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "inspect-frag.docx")
	buildFragmentedTemplateDOCX(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	names, err := docx.InspectTemplate(session)
	if err != nil {
		t.Fatalf("InspectTemplate: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("placeholder count = %d, want 2; got %v", len(names), names)
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	if !nameSet["TITLE"] {
		t.Errorf("missing placeholder TITLE in %v", names)
	}

	if !nameSet["AUTHOR"] {
		t.Errorf("missing placeholder AUTHOR in %v", names)
	}
}

func TestFillTemplatePreservesFormatting(t *testing.T) {
	// The template has {{TITLE}} with bold + size 48 formatting.
	// After filling, the replacement text should be in the same run
	// (preserving the w:rPr with bold and size).
	tmp := filepath.Join(t.TempDir(), "format.docx")
	buildTemplateDOCX(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	values := map[string]string{
		"TITLE": "Formatted Title",
	}

	_, err = docx.FillTemplate(session, values)
	if err != nil {
		t.Fatalf("FillTemplate: %v", err)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and check that the run still has its formatting properties.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	// Read raw XML to verify formatting is preserved.
	raw, err := session2.RawPart("word/document.xml")
	if err != nil {
		t.Fatalf("RawPart: %v", err)
	}
	xmlStr := string(raw)

	// The bold and size attributes should still be present near the replacement text.
	if !strings.Contains(xmlStr, "Formatted Title") {
		t.Errorf("replacement text not found in XML")
	}

	if !strings.Contains(xmlStr, "<w:b/>") && !strings.Contains(xmlStr, "<w:b") {
		t.Errorf("bold formatting lost in XML:\n%s", xmlStr)
	}

	if !strings.Contains(xmlStr, "w:val=\"48\"") {
		t.Errorf("font size formatting lost in XML:\n%s", xmlStr)
	}
}

func TestFillTemplatePartialValues(t *testing.T) {
	// Only provide values for some placeholders; others should remain.
	tmp := filepath.Join(t.TempDir(), "partial.docx")
	buildTemplateDOCX(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	values := map[string]string{
		"TITLE": "Partial Fill",
	}

	n, err := docx.FillTemplate(session, values)
	if err != nil {
		t.Fatalf("FillTemplate: %v", err)
	}

	if n != 1 {
		t.Errorf("replacements = %d, want 1", n)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "Partial Fill") {
		t.Errorf("filled placeholder not found:\n%s", md)
	}
	// Unfilled placeholders should remain.
	if !strings.Contains(md, "{{AUTHOR}}") {
		t.Errorf("unfilled placeholder {{AUTHOR}} should remain:\n%s", md)
	}

	if !strings.Contains(md, "{{BODY}}") {
		t.Errorf("unfilled placeholder {{BODY}} should remain:\n%s", md)
	}
}

func TestFillTemplateInTable(t *testing.T) {
	// Test placeholders inside a table cell.
	tmp := filepath.Join(t.TempDir(), "table-template.docx")
	buildDOCXFromXML(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:tbl>
      <w:tblPr><w:tblStyle w:val="TableGrid"/></w:tblPr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Name</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Value</w:t></w:r></w:p></w:tc>
      </w:tr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Company</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>{{COMPANY}}</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.FillTemplate(session, map[string]string{"COMPANY": "Acme Corp"})
	if err != nil {
		t.Fatalf("FillTemplate: %v", err)
	}

	if n != 1 {
		t.Errorf("replacements = %d, want 1", n)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "Acme Corp") {
		t.Errorf("expected 'Acme Corp' in table:\n%s", md)
	}
}
