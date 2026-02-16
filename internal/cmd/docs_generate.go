package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// DocsGenerateCmd generates a new Google Doc from a template document,
// replacing {{placeholder}} patterns with values from a JSON data file.
type DocsGenerateCmd struct {
	Template string `name:"template" help:"Template document ID or URL" required:""`
	Data     string `name:"data" help:"Path to JSON data file" required:""`
	Title    string `name:"title" help:"Title for new document (default: copy of template)"`
	Folder   string `name:"folder" help:"Folder ID or URL to place the new document in"`
}

func (c *DocsGenerateCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	templateID := normalizeGoogleID(strings.TrimSpace(c.Template))
	if templateID == "" {
		return usage("empty template ID")
	}

	folderID := normalizeGoogleID(strings.TrimSpace(c.Folder))

	// Read and parse the data JSON file.
	dataBytes, err := os.ReadFile(c.Data)
	if err != nil {
		return fmt.Errorf("read --data file: %w", err)
	}

	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(dataBytes))
	decoder.UseNumber()
	if err = decoder.Decode(&data); err != nil {
		return fmt.Errorf("invalid JSON in data file: %w", err)
	}

	if err = dryRunExit(ctx, flags, "docs.generate", map[string]any{
		"template":     templateID,
		"folder":       folderID,
		"title":        c.Title,
		"placeholders": len(data),
	}); err != nil {
		return err
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	driveSvc, err := newDriveService(ctx, account)
	if err != nil {
		return err
	}

	// Step 1: Copy the template doc via Drive API.
	copyMeta := &drive.File{}
	title := strings.TrimSpace(c.Title)
	if title != "" {
		copyMeta.Name = title
	}
	if folderID != "" {
		copyMeta.Parents = []string{folderID}
	}

	created, err := driveSvc.Files.Copy(templateID, copyMeta).
		SupportsAllDrives(true).
		Fields("id, name, mimeType, webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("copy template: %w", err)
	}
	if created == nil {
		return fmt.Errorf("copy template returned nil")
	}

	// Step 2: Replace placeholders in the new doc via Docs API batchUpdate.
	if len(data) > 0 {
		docsSvc, err := newDocsService(ctx, account)
		if err != nil {
			return err
		}

		// Sort keys for deterministic replacement order.
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var requests []*docs.Request
		for _, key := range keys {
			requests = append(requests, &docs.Request{
				ReplaceAllText: &docs.ReplaceAllTextRequest{
					ContainsText: &docs.SubstringMatchCriteria{
						Text:      "{{" + key + "}}",
						MatchCase: true,
					},
					ReplaceText: docsGenerateReplaceValue(data[key]),
				},
			})
		}

		_, err = docsSvc.Documents.BatchUpdate(created.Id, &docs.BatchUpdateDocumentRequest{
			Requests: requests,
		}).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("replace placeholders: %w", err)
		}
	}

	// Step 3: Output the result.
	docURL := docsWebViewLink(created.Id)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"documentId":   created.Id,
			"name":         created.Name,
			"url":          docURL,
			"placeholders": len(data),
		})
	}

	u.Out().Printf("id\t%s", created.Id)
	u.Out().Printf("name\t%s", created.Name)
	u.Out().Printf("url\t%s", docURL)
	u.Out().Printf("placeholders\t%d replaced", len(data))
	return nil
}

func docsGenerateReplaceValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int:
		return strconv.Itoa(v)
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	}
}
