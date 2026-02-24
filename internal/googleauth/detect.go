package googleauth

import (
	"context"
	"net/http"
	"os"
	"time"

	"golang.org/x/term"

	"github.com/namastexlabs/gog-cli/internal/config"
)

// Auth mode constants.
const (
	AuthModeAuto     = "auto"
	AuthModeBrowser  = "browser"
	AuthModeHeadless = "headless"
	AuthModeManual   = "manual"
)

// AuthModeResult holds the resolved auth mode and how it was determined.
type AuthModeResult struct {
	Mode           string // browser, headless, or manual
	Source         string // flag, config, or auto
	CallbackServer string // resolved callback server URL (empty if not applicable)
}

// isTerminal checks if stdin is a terminal. Variable for testability.
var isTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ResolveAuthMode determines the auth mode based on explicit flags, config, and environment.
//
// Precedence:
//  1. Explicit --headless flag → headless
//  2. Explicit --manual/--remote flag → manual
//  3. Config auth_mode=browser → browser
//  4. Config auth_mode=headless → headless (if callback server resolvable)
//  5. Config auth_mode=manual → manual
//  6. Auto: no TTY + callback server reachable → headless; otherwise browser
func ResolveAuthMode(ctx context.Context, explicitHeadless, explicitManual bool, callbackServerFlag string) AuthModeResult {
	// 1. Explicit flags always win
	if explicitHeadless {
		cbURL, _ := CallbackServerURL(callbackServerFlag)

		return AuthModeResult{Mode: AuthModeHeadless, Source: "flag", CallbackServer: cbURL}
	}

	if explicitManual {
		return AuthModeResult{Mode: AuthModeManual, Source: "flag"}
	}

	// 2. Config auth_mode
	cfg, err := config.ReadConfig()
	if err == nil && cfg.AuthMode != "" {
		switch cfg.AuthMode {
		case AuthModeBrowser:
			return AuthModeResult{Mode: AuthModeBrowser, Source: "config"}
		case AuthModeHeadless:
			cbURL, cbErr := CallbackServerURL(callbackServerFlag)
			if cbErr == nil {
				return AuthModeResult{Mode: AuthModeHeadless, Source: "config", CallbackServer: cbURL}
			}

			// Callback server not resolvable, fall back to browser
			return AuthModeResult{Mode: AuthModeBrowser, Source: "config"}
		case AuthModeManual:
			return AuthModeResult{Mode: AuthModeManual, Source: "config"}
		}
	}

	// 3. Auto-detect: no TTY + callback server reachable → headless
	if !isTerminal() {
		cbURL, cbErr := CallbackServerURL(callbackServerFlag)
		if cbErr == nil && callbackServerReachable(ctx, cbURL) {
			return AuthModeResult{Mode: AuthModeHeadless, Source: "auto", CallbackServer: cbURL}
		}
	}

	// 4. Default: browser
	return AuthModeResult{Mode: AuthModeBrowser, Source: "auto"}
}

// callbackServerReachable checks if the callback server's /health endpoint responds.
func callbackServerReachable(ctx context.Context, serverURL string) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/health", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}
