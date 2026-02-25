package docx_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"

	"github.com/automagik-dev/workit/internal/docx"
)

func TestTrackReplace(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "track.docx")
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Hello world</w:t></w:r></w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.TrackReplace(session, "Hello", "Goodbye", "Test Author")
	if err != nil {
		t.Fatalf("TrackReplace: %v", err)
	}

	if n != 1 {
		t.Errorf("replacements = %d, want 1", n)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Verify the XML structure.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	doc, err := session2.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part: %v", err)
	}

	xml, _ := doc.WriteToString()

	// Verify <w:del> is present with old text.
	if !strings.Contains(xml, "delText") {
		t.Errorf("expected <w:delText> in output, got:\n%s", xml)
	}

	if !strings.Contains(xml, "Hello") {
		t.Errorf("expected deleted text 'Hello' in output, got:\n%s", xml)
	}

	// Verify <w:ins> is present with new text.
	if !strings.Contains(xml, "<w:ins") {
		t.Errorf("expected <w:ins> in output, got:\n%s", xml)
	}

	if !strings.Contains(xml, "Goodbye") {
		t.Errorf("expected inserted text 'Goodbye' in output, got:\n%s", xml)
	}

	// Verify author attribution.
	if !strings.Contains(xml, `w:author="Test Author"`) {
		t.Errorf("expected author attribution, got:\n%s", xml)
	}

	// Verify surrounding text " world" is preserved.
	if !strings.Contains(xml, "world") {
		t.Errorf("expected surrounding text 'world' preserved, got:\n%s", xml)
	}
}

func TestAcceptChanges(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "accept.docx")
	// Build a doc with existing tracked changes.
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:del w:id="1" w:author="Bot" w:date="2026-02-24T12:00:00Z">
        <w:r><w:delText xml:space="preserve">old</w:delText></w:r>
      </w:del>
      <w:ins w:id="2" w:author="Bot" w:date="2026-02-24T12:00:00Z">
        <w:r><w:t xml:space="preserve">new</w:t></w:r>
      </w:ins>
    </w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	if acceptErr := docx.AcceptChanges(session); acceptErr != nil {
		t.Fatalf("AcceptChanges: %v", acceptErr)
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

	// After accepting: "old" should be gone, "new" should remain.
	if strings.Contains(md, "old") {
		t.Errorf("deleted text 'old' should be removed after accept, got:\n%s", md)
	}

	if !strings.Contains(md, "new") {
		t.Errorf("inserted text 'new' should remain after accept, got:\n%s", md)
	}

	// Verify no tracked change markers remain.
	doc, _ := session2.Part("word/document.xml")

	xml, _ := doc.WriteToString()
	if strings.Contains(xml, "<w:del") {
		t.Errorf("expected no <w:del> after accept, got:\n%s", xml)
	}

	if strings.Contains(xml, "<w:ins") {
		t.Errorf("expected no <w:ins> after accept, got:\n%s", xml)
	}
}

func TestRejectChanges(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "reject.docx")
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:del w:id="1" w:author="Bot" w:date="2026-02-24T12:00:00Z">
        <w:r><w:delText xml:space="preserve">old</w:delText></w:r>
      </w:del>
      <w:ins w:id="2" w:author="Bot" w:date="2026-02-24T12:00:00Z">
        <w:r><w:t xml:space="preserve">new</w:t></w:r>
      </w:ins>
    </w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	if rejectErr := docx.RejectChanges(session); rejectErr != nil {
		t.Fatalf("RejectChanges: %v", rejectErr)
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

	// After rejecting: "new" should be gone, "old" should remain (as regular text).
	if strings.Contains(md, "new") {
		t.Errorf("inserted text 'new' should be removed after reject, got:\n%s", md)
	}

	if !strings.Contains(md, "old") {
		t.Errorf("deleted text 'old' should be restored after reject, got:\n%s", md)
	}

	// Verify no tracked change markers remain.
	doc, _ := session2.Part("word/document.xml")

	xml, _ := doc.WriteToString()
	if strings.Contains(xml, "<w:del") {
		t.Errorf("expected no <w:del> after reject, got:\n%s", xml)
	}

	if strings.Contains(xml, "<w:ins") {
		t.Errorf("expected no <w:ins> after reject, got:\n%s", xml)
	}
}

