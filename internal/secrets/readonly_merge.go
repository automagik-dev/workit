package secrets

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/99designs/keyring"
)

// dedicatedClientHint is the fix both refusals name: a separate --client keeps
// the read-only token out of the existing token's slot.
const dedicatedClientHint = "use a dedicated --client (e.g. --client brain-ro) for read-only access"

// WriteScopedTokenError reports that a read-only authorization would merge
// into, or overwrite, a stored token that is not provably read-only: its
// recorded scopes include write scopes, or it records no scopes at all
// (ScopesUnknown), as a token brought in by `auth tokens import` can.
type WriteScopedTokenError struct {
	Client        string
	Email         string
	WriteScopes   []string
	ScopesUnknown bool
}

func (e *WriteScopedTokenError) Error() string {
	if e.ScopesUnknown {
		return fmt.Sprintf(
			"refusing to store a read-only token over the existing token for %s (client %s), which records no scopes, "+
				"so it may hold write access; %s",
			e.Email, e.Client, dedicatedClientHint,
		)
	}

	return fmt.Sprintf(
		"refusing to store a read-only token over the existing token for %s (client %s), which holds write scopes (%s); %s",
		e.Email, e.Client, strings.Join(e.WriteScopes, ", "), dedicatedClientHint,
	)
}

// CheckReadOnlyMerge returns a *WriteScopedTokenError when the token stored
// for client/email is not provably read-only: it records a scope that
// isWriteScope reports as write-capable, or it records no scopes, so what it
// can do is unknown. No stored token (first authorization for this client+email) is not an
// error. Any other read failure is returned, so a keyring problem is never
// mistaken for "no token".
func CheckReadOnlyMerge(s Store, client string, email string, isWriteScope func(string) bool) error {
	existing, err := s.GetToken(client, email)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil
		}

		return fmt.Errorf("read existing token: %w", err)
	}

	var write []string

	recorded := 0

	for _, scope := range existing.Scopes {
		if strings.TrimSpace(scope) == "" {
			continue
		}

		recorded++

		if isWriteScope(scope) {
			write = append(write, scope)
		}
	}

	if recorded > 0 && len(write) == 0 {
		return nil
	}

	sort.Strings(write)

	name := existing.Client
	if strings.TrimSpace(name) == "" {
		name = client
	}

	addr := existing.Email
	if strings.TrimSpace(addr) == "" {
		addr = email
	}

	return &WriteScopedTokenError{Client: name, Email: addr, WriteScopes: write, ScopesUnknown: recorded == 0}
}

// MergeReadOnlyToken stores tok, the result of a read-only authorization, like
// Store.MergeToken, but refuses (with a *WriteScopedTokenError) when the token
// already stored for client/email holds write scopes or records none. Merging there would
// replace a working write-capable refresh token with a read-only one while
// the recorded scopes still claimed write access.
func MergeReadOnlyToken(s Store, client string, email string, tok Token, isWriteScope func(string) bool) error {
	if err := CheckReadOnlyMerge(s, client, email, isWriteScope); err != nil {
		return err
	}

	if err := s.MergeToken(client, email, tok); err != nil {
		return fmt.Errorf("store read-only token: %w", err)
	}

	return nil
}
