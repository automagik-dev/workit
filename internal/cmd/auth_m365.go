package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/automagik-dev/workit/internal/msauth"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/secrets"
	"github.com/automagik-dev/workit/internal/ui"
)

func isM365ServicesCSV(value string) bool {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if strings.ToLower(strings.TrimSpace(part)) != "m365" {
			return false
		}
	}
	return true
}

func (c *AuthAddCmd) runM365(ctx context.Context, flags *RootFlags, u *ui.UI) error {
	if !c.Readonly {
		return usage("m365 auth requires explicit --readonly")
	}
	if c.Headless || c.NoPoll || c.CallbackServer != "" {
		return usage("m365 auth uses browser OAuth; headless callback-server mode is not supported yet")
	}
	if c.AuthCode != "" {
		return usage("m365 auth does not accept raw --auth-code; use browser OAuth")
	}
	if c.Step != 0 && c.Step != 1 && c.Step != 2 {
		return usage("step must be 1 or 2")
	}
	if c.Step != 0 && !c.Remote {
		return usage("--step requires --remote")
	}
	if c.Remote || c.Step != 0 || c.AuthURL != "" {
		return usage("m365 remote auth is not supported yet; use browser OAuth on this machine")
	}
	if dryRunErr := dryRunExit(ctx, flags, "auth.add.m365", map[string]any{
		"email":    strings.TrimSpace(c.Email),
		"provider": "microsoft_graph",
		"services": []string{"m365"},
		"scopes":   msauth.PilotAllowedScopes(),
		"readonly": c.Readonly,
	}); dryRunErr != nil {
		return dryRunErr
	}
	if keychainErr := ensureKeychainAccessIfNeeded(); keychainErr != nil {
		return fmt.Errorf("keychain access: %w", keychainErr)
	}
	result, err := authorizeM365(ctx, msauth.AuthorizeOptions{
		ExpectedEmail: strings.TrimSpace(c.Email),
		Readonly:      c.Readonly,
		ForceConsent:  c.ForceConsent,
		Timeout:       c.Timeout,
	})
	if err != nil {
		return err
	}
	if normalizeEmail(result.Email) != normalizeEmail(c.Email) {
		return fmt.Errorf("authorized as %s, expected %s", result.Email, c.Email)
	}
	return storeM365Token(ctx, u, result.Email, result.RefreshToken)
}

func (c *AuthManageCmd) runM365(ctx context.Context) error {
	if !outfmt.IsJSON(ctx) && !c.PrintURL {
		return usage("m365 auth manage requires --print-url")
	}

	result, err := m365ManualAuthURL(ctx, msauth.ManualAuthURLOptions{Readonly: true, ForceConsent: c.ForceConsent})
	if err != nil {
		return err
	}
	if outfmt.IsJSON(ctx) || c.PrintURL {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"provider":   "microsoft_graph",
			"auth_url":   result.URL,
			"state":      result.State,
			"expires_in": result.ExpiresIn,
		})
	}
	return nil
}

func storeM365Token(ctx context.Context, u *ui.UI, email string, refreshToken string) error {
	store, err := openSecretsStore()
	if err != nil {
		return err
	}
	serviceNames := []string{"m365"}
	scopes := append([]string(nil), msauth.PilotAllowedScopes()...)
	sort.Strings(scopes)
	if err := store.MergeToken(msauth.ClientName, email, secrets.Token{
		Client:       msauth.ClientName,
		Email:        email,
		Services:     serviceNames,
		Scopes:       scopes,
		RefreshToken: refreshToken,
	}); err != nil {
		return err
	}
	return writeResult(ctx, u,
		kv("stored", true),
		kv("provider", "microsoft_graph"),
		kv("email", email),
		kv("services", serviceNames),
		kv("client", msauth.ClientName),
	)
}
