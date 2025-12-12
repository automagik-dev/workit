package googleapi

import (
	"context"

	"github.com/99designs/keyring"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/config"
	"github.com/steipete/gogcli/internal/googleauth"
	"github.com/steipete/gogcli/internal/secrets"
)

func tokenSourceForAccount(ctx context.Context, service googleauth.Service, email string) (oauth2.TokenSource, error) {
	creds, err := config.ReadClientCredentials()
	if err != nil {
		return nil, err
	}

	requiredScopes, err := googleauth.Scopes(service)
	if err != nil {
		return nil, err
	}

	store, err := secrets.OpenDefault()
	if err != nil {
		return nil, err
	}
	tok, err := store.GetToken(email)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, &AuthRequiredError{Service: string(service), Email: email, Cause: err}
		}
		return nil, err
	}

	if len(tok.Scopes) > 0 {
		have := make(map[string]struct{}, len(tok.Scopes))
		for _, s := range tok.Scopes {
			have[s] = struct{}{}
		}
		missing := make([]string, 0)
		for _, want := range requiredScopes {
			if _, ok := have[want]; !ok {
				missing = append(missing, want)
			}
		}
		if len(missing) > 0 {
			return nil, &MissingScopesError{Service: string(service), Email: email, Missing: missing}
		}
	}

	cfg := oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       requiredScopes,
	}

	return cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: tok.RefreshToken}), nil
}

func optionsForAccount(ctx context.Context, service googleauth.Service, email string) ([]option.ClientOption, error) {
	ts, err := tokenSourceForAccount(ctx, service, email)
	if err != nil {
		return nil, err
	}
	return []option.ClientOption{option.WithTokenSource(ts)}, nil
}
