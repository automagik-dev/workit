package docx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/docx"
)

func TestReplace(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "replaced.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.Replace(session, "Sample Document", "My Report")
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	if n != 1 {
		t.Errorf("replacements = %d, want 1", n)
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

	if !strings.Contains(md, "My Report") {
		t.Errorf("expected replaced text 'My Report' in markdown:\n%s", md)
	}

	if strings.Contains(md, "Sample Document") {
		t.Errorf("old text 'Sample Document' still present in markdown:\n%s", md)
	}
}

func TestReplaceMultipleOccurrences(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "multi.docx")
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Hello world</w:t></w:r></w:p>
    <w:p><w:r><w:t>Hello again</w:t></w:r></w:p>
    <w:p><w:r><w:t>Goodbye</w:t></w:r></w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.Replace(session, "Hello", "Hi")
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	if n != 2 {
		t.Errorf("replacements = %d, want 2", n)
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

	if !strings.Contains(md, "Hi world") {
		t.Errorf("expected 'Hi world', got:\n%s", md)
	}

	if !strings.Contains(md, "Hi again") {
		t.Errorf("expected 'Hi again', got:\n%s", md)
	}

	if strings.Contains(md, "Hello") {
		t.Errorf("old text 'Hello' still present:\n%s", md)
	}
}

func TestReplaceAcrossRuns(t *testing.T) {
	// Build a DOCX where "Sample Document" is split across three runs.
	tmp := filepath.Join(t.TempDir(), "split-runs.docx")
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:rPr><w:b/></w:rPr><w:t>Sam</w:t></w:r>
      <w:r><w:t>ple Docu</w:t></w:r>
      <w:r><w:t>ment</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.Replace(session, "Sample Document", "Test File")
	if err != nil {
		t.Fatalf("Replace: %v", err)
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

	if !strings.Contains(md, "Test File") {
		t.Errorf("expected 'Test File', got:\n%s", md)
	}

	if strings.Contains(md, "Sample Document") {
		t.Errorf("old text 'Sample Document' still present:\n%s", md)
	}

	if strings.Contains(md, "Sam") && !strings.Contains(md, "Test") {
		t.Errorf("partial old run text 'Sam' still present:\n%s", md)
	}
}

func TestInsertParagraphAfterHeading(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "insert-heading.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.InsertParagraph(session, "heading:Introduction", "This was inserted after the Introduction heading.")
	if err != nil {
		t.Fatalf("InsertParagraph: %v", err)
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

	ds, err := docx.ReadStructure(session2)
	if err != nil {
		t.Fatalf("ReadStructure: %v", err)
	}

	// The new paragraph should appear at index 2 (after title=0, heading1=1).
	if len(ds.Paragraphs) != 8 {
		t.Fatalf("paragraph count = %d, want 8", len(ds.Paragraphs))
	}

	got := ds.Paragraphs[2].Text

	want := "This was inserted after the Introduction heading."
	if got != want {
		t.Errorf("paragraph[2] = %q, want %q", got, want)
	}
}

func TestInsertParagraphByIndex(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "insert-index.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.InsertParagraph(session, "paragraph:0", "Inserted after first paragraph.")
	if err != nil {
		t.Fatalf("InsertParagraph: %v", err)
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

	ds, err := docx.ReadStructure(session2)
	if err != nil {
		t.Fatalf("ReadStructure: %v", err)
	}

	if len(ds.Paragraphs) != 8 {
		t.Fatalf("paragraph count = %d, want 8", len(ds.Paragraphs))
	}

	got := ds.Paragraphs[1].Text

	want := "Inserted after first paragraph."
	if got != want {
		t.Errorf("paragraph[1] = %q, want %q", got, want)
	}
}

func TestDeleteSection(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "delete-section.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Delete the "Details" section (Heading2 + its body paragraphs).
	err = docx.DeleteSection(session, "Details")
	if err != nil {
		t.Fatalf("DeleteSection: %v", err)
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

	if strings.Contains(md, "Details") {
		t.Errorf("deleted heading 'Details' still present:\n%s", md)
	}

	if strings.Contains(md, "Second paragraph") {
		t.Errorf("deleted paragraph 'Second paragraph' still present:\n%s", md)
	}

	if strings.Contains(md, "Third paragraph") {
		t.Errorf("deleted paragraph 'Third paragraph' still present:\n%s", md)
	}

	// The introduction section should remain.
	if !strings.Contains(md, "Introduction") {
		t.Errorf("Introduction heading should remain:\n%s", md)
	}

	if !strings.Contains(md, "first paragraph") {
		t.Errorf("first paragraph should remain:\n%s", md)
	}
}

func TestSetStyle(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "set-style.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Change paragraph 2 (the first body paragraph, index 2) to Heading2 style.
	err = docx.SetStyle(session, 2, "Heading2")
	if err != nil {
		t.Fatalf("SetStyle: %v", err)
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

	ds, err := docx.ReadStructure(session2)
	if err != nil {
		t.Fatalf("ReadStructure: %v", err)
	}

	if ds.Paragraphs[2].Style != "Heading2" {
		t.Errorf("paragraph[2] style = %q, want 'Heading2'", ds.Paragraphs[2].Style)
	}

	// Verify via markdown: should now be rendered as ### heading.
	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "### This is the first paragraph") {
		t.Errorf("expected ### heading for styled paragraph, got:\n%s", md)
	}
}

func TestEditPreservesFormatting(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "preserve-format.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Get structure before edit.
	dsBefore, err := docx.ReadStructure(session)
	if err != nil {
		t.Fatalf("ReadStructure before: %v", err)
	}

	// Replace text in one paragraph.
	_, err = docx.Replace(session, "first paragraph", "opening paragraph")
	if err != nil {
		t.Fatalf("Replace: %v", err)
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

	dsAfter, err := docx.ReadStructure(session2)
	if err != nil {
		t.Fatalf("ReadStructure after: %v", err)
	}

	// All paragraph styles should be preserved.
	if len(dsAfter.Paragraphs) != len(dsBefore.Paragraphs) {
		t.Fatalf("paragraph count changed: %d -> %d", len(dsBefore.Paragraphs), len(dsAfter.Paragraphs))
	}

	for i, before := range dsBefore.Paragraphs {
		after := dsAfter.Paragraphs[i]
		if before.Style != after.Style {
			t.Errorf("paragraph[%d] style changed: %q -> %q", i, before.Style, after.Style)
		}
	}

	// Verify the replace actually took effect.
	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "opening paragraph") {
		t.Errorf("replacement text not found:\n%s", md)
	}
}

func TestReplaceEmptyOld(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	_, err = docx.Replace(session, "", "something")
	if err == nil {
		t.Fatal("expected error for empty old text, got nil")
	}
}

func TestInsertParagraphInvalidReference(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.InsertParagraph(session, "nocolon", "text")
	if err == nil {
		t.Fatal("expected error for invalid reference, got nil")
	}

	err = docx.InsertParagraph(session, "heading:NonexistentHeading", "text")
	if err == nil {
		t.Fatal("expected error for nonexistent heading, got nil")
	}

	err = docx.InsertParagraph(session, "paragraph:999", "text")
	if err == nil {
		t.Fatal("expected error for out-of-range paragraph, got nil")
	}
}

func TestDeleteSectionNotFound(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.DeleteSection(session, "NonexistentSection")
	if err == nil {
		t.Fatal("expected error for nonexistent section, got nil")
	}
}

func TestSetStyleOutOfRange(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.SetStyle(session, 999, "Heading1")
	if err == nil {
		t.Fatal("expected error for out-of-range paragraph index, got nil")
	}
}

func TestSetStyleCreatesProperties(t *testing.T) {
	// Test setting a style on a paragraph that has no w:pPr at all.
	tmp := filepath.Join(t.TempDir(), "no-ppr.docx")
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Plain text</w:t></w:r></w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.SetStyle(session, 0, "Heading1")
	if err != nil {
		t.Fatalf("SetStyle: %v", err)
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

	ds, err := docx.ReadStructure(session2)
	if err != nil {
		t.Fatalf("ReadStructure: %v", err)
	}

	if len(ds.Paragraphs) == 0 {
		t.Fatal("no paragraphs found")
	}

	if ds.Paragraphs[0].Style != "Heading1" {
		t.Errorf("style = %q, want 'Heading1'", ds.Paragraphs[0].Style)
	}
}

func TestReplaceInTable(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "table-replace.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.Replace(session, "Alice", "Carol")
	if err != nil {
		t.Fatalf("Replace: %v", err)
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

	if !strings.Contains(md, "Carol") {
		t.Errorf("expected 'Carol' in table:\n%s", md)
	}

	if strings.Contains(md, "Alice") {
		t.Errorf("old text 'Alice' still present:\n%s", md)
	}
}

// copyFile copies a file from src to dst.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}

	if mkdirErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkdirErr != nil {
		t.Fatalf("mkdir: %v", mkdirErr)
	}

	if writeErr := os.WriteFile(dst, data, 0o644); writeErr != nil {
		t.Fatalf("write %s: %v", dst, writeErr)
	}
}
