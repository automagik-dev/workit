package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/docs/v1"

	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

// DocElement represents a structural element in a Google Doc for the structure command.
type DocElement struct {
	Type           string `json:"type"`
	StartIndex     int64  `json:"startIndex"`
	EndIndex       int64  `json:"endIndex"`
	Style          string `json:"style,omitempty"`
	ContentSummary string `json:"contentSummary,omitempty"`
}

const (
	docElementTypeParagraph = "paragraph"
	docElementTypeUnknown   = "unknown"
)

// DocsStructureCmd inspects the element tree of a Google Doc.
type DocsStructureCmd struct {
	DocID string `arg:"" name:"docId" help:"Document ID or URL"`
}

func (c *DocsStructureCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	id := normalizeGoogleID(strings.TrimSpace(c.DocID))
	if id == "" {
		return usage("empty docId")
	}

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}

	doc, err := svc.Documents.Get(id).
		Context(ctx).
		Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
		}
		return err
	}
	if doc == nil {
		return errors.New("doc not found")
	}

	elements := extractDocStructure(doc)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"documentId": doc.DocumentId,
			"title":      doc.Title,
			"elements":   elements,
		})
	}

	u.Out().Printf("TYPE\tSTART\tEND\tSTYLE\tCONTENT")
	for _, el := range elements {
		u.Out().Printf("%s\t%d\t%d\t%s\t%s",
			el.Type, el.StartIndex, el.EndIndex,
			sanitizeTSVField(el.Style), sanitizeTSVField(el.ContentSummary))
	}
	return nil
}

// extractDocStructure walks doc.Body.Content and extracts structural elements
// with their type, indices, style, and a content summary.
func extractDocStructure(doc *docs.Document) []DocElement {
	if doc == nil || doc.Body == nil {
		return nil
	}

	var elements []DocElement
	for _, el := range doc.Body.Content {
		if el == nil {
			continue
		}
		elements = append(elements, classifyStructuralElement(el))
	}
	return elements
}

// classifyStructuralElement determines the type and extracts metadata from a
// single structural element.
func classifyStructuralElement(el *docs.StructuralElement) DocElement {
	elem := DocElement{
		StartIndex: el.StartIndex,
		EndIndex:   el.EndIndex,
	}

	switch {
	case el.Paragraph != nil:
		elem.Type = docElementTypeParagraph
		if el.Paragraph.ParagraphStyle != nil {
			elem.Style = el.Paragraph.ParagraphStyle.NamedStyleType
		}
		elem.ContentSummary = paragraphContentSummary(el.Paragraph)

	case el.Table != nil:
		elem.Type = "table"
		rows := el.Table.Rows
		cols := el.Table.Columns
		elem.ContentSummary = fmt.Sprintf("Table %dx%d", rows, cols)

	case el.SectionBreak != nil:
		elem.Type = "sectionBreak"

	case el.TableOfContents != nil:
		elem.Type = "tableOfContents"

	default:
		elem.Type = docElementTypeUnknown
	}

	return elem
}

// sanitizeTSVField replaces tab characters with spaces and escapes newlines/carriage
// returns so the value stays within a single TSV column.
func sanitizeTSVField(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// paragraphContentSummary extracts the first ~80 characters of text from a paragraph.
func paragraphContentSummary(p *docs.Paragraph) string {
	const maxSummaryRunes = 80

	summaryRunes := make([]rune, 0, maxSummaryRunes)
	truncated := false

	for _, pe := range p.Elements {
		if pe.TextRun == nil {
			continue
		}

		for _, r := range pe.TextRun.Content {
			if len(summaryRunes) >= maxSummaryRunes {
				truncated = true
				break
			}
			summaryRunes = append(summaryRunes, r)
		}
		if truncated {
			break
		}
	}

	summary := strings.TrimRight(string(summaryRunes), "\n")
	if truncated {
		summary += "..."
	}
	return summary
}
