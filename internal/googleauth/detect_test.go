package googleauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// setupTestConfig writes a config file to the platform-specific config dir.
// On Linux: XDG_CONFIG_HOME/gogcli/config.json
// On macOS: HOME/Library/Application Support/gogcli/config.json
func setupTestConfig(t *testing.T, content string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}

	dir := filepath.Join(cfgDir, "gogcli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// clearTestConfig sets up an empty config dir (no config file).
func clearTestConfig(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}

func TestResolveAuthMode_ExplicitHeadless(t *testing.T) {
	clearTestConfig(t)
	t.Setenv("GOG_CALLBACK_SERVER", "https://example.com")

	result := ResolveAuthMode(context.Background(), true, false, "")
	if result.Mode != AuthModeHeadless {
		t.Fatalf("expected headless, got %s", result.Mode)
	}

	if result.Source != "flag" {
		t.Fatalf("expected source=flag, got %s", result.Source)
	}
}

func TestResolveAuthMode_ExplicitManual(t *testing.T) {
	clearTestConfig(t)

	result := ResolveAuthMode(context.Background(), false, true, "")
	if result.Mode != AuthModeManual {
		t.Fatalf("expected manual, got %s", result.Mode)
	}

	if result.Source != "flag" {
		t.Fatalf("expected source=flag, got %s", result.Source)
	}
}

func TestResolveAuthMode_ConfigBrowser(t *testing.T) {
	setupTestConfig(t, `{"auth_mode":"browser"}`)

	result := ResolveAuthMode(context.Background(), false, false, "")
	if result.Mode != AuthModeBrowser {
		t.Fatalf("expected browser, got %s", result.Mode)
	}

	if result.Source != "config" {
		t.Fatalf("expected source=config, got %s", result.Source)
	}
}

func TestResolveAuthMode_ConfigHeadless(t *testing.T) {
	setupTestConfig(t, `{"auth_mode":"headless","callback_server":"https://test.example.com"}`)
	// Clear env so config callback_server is used
	t.Setenv("GOG_CALLBACK_SERVER", "")

	result := ResolveAuthMode(context.Background(), false, false, "")
	if result.Mode != AuthModeHeadless {
		t.Fatalf("expected headless, got %s", result.Mode)
	}

	if result.Source != "config" {
		t.Fatalf("expected source=config, got %s", result.Source)
	}

	if result.CallbackServer != "https://test.example.com" {
		t.Fatalf("expected callback server from config, got %q", result.CallbackServer)
	}
}

func TestResolveAuthMode_ConfigHeadless_NoCallbackServer(t *testing.T) {
	setupTestConfig(t, `{"auth_mode":"headless"}`)
	t.Setenv("GOG_CALLBACK_SERVER", "")

	result := ResolveAuthMode(context.Background(), false, false, "")
	// Should fall back to browser when callback server not resolvable
	if result.Mode != AuthModeBrowser {
		t.Fatalf("expected browser fallback, got %s", result.Mode)
	}
}

func TestResolveAuthMode_ConfigManual(t *testing.T) {
	setupTestConfig(t, `{"auth_mode":"manual"}`)

	result := ResolveAuthMode(context.Background(), false, false, "")
	if result.Mode != AuthModeManual {
		t.Fatalf("expected manual, got %s", result.Mode)
	}

	if result.Source != "config" {
		t.Fatalf("expected source=config, got %s", result.Source)
	}
}

func TestResolveAuthMode_AutoDetect_NoTTY_ReachableServer(t *testing.T) {
	clearTestConfig(t)
	t.Setenv("GOG_CALLBACK_SERVER", "")

	// Mock non-terminal
	orig := isTerminal

	t.Cleanup(func() { isTerminal = orig })

	isTerminal = func() bool { return false }

	// Start a test server that responds to /health
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))

			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	result := ResolveAuthMode(context.Background(), false, false, srv.URL)
	if result.Mode != AuthModeHeadless {
		t.Fatalf("expected headless via auto-detect, got %s", result.Mode)
	}

	if result.Source != "auto" {
		t.Fatalf("expected source=auto, got %s", result.Source)
	}
}

func TestResolveAuthMode_AutoDetect_WithTTY(t *testing.T) {
	clearTestConfig(t)
	t.Setenv("GOG_CALLBACK_SERVER", "")

	// Mock terminal
	orig := isTerminal

	t.Cleanup(func() { isTerminal = orig })

	isTerminal = func() bool { return true }

	result := ResolveAuthMode(context.Background(), false, false, "")
	if result.Mode != AuthModeBrowser {
		t.Fatalf("expected browser with TTY, got %s", result.Mode)
	}
}

func TestResolveAuthMode_FlagOverridesConfig(t *testing.T) {
	setupTestConfig(t, `{"auth_mode":"browser"}`)
	t.Setenv("GOG_CALLBACK_SERVER", "https://example.com")

	// Explicit headless flag should override config=browser
	result := ResolveAuthMode(context.Background(), true, false, "")
	if result.Mode != AuthModeHeadless {
		t.Fatalf("expected headless (flag override), got %s", result.Mode)
	}
}

func TestCallbackServerReachable_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)

			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	if !callbackServerReachable(context.Background(), srv.URL) {
		t.Fatal("expected reachable")
	}
}

func TestCallbackServerReachable_Unreachable(t *testing.T) {
	if callbackServerReachable(context.Background(), "http://127.0.0.1:1") {
		t.Fatal("expected unreachable")
	}
}
