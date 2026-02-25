package docx

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
)

// Sentinel errors for comment operations.
var (
	errCommentRefMustBeParagraph = errors.New("comment reference must be 'paragraph:N'")
	errNoBody                    = errors.New("no w:body element found in document.xml")
	errParagraphOutOfRange       = errors.New("paragraph index out of range")
	errNoRootContentTypes        = errors.New("no root in [Content_Types].xml")
	errNoRootRels                = errors.New("no root in relationship file")
	errCommentsXMLMissing        = errors.New("no comments.xml found")
	errCommentsXMLNoRoot         = errors.New("comments.xml has no root element")
	errCommentNotFound           = errors.New("comment not found")
	errCommentsExtMissing        = errors.New("no commentsExtended.xml found")
	errCommentsExtNoRoot         = errors.New("commentsExtended.xml has no root element")
	errCommentExNotFound         = errors.New("commentEx entry not found")
	errCommentExNoParaID         = errors.New("commentEx entry has no paraId")
	errCommentsExtNoEntry        = errors.New("commentsExtended.xml has no entry at index")
)

// tagComment is the XML tag name for comment elements.
const tagComment = "comment"

// Comment describes a single comment in a DOCX document.
type Comment struct {
	ID     string `json:"id"`
	Author string `json:"author"`
	Date   string `json:"date"`
	Text   string `json:"text"`
}

// AddComment adds a comment anchored to a paragraph reference.
// Reference format: "paragraph:N" (0-indexed paragraph).
//
// This atomically updates:
//   - word/document.xml -- comment range anchors + reference
//   - word/comments.xml -- comment content
//   - word/commentsExtended.xml -- threading metadata
//   - [Content_Types].xml -- content type entries for new parts
//   - word/_rels/document.xml.rels -- relationship entries
func AddComment(session *EditSession, paragraphRef, text, author string) error {
	refType, refValue, err := parseReference(paragraphRef)
	if err != nil {
		return err
	}

	if refType != "paragraph" {
		return fmt.Errorf("%w, got %q", errCommentRefMustBeParagraph, paragraphRef)
	}

	paraIdx, err := strconv.Atoi(refValue)
	if err != nil {
		return fmt.Errorf("invalid paragraph index %q: %w", refValue, err)
	}

	// 1. Get the document body.
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	p := findParagraphByIndex(body, paraIdx)
	if p == nil {
		return fmt.Errorf("%w: %d", errParagraphOutOfRange, paraIdx)
	}

	// 2. Determine the next comment ID.
	commentID := nextCommentID(session)
	commentIDStr := strconv.Itoa(commentID)
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Generate a paraId for commentsExtended (8-char hex).
	paraID := fmt.Sprintf("%08X", rand.Uint32()) //nolint:gosec // non-cryptographic
	durableID := strconv.Itoa(int(rand.Int31())) //nolint:gosec // non-cryptographic

	// 3. Insert comment anchors into the paragraph in document.xml.
	insertCommentAnchors(p, commentIDStr)

	session.MarkDirty("word/document.xml")

	// 4. Update word/comments.xml.
	addCommentEntry(session, commentIDStr, author, now, text)

	// 5. Update word/commentsExtended.xml.
	addCommentExtended(session, paraID)

	// 6. Update word/commentsIds.xml (optional but we include it).
	addCommentIds(session, paraID, durableID)

	// 7. Ensure content types and relationships are set up.
	if err := ensureCommentContentTypes(session); err != nil {
		return fmt.Errorf("ensure content types: %w", err)
	}

	if err := ensureCommentRelationships(session); err != nil {
		return fmt.Errorf("ensure relationships: %w", err)
	}

	return nil
}

