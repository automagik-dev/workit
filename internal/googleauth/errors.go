package googleauth

import (
	"fmt"
	"strings"
)

// WrapOAuthError appends a human-readable hint to known Google OAuth error codes.
// The original error is preserved via %w for unwrapping.
func WrapOAuthError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unauthorized_client"):
		return fmt.Errorf("%w (hint: refresh token expired — re-run 'gog auth add <email>')", err)
	case strings.Contains(msg, "invalid_grant"):
		return fmt.Errorf("%w (hint: token revoked or invalid — re-run 'gog auth add <email>')", err)
	case strings.Contains(msg, "invalid_client"):
		return fmt.Errorf("%w (hint: client_id/secret invalid — check 'gog auth credentials list')", err)
	}
	return err
}
