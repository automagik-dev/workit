package docx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// Sentinel errors for edit operations.
var (
	errOldTextEmpty     = errors.New("old text must not be empty")
	errHeadingNotFound  = errors.New("heading not found")
	errUnknownRefType   = errors.New("unknown reference type (use heading:... or paragraph:...)")
	errInvalidReference = errors.New("invalid reference (expected type:value, e.g. heading:Summary or paragraph:5)")
)

// Constants for element tag names.
const tagTbl = "tbl"

// Replace finds all occurrences of old text in the document and replaces with newText.
// Text may span multiple <w:r> (run) elements within a paragraph.
// The replacement preserves the formatting (<w:rPr>) of the first run containing the match.
// Returns the number of replacements made.
func Replace(session *EditSession, old, newText string) (int, error) {
	if old == "" {
		return 0, errOldTextEmpty
	}

	doc, err := session.Part("word/document.xml")
	if err != nil {
		return 0, err
	}

	body := findBody(doc)
	if body == nil {
		return 0, errNoBody
	}

	count := 0

	for _, child := range body.ChildElements() {
		switch child.Tag {
		case "p":
			n := replaceParagraph(child, old, newText)
			count += n
		case tagTbl:
			// Process paragraphs inside table cells.
			n := replaceInTable(child, old, newText)
			count += n
		}
	}

	if count > 0 {
		session.MarkDirty("word/document.xml")
	}

	return count, nil
}

// replaceInTable recurses into table rows and cells to replace text in paragraphs.
func replaceInTable(tbl *etree.Element, old, newText string) int {
	count := 0

	for _, tr := range tbl.ChildElements() {
		if tr.Tag != "tr" {
			continue
		}

		for _, tc := range tr.ChildElements() {
			if tc.Tag != "tc" {
				continue
			}

			for _, p := range tc.ChildElements() {
				if p.Tag == "p" {
					count += replaceParagraph(p, old, newText)
				}
			}
		}
	}

	return count
}

// replaceParagraph handles text replacement within a single paragraph.
// It concatenates all run text, finds matches, and handles cross-run spans.
func replaceParagraph(p *etree.Element, old, newText string) int {
	// Collect runs with their text content and positions.
	type runInfo struct {
		elem     *etree.Element
		text     string
		startPos int // position in the concatenated paragraph text
	}

	var runs []runInfo

	pos := 0

	for _, child := range p.ChildElements() {
		if child.Tag != "r" {
			continue
		}

		text := runText(child)
		runs = append(runs, runInfo{elem: child, text: text, startPos: pos})
		pos += len(text)
	}

	if len(runs) == 0 {
		return 0
	}

	// Build full paragraph text.
	fullText := make([]byte, 0, pos)

	for _, r := range runs {
		fullText = append(fullText, r.text...)
	}

	// Find all match positions.
	matches := findAllOccurrences(string(fullText), old)
	if len(matches) == 0 {
		return 0
	}

	// Process matches in reverse order so positions remain valid.
	for i := len(matches) - 1; i >= 0; i-- {
		matchStart := matches[i]
		matchEnd := matchStart + len(old)

		// Find which runs this match spans.
		firstRunIdx := -1
		lastRunIdx := -1

		for j, r := range runs {
			runEnd := r.startPos + len(r.text)
			if firstRunIdx == -1 && matchStart < runEnd && matchStart >= r.startPos {
				firstRunIdx = j
			}

			if matchEnd <= runEnd && matchEnd > r.startPos {
				lastRunIdx = j

				break
			}

			if matchEnd > runEnd && j == len(runs)-1 {
				lastRunIdx = j
			}
		}

		if firstRunIdx == -1 || lastRunIdx == -1 {
			continue
		}

		if firstRunIdx == lastRunIdx {
			// Match is within a single run: simple string replacement.
			r := runs[firstRunIdx]
			localStart := matchStart - r.startPos
			localEnd := matchEnd - r.startPos
			replacedText := r.text[:localStart] + newText + r.text[localEnd:]
			setRunText(r.elem, replacedText)
			runs[firstRunIdx].text = replacedText
		} else {
			// Match spans multiple runs. Merge text into the first run, remove others.
			firstRun := runs[firstRunIdx]
			lastRun := runs[lastRunIdx]

			// Text before match in first run + replacement + text after match in last run.
			localStart := matchStart - firstRun.startPos
			localEnd := matchEnd - lastRun.startPos
			mergedText := firstRun.text[:localStart] + newText + lastRun.text[localEnd:]

			setRunText(firstRun.elem, mergedText)
			runs[firstRunIdx].text = mergedText

			// Remove the intermediate and last runs from the paragraph.
			for j := lastRunIdx; j > firstRunIdx; j-- {
				p.RemoveChild(runs[j].elem)
			}

			// Rebuild runs slice to reflect removals.
			runs = append(runs[:firstRunIdx+1], runs[lastRunIdx+1:]...)
		}
	}

	return len(matches)
}

