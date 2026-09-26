package googleauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Granted-scope sources reported by InspectGrantedScopes.
const (
	GrantedScopesSourceTokenEndpoint = "token_endpoint"
	GrantedScopesSourceTokenInfo     = "tokeninfo"
)

// scopeInfoURL is Google's tokeninfo endpoint, used only when the token
// endpoint's refresh response carries no "scope" field. Variable for tests.
var scopeInfoURL = "https://oauth2.googleapis.com/tokeninfo"

var (
	// ErrRefreshTokenRejected reports that Google refused the stored refresh
	// token (invalid_grant): it was revoked, expired, or issued to another client.
	ErrRefreshTokenRejected = errors.New("refresh token rejected by Google (invalid_grant: revoked or expired)")

	errNoGrantedScopes = errors.New("google returned no granted scopes for this token")
	errTokenInfoStatus = errors.New("tokeninfo request failed")
)

// GrantedScopesReport describes the scopes Google actually granted to a
// refresh token, as opposed to the scopes recorded locally at `auth add` time.
type GrantedScopesReport struct {
	Granted     []string
	WriteScopes []string
	ReadOnly    bool
	Source      string
}

// InspectGrantedScopes refreshes refreshToken once for client and returns the
// scopes Google reports as granted. The token endpoint returns the granted
// scope set in the refresh response's "scope" field; when that field is
// missing, the access token is checked against the tokeninfo endpoint.
//
// A refresh the token endpoint rejects with invalid_grant is reported as
// ErrRefreshTokenRejected so callers can map it to an auth failure.
func InspectGrantedScopes(ctx context.Context, client string, refreshToken string, timeout time.Duration) (GrantedScopesReport, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return GrantedScopesReport{}, errMissingToken
	}

	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	creds, err := readClientCredentials(client)
	if err != nil {
		return GrantedScopesReport{}, fmt.Errorf("read credentials: %w", err)
	}

	cfg := oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     oauthEndpoint,
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpClient := &http.Client{Timeout: timeout}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	tok, err := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_grant" {
			return GrantedScopesReport{}, fmt.Errorf("%w: %w", ErrRefreshTokenRejected, err)
		}

		return GrantedScopesReport{}, fmt.Errorf("refresh access token: %w", err)
	}

	source := GrantedScopesSourceTokenEndpoint
	granted := splitScopeString(tok.Extra("scope"))

	if len(granted) == 0 {
		if strings.TrimSpace(tok.AccessToken) == "" {
			return GrantedScopesReport{}, errMissingAccessToken
		}

		source = GrantedScopesSourceTokenInfo

		granted, err = tokenInfoScopes(ctx, httpClient, tok.AccessToken)
		if err != nil {
			return GrantedScopesReport{}, err
		}
	}

	if len(granted) == 0 {
		return GrantedScopesReport{}, errNoGrantedScopes
	}

	write := WriteScopes(granted)

	return GrantedScopesReport{
		Granted:     granted,
		WriteScopes: write,
		ReadOnly:    len(write) == 0,
		Source:      source,
	}, nil
}

// tokenInfoScopes asks the tokeninfo endpoint which scopes accessToken
// carries. The token is sent in a POST body, never in the URL.
func tokenInfoScopes(ctx context.Context, httpClient *http.Client, accessToken string) ([]string, error) {
	form := url.Values{"access_token": {accessToken}}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, scopeInfoURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build tokeninfo request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tokeninfo request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))

		return nil, fmt.Errorf("%w: status %d", errTokenInfoStatus, resp.StatusCode)
	}

	var payload struct {
		Scope string `json:"scope"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode tokeninfo response: %w", err)
	}

	return splitScopeString(payload.Scope), nil
}

// splitScopeString turns a space-delimited OAuth scope value into a sorted,
// deduplicated list. Non-string values yield nil.
func splitScopeString(raw any) []string {
	s, ok := raw.(string)
	if !ok {
		return nil
	}

	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))

	for _, f := range fields {
		if _, dup := seen[f]; dup {
			continue
		}

		seen[f] = struct{}{}
		out = append(out, f)
	}

	sort.Strings(out)

	return out
}

// WriteScopes returns the scopes that are not on the read-only allowlist,
// deduplicated and sorted. It never returns nil.
func WriteScopes(scopes []string) []string {
	seen := make(map[string]struct{}, len(scopes))
	out := make([]string, 0, len(scopes))

	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if IsReadOnlyScope(scope) {
			continue
		}

		if _, ok := seen[scope]; ok {
			continue
		}

		seen[scope] = struct{}{}
		out = append(out, scope)
	}

	sort.Strings(out)

	return out
}
