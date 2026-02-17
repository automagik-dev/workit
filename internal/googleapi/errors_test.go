package googleapi

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	errBase                = errors.New("base")
	errNotUsedPeople       = errors.New("accessNotConfigured: People API has not been used in project 12345")
	errDisabledClassroom   = errors.New("classroom api: it is disabled")
	errRandom              = errors.New("some random error")
	errUnusedProject       = errors.New("API has not been used in project")
	errSomething           = errors.New("something")
	errAccessNotConfigured = errors.New("accessNotConfigured: API not enabled")
	errPeopleNotUsed       = errors.New("people API has not been used in project")
	errSomeAPIDisabled     = errors.New("some API: it is disabled for this project")
)

func TestErrors_IsHelpers(t *testing.T) {
	if !IsAuthRequiredError(&AuthRequiredError{Service: "gmail", Email: "a@b.com", Cause: errBase}) {
		t.Fatalf("expected IsAuthRequiredError")
	}

	if !IsRateLimitError(&RateLimitError{RetryAfter: time.Second, Retries: 2}) {
		t.Fatalf("expected IsRateLimitError")
	}

	if !IsCircuitBreakerError(&CircuitBreakerError{}) {
		t.Fatalf("expected IsCircuitBreakerError")
	}

	if !IsQuotaExceededError(&QuotaExceededError{Resource: "gmail"}) {
		t.Fatalf("expected IsQuotaExceededError")
	}

	if !IsNotFoundError(&NotFoundError{Resource: "msg", ID: "id"}) {
		t.Fatalf("expected IsNotFoundError")
	}

	if !IsPermissionDeniedError(&PermissionDeniedError{Resource: "file", Action: "read"}) {
		t.Fatalf("expected IsPermissionDeniedError")
	}
}

func TestErrors_Messages(t *testing.T) {
	authErr := &AuthRequiredError{Service: "gmail", Email: "a@b.com", Cause: errBase}
	if got := authErr.Error(); got != "auth required for gmail a@b.com" {
		t.Fatalf("unexpected: %q", got)
	}

	if !errors.Is(authErr, errBase) {
		t.Fatalf("expected unwrap to match base")
	}

	if got := (&RateLimitError{RetryAfter: 2 * time.Second, Retries: 3}).Error(); !strings.Contains(got, "retry after 2s") {
		t.Fatalf("unexpected: %q", got)
	}

	if got := (&RateLimitError{Retries: 1}).Error(); !strings.Contains(got, "after 1 retries") {
		t.Fatalf("unexpected: %q", got)
	}

	if got := (&NotFoundError{Resource: "file"}).Error(); got != "file not found" {
		t.Fatalf("unexpected: %q", got)
	}

	if got := (&NotFoundError{Resource: "file", ID: "id1"}).Error(); got != "file not found: id1" {
		t.Fatalf("unexpected: %q", got)
	}

	if got := (&PermissionDeniedError{Resource: "file"}).Error(); got != "permission denied for file" {
		t.Fatalf("unexpected: %q", got)
	}

	if got := (&PermissionDeniedError{Resource: "file", Action: "delete"}).Error(); got != "permission denied: cannot delete file" {
		t.Fatalf("unexpected: %q", got)
	}

	if got := (&CircuitBreakerError{}).Error(); got == "" {
		t.Fatalf("expected circuit breaker message")
	}

	if got := (&QuotaExceededError{Resource: "drive"}).Error(); got != "API quota exceeded for drive" {
		t.Fatalf("unexpected: %q", got)
	}

	if got := (&QuotaExceededError{}).Error(); got != "API quota exceeded" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestWrapAPIEnablementError_NilError(t *testing.T) {
	if got := WrapAPIEnablementError(nil, "drive"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrapAPIEnablementError_HasNotBeenUsed(t *testing.T) {
	err := errNotUsedPeople
	wrapped := WrapAPIEnablementError(err, "people")

	if !errors.Is(wrapped, err) {
		t.Fatal("expected error to be wrapped")
	}

	msg := wrapped.Error()
	if !strings.Contains(msg, "people API is not enabled") {
		t.Errorf("expected enablement message, got %q", msg)
	}

	if !strings.Contains(msg, "people.googleapis.com") {
		t.Errorf("expected console URL, got %q", msg)
	}
}

func TestWrapAPIEnablementError_IsDisabled(t *testing.T) {
	err := errDisabledClassroom
	wrapped := WrapAPIEnablementError(err, "classroom")

	if !errors.Is(wrapped, err) {
		t.Fatal("expected error to be wrapped")
	}

	if !strings.Contains(wrapped.Error(), "classroom.googleapis.com") {
		t.Errorf("expected console URL, got %q", wrapped.Error())
	}
}

func TestWrapAPIEnablementError_Passthrough(t *testing.T) {
	err := errRandom
	got := WrapAPIEnablementError(err, "drive")

	if !errors.Is(got, err) {
		t.Fatalf("expected original error to be preserved, got %v", got)
	}
}

func TestWrapAPIEnablementError_UnknownService(t *testing.T) {
	err := errUnusedProject

	wrapped := WrapAPIEnablementError(err, "unknownservice")
	if !strings.Contains(wrapped.Error(), "check the Google Cloud Console") {
		t.Errorf("expected fallback message, got %q", wrapped.Error())
	}
}

func TestAPIEnablementLinks_Coverage(t *testing.T) {
	expected := []string{
		"calendar", "drive", "gmail", "docs", "sheets", "slides",
		"forms", "tasks", "chat", "people", "classroom", "cloudidentity",
	}
	for _, svc := range expected {
		if _, ok := APIEnablementLinks[svc]; !ok {
			t.Errorf("missing APIEnablementLinks entry for %q", svc)
		}
	}
}

func TestIsAPINotEnabledError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"random error", errSomething, false},
		{"accessNotConfigured", errAccessNotConfigured, true},
		{"has not been used", errPeopleNotUsed, true},
		{"it is disabled", errSomeAPIDisabled, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAPINotEnabledError(tt.err); got != tt.want {
				t.Errorf("IsAPINotEnabledError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTransientStatusCode(t *testing.T) {
	transient := []int{429, 500, 502, 503, 504}
	for _, code := range transient {
		if !IsTransientStatusCode(code) {
			t.Errorf("expected %d to be transient", code)
		}
	}

	fatal := []int{200, 201, 400, 401, 403, 404, 409}
	for _, code := range fatal {
		if IsTransientStatusCode(code) {
			t.Errorf("expected %d to not be transient", code)
		}
	}
}
