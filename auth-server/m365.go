package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

var (
	errM365Disabled        = errors.New("m365 broker disabled")
	errM365MissingClientID = errors.New("m365 client id missing")
	errM365MissingBaseURL  = errors.New("m365 public base url missing")
	errM365MissingEmail    = errors.New("m365 expected email missing")
	errM365StateNotFound   = errors.New("m365 broker state not found")
	errM365StateExpired    = errors.New("m365 broker state expired")
	errM365EmailMismatch   = errors.New("m365 authorized email mismatch")
)

const defaultM365TenantID = "organizations"

type ServerOptions struct {
	Store              *TokenStore
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	M365Enabled        bool
	M365ClientID       string
	M365TenantID       string
	PublicBaseURL      string
}

type m365Session struct {
	State         string
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
	s.m365TenantID = strings.TrimSpace(opts.M365TenantID)
	if s.m365TenantID == "" {
		s.m365TenantID = defaultM365TenantID
	}
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

	var req struct {
		ExpectedEmail string `json:"expected_email"`
		ForceConsent  bool   `json:"force_consent"`
	}
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
		"status_url":     s.publicBaseURL + "/status/" + session.State,
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
		writeJSONError(w, http.StatusNotFound, err)
		return
	}

	http.Redirect(w, r, session.AuthURL, http.StatusFound)
}

func (s *Server) handleM365Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		s.renderErrorPage(w, "Microsoft 365 authorization failed", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		s.renderErrorPage(w, "Missing Microsoft 365 authorization state or code", http.StatusBadRequest)
		return
	}

	session, err := s.m365Sessions.consume(state)
	if err != nil {
		s.renderErrorPage(w, "Microsoft 365 login link expired or already used", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := s.exchangeM365Code(ctx, session, code)
	if err != nil {
		log.Printf("M365 token exchange failed for state %s: %v", state, err)
		s.renderErrorPage(w, "Failed to exchange Microsoft 365 authorization code", http.StatusInternalServerError)
		return
	}

	email, err := fetchM365Email(ctx, token.AccessToken)
	if err != nil {
		log.Printf("M365 profile fetch failed for state %s: %v", state, err)
		s.renderErrorPage(w, "Failed to validate Microsoft 365 account", http.StatusInternalServerError)
		return
	}
	if err := validateM365Email(session.ExpectedEmail, email); err != nil {
		log.Printf("M365 email mismatch for state %s: %v", state, err)
		s.renderErrorPage(w, "Microsoft 365 account did not match expected email", http.StatusForbidden)
		return
	}

	s.store.Store(state, token)
	s.renderSuccessPage(w, state)
}

func (s *Server) validateM365Config() error {
	if !s.m365Enabled {
		return errM365Disabled
	}
	if s.m365ClientID == "" {
		return errM365MissingClientID
	}
	if s.publicBaseURL == "" {
		return errM365MissingBaseURL
	}

	parsed, err := url.Parse(s.publicBaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errM365MissingBaseURL
	}

	return nil
}

func (s *Server) createM365Session(expectedEmail string, forceConsent bool) (m365Session, error) {
	state, verifier, challenge, err := newM365StateAndPKCE()
	if err != nil {
		return m365Session{}, err
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	redirectURL := s.publicBaseURL + "/m365/callback"
	cfg := s.m365OAuthConfig(redirectURL)
	authURL := cfg.AuthCodeURL(state, m365AuthParams(forceConsent, challenge)...)
	session := m365Session{
		State:         state,
		ExpectedEmail: expectedEmail,
		AuthURL:       authURL,
		CodeVerifier:  verifier,
		ExpiresAt:     expiresAt,
	}

	s.m365Sessions.save(session)

	return session, nil
}

func (s *Server) m365OAuthConfig(redirectURL string) oauth2.Config {
	base := "https://login.microsoftonline.com/" + s.m365TenantID + "/oauth2/v2.0"

	return oauth2.Config{
		ClientID:    s.m365ClientID,
		Endpoint:    oauth2.Endpoint{AuthURL: base + "/authorize", TokenURL: base + "/token"},
		RedirectURL: redirectURL,
		Scopes:      []string{"offline_access", "User.Read", "Mail.Read", "Calendars.Read"},
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

func fetchM365Email(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/v1.0/me?$select=mail,userPrincipalName", nil)
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
		return "", fmt.Errorf("fetch m365 profile status: %d", resp.StatusCode)
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
		return "", errors.New("m365 profile missing email")
	}

	return email, nil
}

func validateM365Email(expected string, actual string) error {
	want := strings.ToLower(strings.TrimSpace(expected))
	got := strings.ToLower(strings.TrimSpace(actual))
	if want == "" || got == "" || want != got {
		return fmt.Errorf("%w: expected %s got %s", errM365EmailMismatch, want, got)
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
