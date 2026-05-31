package msauth

import (
	"errors"
	"fmt"
	"strings"
)

var ErrPilotScopeNotAllowed = errors.New("m365 pilot scope not allowed")

var pilotAllowedScopes = []string{
	"User.Read",
	"Mail.Read",
	"Calendars.Read",
}

func canonicalPilotScope(raw string) (string, bool) {
	for _, scope := range pilotAllowedScopes {
		if strings.EqualFold(raw, scope) {
			return scope, true
		}
	}

	return "", false
}

// PilotAllowedScopes returns a defensive copy of the only Microsoft Graph scopes
// Hapvida/Bernardo's M365 pilot may request. The pilot is intentionally read-only.
func PilotAllowedScopes() []string {
	out := make([]string, len(pilotAllowedScopes))
	copy(out, pilotAllowedScopes)

	return out
}

// GuardPilotScopes normalizes and validates requested Microsoft Graph scopes for
// the M365 pilot. Empty input means the default read-only pilot baseline. Any
// scope outside the explicit allowlist is rejected fail-closed, including known
// write scopes and unknown/ambiguous scopes.
func GuardPilotScopes(requested []string) ([]string, error) {
	if len(requested) == 0 {
		return PilotAllowedScopes(), nil
	}

	seen := make(map[string]bool, len(requested))
	for _, raw := range requested {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		canonical, allowed := canonicalPilotScope(trimmed)
		if !allowed {
			return nil, fmt.Errorf("%w: %s", ErrPilotScopeNotAllowed, trimmed)
		}
		seen[canonical] = true
	}

	out := make([]string, 0, len(pilotAllowedScopes))
	for _, scope := range pilotAllowedScopes {
		if seen[scope] {
			out = append(out, scope)
		}
	}

	if len(out) == 0 {
		return PilotAllowedScopes(), nil
	}

	return out, nil
}
