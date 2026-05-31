package msauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryBrokerStoreConsumesSessionOnce(t *testing.T) {
	store := NewMemoryBrokerStore()
	session := BrokerSession{State: "state", ExpectedEmail: "pilot@example.com", CodeVerifier: "verifier", ExpiresAt: time.Now().Add(time.Minute)}

	if err := store.Save(context.Background(), session); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.Consume(context.Background(), "state")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	if got.ExpectedEmail != "pilot@example.com" || got.CodeVerifier != "verifier" {
		t.Fatalf("session = %#v", got)
	}

	_, err = store.Consume(context.Background(), "state")
	if !errors.Is(err, ErrBrokerStateNotFound) {
		t.Fatalf("expected one-time state miss, got %v", err)
	}
}

func TestMemoryBrokerStoreZeroValueSaveInitializesSessionMap(t *testing.T) {
	var store MemoryBrokerStore
	session := BrokerSession{State: "state", ExpectedEmail: "pilot@example.com", ExpiresAt: time.Now().Add(time.Minute)}

	if err := store.Save(context.Background(), session); err != nil {
		t.Fatalf("save zero-value store: %v", err)
	}

	got, err := store.Consume(context.Background(), "state")
	if err != nil {
		t.Fatalf("consume zero-value store: %v", err)
	}

	if got.ExpectedEmail != "pilot@example.com" {
		t.Fatalf("session = %#v", got)
	}
}

func TestMemoryBrokerStoreRejectsExpiredSession(t *testing.T) {
	store := NewMemoryBrokerStore()
	session := BrokerSession{State: "expired", ExpectedEmail: "pilot@example.com", ExpiresAt: time.Now().Add(-time.Minute)}

	if err := store.Save(context.Background(), session); err != nil {
		t.Fatalf("save: %v", err)
	}

	_, err := store.Consume(context.Background(), "expired")
	if !errors.Is(err, ErrBrokerStateExpired) {
		t.Fatalf("expected expired state, got %v", err)
	}
}

func TestValidateBrokerAuthorizedEmailFailsClosedOnMismatch(t *testing.T) {
	err := ValidateBrokerAuthorizedEmail(BrokerSession{ExpectedEmail: "bernardo@hapvida.com.br"}, "other@hapvida.com.br")
	if !errors.Is(err, ErrBrokerEmailMismatch) {
		t.Fatalf("expected email mismatch, got %v", err)
	}

	if err := ValidateBrokerAuthorizedEmail(BrokerSession{ExpectedEmail: "Bernardo@Hapvida.com.br"}, "bernardo@hapvida.com.br"); err != nil {
		t.Fatalf("expected normalized email match, got %v", err)
	}
}
