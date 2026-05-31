package msauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/automagik-dev/workit/internal/config"
)

const (
	ClientName           = "m365"
	DefaultLocalAuthPort = 8085
)

var localAuthPort = DefaultLocalAuthPort

var (
	ErrMissingClientID     = errors.New("m365 oauth client id missing")
	ErrMissingScopes       = errors.New("m365 oauth scopes missing")
	ErrNoRefreshToken      = errors.New("m365 refresh token missing; ensure offline_access is granted")
	ErrStateMismatch       = errors.New("m365 oauth state mismatch")
	ErrMissingCode         = errors.New("m365 oauth missing code")
	ErrAuthorization       = errors.New("m365 authorization error")
	ErrProfileStatus       = errors.New("fetch m365 profile status error")
	ErrProfileMissingEmail = errors.New("m365 profile missing email")
	ErrContextDone         = errors.New("m365 oauth context done")
)

var (
	openBrowserFn func(context.Context, string) error = openBrowser
	randomStateFn                                     = randomURLToken
	oauthConfigFn                                     = oauthConfig
	graphMeURL                                        = "https://graph.microsoft.com/v1.0/me"
)

type AuthorizeOptions struct {
	ExpectedEmail string
	Readonly      bool
	Manual        bool
	ForceConsent  bool
	Timeout       time.Duration
	AuthURL       string
}

type AuthorizeResult struct {
	Email        string
	RefreshToken string
}

type ManualAuthURLOptions struct {
	Readonly     bool
	ForceConsent bool
}

type ManualAuthURLResult struct {
	URL       string `json:"auth_url"`
	State     string `json:"state"`
	ExpiresIn int    `json:"expires_in"`
}

type oauthSettings struct {
	ClientID string
	TenantID string
}

func Authorize(ctx context.Context, opts AuthorizeOptions) (AuthorizeResult, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}

	settings, err := resolveOAuthSettings()
	if err != nil {
		return AuthorizeResult{}, err
	}

	scopes, err := OAuthScopes(opts.Readonly)
	if err != nil {
		return AuthorizeResult{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	state, verifier, challenge, err := newOAuthStateAndPKCE()
	if err != nil {
		return AuthorizeResult{}, err
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", fmt.Sprintf("localhost:%d", localAuthPort))
	if err != nil {
		return AuthorizeResult{}, fmt.Errorf("listen for m365 callback on port %d: %w", localAuthPort, err)
	}

	defer func() { _ = ln.Close() }()

	redirectURI := fmt.Sprintf("http://localhost:%d/oauth2/callback", localAuthPort)
	cfg := oauthConfigFn(settings, redirectURI, scopes)
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := m365OAuthServer(ctx, state, codeCh, errCh)

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()

	authURL := cfg.AuthCodeURL(state, authParams(opts.ForceConsent, challenge)...)

	fmt.Fprintln(os.Stderr, "Opening browser for Microsoft 365 authorization…")
	fmt.Fprintln(os.Stderr, "If the browser doesn't open, visit this URL:")
	fmt.Fprintln(os.Stderr, authURL)
	_ = openBrowserFn(ctx, authURL)

	select {
	case code := <-codeCh:
		return exchangeCodeAndProfile(ctx, srv, cfg, code, verifier)
	case authErr := <-errCh:
		return AuthorizeResult{}, authErr
	case <-ctx.Done():
		return AuthorizeResult{}, fmt.Errorf("%w: %w", ErrContextDone, ctx.Err())
	}
}

func ManualAuthURL(_ context.Context, opts ManualAuthURLOptions) (ManualAuthURLResult, error) {
	settings, err := resolveOAuthSettings()
	if err != nil {
		return ManualAuthURLResult{}, err
	}

	scopes, err := OAuthScopes(opts.Readonly)
	if err != nil {
		return ManualAuthURLResult{}, err
	}

	state, _, challenge, err := newOAuthStateAndPKCE()
	if err != nil {
		return ManualAuthURLResult{}, err
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/oauth2/callback", localAuthPort)
	cfg := oauthConfigFn(settings, redirectURI, scopes)
	url := cfg.AuthCodeURL(state, authParams(opts.ForceConsent, challenge)...)

	return ManualAuthURLResult{URL: url, State: state, ExpiresIn: 300}, nil
}

func OAuthScopes(readonly bool) ([]string, error) {
	if !readonly {
		return nil, ErrPilotScopeNotAllowed
	}

	scopes := append([]string{"offline_access"}, PilotAllowedScopes()...)
	if len(scopes) == 0 {
		return nil, ErrMissingScopes
	}

	return scopes, nil
}

func FetchEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, graphMeURL, nil)
	if err != nil {
		return "", fmt.Errorf("create m365 profile request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch m365 profile: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: %d", ErrProfileStatus, resp.StatusCode)
	}

	var me struct {
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"` //nolint:tagliatelle // Microsoft Graph field name.
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return "", fmt.Errorf("decode m365 profile: %w", err)
	}

	email := strings.TrimSpace(me.Mail)
	if email == "" {
		email = strings.TrimSpace(me.UserPrincipalName)
	}

	if email == "" {
		return "", ErrProfileMissingEmail
	}

	return email, nil
}

func resolveOAuthSettings() (oauthSettings, error) {
	clientID := strings.TrimSpace(config.DefaultM365ClientID)
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("WK_M365_CLIENT_ID"))
	}

	if clientID == "" {
		return oauthSettings{}, ErrMissingClientID
	}

	tenantID := strings.TrimSpace(config.DefaultM365TenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(os.Getenv("WK_M365_TENANT_ID"))
	}

	if tenantID == "" {
		tenantID = "organizations"
	}

	return oauthSettings{ClientID: clientID, TenantID: tenantID}, nil
}

