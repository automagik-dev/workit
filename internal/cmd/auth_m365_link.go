package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/automagik-dev/workit/internal/googleauth"
	"github.com/automagik-dev/workit/internal/msauth"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

var createM365BrokerSessionOnServer = createRemoteM365BrokerSession

type AuthM365Cmd struct {
	LoginLink AuthM365LoginLinkCmd `cmd:"" name:"login-link" help:"Create a one-click Microsoft 365 read-only login link"`
}

type AuthM365LoginLinkCmd struct {
	Email        string        `arg:"" name:"email" help:"Expected Microsoft 365 account email"`
	BaseURL      string        `name:"base-url" help:"Public HTTPS broker base URL, e.g. https://login.workit.ai"`
	CallbackURL  string        `name:"callback-url" help:"Public HTTPS Microsoft OAuth callback URL"`
	BrokerToken  string        `name:"broker-token" help:"Bearer token allowed to create sessions on the M365 broker"`
	TTL          time.Duration `name:"ttl" help:"Login link validity duration" default:"10m"`
	ForceConsent bool          `name:"force-consent" help:"Force Microsoft consent screen"`
}

type remoteM365BrokerSessionRequest struct {
	ExpectedEmail string `json:"expected_email"`
	ForceConsent  bool   `json:"force_consent"`
}

func createRemoteM365BrokerSession(ctx context.Context, baseURL string, brokerToken string, payload remoteM365BrokerSessionRequest) (msauth.BrokerSession, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return msauth.BrokerSession{}, fmt.Errorf("encode m365 broker session request: %w", err)
	}

	sessionURL := strings.TrimRight(baseURL, "/") + "/m365/sessions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sessionURL, bytes.NewReader(body))
	if err != nil {
		return msauth.BrokerSession{}, fmt.Errorf("create m365 broker session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+brokerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return msauth.BrokerSession{}, fmt.Errorf("create m365 broker session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return msauth.BrokerSession{}, fmt.Errorf("create m365 broker session: status %d", resp.StatusCode)
	}

	var session msauth.BrokerSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return msauth.BrokerSession{}, fmt.Errorf("decode m365 broker session: %w", err)
	}
	if strings.TrimSpace(session.LoginURL) == "" {
		return msauth.BrokerSession{}, fmt.Errorf("decode m365 broker session: missing login_url")
	}

	return session, nil
}

func (c *AuthM365LoginLinkCmd) resolveBrokerURLs() (string, string, error) {
	baseURL := strings.TrimSpace(c.BaseURL)
	callbackURL := strings.TrimSpace(c.CallbackURL)
	if baseURL == "" {
		callbackServer, err := googleauth.CallbackServerURL("")
		if err != nil {
			return "", "", err
		}
		baseURL = strings.TrimRight(callbackServer, "/")
	}
	if callbackURL == "" {
		callbackURL = strings.TrimRight(baseURL, "/") + "/m365/callback"
	}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", usage("invalid m365 broker --base-url")
	}
	parsedCallback, err := url.Parse(callbackURL)
	if err != nil || parsedCallback.Scheme == "" || parsedCallback.Host == "" {
		return "", "", usage("invalid m365 broker --callback-url")
	}

	return baseURL, callbackURL, nil
}

func (c *AuthM365LoginLinkCmd) resolveBrokerToken() (string, error) {
	token := strings.TrimSpace(c.BrokerToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("WK_M365_BROKER_TOKEN"))
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv("WK_BROKER_ADMIN_TOKEN"))
	}
	if token == "" {
		return "", usage("m365 login-link requires --broker-token or WK_M365_BROKER_TOKEN")
	}

	return token, nil
}

func (c *AuthM365LoginLinkCmd) Run(ctx context.Context, _ *RootFlags) error {
	email := strings.TrimSpace(c.Email)
	if email == "" {
		return usage("empty email")
	}
	baseURL, callbackURL, err := c.resolveBrokerURLs()
	if err != nil {
		return err
	}
	_ = callbackURL // Validated for Entra redirect configuration parity; server derives it from base URL.
	brokerToken, err := c.resolveBrokerToken()
	if err != nil {
		return err
	}

	session, err := createM365BrokerSessionOnServer(ctx, baseURL, brokerToken, remoteM365BrokerSessionRequest{
		ExpectedEmail: email,
		ForceConsent:  c.ForceConsent,
	})
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, session)
	}

	u := ui.FromContext(ctx)
	u.Out().Printf("login_url\t%s", session.LoginURL)
	u.Out().Printf("expected_email\t%s", session.ExpectedEmail)
	u.Out().Printf("expires_at\t%s", session.ExpiresAt.Format(time.RFC3339))
	return nil
}
