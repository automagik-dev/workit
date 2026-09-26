package cmd

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/automagik-dev/workit/internal/googleauth"
	"github.com/automagik-dev/workit/internal/secrets"
)

var writeScopedToken = secrets.Token{
	Services: []string{"drive", "gmail"},
	Scopes: []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/gmail.modify",
		"openid",
	},
	RefreshToken: "rt-write",
}

// readonlyAuthHarness stubs the OAuth flow and the secrets store for
// `auth add` tests. authorizeCalls counts browser/manual authorizations.
type readonlyAuthHarness struct {
	store          *memSecretsStore
	authorizeCalls int
	lastOpts       googleauth.AuthorizeOptions
	headlessCalls  int
	lastHeadless   googleauth.HeadlessOptions
	manualURLCalls int
	lastManualURL  googleauth.AuthorizeOptions
}

func newReadonlyAuthHarness(t *testing.T) *readonlyAuthHarness {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	writeAuthModeConfig(t, "browser")

	origAuth := authorizeGoogle
	origOpen := openSecretsStore
	origKeychain := ensureKeychainAccess
	origFetch := fetchAuthorizedEmail
	origHeadless := headlessAuthorize
	origManualURL := manualAuthURL
	origPoll := pollForToken
	origCallback := callbackServerURLFn

	t.Cleanup(func() {
		authorizeGoogle = origAuth
		openSecretsStore = origOpen
		ensureKeychainAccess = origKeychain
		fetchAuthorizedEmail = origFetch
		headlessAuthorize = origHeadless
		manualAuthURL = origManualURL
		pollForToken = origPoll
		callbackServerURLFn = origCallback
	})

	h := &readonlyAuthHarness{store: newMemSecretsStore()}

	ensureKeychainAccess = func() error { return nil }
	openSecretsStore = func() (secrets.Store, error) { return h.store, nil }
	authorizeGoogle = func(_ context.Context, opts googleauth.AuthorizeOptions) (string, error) {
		h.authorizeCalls++
		h.lastOpts = opts

		return "rt-ro", nil
	}
	fetchAuthorizedEmail = func(context.Context, string, string, []string, time.Duration) (string, error) {
		return "user@example.com", nil
	}
	headlessAuthorize = func(_ context.Context, opts googleauth.HeadlessOptions) (googleauth.HeadlessAuthInfo, error) {
		h.headlessCalls++
		h.lastHeadless = opts

		return googleauth.HeadlessAuthInfo{
			AuthURL:   "https://accounts.example.com/auth",
			State:     "state",
			PollURL:   "https://relay.example.com/token/state",
			ExpiresIn: 300,
		}, nil
	}
	pollForToken = func(context.Context, string, string, time.Duration) (string, error) {
		return "rt-ro", nil
	}
	callbackServerURLFn = func(override string) (string, error) {
		if override != "" {
			return override, nil
		}

		return "https://relay.example.com", nil
	}
	manualAuthURL = func(_ context.Context, opts googleauth.AuthorizeOptions) (googleauth.ManualAuthURLResult, error) {
		h.manualURLCalls++
		h.lastManualURL = opts

		return googleauth.ManualAuthURLResult{URL: "https://accounts.example.com/auth?manual=1"}, nil
	}

	return h
}

func writeAuthModeConfig(t *testing.T, mode string) {
	t.Helper()

	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "workit")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"auth_mode":"`+mode+`"}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func runAuthAdd(t *testing.T, args ...string) error {
	t.Helper()

	var err error

	_ = captureStdout(t, func() {
		_ = captureStderr(t, func() {
			err = Execute(append([]string{"--json"}, args...))
		})
	})

	return err
}

