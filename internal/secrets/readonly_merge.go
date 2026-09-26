package secrets

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/99designs/keyring"
)

// WriteScopedTokenError reports that a read-only authorization would merge
// into, or overwrite, a stored token whose recorded scopes include write
// scopes.
type WriteScopedTokenError struct {
	Client      string
	Email       string
	WriteScopes []string
}

func (e *WriteScopedTokenError) Error() string {
	return fmt.Sprintf(
		"refusing to store a read-only token over the existing token for %s (client %s), which holds write scopes (%s); "+
			"use a dedicated --client (e.g. --client brain-ro) for read-only access",
		e.Email, e.Client, strings.Join(e.WriteScopes, ", "),
	)
}

// CheckReadOnlyMerge returns a *WriteScopedTokenError when the token stored
// for client/email records a scope that isWriteScope reports as write-capable.
// No stored token (first authorization for this client+email) is not an
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

	for _, scope := range existing.Scopes {
		if isWriteScope(scope) {
			write = append(write, scope)
		}
	}

	if len(write) == 0 {
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

	return &WriteScopedTokenError{Client: name, Email: addr, WriteScopes: write}
}

// MergeReadOnlyToken stores tok, the result of a read-only authorization, like
// Store.MergeToken, but refuses (with a *WriteScopedTokenError) when the token
// already stored for client/email holds write scopes. Merging there would
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
