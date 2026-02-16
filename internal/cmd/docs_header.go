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

// DocsHeaderCmd gets or sets the default header of a Google Doc.
type DocsHeaderCmd struct {
	DocID string `arg:"" name:"docId" help:"Document ID or URL"`
	Set   string `name:"set" help:"Text to set as the header (creates or replaces)"`
}

func (c *DocsHeaderCmd) Run(ctx context.Context, flags *RootFlags) error {
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

func (c *DocsHeaderCmd) runGet(ctx context.Context, u *ui.UI, account, id string) error {
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

	headerID := ""
	if doc.DocumentStyle != nil {
		headerID = doc.DocumentStyle.DefaultHeaderId
	}

	if headerID == "" {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"docId": id,
				"text":  "no header",
			})
		}
		u.Out().Printf("no header")
		return nil
	}

	header, ok := doc.Headers[headerID]
	if !ok {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"docId": id,
				"text":  "no header",
			})
		}
		u.Out().Printf("no header")
		return nil
	}

	text := extractHeaderText(&header)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"docId":    id,
			"headerId": headerID,
			"text":     text,
		})
	}

	u.Out().Printf("headerId\t%s", headerID)
	u.Out().Printf("text\t%s", text)
	return nil
}

func (c *DocsHeaderCmd) runSet(ctx context.Context, flags *RootFlags, u *ui.UI, account, id string) error {
	if dryRunErr := dryRunExit(ctx, flags, "docs.header.set", map[string]any{
		"docId": id,
		"text":  c.Set,
	}); dryRunErr != nil {
		return dryRunErr
	}

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}

	// Fetch document to check for existing header.
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

	headerID := ""
	if doc.DocumentStyle != nil {
		headerID = doc.DocumentStyle.DefaultHeaderId
	}

	if headerID == "" {
		// No default header exists -- create one and insert text.
		return c.createAndSetHeader(ctx, svc, u, id)
	}

	// Header exists -- clear its content and insert new text.
	return c.updateExistingHeader(ctx, svc, u, id, headerID, doc)
}

func (c *DocsHeaderCmd) createAndSetHeader(ctx context.Context, svc *docs.Service, u *ui.UI, id string) error {
	var requests []*docs.Request

	// Step 1: Create a default header.
	requests = append(requests, &docs.Request{
		CreateHeader: &docs.CreateHeaderRequest{
			Type: "DEFAULT",
		},
	})

	result, err := svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("creating header: %w", err)
	}

	// Get the new header ID from the response.
	newHeaderID := ""
	if len(result.Replies) > 0 && result.Replies[0].CreateHeader != nil {
		newHeaderID = result.Replies[0].CreateHeader.HeaderId
	}
	if newHeaderID == "" {
		return errors.New("failed to get new header ID from API response")
	}

	// Step 2: Insert text into the new header.
	_, err = svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Text: c.Set,
					EndOfSegmentLocation: &docs.EndOfSegmentLocation{
						SegmentId: newHeaderID,
					},
				},
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("inserting header text: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"success":  true,
			"docId":    id,
			"headerId": newHeaderID,
			"text":     c.Set,
			"action":   "created",
		})
	}

	u.Out().Printf("Created header in document %s", id)
	u.Out().Printf("headerId\t%s", newHeaderID)
	u.Out().Printf("text\t%s", c.Set)
	return nil
}

func (c *DocsHeaderCmd) updateExistingHeader(ctx context.Context, svc *docs.Service, u *ui.UI, id, headerID string, doc *docs.Document) error {
	var requests []*docs.Request

	// Delete existing content in the header (if any).
	if header, ok := doc.Headers[headerID]; ok {
		endIdx := headerContentEndIndex(&header)
		if endIdx > 1 {
			requests = append(requests, &docs.Request{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{
						SegmentId:  headerID,
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
				SegmentId: headerID,
			},
		},
	})

	_, err := svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("updating header: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"success":  true,
			"docId":    id,
			"headerId": headerID,
			"text":     c.Set,
			"action":   "updated",
		})
	}

	u.Out().Printf("Updated header in document %s", id)
	u.Out().Printf("headerId\t%s", headerID)
	u.Out().Printf("text\t%s", c.Set)
	return nil
}

// extractHeaderText extracts plain text from a header's structural elements.
func extractHeaderText(header *docs.Header) string {
	if header == nil {
		return ""
	}
	var buf bytes.Buffer
	for _, el := range header.Content {
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

// headerContentEndIndex returns the end index of the last content element in a header.
func headerContentEndIndex(header *docs.Header) int64 {
	if header == nil || len(header.Content) == 0 {
		return 0
	}
	last := header.Content[len(header.Content)-1]
	if last == nil {
		return 0
	}
	return last.EndIndex
}