func oauthConfig(settings oauthSettings, redirectURI string, scopes []string) oauth2.Config {
	base := "https://login.microsoftonline.com/" + settings.TenantID + "/oauth2/v2.0"

	return oauth2.Config{
		ClientID:    settings.ClientID,
		Endpoint:    oauth2.Endpoint{AuthURL: base + "/authorize", TokenURL: base + "/token"},
		RedirectURL: redirectURI,
		Scopes:      scopes,
	}
}

func authParams(forceConsent bool, challenge string) []oauth2.AuthCodeOption {
	params := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if forceConsent {
		params = append(params, oauth2.SetAuthURLParam("prompt", "consent"))
	}

	return params
}

func newOAuthStateAndPKCE() (state string, verifier string, challenge string, err error) {
	state, err = randomStateFn()
	if err != nil {
		return "", "", "", err
	}

	verifier, challenge, err = pkcePair()
	if err != nil {
		return "", "", "", err
	}

	return state, verifier, challenge, nil
}

func pkcePair() (verifier string, challenge string, err error) {
	verifier, err = randomURLToken()
	if err != nil {
		return "", "", err
	}

	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])

	return verifier, challenge, nil
}

func randomURLToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func m365OAuthServer(ctx context.Context, state string, codeCh chan<- string, errCh chan<- error) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleM365OAuthCallback(w, r, state, codeCh, errCh)
	})

	return &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: handler, BaseContext: func(net.Listener) context.Context { return ctx }}
}

func handleM365OAuthCallback(w http.ResponseWriter, r *http.Request, state string, codeCh chan<- string, errCh chan<- error) {
	if r.URL.Path != "/oauth2/callback" {
		http.NotFound(w, r)
		return
	}

	q := r.URL.Query()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if q.Get("error") != "" {
		select {
		case errCh <- fmt.Errorf("%w: %s", ErrAuthorization, q.Get("error")):
		default:
		}

		_, _ = w.Write([]byte("Microsoft 365 authorization failed. You may close this tab."))

		return
	}

	if q.Get("state") != state {
		select {
		case errCh <- ErrStateMismatch:
		default:
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("State mismatch. Please try again."))

		return
	}

	code := q.Get("code")
	if code == "" {
		select {
		case errCh <- ErrMissingCode:
		default:
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Missing authorization code. Please try again."))

		return
	}

	select {
	case codeCh <- code:
	default:
	}

	_, _ = w.Write([]byte("Microsoft 365 authorization complete. You may close this tab."))
}

func exchangeCodeAndProfile(ctx context.Context, srv *http.Server, cfg oauth2.Config, code string, verifier string) (AuthorizeResult, error) {
	tok, err := cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		_ = srv.Close()
		return AuthorizeResult{}, fmt.Errorf("exchange m365 code: %w", err)
	}

	if tok.RefreshToken == "" {
		_ = srv.Close()
		return AuthorizeResult{}, ErrNoRefreshToken
	}

	email, err := FetchEmail(ctx, tok.AccessToken)
	if err != nil {
		_ = srv.Close()
		return AuthorizeResult{}, err
	}

	_ = srv.Shutdown(ctx)

	return AuthorizeResult{Email: email, RefreshToken: tok.RefreshToken}, nil
}
