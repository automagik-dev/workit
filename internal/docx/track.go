package docx

import (
	"strconv"
	"time"

	"github.com/beevik/etree"
)

// Constants for tracked-change element tags.
const (
	tagDel = "del"
	tagIns = "ins"
)

// TrackReplace finds all occurrences of old text and wraps the changes as
// OOXML tracked modifications (<w:del>/<w:ins>). Each replacement records the
// author and a timestamp. Only the matched text is wrapped; surrounding text
// in the same run preserves its original formatting.
// Returns the number of replacements made.
func TrackReplace(session *EditSession, old, newText, author string) (int, error) {
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

	// Find max existing tracked-change ID so we can allocate new ones.
	nextID := maxTrackedChangeID(body) + 1

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	count := 0

	for _, child := range body.ChildElements() {
		switch child.Tag {
		case "p":
			n := trackReplaceParagraph(child, old, newText, author, now, &nextID)
			count += n
		case tagTbl:
			n := trackReplaceInTable(child, old, newText, author, now, &nextID)
			count += n
		}
	}

	if count > 0 {
		session.MarkDirty("word/document.xml")
	}

	return count, nil
}

// AcceptChanges accepts all tracked changes in the document.
// It removes <w:del> elements entirely and unwraps <w:ins> (keeping their content).
func AcceptChanges(session *EditSession) error {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	acceptInElement(body)
	session.MarkDirty("word/document.xml")

	return nil
}

// RejectChanges rejects all tracked changes in the document.
// It removes <w:ins> elements entirely and unwraps <w:del> (converting
// <w:delText> to <w:t> and keeping their content).
func RejectChanges(session *EditSession) error {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	rejectInElement(body)
	session.MarkDirty("word/document.xml")

	return nil
}

// trackReplaceInTable recurses into table rows/cells.
func trackReplaceInTable(tbl *etree.Element, old, newText, author, date string, nextID *int) int {
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
					count += trackReplaceParagraph(p, old, newText, author, date, nextID)
				}
			}
		}
	}

	return count
}

// trackReplaceParagraph handles tracked replacement within a single paragraph.
func trackReplaceParagraph(p *etree.Element, old, newText, author, date string, nextID *int) int {
	type runInfo struct {
		elem     *etree.Element
		text     string
		startPos int
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

	fullText := make([]byte, 0, pos)

	for _, r := range runs {
		fullText = append(fullText, r.text...)
	}

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

		// Extract the formatting (w:rPr) from the first matching run.
		rPr := cloneRunProperties(runs[firstRunIdx].elem)

		// Determine text before, the match, and text after across runs.
		firstRun := runs[firstRunIdx]
		lastRun := runs[lastRunIdx]
		localStart := matchStart - firstRun.startPos
		localEnd := matchEnd - lastRun.startPos

		textBefore := firstRun.text[:localStart]
		textAfter := lastRun.text[localEnd:]

		// Build the <w:del> element.
		delID := *nextID
		*nextID++
		del := etree.NewElement("w:del")
		del.CreateAttr("w:id", strconv.Itoa(delID))
		del.CreateAttr("w:author", author)
		del.CreateAttr("w:date", date)
		delRun := del.CreateElement("w:r")

		if rPr != nil {
			delRun.AddChild(rPr.Copy())
		}

		delText := delRun.CreateElement("w:delText")
		delText.CreateAttr("xml:space", "preserve")
		delText.SetText(old)

		// Build the <w:ins> element.
		insID := *nextID
		*nextID++
		ins := etree.NewElement("w:ins")
		ins.CreateAttr("w:id", strconv.Itoa(insID))
		ins.CreateAttr("w:author", author)
		ins.CreateAttr("w:date", date)
		insRun := ins.CreateElement("w:r")

		if rPr != nil {
			insRun.AddChild(rPr.Copy())
		}

		insT := insRun.CreateElement("w:t")
		insT.CreateAttr("xml:space", "preserve")
		insT.SetText(newText)

		// Now replace the runs. We insert:
		//   [textBefore run (if non-empty)] [del] [ins] [textAfter run (if non-empty)]
		// and remove the original runs that the match spanned.

		// Find insertion point: the first matching run's position.
		refElem := firstRun.elem

		// Build replacement nodes.
		var newNodes []*etree.Element

		if textBefore != "" {
			beforeRun := firstRun.elem.Copy()
			setRunText(beforeRun, textBefore)
			newNodes = append(newNodes, beforeRun)
		}

		newNodes = append(newNodes, del, ins)

		if textAfter != "" {
			afterRun := lastRun.elem.Copy()
			setRunText(afterRun, textAfter)
			newNodes = append(newNodes, afterRun)
		}

		// Insert new nodes before the first matching run.
		refIndex := refElem.Index()

		for k, node := range newNodes {
			p.InsertChildAt(refIndex+k, node)
		}

		// Remove the original runs that were part of the match.
		for j := firstRunIdx; j <= lastRunIdx; j++ {
			p.RemoveChild(runs[j].elem)
		}

		// Rebuild runs slice (we don't need to process further; reverse order handles it).
	}

	return len(matches)
}