// ListComments returns all comments found in word/comments.xml.
func ListComments(session *EditSession) ([]Comment, error) {
	commentsDoc, err := session.Part("word/comments.xml")
	if err != nil {
		// No comments file means no comments.
		return nil, nil //nolint:nilerr // missing comments.xml is expected for docs without comments
	}

	root := commentsDoc.Root()
	if root == nil {
		return nil, nil
	}

	var comments []Comment

	for _, c := range root.ChildElements() {
		if c.Tag != tagComment {
			continue
		}

		id := attrVal(c, "w:id", "id")
		auth := attrVal(c, "w:author", "author")
		date := attrVal(c, "w:date", "date")

		// Extract text from all w:p/w:r/w:t inside the comment.
		var sb strings.Builder

		for _, p := range c.ChildElements() {
			if p.Tag == "p" {
				for _, r := range p.ChildElements() {
					if r.Tag == "r" {
						for _, t := range r.ChildElements() {
							if t.Tag == "t" {
								sb.WriteString(t.Text())
							}
						}
					}
				}
			}
		}

		comments = append(comments, Comment{
			ID:     id,
			Author: auth,
			Date:   date,
			Text:   sb.String(),
		})
	}

	return comments, nil
}

// insertCommentAnchors inserts commentRangeStart, commentRangeEnd, and a
// commentReference run into the paragraph element. The anchors wrap all
// existing content in the paragraph.
func insertCommentAnchors(p *etree.Element, commentID string) {
	// Create commentRangeStart.
	rangeStart := etree.NewElement("w:commentRangeStart")
	rangeStart.CreateAttr("w:id", commentID)

	// Create commentRangeEnd.
	rangeEnd := etree.NewElement("w:commentRangeEnd")
	rangeEnd.CreateAttr("w:id", commentID)

	// Create the commentReference run.
	refRun := etree.NewElement("w:r")
	rPr := refRun.CreateElement("w:rPr")
	rStyle := rPr.CreateElement("w:rStyle")
	rStyle.CreateAttr("w:val", "CommentReference")
	commentRef := refRun.CreateElement("w:commentReference")
	commentRef.CreateAttr("w:id", commentID)

	// Insert rangeStart at the beginning of the paragraph (after pPr if present).
	children := p.ChildElements()
	insertIdx := 0

	if len(children) > 0 && children[0].Tag == "pPr" {
		insertIdx = children[0].Index() + 1
	}

	p.InsertChildAt(insertIdx, rangeStart)

	// Append rangeEnd and reference run at the end.
	p.AddChild(rangeEnd)
	p.AddChild(refRun)
}

// addCommentEntry adds a comment to word/comments.xml, creating the file if needed.
func addCommentEntry(session *EditSession, id, author, date, text string) {
	commentsDoc, err := session.Part("word/comments.xml")
	if err != nil {
		// File doesn't exist; create it.
		commentsDoc = etree.NewDocument()
		commentsDoc.CreateProcInst("xml", `version="1.0" encoding="UTF-8" standalone="yes"`)
		root := commentsDoc.CreateElement("w:comments")
		root.CreateAttr("xmlns:w", nsW)
		root.CreateAttr("xmlns:r", "http://schemas.openxmlformats.org/officeDocument/2006/relationships")

		// Register in session.
		session.AddRawPart("word/comments.xml", []byte{})
		session.parts["word/comments.xml"] = &xmlDoc{doc: commentsDoc}
	}

	root := commentsDoc.Root()
	comment := root.CreateElement("w:comment")
	comment.CreateAttr("w:id", id)
	comment.CreateAttr("w:author", author)
	comment.CreateAttr("w:date", date)

	para := comment.CreateElement("w:p")
	run := para.CreateElement("w:r")
	t := run.CreateElement("w:t")
	t.SetText(text)

	if len(text) > 0 && (text[0] == ' ' || text[len(text)-1] == ' ') {
		t.CreateAttr("xml:space", "preserve")
	}

	session.MarkDirty("word/comments.xml")
}

