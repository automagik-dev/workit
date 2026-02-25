package cmd

import (
	"context"
	"os"
	"strings"

	"google.golang.org/api/drive/v3"

	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
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

	pageToken := ""
	var publicPermission *drive.Permission
	var domainPermission *drive.Permission

	for {
		call := svc.Permissions.List(fileID).
			SupportsAllDrives(true).
			Fields("permissions(id,type,role,emailAddress,domain),nextPageToken").
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return err
		}

		// Track internet-public and domain-wide permissions separately.
		for _, p := range resp.Permissions {
			if p == nil {
				continue
			}
			switch p.Type {
			case driveShareToAnyone:
				if publicPermission == nil {
					publicPermission = p
				}
			case driveShareToDomain:
				if domainPermission == nil {
					domainPermission = p
				}
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"public":        publicPermission != nil,
			"domain_shared": domainPermission != nil,
		}
		if publicPermission != nil {
			payload["permission"] = publicPermission
		}
		if domainPermission != nil {
			payload["domain_permission"] = domainPermission
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	if publicPermission != nil {
		return writeResult(ctx, u,
			kv("public", true),
			kv("domain_shared", domainPermission != nil),
			kv("type", publicPermission.Type),
			kv("role", publicPermission.Role),
		)
	}
	if domainPermission != nil {
		return writeResult(ctx, u,
			kv("public", false),
			kv("domain_shared", true),
			kv("type", domainPermission.Type),
			kv("role", domainPermission.Role),
			kv("domain", domainPermission.Domain),
		)
	}

	return writeResult(ctx, u,
		kv("public", false),
		kv("domain_shared", false),
	)
}
