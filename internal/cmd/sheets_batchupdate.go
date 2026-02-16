package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsBatchUpdateCmd sends a raw batchUpdate request to the Sheets API.
// It reads the JSON request body from --file or stdin.
type SheetsBatchUpdateCmd struct {
	SpreadsheetID string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID or URL"`
	File          string `name:"file" help:"JSON file with requests (if omitted, reads piped stdin)"`
}

func (c *SheetsBatchUpdateCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	if spreadsheetID == "" {
		return usage("empty spreadsheetId")
	}

	// Read JSON input from --file or stdin
	var inputBytes []byte
	var err error

	filePath := strings.TrimSpace(c.File)
	if filePath != "" {
		inputBytes, err = os.ReadFile(filePath) //nolint:gosec // user-provided path
		if err != nil {
			return fmt.Errorf("read --file: %w", err)
		}
	} else {
		stat, statErr := os.Stdin.Stat()
		if statErr != nil {
			return fmt.Errorf("stat stdin: %w", statErr)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return usage("no input provided (use --file or pipe JSON via stdin)")
		}
		inputBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	}

	if len(inputBytes) == 0 {
		return fmt.Errorf("empty request body (provide JSON via --file or stdin)")
	}

	var req sheets.BatchUpdateSpreadsheetRequest
	if err = json.Unmarshal(inputBytes, &req); err != nil {
		return fmt.Errorf("invalid JSON request: %w", err)
	}

	if err = dryRunExit(ctx, flags, "sheets.batch-update", map[string]any{
		"spreadsheet_id": spreadsheetID,
		"requests_count": len(req.Requests),
	}); err != nil {
		return err
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &req).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"spreadsheetId": resp.SpreadsheetId,
			"replies":       resp.Replies,
		})
	}

	u.Out().Printf("Batch update completed for spreadsheet %s (%d replies)", resp.SpreadsheetId, len(resp.Replies))
	return nil
}
