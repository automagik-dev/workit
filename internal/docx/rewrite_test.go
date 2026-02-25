package docx_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

func TestRewrite(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "rewrite.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	markdown := `# New Document Title

## First Section

This is the first paragraph of the new content.

This is the second paragraph.

## Second Section

Final paragraph wrapping up.`

	err = docx.Rewrite(session, markdown)
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
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

	// Check the new content is present.
	if !strings.Contains(md, "New Document Title") {
		t.Errorf("expected new title:\n%s", md)
	}

	if !strings.Contains(md, "First Section") {
		t.Errorf("expected 'First Section':\n%s", md)
	}

	if !strings.Contains(md, "first paragraph of the new content") {
		t.Errorf("expected first paragraph:\n%s", md)
	}

	if !strings.Contains(md, "Final paragraph wrapping up") {
		t.Errorf("expected final paragraph:\n%s", md)
	}

	// Check old content is gone.
	if strings.Contains(md, "Sample Document") {
		t.Errorf("old title should be gone:\n%s", md)
	}

	if strings.Contains(md, "Alice") {
		t.Errorf("old table data should be gone:\n%s", md)
	}

	// Verify structure: Title heading should render as "# New Document Title".
	if !strings.Contains(md, "# New Document Title") {
		t.Errorf("expected Title style to render as # heading:\n%s", md)
	}
}

func TestRewritePreservesStyles(t *testing.T) {
	// Build a DOCX with a sectPr (section properties for page layout).
	tmp := filepath.Join(t.TempDir(), "sectpr.docx")
	buildDOCXFromXML(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Title"/></w:pPr>
      <w:r><w:t>Old Title</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Old content</w:t></w:r>
    </w:p>
    <w:sectPr>
      <w:pgSz w:w="12240" w:h="15840"/>
      <w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/>
    </w:sectPr>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.Rewrite(session, "# New Title\n\nNew content here.")
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and verify section properties are preserved.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	// Check that new content is present.
	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "New Title") {
		t.Errorf("new content not found:\n%s", md)
	}

	if strings.Contains(md, "Old Title") {
		t.Errorf("old content still present:\n%s", md)
	}

	// Verify section properties are preserved in the raw XML.
	raw, err := session2.RawPart("word/document.xml")
	if err != nil {
		t.Fatalf("RawPart: %v", err)
	}

	xmlStr := string(raw)
	if !strings.Contains(xmlStr, "pgSz") {
		t.Errorf("section properties (pgSz) lost after rewrite:\n%s", xmlStr)
	}

	if !strings.Contains(xmlStr, "pgMar") {
		t.Errorf("section properties (pgMar) lost after rewrite:\n%s", xmlStr)
	}
}

func TestRewriteEmptyContent(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "empty-rewrite.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.Rewrite(session, "")
	if err != nil {
		t.Fatalf("Rewrite with empty content: %v", err)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and verify no content paragraphs.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	ds, err := docx.ReadStructure(session2)
	if err != nil {
		t.Fatalf("ReadStructure: %v", err)
	}

	if len(ds.Paragraphs) != 0 {
		t.Errorf("expected no paragraphs after empty rewrite, got %d", len(ds.Paragraphs))
	}
}