// addCommentExtended adds an entry to word/commentsExtended.xml.
func addCommentExtended(session *EditSession, paraID string) {
	const partName = "word/commentsExtended.xml"
	const ns15 = "http://schemas.microsoft.com/office/word/2012/wordml"

	doc, err := session.Part(partName)
	if err != nil {
		doc = etree.NewDocument()
		doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8" standalone="yes"`)
		root := doc.CreateElement("w15:commentsEx")
		root.CreateAttr("xmlns:w15", ns15)
		_ = root

		session.AddRawPart(partName, []byte{})
		session.parts[partName] = &xmlDoc{doc: doc}
	}

	root := doc.Root()
	entry := root.CreateElement("w15:commentEx")
	entry.CreateAttr("w15:paraId", paraID)
	entry.CreateAttr("w15:done", "0")

	session.MarkDirty(partName)
}

// addCommentIds adds an entry to word/commentsIds.xml.
func addCommentIds(session *EditSession, paraID, durableID string) {
	const partName = "word/commentsIds.xml"
	const ns16cid = "http://schemas.microsoft.com/office/word/2016/wordml/cid"

	doc, err := session.Part(partName)
	if err != nil {
		doc = etree.NewDocument()
		doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8" standalone="yes"`)
		root := doc.CreateElement("w16cid:commentsIds")
		root.CreateAttr("xmlns:w16cid", ns16cid)
		_ = root

		session.AddRawPart(partName, []byte{})
		session.parts[partName] = &xmlDoc{doc: doc}
	}

	root := doc.Root()
	entry := root.CreateElement("w16cid:commentId")
	entry.CreateAttr("w16cid:paraId", paraID)
	entry.CreateAttr("w16cid:durableId", durableID)

	session.MarkDirty(partName)
}

// ensureCommentContentTypes adds Override entries for comment parts in [Content_Types].xml.
func ensureCommentContentTypes(session *EditSession) error {
	doc, err := session.Part("[Content_Types].xml")
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return errNoRootContentTypes
	}

	requiredOverrides := map[string]string{
		"/word/comments.xml":         "application/vnd.openxmlformats-officedocument.wordprocessingml.comments+xml",
		"/word/commentsExtended.xml": "application/vnd.openxmlformats-officedocument.wordprocessingml.commentsExtended+xml",
		"/word/commentsIds.xml":      "application/vnd.openxmlformats-officedocument.wordprocessingml.commentsIds+xml",
	}

	// Collect existing overrides.
	existing := make(map[string]bool)

	for _, child := range root.ChildElements() {
		if child.Tag == "Override" {
			partName := ""
			if a := child.SelectAttr("PartName"); a != nil {
				partName = a.Value
			}

			existing[partName] = true
		}
	}

	for partName, contentType := range requiredOverrides {
		if existing[partName] {
			continue
		}

		override := root.CreateElement("Override")
		override.CreateAttr("PartName", partName)
		override.CreateAttr("ContentType", contentType)
	}

	session.MarkDirty("[Content_Types].xml")

	return nil
}

// ensureCommentRelationships adds relationship entries for comment parts
// in word/_rels/document.xml.rels.
func ensureCommentRelationships(session *EditSession) error {
	const relsPath = "word/_rels/document.xml.rels"

	doc, err := session.Part(relsPath)
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("%w: %s", errNoRootRels, relsPath)
	}

	type relInfo struct {
		relType string
		target  string
	}
	requiredRels := []relInfo{
		{
			relType: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/comments",
			target:  "comments.xml",
		},
		{
			relType: "http://schemas.microsoft.com/office/2011/relationships/commentsExtended",
			target:  "commentsExtended.xml",
		},
		{
			relType: "http://schemas.microsoft.com/office/2018/08/relationships/commentsIds",
			target:  "commentsIds.xml",
		},
	}

	// Collect existing relationship types.
	existingTypes := make(map[string]bool)
	maxRelID := 0

	for _, child := range root.ChildElements() {
		if child.Tag == "Relationship" {
			if a := child.SelectAttr("Type"); a != nil {
				existingTypes[a.Value] = true
			}

			if a := child.SelectAttr("Id"); a != nil {
				if strings.HasPrefix(a.Value, "rId") {
					if n, err := strconv.Atoi(a.Value[3:]); err == nil && n > maxRelID {
						maxRelID = n
					}
				}
			}
		}
	}

	for _, rel := range requiredRels {
		if existingTypes[rel.relType] {
			continue
		}

		maxRelID++

		entry := root.CreateElement("Relationship")
		entry.CreateAttr("Id", fmt.Sprintf("rId%d", maxRelID))
		entry.CreateAttr("Type", rel.relType)
		entry.CreateAttr("Target", rel.target)
	}

	session.MarkDirty(relsPath)

	return nil
}

