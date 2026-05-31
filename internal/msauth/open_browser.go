package msauth

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func quoteWindowsStartURL(url string) string {
	return `"` + strings.ReplaceAll(url, `"`, `%22`) + `"`
}

func openBrowser(ctx context.Context, url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", "start", "", quoteWindowsStartURL(url)) //nolint:gosec // URL is quoted for cmd/start; browser launch intentionally uses the OAuth URL.
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	return nil
}
