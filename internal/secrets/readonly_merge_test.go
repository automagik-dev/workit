package secrets

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/99designs/keyring"
)

// isWriteScopeForTest mirrors the googleauth allowlist closely enough for the
// store tests: *.readonly and the OIDC scopes are read-only.
func isWriteScopeForTest(scope string) bool {
	switch scope {
	case "openid", "email", "profile",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile":
		return false
	}

	return !strings.HasSuffix(scope, ".readonly")
}

func TestMergeReadOnlyToken_FirstAuthorizationStores(t *testing.T) {
	t.Parallel()

	store := &KeyringStore{ring: keyring.NewArrayKeyring(nil)}

	err := MergeReadOnlyToken(store, "brain-ro", "a@b.com", Token{
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly", "openid"},
		RefreshToken: "rt-ro",
	}, isWriteScopeForTest)
	if err != nil {
		t.Fatalf("MergeReadOnlyToken: %v", err)
	}

	got, err := store.GetToken("brain-ro", "a@b.com")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}

	if got.RefreshToken != "rt-ro" {
		t.Fatalf("refresh token = %q", got.RefreshToken)
	}
}

func TestMergeReadOnlyToken_MergesIntoReadOnlyToken(t *testing.T) {
	t.Parallel()

	store := &KeyringStore{ring: keyring.NewArrayKeyring(nil)}
	if err := store.SetToken("brain-ro", "a@b.com", Token{
		Services:     []string{"drive"},
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly", "openid"},
		RefreshToken: "rt-old",
	}); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	err := MergeReadOnlyToken(store, "brain-ro", "a@b.com", Token{
		Services:     []string{"sheets"},
		Scopes:       []string{"https://www.googleapis.com/auth/spreadsheets.readonly"},
		RefreshToken: "rt-new",
	}, isWriteScopeForTest)
	if err != nil {
		t.Fatalf("MergeReadOnlyToken: %v", err)
	}

	got, err := store.GetToken("brain-ro", "a@b.com")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}

	if got.RefreshToken != "rt-new" {
		t.Fatalf("refresh token = %q, want rt-new", got.RefreshToken)
	}

	if want := []string{"drive", "sheets"}; !reflect.DeepEqual(got.Services, want) {
		t.Fatalf("services = %v, want %v", got.Services, want)
	}
}

func TestMergeReadOnlyToken_RefusesWriteScopedToken(t *testing.T) {
	t.Parallel()

	store := &KeyringStore{ring: keyring.NewArrayKeyring(nil)}
	existing := Token{
		Services: []string{"drive", "gmail"},
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/drive",
			"openid",
		},
		RefreshToken: "rt-write",
	}

	if err := store.SetToken("default", "a@b.com", existing); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	err := MergeReadOnlyToken(store, "default", "A@B.com", Token{
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
		RefreshToken: "rt-ro",
	}, isWriteScopeForTest)

	var wErr *WriteScopedTokenError
	if !errors.As(err, &wErr) {
		t.Fatalf("expected WriteScopedTokenError, got %v", err)
	}

	if wErr.Client != "default" || wErr.Email != "a@b.com" {
		t.Fatalf("unexpected error target: %+v", wErr)
	}

	wantWrite := []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/gmail.modify",
	}
	if !reflect.DeepEqual(wErr.WriteScopes, wantWrite) {
		t.Fatalf("write scopes = %v, want %v", wErr.WriteScopes, wantWrite)
	}

	if !strings.Contains(err.Error(), "use a dedicated --client (e.g. --client brain-ro)") {
		t.Fatalf("error must name the fix, got %q", err.Error())
	}

	got, getErr := store.GetToken("default", "a@b.com")
	if getErr != nil {
		t.Fatalf("GetToken: %v", getErr)
	}

	if got.RefreshToken != "rt-write" || !reflect.DeepEqual(got.Scopes, existing.Scopes) {
		t.Fatalf("stored token must be untouched, got %+v", got)
	}
}

func TestMergeReadOnlyToken_OtherClientIsIndependent(t *testing.T) {
	t.Parallel()

	store := &KeyringStore{ring: keyring.NewArrayKeyring(nil)}
	if err := store.SetToken("default", "a@b.com", Token{
		Scopes:       []string{"https://www.googleapis.com/auth/drive"},
		RefreshToken: "rt-write",
	}); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	if err := MergeReadOnlyToken(store, "brain-ro", "a@b.com", Token{
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
		RefreshToken: "rt-ro",
	}, isWriteScopeForTest); err != nil {
		t.Fatalf("a dedicated client must be accepted, got %v", err)
	}
}

// A stored token that records no scopes (for example one brought in by
// `auth tokens import`) is not provably read-only, so a read-only
// authorization must not overwrite it.
func TestMergeReadOnlyToken_RefusesTokenWithoutRecordedScopes(t *testing.T) {
	t.Parallel()

	for name, scopes := range map[string][]string{
		"nil":   nil,
		"empty": {},
		"blank": {"", "  "},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store := &KeyringStore{ring: keyring.NewArrayKeyring(nil)}
			existing := Token{Services: []string{"drive"}, Scopes: scopes, RefreshToken: "rt-imported"}

			if err := store.SetToken("default", "a@b.com", existing); err != nil {
				t.Fatalf("SetToken: %v", err)
			}

			err := MergeReadOnlyToken(store, "default", "a@b.com", Token{
				Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
				RefreshToken: "rt-ro",
			}, isWriteScopeForTest)

			var wErr *WriteScopedTokenError
			if !errors.As(err, &wErr) {
				t.Fatalf("expected WriteScopedTokenError, got %v", err)
			}

			if !wErr.ScopesUnknown || len(wErr.WriteScopes) != 0 {
				t.Fatalf("expected an unknown-scopes refusal, got %+v", wErr)
			}

			if !strings.Contains(err.Error(), "records no scopes") ||
				!strings.Contains(err.Error(), "use a dedicated --client (e.g. --client brain-ro)") {
				t.Fatalf("error must say why and name the fix, got %q", err.Error())
			}

			got, getErr := store.GetToken("default", "a@b.com")
			if getErr != nil {
				t.Fatalf("GetToken: %v", getErr)
			}

			if got.RefreshToken != "rt-imported" {
				t.Fatalf("stored token must be untouched, got %+v", got)
			}
		})
	}
}

type getTokenErrStore struct {
	*KeyringStore
	err error
}

func (s getTokenErrStore) GetToken(string, string) (Token, error) {
	return Token{}, s.err
}

var errKeychainLockedForTest = errors.New("keychain locked")

func TestCheckReadOnlyMerge_PropagatesReadErrors(t *testing.T) {
	t.Parallel()

	boom := errKeychainLockedForTest
	store := getTokenErrStore{KeyringStore: &KeyringStore{ring: keyring.NewArrayKeyring(nil)}, err: boom}

	if err := CheckReadOnlyMerge(store, "default", "a@b.com", isWriteScopeForTest); !errors.Is(err, boom) {
		t.Fatalf("expected read error to propagate, got %v", err)
	}
}
