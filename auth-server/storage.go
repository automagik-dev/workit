// Package main provides the auth callback server for headless OAuth.
package main

import (
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// TokenEntry holds the OAuth token along with metadata for TTL management.
type TokenEntry struct {
	Token     *oauth2.Token
	CreatedAt time.Time
	Consumed  bool
}

// TokenStore provides thread-safe in-memory storage for OAuth tokens with TTL.
type TokenStore struct {
	mu       sync.RWMutex
	tokens   map[string]*TokenEntry
	ttl      time.Duration
	stopChan chan struct{}
}

// NewTokenStore creates a new TokenStore with the specified TTL.
func NewTokenStore(ttl time.Duration) *TokenStore {
	return &TokenStore{
		tokens:   make(map[string]*TokenEntry),
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}
}

// Store saves a token with the given state key.
func (s *TokenStore) Store(state string, token *oauth2.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[state] = &TokenEntry{
		Token:     token,
		CreatedAt: time.Now(),
		Consumed:  false,
	}
}

// TokenStatus represents the status of a token lookup.
type TokenStatus int

const (
	// TokenStatusReady means the token is available and ready to be consumed.
	TokenStatusReady TokenStatus = iota
	// TokenStatusPending means the state exists but token has not arrived yet.
	TokenStatusPending
	// TokenStatusNotFound means the state does not exist or has expired.
	TokenStatusNotFound
	// TokenStatusConsumed means the token was already retrieved.
	TokenStatusConsumed
)

// Get retrieves and consumes a token for the given state.
// Returns the token and its status.
func (s *TokenStore) Get(state string) (*oauth2.Token, TokenStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.tokens[state]
	if !exists {
		return nil, TokenStatusNotFound
	}

	// Check if expired
	if time.Since(entry.CreatedAt) > s.ttl {
		delete(s.tokens, state)
		return nil, TokenStatusNotFound
	}

	// Check if already consumed
	if entry.Consumed {
		return nil, TokenStatusConsumed
	}

	// Check if token is still pending (nil)
	if entry.Token == nil {
		return nil, TokenStatusPending
	}

	// Mark as consumed and return
	entry.Consumed = true
	return entry.Token, TokenStatusReady
}

// Status checks the status of a token without consuming it.
func (s *TokenStore) Status(state string) TokenStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.tokens[state]
	if !exists {
		return TokenStatusNotFound
	}

	// Check if expired
	if time.Since(entry.CreatedAt) > s.ttl {
		return TokenStatusNotFound
	}

	if entry.Consumed {
		return TokenStatusConsumed
	}

	if entry.Token != nil {
		return TokenStatusReady
	}

	return TokenStatusPending
}

// MarkPending creates a placeholder entry for a state that is awaiting a token.
// This allows distinguishing between "pending" and "not found" states.
func (s *TokenStore) MarkPending(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[state] = &TokenEntry{
		Token:     nil,
		CreatedAt: time.Now(),
		Consumed:  false,
	}
}

// StartCleanup starts a background goroutine that periodically removes expired entries.
func (s *TokenStore) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.cleanup()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// StopCleanup stops the background cleanup goroutine.
func (s *TokenStore) StopCleanup() {
	close(s.stopChan)
}

// cleanup removes all expired entries from the store.
func (s *TokenStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for state, entry := range s.tokens {
		if now.Sub(entry.CreatedAt) > s.ttl {
			delete(s.tokens, state)
		}
	}
}