// acceptInElement recursively processes an element, accepting tracked changes.
func acceptInElement(elem *etree.Element) {
	// Process all child elements. We iterate over children, collecting
	// modifications, then applying them.
	for {
		changed := false

		for _, child := range elem.ChildElements() {
			switch child.Tag {
			case tagDel:
				// Accept: remove the entire <w:del> element.
				elem.RemoveChild(child)
				changed = true
			case tagIns:
				// Accept: unwrap <w:ins> -- move its children into the parent.
				unwrapElement(elem, child)
				changed = true
			default:
				// Recurse into nested elements (paragraphs inside tables, etc.)
				acceptInElement(child)
			}
		}

		if !changed {
			break
		}
	}
}

// rejectInElement recursively processes an element, rejecting tracked changes.
func rejectInElement(elem *etree.Element) {
	for {
		changed := false

		for _, child := range elem.ChildElements() {
			switch child.Tag {
			case tagIns:
				// Reject: remove the entire <w:ins> element.
				elem.RemoveChild(child)
				changed = true
			case tagDel:
				// Reject: unwrap <w:del>, converting <w:delText> to <w:t>.
				convertDelTextToT(child)
				unwrapElement(elem, child)
				changed = true
			default:
				rejectInElement(child)
			}
		}

		if !changed {
			break
		}
	}
}

// unwrapElement replaces `wrapper` in `parent` with wrapper's children.
func unwrapElement(parent, wrapper *etree.Element) {
	idx := wrapper.Index()
	children := append([]*etree.Element{}, wrapper.ChildElements()...)

	// Remove the wrapper first.
	parent.RemoveChild(wrapper)

	// Insert the children at the wrapper's former position.
	for k, c := range children {
		// Detach from old parent.
		wrapper.RemoveChild(c)
		parent.InsertChildAt(idx+k, c)
	}
}

// convertDelTextToT converts all <w:delText> elements inside elem to <w:t>.
func convertDelTextToT(elem *etree.Element) {
	for _, child := range elem.ChildElements() {
		if child.Tag == "delText" {
			child.Tag = "t"
			// Namespace prefix (Space) is inherited from the parent; no change needed.
		}

		convertDelTextToT(child)
	}
}

// maxTrackedChangeID scans an element tree for w:id attributes on w:del/w:ins
// and returns the highest value found.
func maxTrackedChangeID(elem *etree.Element) int {
	maxID := 0

	for _, child := range elem.ChildElements() {
		if child.Tag == tagDel || child.Tag == tagIns {
			if attr := child.SelectAttr("w:id"); attr != nil {
				if id, err := strconv.Atoi(attr.Value); err == nil && id > maxID {
					maxID = id
				}
			}

			// Also check plain "id" for namespace-stripped parsing.
			if attr := child.SelectAttr("id"); attr != nil {
				if id, err := strconv.Atoi(attr.Value); err == nil && id > maxID {
					maxID = id
				}
			}
		}

		childMax := maxTrackedChangeID(child)
		if childMax > maxID {
			maxID = childMax
		}
	}

	return maxID
}

// cloneRunProperties returns a copy of the w:rPr element from a run, or nil if none.
func cloneRunProperties(r *etree.Element) *etree.Element {
	for _, child := range r.ChildElements() {
		if child.Tag == "rPr" {
			return child.Copy()
		}
	}

	return nil
}
