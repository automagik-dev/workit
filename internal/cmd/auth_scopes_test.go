package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/automagik-dev/workit/internal/googleauth"
	"github.com/automagik-dev/workit/internal/secrets"
)

// stubAuthScopes installs an in-memory secrets store holding one token for
// client+email and a fake granted-scope inspector. inspect receives the client
// and refresh token the command resolved.
func stubAuthScopes(t *testing.T, client string, email string, inspect func(client string, refreshToken string) (googleauth.GrantedScopesReport, error)) {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("WK_ACCOUNT", "")

	origOpen := openSecretsStore
	origInspect := inspectGrantedScopes

	t.Cleanup(func() {
		openSecretsStore = origOpen
		inspectGrantedScopes = origInspect
	})

	store := newMemSecretsStore()
	if email != "" {
		if err := store.SetToken(client, email, secrets.Token{
			Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
			RefreshToken: "rt-" + client,
		}); err != nil {
			t.Fatalf("SetToken: %v", err)
		}
	}

	openSecretsStore = func() (secrets.Store, error) { return store, nil }
	inspectGrantedScopes = func(_ context.Context, client string, refreshToken string, _ time.Duration) (googleauth.GrantedScopesReport, error) {
		return inspect(client, refreshToken)
	}
}

func grantedReport(scopes ...string) googleauth.GrantedScopesReport {
	write := googleauth.WriteScopes(scopes)

	return googleauth.GrantedScopesReport{
		Granted:     scopes,
		WriteScopes: write,
		ReadOnly:    len(write) == 0,
		Source:      googleauth.GrantedScopesSourceTokenEndpoint,
	}
}

type authScopesPayload struct {
	Client      string   `json:"client"`
	Email       string   `json:"email"`
	Granted     []string `json:"granted"`
	WriteScopes []string `json:"write_scopes"`
	ReadOnly    bool     `json:"read_only"`
	Source      string   `json:"source"`
}

func TestAuthScopes_JSON_ReadOnlyClient(t *testing.T) {
	var gotClient, gotRefresh string

	stubAuthScopes(t, "brain-ro", "a@b.com", func(client string, refreshToken string) (googleauth.GrantedScopesReport, error) {
		gotClient, gotRefresh = client, refreshToken

		return grantedReport(
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/userinfo.email",
			"openid",
		), nil
	})

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "--client", "brain-ro", "--account", "A@B.com", "auth", "scopes"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	if gotClient != "brain-ro" || gotRefresh != "rt-brain-ro" {
		t.Fatalf("inspected client=%q refresh=%q", gotClient, gotRefresh)
	}

	var payload authScopesPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}

	if payload.Client != "brain-ro" || payload.Email != "a@b.com" || !payload.ReadOnly {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	if payload.WriteScopes == nil || len(payload.WriteScopes) != 0 {
		t.Fatalf("write_scopes must be an empty array, got %#v (out=%q)", payload.WriteScopes, out)
	}

	if !strings.Contains(out, `"write_scopes": []`) {
		t.Fatalf("write_scopes must serialize as [], out=%q", out)
	}

	if len(payload.Granted) != 3 {
		t.Fatalf("granted = %v", payload.Granted)
	}
}

func TestAuthScopes_JSON_FullDriveIsNotReadOnly(t *testing.T) {
	stubAuthScopes(t, "default", "a@b.com", func(string, string) (googleauth.GrantedScopesReport, error) {
		return grantedReport(
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/spreadsheets.readonly",
			"openid",
		), nil
	})

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "--account", "a@b.com", "auth", "scopes"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var payload authScopesPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}

	if payload.ReadOnly {
		t.Fatalf("expected read_only=false: %#v", payload)
	}

	if want := []string{"https://www.googleapis.com/auth/drive"}; !reflect.DeepEqual(payload.WriteScopes, want) {
		t.Fatalf("write_scopes = %v, want %v", payload.WriteScopes, want)
	}
}

func TestAuthScopes_Text(t *testing.T) {
	stubAuthScopes(t, "default", "a@b.com", func(string, string) (googleauth.GrantedScopesReport, error) {
		return grantedReport(
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/drive.readonly",
		), nil
	})

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--account", "a@b.com", "auth", "scopes"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	for _, want := range []string{
		"client\tdefault",
		"email\ta@b.com",
		"read_only\tfalse",
		"granted\twrite\thttps://www.googleapis.com/auth/drive\n",
		"granted\tread\thttps://www.googleapis.com/auth/drive.readonly\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestAuthScopes_MissingTokenExitsAuth(t *testing.T) {
	stubAuthScopes(t, "default", "", func(string, string) (googleauth.GrantedScopesReport, error) {
		t.Fatalf("inspector must not run without a stored token")
		return googleauth.GrantedScopesReport{}, nil
	})

	var err error

	_ = captureStderr(t, func() {
		err = Execute([]string{"--json", "--client", "brain-ro", "--account", "a@b.com", "auth", "scopes"})
	})

	if got := ExitCode(err); got != exitCodeAuthRequired {
		t.Fatalf("exit = %d, want %d (err=%v)", got, exitCodeAuthRequired, err)
	}

	if err == nil || !strings.Contains(err.Error(), "--client brain-ro auth add a@b.com") {
		t.Fatalf("error must name the fix, got %v", err)
	}
}

