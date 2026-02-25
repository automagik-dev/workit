package docx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/docx"
)

func TestOpen(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	if session.Path() != path {
		t.Errorf("Path() = %q, want %q", session.Path(), path)
	}
}

func TestOpenNotFound(t *testing.T) {
	_, err := docx.Open("/nonexistent/file.docx")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestListParts(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	parts := session.ListParts()
	if len(parts) == 0 {
		t.Fatal("expected at least one part")
	}

	// Verify expected parts are present.
	expected := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"word/document.xml",
		"word/_rels/document.xml.rels",
		"docProps/core.xml",
		"docProps/app.xml",
		"word/styles.xml",
	}

	partSet := make(map[string]bool)
	for _, p := range parts {
		partSet[p] = true
	}

	for _, e := range expected {
		if !partSet[e] {
			t.Errorf("missing expected part %q", e)
		}
	}

	// Verify sorted order.
	for i := 1; i < len(parts); i++ {
		if parts[i] < parts[i-1] {
			t.Errorf("parts not sorted: %q before %q", parts[i-1], parts[i])
		}
	}
}

func TestPart(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	doc, err := session.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part: %v", err)
	}

	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	// Accessing the same part again should return the cached version.
	doc2, err := session.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part (cached): %v", err)
	}

	if doc != doc2 {
		t.Error("expected cached document to be the same pointer")
	}
}

func TestPartNotFound(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	_, err = session.Part("nonexistent/part.xml")
	if err == nil {
		t.Fatal("expected error for nonexistent part")
	}
}

func TestSave(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Parse a part and mark it dirty (no actual modifications, just test the save path).
	_, err = session.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part: %v", err)
	}

	session.MarkDirty("word/document.xml")

	// Save to a temp file.
	outPath := filepath.Join(t.TempDir(), "output.docx")
	if saveErr := session.Save(outPath); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	// Verify the output file exists and is a valid DOCX.
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}

	if info.Size() == 0 {
		t.Error("output file is empty")
	}

	// Re-open the saved file and verify content.
	session2, err := docx.Open(outPath)
	if err != nil {
		t.Fatalf("Open saved: %v", err)
	}
	defer session2.Close()

	parts := session2.ListParts()
	if len(parts) == 0 {
		t.Error("saved DOCX has no parts")
	}

	// Verify document.xml is still valid.
	doc, err := session2.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part from saved: %v", err)
	}

	if doc.Root() == nil {
		t.Error("saved document.xml has no root element")
	}
}

func TestSaveInPlace(t *testing.T) {
	// Copy sample to temp dir so we can modify in place.
	path := ensureSampleDOCX(t)
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "inplace.docx")

	// First save a copy.
	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if saveErr := session.Save(tmpPath); saveErr != nil {
		t.Fatalf("Save copy: %v", saveErr)
	}

	session.Close()

	// Open the copy, modify, and save in place.
	session2, err := docx.Open(tmpPath)
	if err != nil {
		t.Fatalf("Open copy: %v", err)
	}

	doc, err := session2.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part: %v", err)
	}

	// Verify the document has content.
	if doc.Root() == nil {
		t.Fatal("document has no root")
	}

	session2.MarkDirty("word/document.xml")

	if saveErr := session2.SaveInPlace(); saveErr != nil {
		t.Fatalf("SaveInPlace: %v", saveErr)
	}

	session2.Close()

	// Verify the file is still valid.
	session3, err := docx.Open(tmpPath)
	if err != nil {
		t.Fatalf("Open after SaveInPlace: %v", err)
	}
	defer session3.Close()

	if len(session3.ListParts()) == 0 {
		t.Error("SaveInPlace produced empty DOCX")
	}
}

func TestRawPart(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	raw, err := session.RawPart("word/document.xml")
	if err != nil {
		t.Fatalf("RawPart: %v", err)
	}

	if len(raw) == 0 {
		t.Error("raw bytes are empty")
	}

	if !strings.Contains(string(raw), "Sample Document") {
		t.Error("raw bytes don't contain expected text")
	}
}
