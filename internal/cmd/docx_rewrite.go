package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/automagik-dev/workit/internal/docx"
	"github.com/automagik-dev/workit/internal/ui"
)

// DocxRewriteCmd replaces all body content in a DOCX with markdown content.
type DocxRewriteCmd struct {
	File   string `arg:"" help:"DOCX file path"`
	From   string `help:"Markdown file with new content" required:""`
	Output string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx rewrite command.
func (c *DocxRewriteCmd) Run(ctx context.Context) error {
	mdData, err := os.ReadFile(c.From)
	if err != nil {
		return fmt.Errorf("read markdown %s: %w", c.From, err)
	}

	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if err := docx.Rewrite(session, string(mdData)); err != nil {
		return fmt.Errorf("rewrite: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("content rewritten from %s", c.From)
	} else {
		fmt.Fprintf(os.Stderr, "content rewritten from %s\n", c.From)
	}
	return nil
}

// DocxInspectCmd inspects a template DOCX for placeholders.
type DocxInspectCmd struct {
	File string `arg:"" help:"Template DOCX file path"`
}

// Run executes the docx inspect command.
func (c *DocxInspectCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	names, err := docx.InspectTemplate(session)
	if err != nil {
		return fmt.Errorf("inspect template: %w", err)
	}

	u := ui.FromContext(ctx)

	if len(names) == 0 {
		if u != nil {
			u.Out().Print("no placeholders found")
		} else {
			fmt.Println("no placeholders found")
		}
		return nil
	}

	if u != nil {
		u.Out().Printf("found %d placeholder(s):", len(names))
		for _, name := range names {
			u.Out().Printf("  {{%s}}", name)
		}
	} else {
		fmt.Printf("found %d placeholder(s):\n", len(names))
		for _, name := range names {
			fmt.Printf("  {{%s}}\n", name)
		}
	}
	return nil
}
