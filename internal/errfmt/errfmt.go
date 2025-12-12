package errfmt

import (
	"errors"
	"fmt"
	"os"

	"github.com/99designs/keyring"
	ggoogleapi "google.golang.org/api/googleapi"

	"github.com/steipete/gogcli/internal/config"
	gogapi "github.com/steipete/gogcli/internal/googleapi"
)

func Format(err error) string {
	if err == nil {
		return ""
	}

	var authErr *gogapi.AuthRequiredError
	if errors.As(err, &authErr) {
		return fmt.Sprintf("No refresh token for %s %s. Run: gog auth add %s --services %s", authErr.Service, authErr.Email, authErr.Email, authErr.Service)
	}

	var scopesErr *gogapi.MissingScopesError
	if errors.As(err, &scopesErr) {
		return fmt.Sprintf("Missing scopes for %s %s. Re-run: gog auth add %s --services %s --force-consent", scopesErr.Service, scopesErr.Email, scopesErr.Email, scopesErr.Service)
	}

	var credErr *config.CredentialsMissingError
	if errors.As(err, &credErr) {
		return fmt.Sprintf("OAuth credentials missing. Run: gog auth credentials <credentials.json> (expected at %s)", credErr.Path)
	}

	if errors.Is(err, keyring.ErrKeyNotFound) {
		return "Secret not found in keyring (refresh token missing). Run: gog auth add <email>"
	}
	if errors.Is(err, os.ErrNotExist) {
		return err.Error()
	}

	var gerr *ggoogleapi.Error
	if errors.As(err, &gerr) {
		reason := ""
		if len(gerr.Errors) > 0 && gerr.Errors[0].Reason != "" {
			reason = gerr.Errors[0].Reason
		}
		if reason != "" {
			return fmt.Sprintf("Google API error (%d %s): %s", gerr.Code, reason, gerr.Message)
		}
		return fmt.Sprintf("Google API error (%d): %s", gerr.Code, gerr.Message)
	}

	return err.Error()
}
