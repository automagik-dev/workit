package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/automagik-dev/workit/internal/docx"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

// DocxCmd is the top-level command for DOCX document operations.
type DocxCmd struct {
	Cat           DocxCatCmd           `cmd:"" help:"Extract document content as markdown or structured JSON"`
	Info          DocxInfoCmd          `cmd:"" help:"Show document metadata and structure"`
	Replace       DocxReplaceCmd       `cmd:"" help:"Find and replace text in a DOCX file"`
	Insert        DocxInsertCmd        `cmd:"" help:"Insert a paragraph into a DOCX file"`
	Delete        DocxDeleteCmd        `cmd:"" help:"Delete a section from a DOCX file by heading"`
	Style         DocxStyleCmd         `cmd:"" help:"Change the style of a paragraph in a DOCX file"`
	Track         DocxTrackCmd         `cmd:"" help:"Create a tracked replacement in a DOCX file"`
	AcceptChanges DocxAcceptChangesCmd `cmd:"accept-changes" help:"Accept all tracked changes in a DOCX file"`
	RejectChanges DocxRejectChangesCmd `cmd:"reject-changes" help:"Reject all tracked changes in a DOCX file"`
	Comment       DocxCommentCmd       `cmd:"" help:"Add a comment to a DOCX file"`
	ListComments  DocxListCommentsCmd  `cmd:"list-comments" help:"List all comments in a DOCX file"`
	Table         DocxTableCmd         `cmd:"" help:"Table operations (list, add row, update cell, delete row)"`
	Create        DocxCreateCmd        `cmd:"" help:"Create a DOCX from a template + JSON values or from markdown"`
	Rewrite       DocxRewriteCmd       `cmd:"" help:"Replace all body content with markdown content"`
	Inspect       DocxInspectCmd       `cmd:"" help:"Inspect a template DOCX for {{PLACEHOLDER}} patterns"`
	ToPDF         DocxToPDFCmd         `cmd:"to-pdf" help:"Convert DOCX to PDF via LibreOffice"`
}

// DocxCatCmd extracts the text content of a DOCX file.
type DocxCatCmd struct {
	File      string `arg:"" help:"DOCX file path"`
	Structure bool   `help:"Output structured JSON with paragraph IDs and styles" short:"s"`
}

// Run executes the docx cat command.
func (c *DocxCatCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if c.Structure || outfmt.IsJSON(ctx) {
		structure, structErr := docx.ReadStructure(session)
		if structErr != nil {
			return fmt.Errorf("read structure: %w", structErr)
		}
		return outfmt.WriteJSON(ctx, os.Stdout, structure)
	}

	md, err := docx.ReadAsMarkdown(session)
	if err != nil {
		return fmt.Errorf("read content: %w", err)
	}

	u := ui.FromContext(ctx)
	if u != nil {
		u.Out().Print(md)
	} else {
		fmt.Print(md)
	}
	return nil
}

// DocxInfoCmd shows metadata and structure info for a DOCX file.
type DocxInfoCmd struct {
	File string `arg:"" help:"DOCX file path"`
}

// Run executes the docx info command.
func (c *DocxInfoCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	meta, err := docx.ReadMetadata(session)
	if err != nil {
		return fmt.Errorf("read metadata: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, meta)
	}

	u := ui.FromContext(ctx)
	if u != nil {
		if meta.Title != "" {
			u.Out().Printf("title\t%s", meta.Title)
		}
		if meta.Author != "" {
			u.Out().Printf("author\t%s", meta.Author)
		}
		if meta.Description != "" {
			u.Out().Printf("description\t%s", meta.Description)
		}
		if meta.Created != "" {
			u.Out().Printf("created\t%s", meta.Created)
		}
		if meta.Modified != "" {
			u.Out().Printf("modified\t%s", meta.Modified)
		}
		if meta.Pages > 0 {
			u.Out().Printf("pages\t%d", meta.Pages)
		}

		// Show part list.
		parts := session.ListParts()
		u.Out().Printf("parts\t%d", len(parts))
		for _, p := range parts {
			u.Out().Printf("  %s", p)
		}
	}

	return nil
}
