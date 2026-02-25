package docx_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

func TestAddComment(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "comment.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.AddComment(session, "paragraph:2", "Please review this paragraph", "Reviewer")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "comment-out.docx")
	if saveErr := session.Save(outPath); saveErr != nil {
		t.Fatalf("Save: %v", err)
	}

	session.Close()

	// Re-open and verify.
	session2, err := docx.Open(outPath)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	// Check document.xml has comment anchors.
	doc, err := session2.Part("word/document.xml")
	if err != nil {
		t.Fatalf("Part document.xml: %v", err)
	}

	xml, _ := doc.WriteToString()
	if !strings.Contains(xml, "commentRangeStart") {
		t.Errorf("expected commentRangeStart in document.xml, got:\n%s", xml)
	}

	if !strings.Contains(xml, "commentRangeEnd") {
		t.Errorf("expected commentRangeEnd in document.xml, got:\n%s", xml)
	}

	if !strings.Contains(xml, "commentReference") {
		t.Errorf("expected commentReference in document.xml, got:\n%s", xml)
	}

	// Check comments.xml exists and has our comment.
	comments, err := docx.ListComments(session2)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("comment count = %d, want 1", len(comments))
	}

	if comments[0].Text != "Please review this paragraph" {
		t.Errorf("comment text = %q, want 'Please review this paragraph'", comments[0].Text)
	}

	if comments[0].Author != "Reviewer" {
		t.Errorf("comment author = %q, want 'Reviewer'", comments[0].Author)
	}
}

