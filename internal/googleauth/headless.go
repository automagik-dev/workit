package googleauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/namastexlabs/gog-cli/internal/config"
)

var (
	errMissingCallbackServer = errors.New("callback server URL required for headless auth")
	errPollTimeout = errors.New("timeout waiting for token")
	errTokenConsumed         = errors.New("token has already been retrieved")
	errTokenNotFound = errors.New("token not found or expired")
)

// IsPollTimeout reports whether err is a poll timeout error.
// This is needed because errPollTimeout is unexported.
func IsPollTimeout(err error) bool {
	return errors.Is(err, errPollTimeout)
}

// NewPollTimeoutError returns a poll timeout error for testing.
func NewPollTimeoutError() error {
	return errPollTimeout
}

// HeadlessAuthInfo contains the information needed for headless OAuth flow.
type HeadlessAuthInfo struct {
	AuthURL   string `json:"auth_url"`
	State     string `json:"state"`
	PollURL   string `json:"poll_url"`
	ExpiresIn int    `json:"expires_in"`
}

// HeadlessOptions configures the headless OAuth flow.
type HeadlessOptions struct {
	Services       []Service
	Scopes         []string
	ForceConsent   bool
	Client         string
	CallbackServer string
}

// CallbackServerURL returns the callback server URL from the provided sources,
// in order of precedence: override > env var > config file > build-time default.
func CallbackServerURL(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override), nil
	}

	if envURL := os.Getenv("GOG_CALLBACK_SERVER"); strings.TrimSpace(envURL) != "" {
		return strings.TrimSpace(envURL), nil
	}

	// Check config file
	if cfg, cfgErr := config.ReadConfig(); cfgErr == nil && strings.TrimSpace(cfg.CallbackServer) != "" {
		return strings.TrimSpace(cfg.CallbackServer), nil
	}

	if strings.TrimSpace(config.DefaultCallbackServer) != "" {
		return strings.TrimSpace(config.DefaultCallbackServer), nil
	}

	return "", errMissingCallbackServer
}

// HeadlessAuthorize generates an OAuth URL for headless authentication.
// The user must visit the URL and complete authentication in a browser.
// The callback server will receive the token which can be polled for.
func HeadlessAuthorize(ctx context.Context, opts HeadlessOptions) (HeadlessAuthInfo, error) {
	callbackServer, err := CallbackServerURL(opts.CallbackServer)
	if err != nil {
		return HeadlessAuthInfo{}, err
	}

	if len(opts.Scopes) == 0 {
		return HeadlessAuthInfo{}, errMissingScopes
	}

	creds, err := readClientCredentials(opts.Client)
	if err != nil {
		return HeadlessAuthInfo{}, err
	}

	state, err := randomStateFn()
	if err != nil {
		return HeadlessAuthInfo{}, err
	}

	// Build redirect URL pointing to the callback server
	redirectURL := strings.TrimSuffix(callbackServer, "/") + "/callback"

	cfg := oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     oauthEndpoint,
		RedirectURL:  redirectURL,
		Scopes:       opts.Scopes,
	}

	authURL := cfg.AuthCodeURL(state, authURLParams(opts.ForceConsent)...)

	pollURL := strings.TrimSuffix(callbackServer, "/") + "/token/" + state

	return HeadlessAuthInfo{
		AuthURL:   authURL,
		State:     state,
		PollURL:   pollURL,
		ExpiresIn: 300, // 5 minutes default TTL on callback server
	}, nil
}

// PollResponse represents the response from polling the callback server.
type PollResponse struct {
	// Token fields (when ready)
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	Expiry       string `json:"expiry,omitempty"`

	// Status fields (when pending or error)
	Status  string `json:"status,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// PollForToken polls the callback server until a token is ready or timeout.
func PollForToken(ctx context.Context, callbackServer, state string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	pollURL := strings.TrimSuffix(callbackServer, "/") + "/token/" + state

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", errPollTimeout
		case <-ticker.C:
			refreshToken, done, err := pollOnce(ctx, client, pollURL)
			if err != nil {
				return "", err
			}

			if done {
				return refreshToken, nil
			}
		}
	}
}

func pollOnce(ctx context.Context, client *http.Client, pollURL string) (refreshToken string, done bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return "", false, fmt.Errorf("create poll request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Network error, keep trying
		return "", false, nil
	}

	defer func() { _ = resp.Body.Close() }()

	var pollResp PollResponse
	if err := json.NewDecoder(resp.Body).Decode(&pollResp); err != nil {
		// Parse error, keep trying
		return "", false, nil
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// Token is ready
		if pollResp.RefreshToken == "" {
			return "", false, errNoRefreshToken
		}

		return pollResp.RefreshToken, true, nil

	case http.StatusAccepted:
		// Token pending, continue polling
		return "", false, nil

	case http.StatusGone:
		// Token already consumed
		return "", false, errTokenConsumed

	case http.StatusNotFound:
		// Token not found or expired
		return "", false, errTokenNotFound

	default:
		// Unexpected status, keep trying
		return "", false, nil
	}
}
