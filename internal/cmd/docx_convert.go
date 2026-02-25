package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/namastexlabs/workit/internal/docx"
	"github.com/namastexlabs/workit/internal/outfmt"
	"github.com/namastexlabs/workit/internal/ui"
)

// DocxToPDFCmd converts a DOCX file to PDF via LibreOffice.
type DocxToPDFCmd struct {
	File   string `arg:"" help:"DOCX file path"`
	Output string `short:"o" help:"Output PDF file path"`
}

// Run executes the docx to-pdf command.
func (c *DocxToPDFCmd) Run(ctx context.Context) error {
	pdfPath, err := docx.ConvertToPDF(ctx, c.File, c.Output)
	if err != nil {
		return fmt.Errorf("convert to pdf: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]string{
			"path": pdfPath,
		})
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("converted to %s", pdfPath)
	} else {
		fmt.Fprintf(os.Stderr, "converted to %s\n", pdfPath)
	}
	return nil
}
