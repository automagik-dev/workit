package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/namastexlabs/workit/internal/config"
	"github.com/namastexlabs/workit/internal/googleauth"
	"github.com/namastexlabs/workit/internal/outfmt"
	"github.com/namastexlabs/workit/internal/secrets"
	"github.com/namastexlabs/workit/internal/ui"
)

// TestAuthPollCmd_AutoSave_NoEmail verifies that `wk auth poll <state>` without
// --email infers the email via fetchAuthorizedEmail and stores the token.
func TestAuthPollCmd_AutoSave_NoEmail(t *testing.T) {
	origPoll := pollForToken
	origOpen := openSecretsStore
	origFetch := fetchAuthorizedEmail
	origKeychain := ensureKeychainAccess
	origCallback := callbackServerURLFn
	t.Cleanup(func() {
		pollForToken = origPoll
		openSecretsStore = origOpen
		fetchAuthorizedEmail = origFetch
		ensureKeychainAccess = origKeychain
		callbackServerURLFn = origCallback
	})

	ensureKeychainAccess = func() error { return nil }
	callbackServerURLFn = func(s string) (string, error) {
		return "https://callback.example.com", nil
	}

	pollForToken = func(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
		return "rt-from-poll", nil
	}

	var fetchedClient string
	fetchAuthorizedEmail = func(_ context.Context, client string, _ string, _ []string, _ time.Duration) (string, error) {
		fetchedClient = client
		return "inferred@example.com", nil
	}

	store := newMemSecretsStore()
	openSecretsStore = func() (secrets.Store, error) { return store, nil }

	cmd := &AuthPollCmd{
		State:       "test-state-123",
		ServicesCSV: "all",
	}
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			// Create UI inside capture so it writes to the captured pipes.
			u, uiErr := ui.New(ui.Options{Color: "never"})
			if uiErr != nil {
				t.Fatalf("ui.New: %v", uiErr)
			}
			ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})
			if runErr := cmd.Run(ctx); runErr != nil {
				t.Fatalf("Run: %v", runErr)
			}
		})
	})

	// Verify client was resolved from DefaultClientName (not via ResolveClientWithOverride)
	if fetchedClient != config.DefaultClientName {
		t.Fatalf("expected client %q, got %q", config.DefaultClientName, fetchedClient)
	}

	// Verify token was stored
	tok, tokErr := store.GetToken(config.DefaultClientName, "inferred@example.com")
	if tokErr != nil {
		t.Fatalf("GetToken: %v", tokErr)
	}
	if tok.RefreshToken != "rt-from-poll" {
		t.Fatalf("unexpected refresh token: %q", tok.RefreshToken)
	}
	if len(tok.Services) == 0 {
		t.Fatalf("expected services to be set")
	}

	// Verify JSON output includes stored, email, services
	var parsed struct {
		Stored   bool     `json:"stored"`
		Email    string   `json:"email"`
		Services []string `json:"services"`
		Client   string   `json:"client"`
	}
	if jsonErr := json.Unmarshal([]byte(out), &parsed); jsonErr != nil {
		t.Fatalf("json parse: %v\nout=%q", jsonErr, out)
	}
	if !parsed.Stored {
		t.Fatalf("expected stored=true")
	}
	if parsed.Email != "inferred@example.com" {
		t.Fatalf("unexpected email: %q", parsed.Email)
	}
}

