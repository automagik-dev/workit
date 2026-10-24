package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

var (
	errM365Disabled          = errors.New("m365 broker disabled")
	errM365MissingClientID   = errors.New("m365 client id missing")
	errM365MissingBaseURL    = errors.New("m365 public base url missing")
	errM365MissingEmail      = errors.New("m365 expected email missing")
	errM365StateNotFound     = errors.New("m365 broker state not found")
	errM365StateExpired      = errors.New("m365 broker state expired")
	errM365EmailMismatch     = errors.New("m365 authorized email mismatch")
	errM365MissingAdminToken = errors.New("m365 broker admin token missing")
	errM365Unauthorized      = errors.New("m365 broker session creation unauthorized")
	errM365TenantNotPinned   = errors.New("m365 tenant id must name one tenant, not organizations/common/consumers")
	errM365TenantMismatch    = errors.New("m365 authorized tenant mismatch")
)

// Package-level so tests can point them at httptest servers.
var (
	m365LoginBaseURL = "https://login.microsoftonline.com"
	m365GraphBaseURL = "https://graph.microsoft.com/v1.0"
	m365Audit        = slog.New(slog.NewJSONHandler(os.Stderr, nil))
)

// m365Scopes is the only scope set the broker ever requests: read-only Graph plus refresh.
var m365Scopes = []string{"offline_access", "User.Read", "Mail.Read", "Calendars.Read"}

type ServerOptions struct {
	Store              *TokenStore
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	M365Enabled        bool
	M365ClientID       string
	M365TenantID       string
	M365AdminToken     string
	PublicBaseURL      string
}

type m365Session struct {
	State         string
	PollKey       string // secret handed only to the admin caller; keys the token in the store
	ExpectedEmail string
	AuthURL       string
	CodeVerifier  string
	ExpiresAt     time.Time
}

type m365SessionStore struct {
	mu       sync.Mutex
	sessions map[string]m365Session
}

func newM365SessionStore() *m365SessionStore {
	return &m365SessionStore{sessions: make(map[string]m365Session)}
}

func (s *m365SessionStore) save(session m365Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessions == nil {
		s.sessions = make(map[string]m365Session)
	}

	now := time.Now()
	for state, existing := range s.sessions {
		if !existing.ExpiresAt.IsZero() && now.After(existing.ExpiresAt) {
			delete(s.sessions, state)
		}
	}

	s.sessions[session.State] = session
}

func (s *m365SessionStore) get(state string) (m365Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[strings.TrimSpace(state)]
	if !ok {
		return m365Session{}, errM365StateNotFound
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		delete(s.sessions, session.State)
		return m365Session{}, errM365StateExpired
	}

	return session, nil
}

func (s *m365SessionStore) consume(state string) (m365Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(state)
	session, ok := s.sessions[key]
	if ok {
		delete(s.sessions, key)
	}
	if !ok {
		return m365Session{}, errM365StateNotFound
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return m365Session{}, errM365StateExpired
	}

	return session, nil
}

func NewServerWithOptions(opts ServerOptions) *Server {
	store := opts.Store
	if store == nil {
		store = NewTokenStore(DefaultTTL)
	}

	s := NewServer(store, opts.GoogleClientID, opts.GoogleClientSecret, opts.GoogleRedirectURL)
	s.m365Enabled = opts.M365Enabled
	s.m365ClientID = strings.TrimSpace(opts.M365ClientID)
	s.m365AdminToken = strings.TrimSpace(opts.M365AdminToken)
	s.m365TenantID = strings.TrimSpace(opts.M365TenantID)
	s.publicBaseURL = strings.TrimRight(strings.TrimSpace(opts.PublicBaseURL), "/")
	s.m365Sessions = newM365SessionStore()
	s.mux.HandleFunc("/m365/sessions", s.handleM365Sessions)
	s.mux.HandleFunc("/m365/start/", s.handleM365Start)
	s.mux.HandleFunc("/m365/callback", s.handleM365Callback)

	return s
}