func TestAuthAdd_Readonly_PassesReadonlyToConsentURL(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if h.authorizeCalls != 1 || !h.lastOpts.Readonly {
		t.Fatalf("expected one read-only authorization, got calls=%d opts=%+v", h.authorizeCalls, h.lastOpts)
	}

	if err := runAuthAdd(t, "--client", "rw", "auth", "add", "user@example.com", "--services", "drive"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if h.lastOpts.Readonly {
		t.Fatalf("a full authorization must keep include_granted_scopes (Readonly=false)")
	}
}

func TestAuthAdd_Readonly_RefusesWriteScopedTokenBeforeOAuth(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := h.store.SetToken("default", "user@example.com", writeScopedToken); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly")
	if got := ExitCode(err); got != 2 {
		t.Fatalf("exit = %d, want 2 (err=%v)", got, err)
	}

	if !strings.Contains(err.Error(), "use a dedicated --client (e.g. --client brain-ro)") {
		t.Fatalf("error must name the fix, got %v", err)
	}

	if h.authorizeCalls != 0 {
		t.Fatalf("OAuth must not start when the merge would be refused (calls=%d)", h.authorizeCalls)
	}

	got, getErr := h.store.GetToken("default", "user@example.com")
	if getErr != nil {
		t.Fatalf("GetToken: %v", getErr)
	}

	if got.RefreshToken != "rt-write" || !reflect.DeepEqual(got.Scopes, writeScopedToken.Scopes) {
		t.Fatalf("existing token must be untouched, got %+v", got)
	}
}

// A token brought in by `auth tokens import` can record no scopes. It is not
// provably read-only, so a --readonly add refuses it like a write-scoped one.
func TestAuthAdd_Readonly_RefusesTokenWithoutRecordedScopesBeforeOAuth(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	imported := secrets.Token{Services: []string{"drive"}, RefreshToken: "rt-imported"}
	if err := h.store.SetToken("default", "user@example.com", imported); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly")
	if got := ExitCode(err); got != 2 {
		t.Fatalf("exit = %d, want 2 (err=%v)", got, err)
	}

	if !strings.Contains(err.Error(), "use a dedicated --client (e.g. --client brain-ro)") {
		t.Fatalf("error must name the fix, got %v", err)
	}

	if h.authorizeCalls != 0 {
		t.Fatalf("OAuth must not start when the merge would be refused (calls=%d)", h.authorizeCalls)
	}

	got, getErr := h.store.GetToken("default", "user@example.com")
	if getErr != nil || got.RefreshToken != "rt-imported" {
		t.Fatalf("existing token must be untouched, got %+v err=%v", got, getErr)
	}
}

func TestAuthAdd_Readonly_DedicatedClientIsAccepted(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := h.store.SetToken("default", "user@example.com", writeScopedToken); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	if err := runAuthAdd(t, "--client", "brain-ro", "auth", "add", "user@example.com", "--services", "drive", "--readonly"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	ro, err := h.store.GetToken("brain-ro", "user@example.com")
	if err != nil {
		t.Fatalf("GetToken brain-ro: %v", err)
	}

	if ro.RefreshToken != "rt-ro" {
		t.Fatalf("brain-ro refresh token = %q", ro.RefreshToken)
	}

	rw, err := h.store.GetToken("default", "user@example.com")
	if err != nil || rw.RefreshToken != "rt-write" {
		t.Fatalf("default token must be untouched, got %+v err=%v", rw, err)
	}
}

func TestAuthAdd_Readonly_MergesIntoReadOnlyToken(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := h.store.SetToken("default", "user@example.com", secrets.Token{
		Services:     []string{"sheets"},
		Scopes:       []string{"https://www.googleapis.com/auth/spreadsheets.readonly", "openid"},
		RefreshToken: "rt-old",
	}); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	if err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, err := h.store.GetToken("default", "user@example.com")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}

	if got.RefreshToken != "rt-ro" || !reflect.DeepEqual(got.Services, []string{"drive", "sheets"}) {
		t.Fatalf("expected merged read-only token, got %+v", got)
	}
}

func TestAuthAdd_ReadonlyRemoteStep1_RefusesWriteScopedToken(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly", "--remote", "--step", "1"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if h.manualURLCalls != 1 || !h.lastManualURL.Readonly {
		t.Fatalf("expected a read-only remote URL, got calls=%d opts=%+v", h.manualURLCalls, h.lastManualURL)
	}

	if err := h.store.SetToken("default", "user@example.com", writeScopedToken); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly", "--remote", "--step", "1")
	if got := ExitCode(err); got != 2 {
		t.Fatalf("exit = %d, want 2 (err=%v)", got, err)
	}

	if h.manualURLCalls != 1 {
		t.Fatalf("remote step 1 must not print a URL when the merge would be refused")
	}
}

