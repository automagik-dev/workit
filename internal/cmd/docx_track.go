package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/namastexlabs/workit/internal/docx"
	"github.com/namastexlabs/workit/internal/outfmt"
	"github.com/namastexlabs/workit/internal/ui"
)

// DocxTrackCmd creates tracked replacements in a DOCX file.
type DocxTrackCmd struct {
	File    string `arg:"" help:"DOCX file path"`
	Replace string `help:"Text to find for tracked replacement" required:""`
	New     string `help:"Replacement text (used with --replace)" required:""`
	Author  string `help:"Author attribution" default:"Workit"`
	Output  string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx track command.
func (c *DocxTrackCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	n, err := docx.TrackReplace(session, c.Replace, c.New, c.Author)
	if err != nil {
		return fmt.Errorf("track replace: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("tracked %d replacement(s)", n)
	} else {
		fmt.Fprintf(os.Stderr, "tracked %d replacement(s)\n", n)
	}
	return nil
}

// DocxAcceptChangesCmd accepts all tracked changes in a DOCX file.
type DocxAcceptChangesCmd struct {
	File   string `arg:"" help:"DOCX file path"`
	Output string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx accept-changes command.
func (c *DocxAcceptChangesCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if err := docx.AcceptChanges(session); err != nil {
		return fmt.Errorf("accept changes: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Print("all tracked changes accepted")
	} else {
		fmt.Fprintln(os.Stderr, "all tracked changes accepted")
	}
	return nil
}

// DocxRejectChangesCmd rejects all tracked changes in a DOCX file.
type DocxRejectChangesCmd struct {
	File   string `arg:"" help:"DOCX file path"`
	Output string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx reject-changes command.
func (c *DocxRejectChangesCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if err := docx.RejectChanges(session); err != nil {
		return fmt.Errorf("reject changes: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Print("all tracked changes rejected")
	} else {
		fmt.Fprintln(os.Stderr, "all tracked changes rejected")
	}
	return nil
}

// DocxCommentCmd adds a comment to a DOCX file.
type DocxCommentCmd struct {
	File   string `arg:"" help:"DOCX file path"`
	At     string `help:"Comment location (e.g. 'paragraph:1')" required:""`
	Text   string `help:"Comment text" required:""`
	Author string `help:"Comment author" default:"Workit"`
	Output string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx comment command.
func (c *DocxCommentCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	if err := docx.AddComment(session, c.At, c.Text, c.Author); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}

	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("comment added at %s", c.At)
	} else {
		fmt.Fprintf(os.Stderr, "comment added at %s\n", c.At)
	}
	return nil
}

// DocxListCommentsCmd lists all comments in a DOCX file.
type DocxListCommentsCmd struct {
	File string `arg:"" help:"DOCX file path"`
}

// Run executes the docx list-comments command.
func (c *DocxListCommentsCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	comments, err := docx.ListComments(session)
	if err != nil {
		return fmt.Errorf("list comments: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, comments)
	}

	u := ui.FromContext(ctx)
	if len(comments) == 0 {
		if u != nil {
			u.Out().Print("no comments")
		} else {
			fmt.Println("no comments")
		}
		return nil
	}

	for _, c := range comments {
		line := fmt.Sprintf("[%s] %s (%s): %s", c.ID, c.Author, c.Date, c.Text)
		if u != nil {
			u.Out().Print(line)
		} else {
			fmt.Println(line)
		}
	}
	return nil
}
