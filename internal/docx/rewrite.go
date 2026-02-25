package docx

import (
	"strings"

	"github.com/beevik/etree"
)

// Rewrite replaces all body content in a DOCX with new content from markdown.
// Preserves: styles, template, headers, footers, page layout.
// Replaces: all paragraphs and tables in w:body.
func Rewrite(session *EditSession, markdownContent string) error {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	// Save the final w:sectPr (section properties = page layout) if present.
	var sectPr *etree.Element

	children := body.ChildElements()
	if len(children) > 0 {
		last := children[len(children)-1]
		if last.Tag == "sectPr" {
			sectPr = last.Copy()
		}
	}

	// Clear all existing children from body.
	for _, child := range body.ChildElements() {
		body.RemoveChild(child)
	}

	// Parse markdown into paragraphs and create new w:p elements.
	blocks := splitMarkdownBlocks(markdownContent)
	for _, block := range blocks {
		if block == "" {
			continue
		}

		style, text := parseMarkdownBlock(block)
		p := buildStyledParagraph(text, style)
		body.AddChild(p)
	}

	// Restore section properties at the end.
	if sectPr != nil {
		body.AddChild(sectPr)
	}

	session.MarkDirty("word/document.xml")

	return nil
}

// splitMarkdownBlocks splits markdown content on double newlines into blocks.
func splitMarkdownBlocks(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	// Split on double newlines (paragraph separators).
	blocks := strings.Split(content, "\n\n")
	var result []string

	for _, b := range blocks {
		trimmed := strings.TrimSpace(b)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// parseMarkdownBlock detects the heading level and extracts the text.
// Returns (style, text) where style is "Title", "Heading1", "Heading2", etc.
// or "" for normal paragraphs.
func parseMarkdownBlock(block string) (string, string) {
	if !strings.HasPrefix(block, "#") {
		return "", block
	}

	// Count heading level.
	level := 0

	for _, ch := range block {
		if ch == '#' {
			level++
		} else {
			break
		}
	}

	if level > 6 {
		level = 6
	}

	text := strings.TrimSpace(block[level:])

	switch level {
	case 1:
		return "Title", text
	case 2:
		return "Heading1", text
	case 3:
		return "Heading2", text
	case 4:
		return "Heading3", text
	case 5:
		return "Heading4", text
	case 6:
		return "Heading5", text
	default:
		return "", text
	}
}

// buildStyledParagraph creates a w:p element with the given text and optional style.
func buildStyledParagraph(text, style string) *etree.Element {
	p := buildParagraph(text)
	if style != "" {
		setParagraphStyle(p, style)
	}

	return p
}
