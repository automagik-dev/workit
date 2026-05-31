package msauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrBrokerStateNotFound = errors.New("m365 broker state not found")
	ErrBrokerStateExpired  = errors.New("m365 broker state expired")
	ErrBrokerEmailMismatch = errors.New("m365 broker authorized email mismatch")
)

type BrokerStore interface {
	Save(context.Context, BrokerSession) error
	Consume(context.Context, string) (BrokerSession, error)
}

type MemoryBrokerStore struct {
	mu       sync.Mutex
	sessions map[string]BrokerSession
}

func NewMemoryBrokerStore() *MemoryBrokerStore {
	return &MemoryBrokerStore{sessions: make(map[string]BrokerSession)}
}

func (s *MemoryBrokerStore) Save(_ context.Context, session BrokerSession) error {
	state := strings.TrimSpace(session.State)
	if state == "" {
		return ErrBrokerStateNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessions == nil {
		s.sessions = make(map[string]BrokerSession)
	}

	s.sessions[state] = session

	return nil
}

func (s *MemoryBrokerStore) Consume(_ context.Context, state string) (BrokerSession, error) {
	key := strings.TrimSpace(state)
	if key == "" {
		return BrokerSession{}, ErrBrokerStateNotFound
	}

	s.mu.Lock()

	session, ok := s.sessions[key]
	if ok {
		delete(s.sessions, key)
	}

	s.mu.Unlock()

	if !ok {
		return BrokerSession{}, ErrBrokerStateNotFound
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return BrokerSession{}, ErrBrokerStateExpired
	}

	return session, nil
}

func ValidateBrokerAuthorizedEmail(session BrokerSession, authorizedEmail string) error {
	expected := strings.ToLower(strings.TrimSpace(session.ExpectedEmail))
	actual := strings.ToLower(strings.TrimSpace(authorizedEmail))

	if expected == "" || actual == "" || expected != actual {
		return fmt.Errorf("%w: expected %s got %s", ErrBrokerEmailMismatch, expected, actual)
	}

	return nil
}
