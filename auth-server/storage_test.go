package main

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenStore_StoreAndGet(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)

	token := &oauth2.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Store token
	store.Store("state-abc", token)

	// Get token
	gotToken, status := store.Get("state-abc")
	if status != TokenStatusReady {
		t.Errorf("Expected TokenStatusReady, got %v", status)
	}
	if gotToken.AccessToken != "access-123" {
		t.Errorf("Expected access token 'access-123', got '%s'", gotToken.AccessToken)
	}
	if gotToken.RefreshToken != "refresh-456" {
		t.Errorf("Expected refresh token 'refresh-456', got '%s'", gotToken.RefreshToken)
	}

	// Second get should return consumed
	_, status = store.Get("state-abc")
	if status != TokenStatusConsumed {
		t.Errorf("Expected TokenStatusConsumed, got %v", status)
	}
}

func TestTokenStore_NotFound(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)

	_, status := store.Get("nonexistent")
	if status != TokenStatusNotFound {
		t.Errorf("Expected TokenStatusNotFound, got %v", status)
	}
}

func TestTokenStore_Pending(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)

	// Mark pending (no token yet)
	store.MarkPending("pending-state")

	status := store.Status("pending-state")
	if status != TokenStatusPending {
		t.Errorf("Expected TokenStatusPending, got %v", status)
	}
}

func TestTokenStore_Status(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)

	token := &oauth2.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Status of nonexistent
	status := store.Status("nonexistent")
	if status != TokenStatusNotFound {
		t.Errorf("Expected TokenStatusNotFound, got %v", status)
	}

	// Store and check status
	store.Store("state-xyz", token)
	status = store.Status("state-xyz")
	if status != TokenStatusReady {
		t.Errorf("Expected TokenStatusReady, got %v", status)
	}

	// Consume and check status
	store.Get("state-xyz")
	status = store.Status("state-xyz")
	if status != TokenStatusConsumed {
		t.Errorf("Expected TokenStatusConsumed, got %v", status)
	}
}

func TestTokenStore_TTL(t *testing.T) {
	// Use very short TTL for testing
	store := NewTokenStore(10 * time.Millisecond)

	token := &oauth2.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	store.Store("expiring-state", token)

	// Should be available immediately
	status := store.Status("expiring-state")
	if status != TokenStatusReady {
		t.Errorf("Expected TokenStatusReady, got %v", status)
	}

	// Wait for TTL to expire
	time.Sleep(20 * time.Millisecond)

	// Should now be expired (not found)
	_, getStatus := store.Get("expiring-state")
	if getStatus != TokenStatusNotFound {
		t.Errorf("Expected TokenStatusNotFound after TTL, got %v", getStatus)
	}
}

func TestTokenStore_Concurrent(t *testing.T) {
	store := NewTokenStore(15 * time.Minute)
	done := make(chan bool)

	// Start multiple goroutines storing tokens
	for i := 0; i < 100; i++ {
		go func(id int) {
			token := &oauth2.Token{
				AccessToken: "access",
			}
			store.Store("state", token)
			store.Status("state")
			store.Get("state")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}