// nextCommentID scans word/comments.xml (if it exists) for the highest
// existing comment ID and returns max+1. Falls back to 1 if no comments exist.
func nextCommentID(session *EditSession) int {
	commentsDoc, err := session.Part("word/comments.xml")
	if err != nil {
		return 1
	}

	root := commentsDoc.Root()
	if root == nil {
		return 1
	}

	maxID := 0

	for _, c := range root.ChildElements() {
		if c.Tag == tagComment {
			id := attrVal(c, "w:id", "id")
			if n, err := strconv.Atoi(id); err == nil && n > maxID {
				maxID = n
			}
		}
	}

	return maxID + 1
}

// ReplyToComment adds a reply to an existing comment.
// It creates a new comment entry in word/comments.xml that references the same
// paragraph, and links the reply to its parent in word/commentsExtended.xml
// via the w15:paraIdParent attribute.
func ReplyToComment(session *EditSession, commentID int, text, author string) error {
	// 1. Verify the parent comment exists in word/comments.xml.
	commentsDoc, err := session.Part("word/comments.xml")
	if err != nil {
		return fmt.Errorf("%w: cannot reply to comment %d", errCommentsXMLMissing, commentID)
	}

	root := commentsDoc.Root()
	if root == nil {
		return errCommentsXMLNoRoot
	}

	parentIDStr := strconv.Itoa(commentID)
	var parentFound bool

	for _, c := range root.ChildElements() {
		if c.Tag == tagComment {
			id := attrVal(c, "w:id", "id")
			if id == parentIDStr {
				parentFound = true

				break
			}
		}
	}

	if !parentFound {
		return fmt.Errorf("%w: %d", errCommentNotFound, commentID)
	}

	// 2. Find the parent comment's paraId in commentsExtended.xml.
	parentParaID, err := findCommentParaID(session, commentID)
	if err != nil {
		return fmt.Errorf("find parent paraId: %w", err)
	}

	// 3. Determine the next comment ID and create the reply.
	replyID := nextCommentID(session)
	replyIDStr := strconv.Itoa(replyID)
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Generate a paraId for the reply's commentsExtended entry.
	replyParaID := fmt.Sprintf("%08X", rand.Uint32()) //nolint:gosec // non-cryptographic
	replyDurableID := strconv.Itoa(int(rand.Int31())) //nolint:gosec // non-cryptographic

	// 4. Add comment entry in word/comments.xml.
	addCommentEntry(session, replyIDStr, author, now, text)

	// 5. Add commentsExtended entry with paraIdParent linking to parent.
	addCommentExtendedWithParent(session, replyParaID, parentParaID)

	// 6. Add commentsIds entry.
	addCommentIds(session, replyParaID, replyDurableID)

	return nil
}

