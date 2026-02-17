package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/steipete/gogcli/internal/officetext"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

const (
	// maxCatSize is the maximum file size (100 MB) that drive cat will read.
	// Files larger than this are truncated with a warning on stderr.
	maxCatSize int64 = 100 << 20
)

// DriveCatCmd outputs the text content of a Drive file to stdout.
// For Google Docs native formats (Docs/Sheets/Slides), it exports to plain text.
// For DOCX/XLSX/PPTX, it extracts text using the officetext package.
// For other formats, it outputs the raw content.
type DriveCatCmd struct {
	FileID string `arg:"" name:"file-id" help:"Drive file ID or name to read"`
}

func (c *DriveCatCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	fileID := strings.TrimSpace(c.FileID)
	if fileID == "" {
		return usage("empty file-id")
	}

	svc, err := newDriveService(ctx, account)
	if err != nil {
		return err
	}

	// Fetch file metadata.
	meta, err := svc.Files.Get(fileID).
		SupportsAllDrives(true).
		Fields("id, name, mimeType, size").
		Context(ctx).
		Do()
	if err != nil {
		return err
	}

	isGoogleNative := strings.HasPrefix(meta.MimeType, "application/vnd.google-apps.")

	var content string

	if isGoogleNative {
		// Export Google Docs native formats to plain text.
		exportMime := googleNativeExportMime(meta.MimeType)
		resp, dlErr := driveExportDownload(ctx, svc, meta.Id, exportMime)
		if dlErr != nil {
			return fmt.Errorf("export %s: %w", meta.Name, dlErr)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxCatSize+1))
		if readErr != nil {
			return fmt.Errorf("read export: %w", readErr)
		}
		if int64(len(body)) > maxCatSize {
			u.Err().Printf("warning: exported content truncated at %d MB", maxCatSize>>20)
			body = body[:maxCatSize]
		}
		content = string(body)
	} else {
		// Download raw file bytes.
		resp, dlErr := driveDownload(ctx, svc, meta.Id)
		if dlErr != nil {
			return fmt.Errorf("download %s: %w", meta.Name, dlErr)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxCatSize+1))
		if readErr != nil {
			return fmt.Errorf("read download: %w", readErr)
		}
		if int64(len(body)) > maxCatSize {
			u.Err().Printf("warning: file %s truncated at %d MB", meta.Name, maxCatSize>>20)
			body = body[:maxCatSize]
		}

		// Try Office text extraction if applicable.
		if officetext.IsSupportedMIME(meta.MimeType) || officetext.IsSupportedExtension(meta.Name) {
			extracted, extErr := officetext.ExtractText(bytes.NewReader(body), meta.Name)
			if extErr != nil {
				// Fall back to raw content with a warning on stderr.
				u.Err().Printf("warning: text extraction failed for %s: %s (showing raw content)", meta.Name, extErr)
				content = string(body)
			} else {
				content = extracted
			}
		} else {
			content = string(body)
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"file_id":   meta.Id,
			"name":      meta.Name,
			"mime_type": meta.MimeType,
			"content":   content,
		})
	}

	// Plain/default output: just the text content.
	fmt.Fprint(os.Stdout, content)
	return nil
}

// googleNativeExportMime returns the best plain-text export MIME type
// for Google Docs native formats.
func googleNativeExportMime(googleMimeType string) string {
	switch googleMimeType {
	case driveMimeGoogleDoc:
		return mimeTextPlain
	case driveMimeGoogleSheet:
		return mimeCSV
	case driveMimeGoogleSlides:
		return mimeTextPlain
	default:
		return mimeTextPlain
	}
}