func TestAuthAdd_ReadonlyHeadless_RefusesWriteScopedTokenAndDropsIncludeGranted(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly", "--headless"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if h.headlessCalls != 1 || !h.lastHeadless.Readonly {
		t.Fatalf("expected a read-only headless URL, got calls=%d opts=%+v", h.headlessCalls, h.lastHeadless)
	}

	if err := h.store.SetToken("other", "user@example.com", writeScopedToken); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	err := runAuthAdd(t, "--client", "other", "auth", "add", "user@example.com", "--services", "drive", "--readonly", "--headless")
	if got := ExitCode(err); got != 2 {
		t.Fatalf("exit = %d, want 2 (err=%v)", got, err)
	}

	if h.headlessCalls != 1 {
		t.Fatalf("headless flow must not start when the merge would be refused")
	}
}

func TestAuthPoll_Readonly_RefusesWriteScopedToken(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := h.store.SetToken("default", "user@example.com", writeScopedToken); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	err := runAuthAdd(t, "auth", "poll", "state", "--email", "user@example.com", "--services", "drive", "--readonly")
	if got := ExitCode(err); got != 2 {
		t.Fatalf("exit = %d, want 2 (err=%v)", got, err)
	}

	got, getErr := h.store.GetToken("default", "user@example.com")
	if getErr != nil || got.RefreshToken != "rt-write" {
		t.Fatalf("existing token must be untouched, got %+v err=%v", got, getErr)
	}
}

func TestAuthAdd_NoRelay_ForcesLoopbackOverConfigHeadless(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	writeAuthModeConfig(t, "headless")
	t.Setenv("WK_CALLBACK_SERVER", "https://relay.example.com")

	// Control: config auth_mode=headless takes the relay.
	if err := runAuthAdd(t, "--client", "control", "auth", "add", "user@example.com", "--services", "drive", "--readonly"); err != nil {
		t.Fatalf("Execute (control): %v", err)
	}

	if h.headlessCalls != 1 || h.authorizeCalls != 0 {
		t.Fatalf("control run should use the relay, got headless=%d authorize=%d", h.headlessCalls, h.authorizeCalls)
	}

	if err := runAuthAdd(t, "--client", "brain-ro", "auth", "add", "user@example.com", "--services", "drive", "--readonly", "--no-relay"); err != nil {
		t.Fatalf("Execute (--no-relay): %v", err)
	}

	if h.headlessCalls != 1 {
		t.Fatalf("--no-relay must not use the relay (headless calls=%d)", h.headlessCalls)
	}

	if h.authorizeCalls != 1 || h.lastOpts.Manual {
		t.Fatalf("--no-relay must use the loopback browser flow, got calls=%d opts=%+v", h.authorizeCalls, h.lastOpts)
	}
}

func TestAuthAdd_NoRelay_RejectsRelayFlags(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	for _, extra := range [][]string{
		{"--headless"},
		{"--callback-server", "https://relay.example.com"},
		{"--no-poll"},
	} {
		args := append([]string{"auth", "add", "user@example.com", "--services", "drive", "--readonly", "--no-relay"}, extra...)

		err := runAuthAdd(t, args...)
		if got := ExitCode(err); got != 2 {
			t.Fatalf("%v: exit = %d, want 2 (err=%v)", extra, got, err)
		}

		if !strings.Contains(err.Error(), "--no-relay") {
			t.Fatalf("%v: error should name --no-relay, got %v", extra, err)
		}
	}

	if h.headlessCalls != 0 || h.authorizeCalls != 0 {
		t.Fatalf("no flow may start, got headless=%d authorize=%d", h.headlessCalls, h.authorizeCalls)
	}
}

func TestAuthAdd_NoRelay_KeepsManual(t *testing.T) {
	h := newReadonlyAuthHarness(t)

	if err := runAuthAdd(t, "auth", "add", "user@example.com", "--services", "drive", "--readonly", "--no-relay", "--manual"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if h.authorizeCalls != 1 || !h.lastOpts.Manual || h.headlessCalls != 0 {
		t.Fatalf("expected the manual flow, got calls=%d opts=%+v headless=%d", h.authorizeCalls, h.lastOpts, h.headlessCalls)
	}
}
