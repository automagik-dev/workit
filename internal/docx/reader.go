package docx

import (
	"strings"

	"github.com/beevik/etree"
)

const (
	// OOXML namespace for wordprocessingml.
	nsW = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
)

// DocumentStructure is a structured representation of a DOCX document's content.
type DocumentStructure struct {
	Paragraphs []ParagraphInfo `json:"paragraphs"`
	Tables     []TableInfo     `json:"tables"`
}

// ParagraphInfo describes a single paragraph.
type ParagraphInfo struct {
	Index int    `json:"index"`
	Style string `json:"style"`
	Text  string `json:"text"`
}

// TableInfo describes a table's dimensions.
type TableInfo struct {
	Index int `json:"index"`
	Rows  int `json:"rows"`
	Cols  int `json:"cols"`
}

// ReadAsMarkdown extracts document content as markdown text.
//   - Headings become # / ## / ### based on style
//   - Paragraphs become text blocks separated by blank lines
//   - Tables become markdown tables
//   - Images become [image: filename] placeholders
func ReadAsMarkdown(session *EditSession) (string, error) {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return "", err
	}

	body := findBody(doc)
	if body == nil {
		return "", errNoBody
	}

	var sb strings.Builder

	for _, child := range body.ChildElements() {
		tag := child.Tag
		if child.Space != "" {
			tag = child.Tag // etree strips the prefix; we match on local name
		}

		switch tag {
		case "p":
			text := paragraphText(child)
			style := paragraphStyle(child)

			md := formatParagraphMarkdown(text, style)
			if md != "" {
				sb.WriteString(md)
				sb.WriteString("\n\n")
			}
		case "tbl":
			md := formatTableMarkdown(child)
			if md != "" {
				sb.WriteString(md)
				sb.WriteString("\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n", nil
}

// ReadStructure extracts a structured JSON-friendly representation of the document.
func ReadStructure(session *EditSession) (*DocumentStructure, error) {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return nil, err
	}

	body := findBody(doc)
	if body == nil {
		return nil, errNoBody
	}

	ds := &DocumentStructure{}
	pIdx := 0
	tIdx := 0

	for _, child := range body.ChildElements() {
		switch child.Tag {
		case "p":
			text := paragraphText(child)
			style := paragraphStyle(child)
			ds.Paragraphs = append(ds.Paragraphs, ParagraphInfo{
				Index: pIdx,
				Style: style,
				Text:  text,
			})
			pIdx++
		case "tbl":
			rows, cols := tableDimensions(child)
			ds.Tables = append(ds.Tables, TableInfo{
				Index: tIdx,
				Rows:  rows,
				Cols:  cols,
			})
			tIdx++
		}
	}

	return ds, nil
}

// findBody locates the w:body element in the document.
func findBody(doc *etree.Document) *etree.Element {
	root := doc.Root()
	if root == nil {
		return nil
	}
	// The root is w:document; body is a direct child.
	for _, child := range root.ChildElements() {
		if child.Tag == "body" {
			return child
		}
	}

	return nil
}

// paragraphText extracts all text from w:r/w:t elements within a paragraph.
func paragraphText(p *etree.Element) string {
	var sb strings.Builder

	for _, r := range p.ChildElements() {
		if r.Tag == "r" {
			for _, t := range r.ChildElements() {
				if t.Tag == "t" {
					sb.WriteString(t.Text())
				}
			}
		}
	}

	return sb.String()
}

// paragraphStyle extracts the style name from w:pPr/w:pStyle[@w:val].
func paragraphStyle(p *etree.Element) string {
	for _, child := range p.ChildElements() {
		if child.Tag == "pPr" {
			for _, prop := range child.ChildElements() {
				if prop.Tag == "pStyle" {
					val := prop.SelectAttr("val")
					if val == nil {
						// Try with namespace prefix.
						val = prop.SelectAttr("w:val")
					}

					if val != nil {
						return val.Value
					}
				}
			}
		}
	}

	return ""
}

// formatParagraphMarkdown converts a paragraph to markdown based on its style.
func formatParagraphMarkdown(text, style string) string {
	if text == "" {
		return ""
	}

	switch strings.ToLower(style) {
	case "title":
		return "# " + text
	case "heading1":
		return "## " + text
	case "heading2":
		return "### " + text
	case "heading3":
		return "#### " + text
	case "heading4":
		return "##### " + text
	case "heading5", "heading6":
		return "###### " + text
	default:
		return text
	}
}

// formatTableMarkdown renders a w:tbl element as a markdown table.
func formatTableMarkdown(tbl *etree.Element) string {
	var rows [][]string

	for _, tr := range tbl.ChildElements() {
		if tr.Tag != "tr" {
			continue
		}
		var cells []string

		for _, tc := range tr.ChildElements() {
			if tc.Tag != "tc" {
				continue
			}

			// A cell can contain multiple paragraphs; join with space.
			var cellText []string

			for _, p := range tc.ChildElements() {
				if p.Tag == "p" {
					t := paragraphText(p)
					if t != "" {
						cellText = append(cellText, t)
					}
				}
			}

			cells = append(cells, strings.Join(cellText, " "))
		}

		rows = append(rows, cells)
	}

	if len(rows) == 0 {
		return ""
	}

	// Normalize column count to the maximum.
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	for i := range rows {
		for len(rows[i]) < maxCols {
			rows[i] = append(rows[i], "")
		}
	}

	var sb strings.Builder

	// Header row.
	sb.WriteString("| ")
	sb.WriteString(strings.Join(rows[0], " | "))
	sb.WriteString(" |\n")

	// Separator.
	sep := make([]string, maxCols)
	for i := range sep {
		sep[i] = "---"
	}

	sb.WriteString("| ")
	sb.WriteString(strings.Join(sep, " | "))
	sb.WriteString(" |\n")

	// Data rows.
	for _, row := range rows[1:] {
		sb.WriteString("| ")
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteString(" |\n")
	}

	return sb.String()
}

// tableDimensions returns the number of rows and the maximum number of columns.
func tableDimensions(tbl *etree.Element) (rows, cols int) {
	maxCols := 0

	for _, tr := range tbl.ChildElements() {
		if tr.Tag != "tr" {
			continue
		}

		rows++

		cellCount := 0

		for _, tc := range tr.ChildElements() {
			if tc.Tag == "tc" {
				cellCount++
			}
		}

		if cellCount > maxCols {
			maxCols = cellCount
		}
	}

	return rows, maxCols
}