func TestAuthScopes_RevokedTokenExitsAuth(t *testing.T) {
	stubAuthScopes(t, "default", "a@b.com", func(string, string) (googleauth.GrantedScopesReport, error) {
		return googleauth.GrantedScopesReport{}, fmt.Errorf("%w: oauth2: invalid_grant", googleauth.ErrRefreshTokenRejected)
	})

	var err error

	_ = captureStderr(t, func() {
		err = Execute([]string{"--json", "--account", "a@b.com", "auth", "scopes"})
	})

	if got := ExitCode(err); got != exitCodeAuthRequired {
		t.Fatalf("exit = %d, want %d (err=%v)", got, exitCodeAuthRequired, err)
	}
}

func TestAuthScopes_OtherRefreshErrorIsNotAuth(t *testing.T) {
	stubAuthScopes(t, "default", "a@b.com", func(string, string) (googleauth.GrantedScopesReport, error) {
		return googleauth.GrantedScopesReport{}, errors.New("refresh access token: oauth2: invalid_client")
	})

	var err error

	_ = captureStderr(t, func() {
		err = Execute([]string{"--json", "--account", "a@b.com", "auth", "scopes"})
	})

	if err == nil {
		t.Fatalf("expected error")
	}

	if got := ExitCode(err); got == exitCodeAuthRequired || got == 0 {
		t.Fatalf("exit = %d, want a non-auth failure (err=%v)", got, err)
	}
}

// A stored record with an empty refresh token cannot be inspected. It must
// exit 4 like a missing token, not 1. The real inspector runs here: it
// rejects the empty token before any network call.
func TestAuthScopes_EmptyRefreshTokenExitsAuth(t *testing.T) {
	stubAuthScopes(t, "brain-ro", "", nil)
	inspectGrantedScopes = googleauth.InspectGrantedScopes

	// SetToken refuses an empty refresh token, but a keyring record decoded by
	// GetToken can still carry one, so the record is placed directly.
	store := newMemSecretsStore()
	store.tokens["brain-ro:a@b.com"] = secrets.Token{
		Client:       "brain-ro",
		Email:        "a@b.com",
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
		RefreshToken: "   ",
	}
	openSecretsStore = func() (secrets.Store, error) { return store, nil }

	var execErr error

	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			execErr = Execute([]string{"--json", "--client", "brain-ro", "--account", "a@b.com", "auth", "scopes"})
		})
	})

	if got := ExitCode(execErr); got != exitCodeAuthRequired {
		t.Fatalf("exit = %d, want %d (err=%v)", got, exitCodeAuthRequired, execErr)
	}

	if !strings.Contains(execErr.Error(), "--client brain-ro auth add a@b.com --force-consent") {
		t.Fatalf("error must name the fix, got %v", execErr)
	}

	if body := parseJSONErrorLine(t, stderr); body.Kind != "auth_required" {
		t.Fatalf("kind = %q, want auth_required", body.Kind)
	}
}

// The JSON `source` field says how the granted set was read: from the token
// endpoint's refresh response, or from tokeninfo when that response carried
// no scope field.
func TestAuthScopes_JSON_Source(t *testing.T) {
	for _, source := range []string{googleauth.GrantedScopesSourceTokenEndpoint, googleauth.GrantedScopesSourceTokenInfo} {
		t.Run(source, func(t *testing.T) {
			stubAuthScopes(t, "default", "a@b.com", func(string, string) (googleauth.GrantedScopesReport, error) {
				report := grantedReport("https://www.googleapis.com/auth/drive.readonly")
				report.Source = source

				return report, nil
			})

			out := captureStdout(t, func() {
				_ = captureStderr(t, func() {
					if err := Execute([]string{"--json", "--account", "a@b.com", "auth", "scopes"}); err != nil {
						t.Fatalf("Execute: %v", err)
					}
				})
			})

			var payload authScopesPayload
			if err := json.Unmarshal([]byte(out), &payload); err != nil {
				t.Fatalf("json parse: %v\nout=%q", err, out)
			}

			if payload.Source != source {
				t.Fatalf("source = %q, want %q (out=%q)", payload.Source, source, out)
			}
		})
	}
}