// TestAuthPollCmd_AutoSave_UserinfoFailure verifies that when fetchAuthorizedEmail
// fails, the poll command falls back to printing the raw token + a WARNING.
func TestAuthPollCmd_AutoSave_UserinfoFailure(t *testing.T) {
	origPoll := pollForToken
	origOpen := openSecretsStore
	origFetch := fetchAuthorizedEmail
	origKeychain := ensureKeychainAccess
	origCallback := callbackServerURLFn
	t.Cleanup(func() {
		pollForToken = origPoll
		openSecretsStore = origOpen
		fetchAuthorizedEmail = origFetch
		ensureKeychainAccess = origKeychain
		callbackServerURLFn = origCallback
	})

	ensureKeychainAccess = func() error { return nil }
	callbackServerURLFn = func(s string) (string, error) {
		return "https://callback.example.com", nil
	}

	pollForToken = func(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
		return "rt-raw-token", nil
	}

	fetchAuthorizedEmail = func(_ context.Context, _ string, _ string, _ []string, _ time.Duration) (string, error) {
		return "", errors.New("userinfo endpoint unreachable")
	}

	store := newMemSecretsStore()
	openSecretsStore = func() (secrets.Store, error) { return store, nil }

	cmd := &AuthPollCmd{
		State:       "test-state-456",
		ServicesCSV: "all",
	}

	var stdoutBuf, stderrBuf strings.Builder
	u, uiErr := ui.New(ui.Options{Stdout: &stdoutBuf, Stderr: &stderrBuf, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)

	if runErr := cmd.Run(ctx); runErr != nil {
		t.Fatalf("Run should not error on userinfo failure: %v", runErr)
	}

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	// Check WARNING printed to stderr
	if !strings.Contains(stderr, "WARNING") {
		t.Fatalf("expected WARNING in stderr, got: %q", stderr)
	}
	// Check re-poll command hint
	if !strings.Contains(stderr, "wk auth poll") || !strings.Contains(stderr, "--email") {
		t.Fatalf("expected re-poll command hint in stderr, got: %q", stderr)
	}

	// Verify the raw refresh token is printed to stdout
	if !strings.Contains(stdout, "rt-raw-token") {
		t.Fatalf("expected raw token in stdout, got: %q", stdout)
	}

	// Verify NO token was stored (since email couldn't be inferred)
	tokens, _ := store.ListTokens()
	if len(tokens) != 0 {
		t.Fatalf("expected no tokens stored, got %d", len(tokens))
	}
}

// TestAuthAddCmd_DefaultServicesAll verifies that auth add without --services
// defaults to "all" (which resolves to UserServices, including drive/calendar/gmail).
func TestAuthAddCmd_DefaultServicesAll(t *testing.T) {
	origAuth := authorizeGoogle
	origOpen := openSecretsStore
	origKeychain := ensureKeychainAccess
	origFetch := fetchAuthorizedEmail
	t.Cleanup(func() {
		authorizeGoogle = origAuth
		openSecretsStore = origOpen
		ensureKeychainAccess = origKeychain
		fetchAuthorizedEmail = origFetch
	})

	ensureKeychainAccess = func() error { return nil }

	store := newMemSecretsStore()
	openSecretsStore = func() (secrets.Store, error) { return store, nil }

	var gotServices []googleauth.Service
	authorizeGoogle = func(_ context.Context, opts googleauth.AuthorizeOptions) (string, error) {
		gotServices = append([]googleauth.Service{}, opts.Services...)
		return "rt", nil
	}
	fetchAuthorizedEmail = func(context.Context, string, string, []string, time.Duration) (string, error) {
		return "user@example.com", nil
	}

	_ = captureStdout(t, func() {
		_ = captureStderr(t, func() {
			// No --services flag = should use default "all"
			if err := Execute([]string{"--json", "auth", "add", "user@example.com"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	// Verify that services include drive, calendar, gmail (i.e., all user services)
	serviceSet := make(map[googleauth.Service]bool)
	for _, s := range gotServices {
		serviceSet[s] = true
	}
	if !serviceSet[googleauth.ServiceDrive] {
		t.Fatalf("expected drive in default services, got: %v", gotServices)
	}
	if !serviceSet[googleauth.ServiceCalendar] {
		t.Fatalf("expected calendar in default services, got: %v", gotServices)
	}
	if !serviceSet[googleauth.ServiceGmail] {
		t.Fatalf("expected gmail in default services, got: %v", gotServices)
	}
}

// TestParseAuthServices_DistinctPaths verifies that "all" and "user" both
// return UserServices() but travel through distinct code paths.
func TestParseAuthServices_DistinctPaths(t *testing.T) {
	allServices, err := parseAuthServices("all")
	if err != nil {
		t.Fatalf("parseAuthServices(all): %v", err)
	}

	userServices, err := parseAuthServices("user")
	if err != nil {
		t.Fatalf("parseAuthServices(user): %v", err)
	}

	// Both should return the same set of services (UserServices)
	if len(allServices) != len(userServices) {
		t.Fatalf("expected same length, got all=%d user=%d", len(allServices), len(userServices))
	}

	expected := googleauth.UserServices()
	if len(allServices) != len(expected) {
		t.Fatalf("expected %d services, got %d", len(expected), len(allServices))
	}

	// Empty should also return UserServices
	emptyServices, err := parseAuthServices("")
	if err != nil {
		t.Fatalf("parseAuthServices(''): %v", err)
	}
	if len(emptyServices) != len(expected) {
		t.Fatalf("expected %d services for empty, got %d", len(expected), len(emptyServices))
	}
}

// TestAuthAddCmd_PollTimeoutSoftFailure_JSON verifies that poll timeout
// exits 0 and outputs stored=false in JSON mode.
func TestAuthAddCmd_PollTimeoutSoftFailure_JSON(t *testing.T) {
	origHeadless := headlessAuthorize
	origPoll := pollForToken
	origCallback := callbackServerURLFn
	t.Cleanup(func() {
		headlessAuthorize = origHeadless
		pollForToken = origPoll
		callbackServerURLFn = origCallback
	})

	callbackServerURLFn = func(s string) (string, error) {
		return "https://callback.example.com", nil
	}

	headlessAuthorize = func(_ context.Context, _ googleauth.HeadlessOptions) (googleauth.HeadlessAuthInfo, error) {
		return googleauth.HeadlessAuthInfo{
			AuthURL:   "https://accounts.google.com/o/oauth2/auth",
			State:     "timeout-state",
			PollURL:   "https://callback.example.com/poll/timeout-state",
			ExpiresIn: 600,
		}, nil
	}

	pollForToken = func(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
		return "", googleauth.NewPollTimeoutError()
	}

	cmd := &AuthAddCmd{
		Email:       "user@example.com",
		Headless:    true,
		ServicesCSV: "gmail",
	}

	var stderrBuf strings.Builder
	u, uiErr := ui.New(ui.Options{Stdout: nil, Stderr: &stderrBuf, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})

	stdout := captureStdout(t, func() {
		err := cmd.Run(ctx, &RootFlags{})
		if err != nil {
			t.Fatalf("expected nil error on timeout (soft failure), got: %v", err)
		}
	})

	// runHeadless outputs two JSON objects: initial auth info, then timeout result.
	// Parse the last JSON object which is the timeout response.
	dec := json.NewDecoder(strings.NewReader(stdout))
	var last map[string]any
	for dec.More() {
		var obj map[string]any
		if jsonErr := dec.Decode(&obj); jsonErr != nil {
			t.Fatalf("json decode: %v\nout=%q", jsonErr, stdout)
		}
		last = obj
	}
	if last == nil {
		t.Fatalf("no JSON objects in output: %q", stdout)
	}
	if stored, ok := last["stored"].(bool); !ok || stored {
		t.Fatalf("expected stored=false, got %v", last["stored"])
	}
	if state, ok := last["state"].(string); !ok || state != "timeout-state" {
		t.Fatalf("expected state=timeout-state, got %v", last["state"])
	}
	if timeout, ok := last["timeout"].(bool); !ok || !timeout {
		t.Fatalf("expected timeout=true, got %v", last["timeout"])
	}
}

// TestAuthAddCmd_PollTimeoutSoftFailure_Text verifies that poll timeout
// exits 0 and prints hints in text mode.
func TestAuthAddCmd_PollTimeoutSoftFailure_Text(t *testing.T) {
	origHeadless := headlessAuthorize
	origPoll := pollForToken
	origCallback := callbackServerURLFn
	t.Cleanup(func() {
		headlessAuthorize = origHeadless
		pollForToken = origPoll
		callbackServerURLFn = origCallback
	})

	callbackServerURLFn = func(s string) (string, error) {
		return "https://callback.example.com", nil
	}

	headlessAuthorize = func(_ context.Context, _ googleauth.HeadlessOptions) (googleauth.HeadlessAuthInfo, error) {
		return googleauth.HeadlessAuthInfo{
			AuthURL:   "https://accounts.google.com/o/oauth2/auth",
			State:     "timeout-state-2",
			PollURL:   "https://callback.example.com/poll/timeout-state-2",
			ExpiresIn: 600,
		}, nil
	}

	pollForToken = func(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
		return "", googleauth.NewPollTimeoutError()
	}

	cmd := &AuthAddCmd{
		Email:       "user@example.com",
		Headless:    true,
		ServicesCSV: "gmail",
	}

	var stderrBuf strings.Builder
	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: &stderrBuf, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)

	err := cmd.Run(ctx, &RootFlags{})
	if err != nil {
		t.Fatalf("expected nil error on timeout (soft failure), got: %v", err)
	}

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "timed out") {
		t.Fatalf("expected timeout message in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "timeout-state-2") {
		t.Fatalf("expected state in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "wk auth poll") {
		t.Fatalf("expected poll hint in stderr, got: %q", stderr)
	}
}

// TestAuthPollCmd_HintSaysSaveToken verifies that the --no-poll hint in
// runHeadless says "save the token" not "retrieve the token".
func TestAuthPollCmd_HintSaysSaveToken(t *testing.T) {
	origHeadless := headlessAuthorize
	origCallback := callbackServerURLFn
	t.Cleanup(func() {
		headlessAuthorize = origHeadless
		callbackServerURLFn = origCallback
	})

	callbackServerURLFn = func(s string) (string, error) {
		return "https://callback.example.com", nil
	}

	headlessAuthorize = func(_ context.Context, _ googleauth.HeadlessOptions) (googleauth.HeadlessAuthInfo, error) {
		return googleauth.HeadlessAuthInfo{
			AuthURL:   "https://accounts.google.com/o/oauth2/auth",
			State:     "test-state",
			PollURL:   "https://callback.example.com/poll/test-state",
			ExpiresIn: 600,
		}, nil
	}

	cmd := &AuthAddCmd{
		Email:       "user@example.com",
		Headless:    true,
		NoPoll:      true,
		ServicesCSV: "gmail",
	}

	var stderrBuf strings.Builder
	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: &stderrBuf, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)

	// Run the command. It may fail (e.g., auth mode resolution) but we care about stderr.
	_ = cmd.Run(ctx, &RootFlags{})

	stderr := stderrBuf.String()
	if strings.Contains(stderr, "retrieve the token") {
		t.Fatalf("hint should NOT say 'retrieve the token', got: %q", stderr)
	}
	if !strings.Contains(stderr, "save the token") {
		t.Fatalf("hint should say 'save the token', got: %q", stderr)
	}
}
