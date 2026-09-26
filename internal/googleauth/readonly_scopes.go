package googleauth

import "strings"

// readOnlyScopeSuffix marks a Google OAuth scope as read-only by name
// (e.g. drive.readonly, spreadsheets.readonly, gmail.readonly).
const readOnlyScopeSuffix = ".readonly"

// readOnlyIdentityScopes are the OIDC / identity scopes that grant no write
// access to user data.
var readOnlyIdentityScopes = map[string]struct{}{
	"openid":  {},
	"email":   {},
	"profile": {},
	"https://www.googleapis.com/auth/userinfo.email":   {},
	"https://www.googleapis.com/auth/userinfo.profile": {},
}

// IsReadOnlyScope reports whether scope is on the explicit read-only
// allowlist: any scope whose name ends in ".readonly", plus the OIDC identity
// scopes (openid, email, profile, userinfo.email, userinfo.profile).
//
// Everything else is treated as write-capable, including scopes that are
// read-only in practice but not named so (for example drive.metadata or
// gmail.metadata). The allowlist is deliberately conservative: a caller that
// needs a least-privilege guarantee must never be told "read-only" for a
// scope it cannot prove is read-only.
func IsReadOnlyScope(scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return true
	}

	if _, ok := readOnlyIdentityScopes[scope]; ok {
		return true
	}

	return strings.HasSuffix(scope, readOnlyScopeSuffix)
}
