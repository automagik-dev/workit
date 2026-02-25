package docx_test

import (
	"archive/zip"
	"os"
	"strings"
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

func TestReadAsMarkdown(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	md, err := docx.ReadAsMarkdown(session)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	// Verify title is rendered as # heading.
	if !strings.Contains(md, "# Sample Document") {
		t.Errorf("missing title heading; got:\n%s", md)
	}

	// Verify Heading1 is rendered as ##.
	if !strings.Contains(md, "## Introduction") {
		t.Errorf("missing Heading1; got:\n%s", md)
	}

	// Verify Heading2 is rendered as ###.
	if !strings.Contains(md, "### Details") {
		t.Errorf("missing Heading2; got:\n%s", md)
	}

	// Verify body paragraphs.
	if !strings.Contains(md, "first paragraph") {
		t.Error("missing first paragraph text")
	}

	if !strings.Contains(md, "Second paragraph") {
		t.Error("missing second paragraph text")
	}

	if !strings.Contains(md, "Third paragraph") {
		t.Error("missing third paragraph text")
	}

	// Verify table is rendered as markdown.
	if !strings.Contains(md, "| Name | Role | Status |") {
		t.Errorf("missing table header; got:\n%s", md)
	}

	if !strings.Contains(md, "| --- | --- | --- |") {
		t.Errorf("missing table separator; got:\n%s", md)
	}

	if !strings.Contains(md, "| Alice | Engineer | Active |") {
		t.Error("missing table data row")
	}

	// Verify final paragraph after table.
	if !strings.Contains(md, "Final paragraph after the table.") {
		t.Error("missing final paragraph")
	}
}

func TestReadStructure(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	ds, err := docx.ReadStructure(session)
	if err != nil {
		t.Fatalf("ReadStructure: %v", err)
	}

	// We expect 7 paragraphs: title, heading1, body1, heading2, body2, body3, final.
	if len(ds.Paragraphs) != 7 {
		t.Errorf("paragraph count = %d, want 7", len(ds.Paragraphs))
	}

	// Verify first paragraph is the title.
	if len(ds.Paragraphs) > 0 {
		p := ds.Paragraphs[0]
		if p.Style != "Title" {
			t.Errorf("first paragraph style = %q, want %q", p.Style, "Title")
		}

		if p.Text != "Sample Document" {
			t.Errorf("first paragraph text = %q, want %q", p.Text, "Sample Document")
		}

		if p.Index != 0 {
			t.Errorf("first paragraph index = %d, want 0", p.Index)
		}
	}

	// Verify heading1.
	if len(ds.Paragraphs) > 1 {
		p := ds.Paragraphs[1]
		if p.Style != "Heading1" {
			t.Errorf("second paragraph style = %q, want %q", p.Style, "Heading1")
		}
	}

	// Verify we found the table.
	if len(ds.Tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(ds.Tables))
	}

	tbl := ds.Tables[0]
	if tbl.Rows != 3 {
		t.Errorf("table rows = %d, want 3", tbl.Rows)
	}

	if tbl.Cols != 3 {
		t.Errorf("table cols = %d, want 3", tbl.Cols)
	}
}

func TestReadAsMarkdownEmptyDoc(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/empty.docx"
	buildMinimalDOCX(t, path, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body/>
</w:document>`)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	md, err := docx.ReadAsMarkdown(session)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if strings.TrimSpace(md) != "" {
		t.Errorf("expected empty markdown, got %q", md)
	}
}

// buildMinimalDOCX creates a DOCX with custom document.xml content.
func buildMinimalDOCX(t *testing.T, path string, documentXMLContent string) {
	t.Helper()

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
		"word/document.xml":   documentXMLContent,
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