func (s *Server) handleM365Sessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.validateM365Config(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}
	if !s.authorizeM365SessionRequest(r) {
		writeJSONError(w, http.StatusUnauthorized, errM365Unauthorized)
		return
	}

	var req struct {
		ExpectedEmail string `json:"expected_email"`
		ForceConsent  bool   `json:"force_consent"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	expectedEmail := strings.ToLower(strings.TrimSpace(req.ExpectedEmail))
	if expectedEmail == "" {
		writeJSONError(w, http.StatusBadRequest, errM365MissingEmail)
		return
	}

	session, err := s.createM365Session(expectedEmail, req.ForceConsent)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"state":          session.State,
		"expected_email": session.ExpectedEmail,
		"login_url":      s.publicBaseURL + "/m365/start/" + session.State,
		"poll_key":       session.PollKey,
		"status_url":     s.publicBaseURL + "/status/" + session.PollKey,
		"expires_at":     session.ExpiresAt.Format(time.RFC3339),
	}); err != nil {
		log.Printf("Error encoding m365 session response: %v", err)
	}
}

func (s *Server) handleM365Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := strings.TrimPrefix(r.URL.Path, "/m365/start/")
	session, err := s.m365Sessions.get(state)
	if err != nil {
		if errors.Is(err, errM365StateExpired) {
			auditM365(slog.LevelWarn, "m365.state_expired", state)
		}
		s.renderErrorPage(w, "Microsoft 365 login link expired or not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, session.AuthURL, http.StatusFound)
}

func (s *Server) handleM365Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state := r.URL.Query().Get("state")
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		// The state is not consumed: the user may cancel and click the link again.
		auditM365(slog.LevelWarn, "m365.callback_failed", state, "reason", "authorization_error", "error", errParam)
		s.renderErrorPage(w, "Microsoft 365 authorization failed", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		auditM365(slog.LevelWarn, "m365.callback_failed", state, "reason", "missing_state_or_code")
		s.renderErrorPage(w, "Missing Microsoft 365 authorization state or code", http.StatusBadRequest)
		return
	}

	// Consumed before anything else, so a replayed or failed state can never be retried.
	session, err := s.m365Sessions.consume(state)
	if err != nil {
		if errors.Is(err, errM365StateExpired) {
			auditM365(slog.LevelWarn, "m365.state_expired", state)
		} else {
			auditM365(slog.LevelWarn, "m365.callback_failed", state, "reason", "state_not_found_or_reused")
		}
		s.renderErrorPage(w, "Microsoft 365 login link expired or already used", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	fail := func(reason string, err error, page string, status int) {
		auditM365(slog.LevelWarn, "m365.callback_failed", state, "reason", reason, "email", maskEmail(session.ExpectedEmail), "error", err.Error())
		s.renderErrorPage(w, page, status)
	}

	token, err := s.exchangeM365Code(ctx, session, code)
	if err != nil {
		fail("token_exchange", err, "Failed to exchange Microsoft 365 authorization code", http.StatusInternalServerError)
		return
	}

	email, err := fetchM365Email(ctx, token.AccessToken)
	if err != nil {
		fail("profile", err, "Failed to validate Microsoft 365 account", http.StatusInternalServerError)
		return
	}
	if err := validateM365Email(session.ExpectedEmail, email); err != nil {
		fail("email_mismatch", err, "Microsoft 365 account did not match expected email", http.StatusForbidden)
		return
	}

	tenantID, domains, err := fetchM365Tenant(ctx, token.AccessToken)
	if err != nil {
		fail("tenant_lookup", err, "Failed to validate Microsoft 365 account", http.StatusInternalServerError)
		return
	}
	if err := validateM365Tenant(s.m365TenantID, tenantID, domains); err != nil {
		fail("tenant_mismatch", err, "Microsoft 365 account is not in the expected organization", http.StatusForbidden)
		return
	}

	s.store.Store(session.PollKey, token)
	auditM365(slog.LevelInfo, "m365.callback_ok", state, "email", maskEmail(session.ExpectedEmail), "tenant", tenantID)
	renderM365SuccessPage(w)
}

func (s *Server) validateM365Config() error {
	if !s.m365Enabled {
		return errM365Disabled
	}
	if s.m365ClientID == "" {
		return errM365MissingClientID
	}
	if s.m365AdminToken == "" {
		return errM365MissingAdminToken
	}
	if !m365TenantPinned(s.m365TenantID) {
		return errM365TenantNotPinned
	}

	parsed, err := url.Parse(s.publicBaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errM365MissingBaseURL
	}

	return nil
}

func (s *Server) authorizeM365SessionRequest(r *http.Request) bool {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}

	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == s.m365AdminToken
}

func (s *Server) createM365Session(expectedEmail string, forceConsent bool) (m365Session, error) {
	state, verifier, challenge, err := newM365StateAndPKCE()
	if err != nil {
		return m365Session{}, err
	}
	pollKey, err := randomURLToken()
	if err != nil {
		return m365Session{}, err
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	redirectURL := s.publicBaseURL + "/m365/callback"
	cfg := s.m365OAuthConfig(redirectURL)
	authURL := cfg.AuthCodeURL(state, m365AuthParams(forceConsent, challenge)...)
	session := m365Session{
		State:         state,
		PollKey:       pollKey,
		ExpectedEmail: expectedEmail,
		AuthURL:       authURL,
		CodeVerifier:  verifier,
		ExpiresAt:     expiresAt,
	}

	s.m365Sessions.save(session)
	s.store.MarkPending(pollKey)
	auditM365(slog.LevelInfo, "m365.link_created", state, "email", maskEmail(expectedEmail), "expires_at", expiresAt.Format(time.RFC3339))

	return session, nil
}

func (s *Server) m365OAuthConfig(redirectURL string) oauth2.Config {
	base := m365LoginBaseURL + "/" + s.m365TenantID + "/oauth2/v2.0"

	return oauth2.Config{
		ClientID:    s.m365ClientID,
		Endpoint:    oauth2.Endpoint{AuthURL: base + "/authorize", TokenURL: base + "/token", AuthStyle: oauth2.AuthStyleInParams},
		RedirectURL: redirectURL,
		Scopes:      m365Scopes,
	}
}

func (s *Server) exchangeM365Code(ctx context.Context, session m365Session, code string) (*oauth2.Token, error) {
	cfg := s.m365OAuthConfig(s.publicBaseURL + "/m365/callback")
	token, err := cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", session.CodeVerifier))
	if err != nil {
		return nil, fmt.Errorf("exchange m365 code: %w", err)
	}
	if token.RefreshToken == "" {
		return nil, errors.New("m365 refresh token missing")
	}

	return token, nil
}

func m365AuthParams(forceConsent bool, challenge string) []oauth2.AuthCodeOption {
	params := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if forceConsent {
		params = append(params, oauth2.SetAuthURLParam("prompt", "consent"))
	}

	return params
}

func newM365StateAndPKCE() (string, string, string, error) {
	state, err := randomURLToken()
	if err != nil {
		return "", "", "", err
	}

	verifier, err := randomURLToken()
	if err != nil {
		return "", "", "", err
	}

	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	return state, verifier, challenge, nil
}

func randomURLToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func getM365Graph(ctx context.Context, accessToken string, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m365GraphBaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create graph request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("graph request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("graph %s status: %d", strings.SplitN(path, "?", 2)[0], resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode graph response: %w", err)
	}

	return nil
}

func fetchM365Email(ctx context.Context, accessToken string) (string, error) {
	var me struct {
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"` //nolint:tagliatelle // Microsoft Graph field name.
	}
	if err := getM365Graph(ctx, accessToken, "/me?$select=mail,userPrincipalName", &me); err != nil {
		return "", err
	}

	email := strings.TrimSpace(me.Mail)
	if email == "" {
		email = strings.TrimSpace(me.UserPrincipalName)
	}
	if email == "" {
		return "", errors.New("m365 profile missing email")
	}

	return email, nil
}