func TestAddMultipleComments(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "multi-comment.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.AddComment(session, "paragraph:0", "First comment", "Author1")
	if err != nil {
		t.Fatalf("AddComment 1: %v", err)
	}

	err = docx.AddComment(session, "paragraph:1", "Second comment", "Author2")
	if err != nil {
		t.Fatalf("AddComment 2: %v", err)
	}

	err = docx.AddComment(session, "paragraph:2", "Third comment", "Author1")
	if err != nil {
		t.Fatalf("AddComment 3: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "multi-comment-out.docx")
	if saveErr := session.Save(outPath); saveErr != nil {
		t.Fatalf("Save: %v", err)
	}

	session.Close()

	session2, err := docx.Open(outPath)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	comments, err := docx.ListComments(session2)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}

	if len(comments) != 3 {
		t.Fatalf("comment count = %d, want 3", len(comments))
	}

	// Verify sequential IDs.
	ids := make(map[string]bool)

	for _, c := range comments {
		if c.ID == "" {
			t.Error("comment has empty ID")
		}

		if ids[c.ID] {
			t.Errorf("duplicate comment ID: %s", c.ID)
		}
		ids[c.ID] = true
	}

	// Verify texts.
	if comments[0].Text != "First comment" {
		t.Errorf("comment[0].Text = %q, want 'First comment'", comments[0].Text)
	}

	if comments[1].Text != "Second comment" {
		t.Errorf("comment[1].Text = %q, want 'Second comment'", comments[1].Text)
	}

	if comments[2].Text != "Third comment" {
		t.Errorf("comment[2].Text = %q, want 'Third comment'", comments[2].Text)
	}
}

func TestListComments(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "list-comments.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// List on a document with no comments should return nil/empty.
	comments, err := docx.ListComments(session)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}

	if len(comments) != 0 {
		t.Errorf("expected 0 comments on fresh doc, got %d", len(comments))
	}

	// Add a comment and verify list.
	err = docx.AddComment(session, "paragraph:0", "Test comment", "Tester")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	comments, err = docx.ListComments(session)
	if err != nil {
		t.Fatalf("ListComments after add: %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}

	if comments[0].Text != "Test comment" {
		t.Errorf("comment text = %q, want 'Test comment'", comments[0].Text)
	}

	if comments[0].Author != "Tester" {
		t.Errorf("comment author = %q, want 'Tester'", comments[0].Author)
	}

	if comments[0].Date == "" {
		t.Error("comment date should not be empty")
	}
}

func TestCommentFileSync(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "sync-comment.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.AddComment(session, "paragraph:0", "Sync test", "Bot")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "sync-comment-out.docx")
	if saveErr := session.Save(outPath); saveErr != nil {
		t.Fatalf("Save: %v", err)
	}

	session.Close()

	// Re-open and verify all parts exist.
	session2, err := docx.Open(outPath)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	// Verify the required parts are present.
	parts := session2.ListParts()

	partsMap := make(map[string]bool)
	for _, p := range parts {
		partsMap[p] = true
	}

	requiredParts := []string{
		"word/document.xml",
		"word/comments.xml",
		"word/commentsExtended.xml",
		"word/commentsIds.xml",
	}

	for _, rp := range requiredParts {
		if !partsMap[rp] {
			t.Errorf("expected part %q in output, got parts: %v", rp, parts)
		}
	}

	// Verify [Content_Types].xml has overrides for comment parts.
	ctDoc, err := session2.Part("[Content_Types].xml")
	if err != nil {
		t.Fatalf("Part [Content_Types].xml: %v", err)
	}

	ctXML, _ := ctDoc.WriteToString()
	if !strings.Contains(ctXML, "comments+xml") {
		t.Errorf("expected comments content type override, got:\n%s", ctXML)
	}

	if !strings.Contains(ctXML, "commentsExtended+xml") {
		t.Errorf("expected commentsExtended content type override, got:\n%s", ctXML)
	}

	// Verify word/_rels/document.xml.rels has relationship entries.
	relsDoc, err := session2.Part("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatalf("Part document.xml.rels: %v", err)
	}

	relsXML, _ := relsDoc.WriteToString()
	if !strings.Contains(relsXML, "comments.xml") {
		t.Errorf("expected comments.xml relationship, got:\n%s", relsXML)
	}

	if !strings.Contains(relsXML, "commentsExtended.xml") {
		t.Errorf("expected commentsExtended.xml relationship, got:\n%s", relsXML)
	}

	// Verify commentsExtended.xml has an entry.
	extDoc, err := session2.Part("word/commentsExtended.xml")
	if err != nil {
		t.Fatalf("Part commentsExtended.xml: %v", err)
	}

	extXML, _ := extDoc.WriteToString()
	if !strings.Contains(extXML, "commentEx") {
		t.Errorf("expected commentEx entry in commentsExtended.xml, got:\n%s", extXML)
	}

	// Verify commentsIds.xml has an entry.
	idsDoc, err := session2.Part("word/commentsIds.xml")
	if err != nil {
		t.Fatalf("Part commentsIds.xml: %v", err)
	}

	idsXML, _ := idsDoc.WriteToString()
	if !strings.Contains(idsXML, "commentId") {
		t.Errorf("expected commentId entry in commentsIds.xml, got:\n%s", idsXML)
	}
}

func TestCommentInvalidReference(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad-ref.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Missing colon.
	err = docx.AddComment(session, "nocolon", "text", "author")
	if err == nil {
		t.Fatal("expected error for invalid reference, got nil")
	}

	// Wrong type.
	err = docx.AddComment(session, "heading:Intro", "text", "author")
	if err == nil {
		t.Fatal("expected error for non-paragraph reference, got nil")
	}

	// Out of range.
	err = docx.AddComment(session, "paragraph:999", "text", "author")
	if err == nil {
		t.Fatal("expected error for out-of-range paragraph, got nil")
	}
}

func TestReplyToComment(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "reply.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Add a parent comment first.
	err = docx.AddComment(session, "paragraph:0", "Original comment", "Author1")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	// Save and reopen to ensure parts are properly written.
	outPath := filepath.Join(t.TempDir(), "reply-out.docx")
	if saveErr := session.Save(outPath); saveErr != nil {
		t.Fatalf("Save: %v", err)
	}

	session.Close()

	session2, err := docx.Open(outPath)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	// Get the parent comment ID.
	comments, err := docx.ListComments(session2)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}

	parentID := 1 // First comment gets ID 1.

	// Reply to the comment.
	err = docx.ReplyToComment(session2, parentID, "This is a reply", "Author2")
	if err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}

	// Save and verify.
	replyPath := filepath.Join(t.TempDir(), "reply-final.docx")
	if saveErr := session2.Save(replyPath); saveErr != nil {
		t.Fatalf("Save reply: %v", saveErr)
	}

	session2.Close()

	session3, err := docx.Open(replyPath)
	if err != nil {
		t.Fatalf("Open reply: %v", err)
	}
	defer session3.Close()

	// Should have 2 comments now (original + reply).
	allComments, err := docx.ListComments(session3)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}

	if len(allComments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(allComments))
	}

	if allComments[0].Text != "Original comment" {
		t.Errorf("comment[0].Text = %q, want 'Original comment'", allComments[0].Text)
	}

	if allComments[1].Text != "This is a reply" {
		t.Errorf("comment[1].Text = %q, want 'This is a reply'", allComments[1].Text)
	}

	// Verify commentsExtended.xml has the reply with paraIdParent.
	extDoc, err := session3.Part("word/commentsExtended.xml")
	if err != nil {
		t.Fatalf("Part commentsExtended.xml: %v", err)
	}

	extXML, _ := extDoc.WriteToString()
	if !strings.Contains(extXML, "paraIdParent") {
		t.Errorf("expected paraIdParent in commentsExtended.xml, got:\n%s", extXML)
	}
}

