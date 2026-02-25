package googleapi

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/api/forms/v1"

	"github.com/automagik-dev/workit/internal/googleauth"
)

func NewForms(ctx context.Context, email string) (*forms.Service, error) {
	if opts, err := optionsForAccount(ctx, googleauth.ServiceForms, email); err != nil {
		return nil, fmt.Errorf("forms options: %w", err)
	} else if svc, err := forms.NewService(ctx, opts...); err != nil {
		return nil, fmt.Errorf("create forms service: %w", err)
	} else {
		return svc, nil
	}
}

// NewFormsHTTPClient returns an authenticated *http.Client for the Forms API.
// Use this for raw HTTP calls to endpoints not covered by the generated client
// (e.g., setPublishSettings).
func NewFormsHTTPClient(ctx context.Context, email string) (*http.Client, error) {
	return HTTPClientForService(ctx, googleauth.ServiceForms, email)
}

// HTTPClientForService returns an authenticated *http.Client for the given
// Google API service. Useful for making raw HTTP calls to endpoints that the
// generated Go client library does not cover.
func HTTPClientForService(ctx context.Context, service googleauth.Service, email string) (*http.Client, error) {
	scopes, err := googleauth.Scopes(service)
	if err != nil {
		return nil, fmt.Errorf("resolve scopes for %s: %w", service, err)
	}

	return httpClientForScopes(ctx, string(service), email, scopes)
}
