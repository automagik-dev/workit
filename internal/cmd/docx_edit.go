package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/automagik-dev/workit/internal/docx"
	"github.com/automagik-dev/workit/internal/ui"
)

// DocxReplaceCmd replaces text in a DOCX file.
type DocxReplaceCmd struct {
	File   string `arg:"" help:"DOCX file path"`
	Old    string `arg:"" help:"Text to find"`
	New    string `arg:"" help:"Replacement text"`
	Output string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx replace command.
func (c *DocxReplaceCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	n, err := docx.Replace(session, c.Old, c.New)
	if err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("replaced %d occurrence(s)", n)
	} else {
		fmt.Fprintf(os.Stderr, "replaced %d occurrence(s)\n", n)
	}
	return nil
}

// DocxInsertCmd inserts a paragraph into a DOCX file.
type DocxInsertCmd struct {
	File   string `arg:"" help:"DOCX file path"`
	After  string `help:"Insert after reference (e.g. 'heading:Summary' or 'paragraph:5')" required:""`
	Text   string `help:"Text to insert" required:""`
	Output string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx insert command.
func (c *DocxInsertCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if err := docx.InsertParagraph(session, c.After, c.Text); err != nil {
		return fmt.Errorf("insert paragraph: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Print("paragraph inserted")
	} else {
		fmt.Fprintln(os.Stderr, "paragraph inserted")
	}
	return nil
}

// DocxDeleteCmd deletes a section from a DOCX file.
type DocxDeleteCmd struct {
	File    string `arg:"" help:"DOCX file path"`
	Section string `help:"Section heading to delete" required:""`
	Output  string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx delete command.
func (c *DocxDeleteCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if err := docx.DeleteSection(session, c.Section); err != nil {
		return fmt.Errorf("delete section: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("section %q deleted", c.Section)
	} else {
		fmt.Fprintf(os.Stderr, "section %q deleted\n", c.Section)
	}
	return nil
}

// DocxStyleCmd changes the style of a paragraph in a DOCX file.
type DocxStyleCmd struct {
	File      string `arg:"" help:"DOCX file path"`
	Paragraph int    `help:"Paragraph index (0-based)" required:""`
	Style     string `help:"Style name to apply" required:""`
	Output    string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx style command.
func (c *DocxStyleCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if err := docx.SetStyle(session, c.Paragraph, c.Style); err != nil {
		return fmt.Errorf("set style: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("paragraph %d styled as %q", c.Paragraph, c.Style)
	} else {
		fmt.Fprintf(os.Stderr, "paragraph %d styled as %q\n", c.Paragraph, c.Style)
	}
	return nil
}
