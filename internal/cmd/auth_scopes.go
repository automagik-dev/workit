package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/99designs/keyring"

	"github.com/automagik-dev/workit/internal/googleauth"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

var inspectGrantedScopes = googleauth.InspectGrantedScopes

// AuthScopesCmd reports the OAuth scopes Google actually granted to the stored
// refresh token of the selected --client/--account. The scopes recorded in the
// keyring at `auth add` time are what was requested; Google can grant more
// (for example through incremental authorization), so least-privilege checks
// must read the granted set.
type AuthScopesCmd struct {
	Timeout time.Duration `name:"timeout" help:"Token refresh timeout" default:"15s"`
}

func (c *AuthScopesCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	email := normalizeEmail(account)

	client, err := resolveClientForEmail(email, flags, "")
	if err != nil {
		return err
	}

	store, err := openSecretsStore()
	if err != nil {
		return err
	}

	tok, err := store.GetToken(client, email)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return &ExitError{
				Code: exitCodeAuthRequired,
				Err:  fmt.Errorf("no stored token for %s (client %s); run: wk --client %s auth add %s: %w", email, client, client, email, err),
			}
		}

		return err
	}

	report, err := inspectGrantedScopes(ctx, client, tok.RefreshToken, c.Timeout)
	if err != nil {
		if errors.Is(err, googleauth.ErrRefreshTokenRejected) {
			return &ExitError{
				Code: exitCodeAuthRequired,
				Err:  fmt.Errorf("stored token for %s (client %s) no longer works; re-authorize with: wk --client %s auth add %s --force-consent: %w", email, client, client, email, err),
			}
		}

		return googleauth.WrapOAuthError(err)
	}

	granted := report.Granted
	if granted == nil {
		granted = []string{}
	}

	writeScopes := report.WriteScopes
	if writeScopes == nil {
		writeScopes = []string{}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"client":       client,
			"email":        email,
			"granted":      granted,
			"write_scopes": writeScopes,
			"read_only":    len(writeScopes) == 0,
			"source":       report.Source,
		})
	}

	u.Out().Printf("client\t%s", client)
	u.Out().Printf("email\t%s", email)
	u.Out().Printf("read_only\t%t", len(writeScopes) == 0)
	u.Out().Printf("source\t%s", report.Source)

	for _, scope := range granted {
		kind := "read"
		if !googleauth.IsReadOnlyScope(scope) {
			kind = "write"
		}

		u.Out().Printf("granted\t%s\t%s", kind, scope)
	}

	return nil
}