func TestReplyToCommentNotFound(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "reply-notfound.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Try to reply to a non-existent comment.
	err = docx.ReplyToComment(session, 999, "Reply", "Author")
	if err == nil {
		t.Fatal("expected error for reply to non-existent comment")
	}
}

func TestResolveComment(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "resolve.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Add a comment.
	err = docx.AddComment(session, "paragraph:0", "Needs review", "Reviewer")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	// Save and reopen.
	outPath := filepath.Join(t.TempDir(), "resolve-out.docx")
	if saveErr := session.Save(outPath); saveErr != nil {
		t.Fatalf("Save: %v", err)
	}

	session.Close()

	session2, err := docx.Open(outPath)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	// Verify the comment exists and is not resolved.
	extDoc, err := session2.Part("word/commentsExtended.xml")
	if err != nil {
		t.Fatalf("Part commentsExtended.xml: %v", err)
	}

	extXML, _ := extDoc.WriteToString()
	if !strings.Contains(extXML, `done="0"`) {
		t.Errorf("expected done=\"0\" before resolve, got:\n%s", extXML)
	}

	// Resolve the comment.
	err = docx.ResolveComment(session2, 1)
	if err != nil {
		t.Fatalf("ResolveComment: %v", err)
	}

	// Save and verify.
	resolvedPath := filepath.Join(t.TempDir(), "resolve-final.docx")
	if saveErr := session2.Save(resolvedPath); saveErr != nil {
		t.Fatalf("Save resolved: %v", saveErr)
	}

	session2.Close()

	session3, err := docx.Open(resolvedPath)
	if err != nil {
		t.Fatalf("Open resolved: %v", err)
	}
	defer session3.Close()

	extDoc2, err := session3.Part("word/commentsExtended.xml")
	if err != nil {
		t.Fatalf("Part commentsExtended.xml: %v", err)
	}

	extXML2, _ := extDoc2.WriteToString()
	if !strings.Contains(extXML2, `done="1"`) {
		t.Errorf("expected done=\"1\" after resolve, got:\n%s", extXML2)
	}
}

func TestResolveCommentNotFound(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "resolve-notfound.docx")
	buildSampleDOCXWithRels(t, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Try to resolve a non-existent comment.
	err = docx.ResolveComment(session, 999)
	if err == nil {
		t.Fatal("expected error for resolve of non-existent comment")
	}
}

// buildSampleDOCXWithRels creates a DOCX that includes document.xml.rels and
// [Content_Types].xml, needed for comment tests that modify these files.
func buildSampleDOCXWithRels(t *testing.T, path string) {
	t.Helper()
	buildSampleDOCX(t, path)
}
