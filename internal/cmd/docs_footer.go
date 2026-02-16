package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/docs/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// DocsFooterCmd gets or sets the default footer of a Google Doc.
type DocsFooterCmd struct {
	DocID string `arg:"" name:"docId" help:"Document ID or URL"`
	Set   string `name:"set" help:"Text to set as the footer (creates or replaces)"`
}

func (c *DocsFooterCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}
	id = normalizeGoogleID(id)

	if c.Set != "" {
		return c.runSet(ctx, flags, u, account, id)
	}
	return c.runGet(ctx, u, account, id)
}

func (c *DocsFooterCmd) runGet(ctx context.Context, u *ui.UI, account, id string) error {
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}

	doc, err := svc.Documents.Get(id).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
		}
		return err
	}
	if doc == nil {
		return errors.New("doc not found")
	}

	footerID := ""
	if doc.DocumentStyle != nil {
		footerID = doc.DocumentStyle.DefaultFooterId
	}

	if footerID == "" {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"docId": id,
				"text":  "no footer",
			})
		}
		u.Out().Printf("no footer")
		return nil
	}

	footer, ok := doc.Footers[footerID]
	if !ok {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"docId": id,
				"text":  "no footer",
			})
		}
		u.Out().Printf("no footer")
		return nil
	}

	text := extractFooterText(&footer)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"docId":    id,
			"footerId": footerID,
			"text":     text,
		})
	}

	u.Out().Printf("footerId\t%s", footerID)
	u.Out().Printf("text\t%s", text)
	return nil
}

func (c *DocsFooterCmd) runSet(ctx context.Context, flags *RootFlags, u *ui.UI, account, id string) error {
	if dryRunErr := dryRunExit(ctx, flags, "docs.footer.set", map[string]any{
		"docId": id,
		"text":  c.Set,
	}); dryRunErr != nil {
		return dryRunErr
	}

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}

	// Fetch document to check for existing footer.
	doc, err := svc.Documents.Get(id).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
		}
		return err
	}
	if doc == nil {
		return errors.New("doc not found")
	}

	footerID := ""
	if doc.DocumentStyle != nil {
		footerID = doc.DocumentStyle.DefaultFooterId
	}

	if footerID == "" {
		// No default footer exists -- create one and insert text.
		return c.createAndSetFooter(ctx, svc, u, id)
	}

	// Footer exists -- clear its content and insert new text.
	return c.updateExistingFooter(ctx, svc, u, id, footerID, doc)
}

func (c *DocsFooterCmd) createAndSetFooter(ctx context.Context, svc *docs.Service, u *ui.UI, id string) error {
	var requests []*docs.Request

	// Step 1: Create a default footer.
	requests = append(requests, &docs.Request{
		CreateFooter: &docs.CreateFooterRequest{
			Type: "DEFAULT",
		},
	})

	result, err := svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("creating footer: %w", err)
	}

	// Get the new footer ID from the response.
	newFooterID := ""
	if len(result.Replies) > 0 && result.Replies[0].CreateFooter != nil {
		newFooterID = result.Replies[0].CreateFooter.FooterId
	}
	if newFooterID == "" {
		return errors.New("failed to get new footer ID from API response")
	}

	// Step 2: Insert text into the new footer.
	_, err = svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Text: c.Set,
					EndOfSegmentLocation: &docs.EndOfSegmentLocation{
						SegmentId: newFooterID,
					},
				},
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("inserting footer text: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"success":  true,
			"docId":    id,
			"footerId": newFooterID,
			"text":     c.Set,
			"action":   "created",
		})
	}

	u.Out().Printf("Created footer in document %s", id)
	u.Out().Printf("footerId\t%s", newFooterID)
	u.Out().Printf("text\t%s", c.Set)
	return nil
}

func (c *DocsFooterCmd) updateExistingFooter(ctx context.Context, svc *docs.Service, u *ui.UI, id, footerID string, doc *docs.Document) error {
	var requests []*docs.Request

	// Delete existing content in the footer (if any).
	if footer, ok := doc.Footers[footerID]; ok {
		endIdx := footerContentEndIndex(&footer)
		if endIdx > 1 {
			requests = append(requests, &docs.Request{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{
						SegmentId:  footerID,
						StartIndex: 0,
						EndIndex:   endIdx - 1,
					},
				},
			})
		}
	}

	// Insert new text.
	requests = append(requests, &docs.Request{
		InsertText: &docs.InsertTextRequest{
			Text: c.Set,
			EndOfSegmentLocation: &docs.EndOfSegmentLocation{
				SegmentId: footerID,
			},
		},
	})

	_, err := svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("updating footer: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"success":  true,
			"docId":    id,
			"footerId": footerID,
			"text":     c.Set,
			"action":   "updated",
		})
	}

	u.Out().Printf("Updated footer in document %s", id)
	u.Out().Printf("footerId\t%s", footerID)
	u.Out().Printf("text\t%s", c.Set)
	return nil
}

// extractFooterText extracts plain text from a footer's structural elements.
func extractFooterText(footer *docs.Footer) string {
	if footer == nil {
		return ""
	}
	var buf bytes.Buffer
	for _, el := range footer.Content {
		if el == nil || el.Paragraph == nil {
			continue
		}
		for _, pe := range el.Paragraph.Elements {
			if pe.TextRun == nil {
				continue
			}
			buf.WriteString(pe.TextRun.Content)
		}
	}
	return strings.TrimRight(buf.String(), "\n")
}

// footerContentEndIndex returns the end index of the last content element in a footer.
func footerContentEndIndex(footer *docs.Footer) int64 {
	if footer == nil || len(footer.Content) == 0 {
		return 0
	}
	last := footer.Content[len(footer.Content)-1]
	if last == nil {
		return 0
	}
	return last.EndIndex
}
