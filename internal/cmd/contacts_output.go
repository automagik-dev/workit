package cmd

import (
	"context"

	"github.com/namastexlabs/gog-cli/internal/ui"
)

func writeDeleteResult(ctx context.Context, u *ui.UI, resourceName string) error {
	return writeResult(ctx, u,
		kv("deleted", true),
		kv("resource", resourceName),
	)
}
