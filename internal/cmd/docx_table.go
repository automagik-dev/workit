package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/automagik-dev/workit/internal/docx"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

// DocxTableCmd provides table operations on DOCX files.
type DocxTableCmd struct {
	File       string `arg:"" help:"DOCX file path"`
	List       bool   `help:"List all tables"`
	ID         int    `help:"Table index (0-based)" default:"-1"`
	AddRow     string `help:"Add row with comma-separated values"`
	UpdateCell string `help:"Update cell at row,col (1-based) with value (format: row,col,value)"`
	DeleteRow  int    `help:"Delete row at index (1-based)" default:"-1"`
	Output     string `short:"o" help:"Output file path (default: overwrite input)"`
}

// Run executes the docx table command.
func (c *DocxTableCmd) Run(ctx context.Context) error {
	session, err := docx.Open(c.File)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	defer session.Close()

	// List mode.
	if c.List {
		return c.runList(ctx, session)
	}

	// Determine target table index.
	tableIdx := c.ID
	if tableIdx < 0 {
		tableIdx = 0 // default to first table
	}

	// Add row.
	if c.AddRow != "" {
		if err := docx.AddTableRow(session, tableIdx, c.AddRow); err != nil {
			return fmt.Errorf("add row: %w", err)
		}
		return c.saveAndReport(ctx, session, "row added to table %d", tableIdx)
	}

	// Update cell.
	if c.UpdateCell != "" {
		row, col, value, err := parseUpdateCell(c.UpdateCell)
		if err != nil {
			return fmt.Errorf("parse update-cell: %w", err)
		}
		if err := docx.UpdateTableCell(session, tableIdx, row, col, value); err != nil {
			return fmt.Errorf("update cell: %w", err)
		}
		return c.saveAndReport(ctx, session, "cell [%d,%d] updated in table %d", row, col, tableIdx)
	}

	// Delete row.
	if c.DeleteRow > 0 {
		if err := docx.DeleteTableRow(session, tableIdx, c.DeleteRow); err != nil {
			return fmt.Errorf("delete row: %w", err)
		}
		return c.saveAndReport(ctx, session, "row %d deleted from table %d", c.DeleteRow, tableIdx)
	}

	// Default: list tables if no operation specified.
	return c.runList(ctx, session)
}

func (c *DocxTableCmd) runList(ctx context.Context, session *docx.EditSession) error {
	tables, err := docx.ListTables(session)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, tables)
	}

	u := ui.FromContext(ctx)
	if len(tables) == 0 {
		if u != nil {
			u.Out().Print("no tables found")
		} else {
			fmt.Println("no tables found")
		}
		return nil
	}

	for _, tbl := range tables {
		if u != nil {
			u.Out().Printf("table %d: %d rows x %d cols", tbl.Index, tbl.Rows, tbl.Cols)
			for _, row := range tbl.Data {
				u.Out().Printf("  %v", row)
			}
		} else {
			fmt.Printf("table %d: %d rows x %d cols\n", tbl.Index, tbl.Rows, tbl.Cols)
			for _, row := range tbl.Data {
				fmt.Printf("  %v\n", row)
			}
		}
	}
	return nil
}

func (c *DocxTableCmd) saveAndReport(ctx context.Context, session *docx.EditSession, msg string, args ...any) error {
	output := c.Output
	if output == "" {
		output = c.File
	}
	if err := session.Save(output); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf(msg, args...)
	} else {
		fmt.Fprintf(os.Stderr, msg+"\n", args...)
	}
	return nil
}

// parseUpdateCell parses "row,col,value" format.
func parseUpdateCell(s string) (row, col int, value string, err error) {
	// Find first comma.
	first := -1
	for i, ch := range s {
		if ch == ',' {
			first = i
			break
		}
	}
	if first == -1 {
		return 0, 0, "", fmt.Errorf("expected format row,col,value")
	}

	// Find second comma.
	second := -1
	for i := first + 1; i < len(s); i++ {
		if s[i] == ',' {
			second = i
			break
		}
	}
	if second == -1 {
		return 0, 0, "", fmt.Errorf("expected format row,col,value")
	}

	_, err = fmt.Sscanf(s[:first], "%d", &row)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid row: %w", err)
	}
	_, err = fmt.Sscanf(s[first+1:second], "%d", &col)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid col: %w", err)
	}
	value = s[second+1:]
	return row, col, value, nil
}