// ResolveComment marks a comment as done/resolved by setting w15:done="1"
// on the corresponding w15:commentEx element in word/commentsExtended.xml.
func ResolveComment(session *EditSession, commentID int) error {
	// Find the comment's paraId from commentsExtended.xml.
	// The commentsExtended entries are indexed positionally to match comments.
	const partName = "word/commentsExtended.xml"

	doc, err := session.Part(partName)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve comment %d", errCommentsExtMissing, commentID)
	}

	root := doc.Root()
	if root == nil {
		return errCommentsExtNoRoot
	}

	// We need to find the paraId for this comment, then set done="1".
	paraID, err := findCommentParaID(session, commentID)
	if err != nil {
		return fmt.Errorf("find paraId for comment %d: %w", commentID, err)
	}

	// Find the commentEx element with matching paraId and set done="1".
	for _, entry := range root.ChildElements() {
		if entry.Tag != "commentEx" {
			continue
		}

		pid := attrVal(entry, "w15:paraId", "paraId")
		if pid == paraID {
			// Update or create the done attribute.
			if a := entry.SelectAttr("w15:done"); a != nil {
				a.Value = "1"
			} else if a := entry.SelectAttr("done"); a != nil {
				a.Value = "1"
			} else {
				entry.CreateAttr("w15:done", "1")
			}

			session.MarkDirty(partName)

			return nil
		}
	}

	return fmt.Errorf("%w: comment %d (paraId %s)", errCommentExNotFound, commentID, paraID)
}

// findCommentParaID finds the paraId for a comment by its numeric ID.
// Comments and commentsExtended entries are positionally correlated:
// the Nth comment in comments.xml corresponds to the Nth commentEx
// in commentsExtended.xml.
func findCommentParaID(session *EditSession, commentID int) (string, error) {
	commentsDoc, err := session.Part("word/comments.xml")
	if err != nil {
		return "", errCommentsXMLMissing
	}

	root := commentsDoc.Root()
	if root == nil {
		return "", errCommentsXMLNoRoot
	}

	// Find the positional index of the comment with this ID.
	commentIDStr := strconv.Itoa(commentID)
	idx := -1
	i := 0

	for _, c := range root.ChildElements() {
		if c.Tag == tagComment {
			if attrVal(c, "w:id", "id") == commentIDStr {
				idx = i

				break
			}

			i++
		}
	}

	if idx < 0 {
		return "", fmt.Errorf("%w: %d in comments.xml", errCommentNotFound, commentID)
	}

	// Find the corresponding entry in commentsExtended.xml.
	extDoc, err := session.Part("word/commentsExtended.xml")
	if err != nil {
		return "", errCommentsExtMissing
	}

	extRoot := extDoc.Root()
	if extRoot == nil {
		return "", errCommentsExtNoRoot
	}

	j := 0

	for _, entry := range extRoot.ChildElements() {
		if entry.Tag == "commentEx" {
			if j == idx {
				paraID := attrVal(entry, "w15:paraId", "paraId")
				if paraID == "" {
					return "", fmt.Errorf("%w: at index %d", errCommentExNoParaID, idx)
				}

				return paraID, nil
			}

			j++
		}
	}

	return "", fmt.Errorf("%w: %d", errCommentsExtNoEntry, idx)
}

// addCommentExtendedWithParent adds an entry to word/commentsExtended.xml
// with a paraIdParent attribute linking it to a parent comment.
func addCommentExtendedWithParent(session *EditSession, paraID, parentParaID string) {
	const partName = "word/commentsExtended.xml"
	const ns15 = "http://schemas.microsoft.com/office/word/2012/wordml"

	doc, err := session.Part(partName)
	if err != nil {
		doc = etree.NewDocument()
		doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8" standalone="yes"`)
		root := doc.CreateElement("w15:commentsEx")
		root.CreateAttr("xmlns:w15", ns15)
		_ = root

		session.AddRawPart(partName, []byte{})
		session.parts[partName] = &xmlDoc{doc: doc}
	}

	root := doc.Root()
	entry := root.CreateElement("w15:commentEx")
	entry.CreateAttr("w15:paraId", paraID)
	entry.CreateAttr("w15:paraIdParent", parentParaID)
	entry.CreateAttr("w15:done", "0")

	session.MarkDirty(partName)
}

// attrVal returns the value of an attribute, trying multiple possible names.
func attrVal(elem *etree.Element, names ...string) string {
	for _, name := range names {
		if a := elem.SelectAttr(name); a != nil {
			return a.Value
		}
	}

	return ""
}
