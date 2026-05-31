package msauth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var (
	ErrBrokerHTTPSRequired = errors.New("m365 broker URL must use https")
	ErrBrokerMissingEmail  = errors.New("m365 broker expected email missing")
	ErrBrokerURLRequired   = errors.New("m365 broker URL required")
)

type BrokerSessionOptions struct {
	ExpectedEmail string
	BaseURL       string
	CallbackURL   string
	Readonly      bool
	ForceConsent  bool
	TTL           time.Duration
}

type BrokerSession struct {
	State         string    `json:"state"`
	ExpectedEmail string    `json:"expected_email"`
	LoginURL      string    `json:"login_url"`
	AuthURL       string    `json:"auth_url,omitempty"`
	CodeVerifier  string    `json:"-"`
	CodeChallenge string    `json:"-"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func CreateBrokerSession(_ context.Context, opts BrokerSessionOptions) (BrokerSession, error) {
	expectedEmail := strings.ToLower(strings.TrimSpace(opts.ExpectedEmail))
	if expectedEmail == "" {
		return BrokerSession{}, ErrBrokerMissingEmail
	}

	if !opts.Readonly {
		return BrokerSession{}, fmt.Errorf("%w: one-click broker is read-only only", ErrPilotScopeNotAllowed)
	}

	baseURL, err := parseHTTPSURL(opts.BaseURL, "base-url")
	if err != nil {
		return BrokerSession{}, err
	}

	callbackURL, err := parseHTTPSURL(opts.CallbackURL, "callback-url")
	if err != nil {
		return BrokerSession{}, err
	}

	settings, err := resolveOAuthSettings()
	if err != nil {
		return BrokerSession{}, err
	}

	scopes, err := OAuthScopes(opts.Readonly)
	if err != nil {
		return BrokerSession{}, err
	}

	state, verifier, challenge, err := newOAuthStateAndPKCE()
	if err != nil {
		return BrokerSession{}, err
	}

	if opts.TTL <= 0 {
		opts.TTL = 10 * time.Minute
	}

	loginURL := baseURL.JoinPath("m365", "start", state).String()
	cfg := oauthConfigFn(settings, callbackURL.String(), scopes)
	authURL := cfg.AuthCodeURL(state, authParams(opts.ForceConsent, challenge)...)

	return BrokerSession{
		State:         state,
		ExpectedEmail: expectedEmail,
		LoginURL:      loginURL,
		AuthURL:       authURL,
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
		ExpiresAt:     time.Now().UTC().Add(opts.TTL),
	}, nil
}

func parseHTTPSURL(raw string, field string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, fmt.Errorf("%w: %s", ErrBrokerURLRequired, field)
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse m365 broker %s: %w", field, err)
	}

	if parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: %s", ErrBrokerHTTPSRequired, field)
	}

	return parsed, nil
}