// fetchM365Tenant returns the signed-in user's tenant id and verified domains.
// GET /organization only needs User.Read.
func fetchM365Tenant(ctx context.Context, accessToken string) (string, []string, error) {
	var org struct {
		Value []struct {
			ID              string `json:"id"`
			VerifiedDomains []struct {
				Name string `json:"name"`
			} `json:"verifiedDomains"` //nolint:tagliatelle // Microsoft Graph field name.
		} `json:"value"`
	}
	if err := getM365Graph(ctx, accessToken, "/organization?$select=id,verifiedDomains", &org); err != nil {
		return "", nil, err
	}
	if len(org.Value) != 1 || org.Value[0].ID == "" {
		return "", nil, errors.New("m365 organization missing")
	}

	domains := make([]string, 0, len(org.Value[0].VerifiedDomains))
	for _, d := range org.Value[0].VerifiedDomains {
		domains = append(domains, d.Name)
	}

	return org.Value[0].ID, domains, nil
}

// validateM365Tenant accepts the configured tenant as either the tenant id or one of its verified domains,
// the two forms login.microsoftonline.com accepts.
func validateM365Tenant(configured string, tenantID string, domains []string) error {
	if strings.EqualFold(configured, tenantID) {
		return nil
	}
	for _, d := range domains {
		if strings.EqualFold(configured, d) {
			return nil
		}
	}

	return errM365TenantMismatch
}

func m365TenantPinned(tenant string) bool {
	switch strings.ToLower(strings.TrimSpace(tenant)) {
	case "", "organizations", "common", "consumers":
		return false
	}

	return true
}

func validateM365Email(expected string, actual string) error {
	want := strings.ToLower(strings.TrimSpace(expected))
	got := strings.ToLower(strings.TrimSpace(actual))
	if want == "" || got == "" || want != got {
		return fmt.Errorf("%w: email mismatch", errM365EmailMismatch)
	}

	return nil
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encodeErr != nil {
		log.Printf("Error encoding error response: %v", encodeErr)
	}
}

// auditM365 writes one structured audit line. It never receives tokens, the poll key or mail content;
// the state is reduced to a short hash so log readers cannot reuse a live link.
func auditM365(level slog.Level, event string, state string, attrs ...any) {
	m365Audit.Log(context.Background(), level, event, append([]any{"session", m365SessionID(state)}, attrs...)...)
}

func m365SessionID(state string) string {
	if state == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(state))

	return hex.EncodeToString(sum[:6])
}

func maskEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return "***"
	}

	return local[:1] + "***@" + domain
}

func renderM365SuccessPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Microsoft 365 connected</title>
</head>
<body style="font-family: sans-serif; text-align: center; padding: 48px 16px;">
    <h1>Microsoft 365 connected read-only</h1>
    <p>You can close this window.</p>
</body>
</html>`)
}