func TestTrackReplacePreservesFormatting(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "format.docx")
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:rPr><w:b/></w:rPr>
        <w:t>Bold Hello world</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.TrackReplace(session, "Hello", "Hi", "Reviewer")
	if err != nil {
		t.Fatalf("TrackReplace: %v", err)
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

	doc, err := session2.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part: %v", err)
	}

	xml, _ := doc.WriteToString()

	// The bold formatting should be preserved in the del/ins runs.
	// The <w:b/> element should appear inside both the del and ins run properties.
	body := findBodyFromDoc(doc)
	if body == nil {
		t.Fatal("no body")
	}

	p := firstParagraph(body)
	if p == nil {
		t.Fatal("no paragraph")
	}

	// Check that "Bold " (text before match) preserves bold formatting.
	foundBoldBefore := false

	for _, child := range p.ChildElements() {
		if child.Tag == "r" {
			txt := extractRunText(child)
			if strings.Contains(txt, "Bold") {
				if hasBoldFormatting(child) {
					foundBoldBefore = true
				}
			}
		}
	}

	if !foundBoldBefore {
		t.Errorf("text before match should preserve bold formatting, got:\n%s", xml)
	}

	// Check that <w:del> and <w:ins> runs have bold formatting.
	foundBoldDel := false
	foundBoldIns := false

	for _, child := range p.ChildElements() {
		if child.Tag == "del" {
			for _, r := range child.ChildElements() {
				if r.Tag == "r" && hasBoldFormatting(r) {
					foundBoldDel = true
				}
			}
		}

		if child.Tag == "ins" {
			for _, r := range child.ChildElements() {
				if r.Tag == "r" && hasBoldFormatting(r) {
					foundBoldIns = true
				}
			}
		}
	}

	if !foundBoldDel {
		t.Errorf("deleted run should preserve bold formatting, got:\n%s", xml)
	}

	if !foundBoldIns {
		t.Errorf("inserted run should preserve bold formatting, got:\n%s", xml)
	}
}

func TestTrackReplaceRoundTrip(t *testing.T) {
	// TrackReplace followed by AcceptChanges should produce the same result
	// as a plain Replace.
	tmp := filepath.Join(t.TempDir(), "roundtrip.docx")
	buildMinimalDOCX(t, tmp, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>The quick brown fox</w:t></w:r></w:p>
  </w:body>
</w:document>`)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	n, err := docx.TrackReplace(session, "quick brown", "slow red", "Bot")
	if err != nil {
		t.Fatalf("TrackReplace: %v", err)
	}

	if n != 1 {
		t.Errorf("replacements = %d, want 1", n)
	}

	if acceptErr := docx.AcceptChanges(session); acceptErr != nil {
		t.Fatalf("AcceptChanges: %v", acceptErr)
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

	if !strings.Contains(md, "The slow red fox") {
		t.Errorf("expected 'The slow red fox', got:\n%s", md)
	}
}

// Test helpers.
func findBodyFromDoc(doc *etree.Document) *etree.Element {
	root := doc.Root()
	if root == nil {
		return nil
	}

	for _, child := range root.ChildElements() {
		if child.Tag == "body" {
			return child
		}
	}

	return nil
}

func firstParagraph(body *etree.Element) *etree.Element {
	for _, child := range body.ChildElements() {
		if child.Tag == "p" {
			return child
		}
	}

	return nil
}

func extractRunText(r *etree.Element) string {
	var sb strings.Builder

	for _, child := range r.ChildElements() {
		if child.Tag == "t" {
			sb.WriteString(child.Text())
		}
	}

	return sb.String()
}

func hasBoldFormatting(r *etree.Element) bool {
	for _, child := range r.ChildElements() {
		if child.Tag == "rPr" {
			for _, prop := range child.ChildElements() {
				if prop.Tag == "b" {
					return true
				}
			}
		}
	}

	return false
}
