package googleapi

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type AuthRequiredError struct {
	Service string
	Email   string
	Client  string
	Cause   error
}

func (e *AuthRequiredError) Error() string {
	if e.Client != "" {
		return fmt.Sprintf("auth required for %s %s (client %s)", e.Service, e.Email, e.Client)
	}

	return fmt.Sprintf("auth required for %s %s", e.Service, e.Email)
}

func (e *AuthRequiredError) Unwrap() error {
	return e.Cause
}

// RateLimitError indicates rate limit was exceeded
type RateLimitError struct {
	RetryAfter time.Duration
	Retries    int
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded, retry after %s (attempted %d retries)", e.RetryAfter, e.Retries)
	}

	return fmt.Sprintf("rate limit exceeded after %d retries", e.Retries)
}

// CircuitBreakerError indicates the circuit breaker is open
type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "circuit breaker is open, too many recent failures - try again later"
}

// QuotaExceededError indicates API quota was exceeded
type QuotaExceededError struct {
	Resource string
}

func (e *QuotaExceededError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("API quota exceeded for %s", e.Resource)
	}

	return "API quota exceeded"
}

// NotFoundError indicates the requested resource was not found
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}

	return fmt.Sprintf("%s not found", e.Resource)
}

// PermissionDeniedError indicates insufficient permissions
type PermissionDeniedError struct {
	Resource string
	Action   string
}

func (e *PermissionDeniedError) Error() string {
	if e.Action != "" {
		return fmt.Sprintf("permission denied: cannot %s %s", e.Action, e.Resource)
	}

	return fmt.Sprintf("permission denied for %s", e.Resource)
}

// IsAuthRequiredError checks if the error is an auth required error
func IsAuthRequiredError(err error) bool {
	var e *AuthRequiredError
	return errors.As(err, &e)
}

// IsRateLimitError checks if the error is a rate limit error
func IsRateLimitError(err error) bool {
	var e *RateLimitError
	return errors.As(err, &e)
}

// IsCircuitBreakerError checks if the error is a circuit breaker error
func IsCircuitBreakerError(err error) bool {
	var e *CircuitBreakerError
	return errors.As(err, &e)
}

// IsQuotaExceededError checks if the error is a quota exceeded error
func IsQuotaExceededError(err error) bool {
	var e *QuotaExceededError
	return errors.As(err, &e)
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	var e *NotFoundError
	return errors.As(err, &e)
}

// IsPermissionDeniedError checks if the error is a permission denied error
func IsPermissionDeniedError(err error) bool {
	var e *PermissionDeniedError
	return errors.As(err, &e)
}

// APIEnablementLinks maps service names to their GCP console enablement URLs.
var APIEnablementLinks = map[string]string{
	"calendar":      "https://console.developers.google.com/apis/api/calendar-json.googleapis.com/overview",
	"drive":         "https://console.developers.google.com/apis/api/drive.googleapis.com/overview",
	"gmail":         "https://console.developers.google.com/apis/api/gmail.googleapis.com/overview",
	"docs":          "https://console.developers.google.com/apis/api/docs.googleapis.com/overview",
	"sheets":        "https://console.developers.google.com/apis/api/sheets.googleapis.com/overview",
	"slides":        "https://console.developers.google.com/apis/api/slides.googleapis.com/overview",
	"forms":         "https://console.developers.google.com/apis/api/forms.googleapis.com/overview",
	"tasks":         "https://console.developers.google.com/apis/api/tasks.googleapis.com/overview",
	"chat":          "https://console.developers.google.com/apis/api/chat.googleapis.com/overview",
	"people":        "https://console.developers.google.com/apis/api/people.googleapis.com/overview",
	"classroom":     "https://console.developers.google.com/apis/api/classroom.googleapis.com/overview",
	"cloudidentity": "https://console.developers.google.com/apis/api/cloudidentity.googleapis.com/overview",
}

// IsAPINotEnabledError checks whether an error message indicates an API that
// has not been enabled in the GCP project.
func IsAPINotEnabledError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "accessNotConfigured") ||
		strings.Contains(msg, "has not been used") ||
		strings.Contains(msg, "it is disabled") ||
		strings.Contains(msg, "API has not been used")
}

// WrapAPIEnablementError checks if err is an "API not enabled" error and wraps it
// with a helpful hint including the GCP console URL. Returns the original error
// if not applicable.
func WrapAPIEnablementError(err error, serviceName string) error {
	if err == nil {
		return nil
	}
	if !IsAPINotEnabledError(err) {
		return err
	}
	link, ok := APIEnablementLinks[serviceName]
	if !ok {
		return fmt.Errorf("%s API is not enabled; check the Google Cloud Console to enable it (%w)", serviceName, err)
	}
	return fmt.Errorf("%s API is not enabled; enable it at: %s (%w)", serviceName, link, err)
}

// IsTransientStatusCode returns true if the HTTP status code indicates a transient
// (retryable) error: 429 (rate limit), 500, 502, 503, 504 (server errors).
func IsTransientStatusCode(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}
