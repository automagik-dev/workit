package docx

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/beevik/etree"
)

// placeholderRe matches {{PLACEHOLDER}} patterns in concatenated text.
var placeholderRe = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// FillTemplate opens a template DOCX, finds {{PLACEHOLDER}} patterns in the XML,
// and replaces them with provided values. All non-content ZIP entries (logos, backgrounds,
// headers, footers, fonts, page layout) are preserved byte-for-byte.
//
// Handles Word's run fragmentation: {{TITLE}} may be split across multiple <w:r> elements.
// Before replacement, fragments are merged so the placeholder text is in a single run.
//
// The replacement preserves the placeholder's <w:rPr> formatting (font, size, bold, etc.).
func FillTemplate(session *EditSession, values map[string]string) (int, error) {
	totalCount := 0

	// Process document.xml
	n, err := fillTemplatePart(session, "word/document.xml", values)
	if err != nil {
		return 0, err
	}
	totalCount += n

	// Also scan header/footer XML files.
	for _, partName := range session.ListParts() {
		if isHeaderFooterPart(partName) {
			n, err := fillTemplatePart(session, partName, values)
			if err != nil {
				return totalCount, fmt.Errorf("fill %s: %w", partName, err)
			}
			totalCount += n
		}
	}

	return totalCount, nil
}

// isHeaderFooterPart checks if a part name is a header or footer XML file.
func isHeaderFooterPart(name string) bool {
	return (strings.HasPrefix(name, "word/header") || strings.HasPrefix(name, "word/footer")) &&
		strings.HasSuffix(name, ".xml")
}

// fillTemplatePart fills placeholders in a single XML part.
func fillTemplatePart(session *EditSession, partName string, values map[string]string) (int, error) {
	doc, err := session.Part(partName)
	if err != nil {
		return 0, err
	}

	root := doc.Root()
	if root == nil {
		return 0, nil
	}

	count := fillElementRecursive(root, values)
	if count > 0 {
		session.MarkDirty(partName)
	}

	return count, nil
}

// fillElementRecursive finds paragraphs anywhere in the element tree and fills placeholders.
func fillElementRecursive(elem *etree.Element, values map[string]string) int {
	count := 0

	for _, child := range elem.ChildElements() {
		switch child.Tag {
		case "p":
			count += fillParagraphPlaceholders(child, values)
		case "tbl":
			count += fillElementRecursive(child, values)
		case "tr":
			count += fillElementRecursive(child, values)
		case "tc":
			count += fillElementRecursive(child, values)
		case "body":
			count += fillElementRecursive(child, values)
		case "hdr", "ftr":
			count += fillElementRecursive(child, values)
		default:
			// Recurse into any other structural elements.
			count += fillElementRecursive(child, values)
		}
	}

	return count
}

// fillParagraphPlaceholders handles placeholder replacement in a single paragraph.
// It merges fragmented runs that contain parts of the same placeholder, then replaces.
func fillParagraphPlaceholders(p *etree.Element, values map[string]string) int {
	// Step 1: Collect runs with text content and positions.
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

	// Step 2: Build full paragraph text.
	var fullText strings.Builder
	for _, r := range runs {
		fullText.WriteString(r.text)
	}
	paraText := fullText.String()

	// Step 3: Find all placeholder matches.
	matches := placeholderRe.FindAllStringIndex(paraText, -1)
	if len(matches) == 0 {
		return 0
	}

	count := 0

	// Step 4: Process matches in reverse order so positions remain valid.
	for i := len(matches) - 1; i >= 0; i-- {
		matchStart := matches[i][0]
		matchEnd := matches[i][1]
		placeholder := paraText[matchStart:matchEnd]

		// Extract the key name (strip {{ and }}).
		key := placeholder[2 : len(placeholder)-2]

		replacement, ok := values[key]
		if !ok {
			// No value provided for this placeholder; skip it.
			continue
		}

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

			if j == len(runs)-1 {
				lastRunIdx = j
			}
		}

		if firstRunIdx == -1 || lastRunIdx == -1 {
			continue
		}

		if firstRunIdx == lastRunIdx {
			// Match within a single run.
			r := runs[firstRunIdx]
			localStart := matchStart - r.startPos
			localEnd := matchEnd - r.startPos
			newText := r.text[:localStart] + replacement + r.text[localEnd:]
			setRunText(r.elem, newText)
			runs[firstRunIdx].text = newText
		} else {
			// Match spans multiple runs. Merge into the first run, remove others.
			firstRun := runs[firstRunIdx]
			lastRun := runs[lastRunIdx]

			localStart := matchStart - firstRun.startPos
			localEnd := matchEnd - lastRun.startPos
			mergedText := firstRun.text[:localStart] + replacement + lastRun.text[localEnd:]

			// Preserve the formatting (w:rPr) of the first run.
			setRunText(firstRun.elem, mergedText)
			runs[firstRunIdx].text = mergedText

			// Remove intermediate and last runs.
			for j := lastRunIdx; j > firstRunIdx; j-- {
				p.RemoveChild(runs[j].elem)
			}

			// Rebuild runs slice.
			runs = append(runs[:firstRunIdx+1], runs[lastRunIdx+1:]...)
		}

		count++
	}

	return count
}

// InspectTemplate scans a DOCX for all {{PLACEHOLDER}} patterns and returns their names.
func InspectTemplate(session *EditSession) ([]string, error) {
	seen := make(map[string]bool)
	var names []string

	// Scan document.xml
	if err := inspectPart(session, "word/document.xml", seen, &names); err != nil {
		return nil, err
	}

	// Scan header/footer XML files.
	for _, partName := range session.ListParts() {
		if isHeaderFooterPart(partName) {
			if err := inspectPart(session, partName, seen, &names); err != nil {
				return nil, fmt.Errorf("inspect %s: %w", partName, err)
			}
		}
	}

	return names, nil
}

// inspectPart scans a single XML part for placeholder names.
func inspectPart(session *EditSession, partName string, seen map[string]bool, names *[]string) error {
	doc, err := session.Part(partName)
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return nil
	}

	inspectElementRecursive(root, seen, names)

	return nil
}

// inspectElementRecursive finds paragraphs and extracts placeholder names.
func inspectElementRecursive(elem *etree.Element, seen map[string]bool, names *[]string) {
	for _, child := range elem.ChildElements() {
		switch child.Tag {
		case "p":
			extractPlaceholderNames(child, seen, names)
		default:
			inspectElementRecursive(child, seen, names)
		}
	}
}

// extractPlaceholderNames extracts placeholder names from a paragraph,
// handling run fragmentation by concatenating all run text first.
func extractPlaceholderNames(p *etree.Element, seen map[string]bool, names *[]string) {
	// Concatenate all run text in the paragraph.
	var sb strings.Builder

	for _, child := range p.ChildElements() {
		if child.Tag == "r" {
			sb.WriteString(runText(child))
		}
	}

	paraText := sb.String()

	matches := placeholderRe.FindAllStringSubmatch(paraText, -1)
	for _, m := range matches {
		key := m[1]
		if !seen[key] {
			seen[key] = true
			*names = append(*names, key)
		}
	}
}
