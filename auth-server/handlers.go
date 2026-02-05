package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Server holds the HTTP server configuration and dependencies.
type Server struct {
	store        *TokenStore
	oauthConfig  *oauth2.Config
	mux          *http.ServeMux
	exchangeFunc func(ctx context.Context, code string) (*oauth2.Token, error)
}

// NewServer creates a new Server with the given configuration.
func NewServer(store *TokenStore, clientID, clientSecret, redirectURL string) *Server {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/calendar",
		},
	}

	s := &Server{
		store:       store,
		oauthConfig: config,
		mux:         http.NewServeMux(),
	}

	// Default exchange function uses the real OAuth config
	s.exchangeFunc = func(ctx context.Context, code string) (*oauth2.Token, error) {
		return config.Exchange(ctx, code, oauth2.AccessTypeOffline)
	}

	s.registerRoutes()
	return s
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/callback", s.handleCallback)
	s.mux.HandleFunc("/token/", s.handleToken)
	s.mux.HandleFunc("/status/", s.handleStatus)
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// HealthResponse represents the JSON response from /health.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// handleCallback processes the OAuth callback from Google.
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		s.renderErrorPage(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	if state == "" {
		s.renderErrorPage(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Check for OAuth error response
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		s.renderErrorPage(w, fmt.Sprintf("OAuth error: %s - %s", errParam, errDesc), http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for tokens
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := s.exchangeFunc(ctx, code)
	if err != nil {
		log.Printf("Token exchange failed for state %s: %v", state, err)
		s.renderErrorPage(w, "Failed to exchange authorization code for token", http.StatusInternalServerError)
		return
	}

	// Store the token
	s.store.Store(state, token)
	log.Printf("Token stored for state: %s", state)

	// Return success HTML page
	s.renderSuccessPage(w, state)
}

// handleToken returns the token for the given state.
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract state from path: /token/{state}
	state := strings.TrimPrefix(r.URL.Path, "/token/")
	if state == "" {
		http.Error(w, `{"error": "Missing state parameter"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	token, status := s.store.Get(state)

	switch status {
	case TokenStatusReady:
		resp := TokenResponse{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    token.TokenType,
			Expiry:       token.Expiry.Format(time.RFC3339),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding token response: %v", err)
		}

	case TokenStatusPending:
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "pending",
			"message": "Token not yet available, please try again",
		}); err != nil {
			log.Printf("Error encoding pending response: %v", err)
		}

	case TokenStatusConsumed:
		w.WriteHeader(http.StatusGone)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error":   "consumed",
			"message": "Token has already been retrieved",
		}); err != nil {
			log.Printf("Error encoding consumed response: %v", err)
		}

	case TokenStatusNotFound:
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "Token not found or expired",
		}); err != nil {
			log.Printf("Error encoding not found response: %v", err)
		}
	}
}

// handleStatus checks the status of a token without consuming it.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract state from path: /status/{state}
	state := strings.TrimPrefix(r.URL.Path, "/status/")
	if state == "" {
		http.Error(w, `{"error": "Missing state parameter"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	status := s.store.Status(state)

	var statusStr string
	switch status {
	case TokenStatusReady:
		statusStr = "ready"
	case TokenStatusPending:
		statusStr = "pending"
	case TokenStatusConsumed:
		statusStr = "consumed"
	case TokenStatusNotFound:
		statusStr = "not_found"
	}

	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": statusStr,
	}); err != nil {
		log.Printf("Error encoding status response: %v", err)
	}
}

// TokenResponse represents the JSON response containing OAuth tokens.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expiry       string `json:"expiry"`
}

// renderSuccessPage renders an HTML success page for the OAuth callback.
func (s *Server) renderSuccessPage(w http.ResponseWriter, state string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authorization Successful</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            text-align: center;
            max-width: 400px;
        }
        .success-icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #22c55e;
            margin-bottom: 16px;
        }
        p {
            color: #666;
            line-height: 1.6;
        }
        .state {
            font-family: monospace;
            background: #f3f4f6;
            padding: 8px 12px;
            border-radius: 6px;
            font-size: 14px;
            word-break: break-all;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success-icon">&#x2705;</div>
        <h1>Authorization Successful</h1>
        <p>You have successfully authorized the application.</p>
        <p>You can close this window and return to your terminal.</p>
        <p style="margin-top: 24px; font-size: 12px; color: #999;">
            State: <span class="state">%s</span>
        </p>
    </div>
</body>
</html>`, state)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Error writing success page: %v", err)
	}
}

// renderErrorPage renders an HTML error page.
func (s *Server) renderErrorPage(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authorization Failed</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            text-align: center;
            max-width: 400px;
        }
        .error-icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #ef4444;
            margin-bottom: 16px;
        }
        p {
            color: #666;
            line-height: 1.6;
        }
        .error-message {
            background: #fef2f2;
            color: #b91c1c;
            padding: 12px;
            border-radius: 8px;
            margin-top: 16px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-icon">&#x274C;</div>
        <h1>Authorization Failed</h1>
        <p>There was a problem completing the authorization.</p>
        <div class="error-message">%s</div>
        <p style="margin-top: 24px;">Please try again or contact support.</p>
    </div>
</body>
</html>`, message)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Error writing error page: %v", err)
	}
}
