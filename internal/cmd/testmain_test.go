package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/automagik-dev/workit/internal/googleauth"
)

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "workit-tests-*")
	if err != nil {
		panic(err)
	}

	oldHome := os.Getenv("HOME")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")

	home := filepath.Join(root, "home")
	xdg := filepath.Join(root, "xdg")
	_ = os.MkdirAll(home, 0o755)
	_ = os.MkdirAll(xdg, 0o755)
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CONFIG_HOME", xdg)

	// Default test stubs: prevent tests from reading real credentials off disk
	// or making real network calls. Individual tests override these as needed.
	origCallback := callbackServerURLFn
	origHeadless := headlessAuthorize
	origPoll := pollForToken
	callbackServerURLFn = func(override string) (string, error) {
		if override != "" {
			return override, nil
		}
		return "https://auth.automagik.dev", nil
	}
	headlessAuthorize = func(ctx context.Context, opts googleauth.HeadlessOptions) (googleauth.HeadlessAuthInfo, error) {
		// Delegate to authorizeGoogle so tests that mock authorizeGoogle still work.
		rt, err := authorizeGoogle(ctx, googleauth.AuthorizeOptions{
			Services: opts.Services,
			Scopes:   opts.Scopes,
			Client:   opts.Client,
		})
		if err != nil {
			return googleauth.HeadlessAuthInfo{}, err
		}
		_ = rt // refresh token handled via pollForToken mock below
		return googleauth.HeadlessAuthInfo{
			AuthURL:   "https://accounts.google.com/o/oauth2/auth?test=1",
			State:     "test-state",
			PollURL:   "https://auth.automagik.dev/token/test-state",
			ExpiresIn: 300,
		}, nil
	}
	pollForToken = func(_ context.Context, _, _ string, _ time.Duration) (string, error) {
		return "test-refresh-token", nil
	}

	code := m.Run()

	callbackServerURLFn = origCallback
	headlessAuthorize = origHeadless
	pollForToken = origPoll

	if oldHome == "" {
		_ = os.Unsetenv("HOME")
	} else {
		_ = os.Setenv("HOME", oldHome)
	}
	if oldXDG == "" {
		_ = os.Unsetenv("XDG_CONFIG_HOME")
	} else {
		_ = os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}
	_ = os.RemoveAll(root)
	os.Exit(code)
}