// findAllOccurrences returns all starting indices of needle in haystack.
func findAllOccurrences(haystack, needle string) []int {
	var positions []int

	start := 0

	for {
		idx := strings.Index(haystack[start:], needle)
		if idx == -1 {
			break
		}

		positions = append(positions, start+idx)
		start += idx + len(needle)
	}

	return positions
}

// runText extracts the concatenated text from all w:t elements in a run.
func runText(r *etree.Element) string {
	var sb strings.Builder

	for _, child := range r.ChildElements() {
		if child.Tag == "t" {
			sb.WriteString(child.Text())
		}
	}

	return sb.String()
}

// setRunText sets the text of a run element. If multiple w:t elements exist,
// they are consolidated into one. The xml:space="preserve" attribute is set
// when the text has leading or trailing whitespace.
func setRunText(r *etree.Element, text string) {
	// Remove all existing w:t elements.
	var toRemove []*etree.Element

	for _, child := range r.ChildElements() {
		if child.Tag == "t" {
			toRemove = append(toRemove, child)
		}
	}

	for _, child := range toRemove {
		r.RemoveChild(child)
	}

	// Create a single w:t element with the new text.
	t := r.CreateElement("w:t")
	t.SetText(text)

	if len(text) > 0 && (text[0] == ' ' || text[len(text)-1] == ' ') {
		t.CreateAttr("xml:space", "preserve")
	}
}

// InsertParagraph inserts a new paragraph after a reference point.
// Reference format: "heading:Summary" (after heading containing "Summary")
//
//	"paragraph:5" (after 5th paragraph, 0-indexed)
func InsertParagraph(session *EditSession, after string, text string) error {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	refType, refValue, err := parseReference(after)
	if err != nil {
		return err
	}

	// Find the reference element.
	var refElem *etree.Element

	switch refType {
	case "heading":
		refElem = findHeadingParagraph(body, refValue)
		if refElem == nil {
			return fmt.Errorf("%w: %q", errHeadingNotFound, refValue)
		}
	case "paragraph":
		idx, err := strconv.Atoi(refValue)
		if err != nil {
			return fmt.Errorf("invalid paragraph index %q: %w", refValue, err)
		}

		refElem = findParagraphByIndex(body, idx)
		if refElem == nil {
			return fmt.Errorf("%w: %d", errParagraphOutOfRange, idx)
		}
	default:
		return fmt.Errorf("%w: %q", errUnknownRefType, refType)
	}

	// Build the new paragraph element.
	newPara := buildParagraph(text)

	// Insert after the reference element.
	insertAfter(body, refElem, newPara)

	session.MarkDirty("word/document.xml")

	return nil
}

// DeleteSection removes a section identified by its heading.
// Deletes from the heading paragraph through to (but not including) the next
// heading of same or higher level.
func DeleteSection(session *EditSession, headingName string) error {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	// Find the heading paragraph.
	headingElem := findHeadingParagraph(body, headingName)
	if headingElem == nil {
		return fmt.Errorf("%w: %q", errHeadingNotFound, headingName)
	}

	headingLevel := headingStyleLevel(paragraphStyle(headingElem))
	if headingLevel == 0 {
		// Treat Title as level 0 (highest).
		headingLevel = 0
	}

	// Collect elements to remove: from the heading to (but not including)
	// the next heading of same or higher level.
	var toRemove []*etree.Element
	toRemove = append(toRemove, headingElem)

	found := false

	for _, child := range body.ChildElements() {
		if !found {
			if child == headingElem {
				found = true
			}

			continue
		}

		// Stop at next heading of same or higher level.
		if child.Tag == "p" {
			style := paragraphStyle(child)
			level := headingStyleLevel(style)

			if level > 0 && level <= headingLevel {
				break
			}
		}

		toRemove = append(toRemove, child)
	}

	for _, elem := range toRemove {
		body.RemoveChild(elem)
	}

	session.MarkDirty("word/document.xml")

	return nil
}

