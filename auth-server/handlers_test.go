package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestHealthEndpoint(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Status)
	}
}

func TestHealthEndpoint_WrongMethod(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestTokenEndpoint_NotFound(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/token/unknown-state", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestTokenEndpoint_Ready(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	// Store a token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	store.Store("test-state", token)

	req := httptest.NewRequest(http.MethodGet, "/token/test-state", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.AccessToken != "test-access-token" {
		t.Errorf("Expected access token 'test-access-token', got '%s'", resp.AccessToken)
	}
	if resp.RefreshToken != "test-refresh-token" {
		t.Errorf("Expected refresh token 'test-refresh-token', got '%s'", resp.RefreshToken)
	}
}

func TestTokenEndpoint_Consumed(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	// Store a token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	store.Store("consumed-state", token)

	// First request - should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/token/consumed-state", nil)
	w1 := httptest.NewRecorder()
	server.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request: expected status 200, got %d", w1.Code)
	}

	// Second request - should return 410 Gone
	req2 := httptest.NewRequest(http.MethodGet, "/token/consumed-state", nil)
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	if w2.Code != http.StatusGone {
		t.Errorf("Second request: expected status 410, got %d", w2.Code)
	}
}

func TestTokenEndpoint_Pending(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	// Mark a state as pending (token not yet received)
	store.MarkPending("pending-state")

	req := httptest.NewRequest(http.MethodGet, "/token/pending-state", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}
}

func TestStatusEndpoint(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	// Test not found
	req := httptest.NewRequest(http.MethodGet, "/status/unknown", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "not_found" {
		t.Errorf("Expected status 'not_found', got '%s'", resp["status"])
	}
}

func TestStatusEndpoint_Ready(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	// Store a token
	token := &oauth2.Token{
		AccessToken: "test-token",
	}
	store.Store("ready-state", token)

	req := httptest.NewRequest(http.MethodGet, "/status/ready-state", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", resp["status"])
	}
}

func TestCallbackEndpoint_MissingCode(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/callback?state=test-state", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCallbackEndpoint_MissingState(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/callback?code=test-code", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCallbackEndpoint_Success(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	server := NewServer(store, "client-id", "client-secret", "http://localhost/callback")

	// Mock the exchange function
	server.exchangeFunc = func(ctx context.Context, code string) (*oauth2.Token, error) {
		return &oauth2.Token{
			AccessToken:  "exchanged-access-token",
			RefreshToken: "exchanged-refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour),
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/callback?code=auth-code&state=callback-state", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify token was stored
	token, status := store.Get("callback-state")
	if status != TokenStatusReady {
		t.Errorf("Expected TokenStatusReady, got %v", status)
	}
	if token.AccessToken != "exchanged-access-token" {
		t.Errorf("Expected access token 'exchanged-access-token', got '%s'", token.AccessToken)
	}

	// Verify HTML response
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/html; charset=utf-8', got '%s'", contentType)
	}
}
