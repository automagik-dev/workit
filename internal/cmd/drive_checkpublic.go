package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// DriveCheckPublicCmd checks whether a Drive file is publicly accessible.
type DriveCheckPublicCmd struct {
	FileID string `arg:"" name:"fileId" help:"File ID or URL to check"`
}

func (c *DriveCheckPublicCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	fileID := normalizeGoogleID(c.FileID)
	if strings.TrimSpace(fileID) == "" {
		return usage("empty fileId")
	}

	svc, err := newDriveService(ctx, account)
	if err != nil {
		return err
	}

	resp, err := svc.Permissions.List(fileID).
		SupportsAllDrives(true).
		Fields("permissions(id,type,role,emailAddress,domain)").
		Context(ctx).
		Do()
	if err != nil {
		return err
	}

	// Check if any permission grants public or domain-wide access.
	for _, p := range resp.Permissions {
		if p.Type == "anyone" || p.Type == "domain" {
			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
					"public":     true,
					"permission": p,
				})
			}
			return writeResult(ctx, u,
				kv("public", true),
				kv("type", p.Type),
				kv("role", p.Role),
			)
		}
	}

	// No public permissions found.
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"public": false,
		})
	}
	return writeResult(ctx, u,
		kv("public", false),
	)
}