// SetStyle changes the paragraph style of a specific paragraph.
func SetStyle(session *EditSession, paragraphIdx int, styleName string) error {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	p := findParagraphByIndex(body, paragraphIdx)
	if p == nil {
		return fmt.Errorf("%w: %d", errParagraphOutOfRange, paragraphIdx)
	}

	setParagraphStyle(p, styleName)

	session.MarkDirty("word/document.xml")

	return nil
}

// parseReference parses a reference string like "heading:Summary" or "paragraph:5".
func parseReference(ref string) (refType, refValue string, err error) {
	idx := strings.IndexByte(ref, ':')
	if idx == -1 {
		return "", "", fmt.Errorf("%w: %q", errInvalidReference, ref)
	}

	return ref[:idx], ref[idx+1:], nil
}

// findHeadingParagraph finds the first paragraph whose style is a heading type
// and whose text contains the given string.
func findHeadingParagraph(body *etree.Element, text string) *etree.Element {
	for _, child := range body.ChildElements() {
		if child.Tag != "p" {
			continue
		}

		style := paragraphStyle(child)
		if headingStyleLevel(style) == 0 && !strings.EqualFold(style, "title") {
			continue
		}

		pText := paragraphText(child)
		if strings.Contains(pText, text) {
			return child
		}
	}

	return nil
}

// findParagraphByIndex returns the paragraph at the given 0-based index
// (counting only w:p direct children of body).
func findParagraphByIndex(body *etree.Element, idx int) *etree.Element {
	count := 0

	for _, child := range body.ChildElements() {
		if child.Tag == "p" {
			if count == idx {
				return child
			}

			count++
		}
	}

	return nil
}

// headingStyleLevel returns the heading level for a style name.
// "Heading1" -> 1, "Heading2" -> 2, etc. "Title" -> 0 (special). Unknown -> 0.
func headingStyleLevel(style string) int {
	lower := strings.ToLower(style)
	if lower == "title" {
		return 0
	}

	if strings.HasPrefix(lower, "heading") {
		numStr := strings.TrimPrefix(lower, "heading")
		if n, err := strconv.Atoi(numStr); err == nil {
			return n
		}
	}

	return 0
}

// buildParagraph creates a new w:p element with a single run containing the given text.
func buildParagraph(text string) *etree.Element {
	p := etree.NewElement("w:p")
	r := p.CreateElement("w:r")
	t := r.CreateElement("w:t")
	t.SetText(text)

	if len(text) > 0 && (text[0] == ' ' || text[len(text)-1] == ' ') {
		t.CreateAttr("xml:space", "preserve")
	}

	return p
}

// insertAfter inserts newChild immediately after refChild within parent.
func insertAfter(parent, refChild, newChild *etree.Element) {
	// Find the position of refChild and insert after it.
	children := parent.ChildElements()
	refIdx := -1

	for i, child := range children {
		if child == refChild {
			refIdx = i

			break
		}
	}

	if refIdx == -1 || refIdx == len(children)-1 {
		// Append at end if not found or it's the last element.
		parent.AddChild(newChild)

		return
	}

	// Insert before the next sibling element.
	nextSibling := children[refIdx+1]
	parent.InsertChildAt(nextSibling.Index(), newChild)
}

// setParagraphStyle sets or replaces the paragraph style. Creates w:pPr and
// w:pStyle elements if they don't exist.
func setParagraphStyle(p *etree.Element, styleName string) {
	// Find or create w:pPr.
	var pPr *etree.Element

	for _, child := range p.ChildElements() {
		if child.Tag == "pPr" {
			pPr = child

			break
		}
	}

	if pPr == nil {
		// w:pPr should be the first child of w:p.
		pPr = etree.NewElement("w:pPr")
		if len(p.ChildElements()) == 0 {
			p.AddChild(pPr)
		} else {
			first := p.ChildElements()[0]
			p.InsertChildAt(first.Index(), pPr)
		}
	}

	// Find or create w:pStyle.
	var pStyle *etree.Element

	for _, child := range pPr.ChildElements() {
		if child.Tag == "pStyle" {
			pStyle = child

			break
		}
	}

	if pStyle == nil {
		pStyle = pPr.CreateElement("w:pStyle")
	}

	// Set the style value. Use w:val attribute.
	pStyle.RemoveAttr("val")
	pStyle.RemoveAttr("w:val")
	pStyle.CreateAttr("w:val", styleName)
}
