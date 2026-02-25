package docx_test

import (
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

func TestReadMetadata(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	meta, err := docx.ReadMetadata(session)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}

	if meta.Title != "Sample Document" {
		t.Errorf("Title = %q, want %q", meta.Title, "Sample Document")
	}

	if meta.Author != "Test Author" {
		t.Errorf("Author = %q, want %q", meta.Author, "Test Author")
	}

	if meta.Description != "A sample document for testing" {
		t.Errorf("Description = %q, want %q", meta.Description, "A sample document for testing")
	}

	if meta.Created != "2026-01-15T10:30:00Z" {
		t.Errorf("Created = %q, want %q", meta.Created, "2026-01-15T10:30:00Z")
	}

	if meta.Modified != "2026-02-20T14:45:00Z" {
		t.Errorf("Modified = %q, want %q", meta.Modified, "2026-02-20T14:45:00Z")
	}

	if meta.Pages != 2 {
		t.Errorf("Pages = %d, want 2", meta.Pages)
	}
}

func TestReadMetadataMissingCoreXML(t *testing.T) {
	// Build a DOCX without core.xml or app.xml.
	tmpDir := t.TempDir()
	path := tmpDir + "/nocore.docx"
	buildMinimalDOCX(t, path, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>Hello</w:t></w:r></w:p></w:body>
</w:document>`)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	meta, err := docx.ReadMetadata(session)
	if err != nil {
		t.Fatalf("ReadMetadata should not error for missing core/app.xml: %v", err)
	}

	// All fields should be zero values since there's no metadata.
	if meta.Title != "" {
		t.Errorf("Title = %q, want empty", meta.Title)
	}

	if meta.Pages != 0 {
		t.Errorf("Pages = %d, want 0", meta.Pages)
	}
}
