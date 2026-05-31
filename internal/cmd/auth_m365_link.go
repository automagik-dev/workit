package cmd

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/automagik-dev/workit/internal/msauth"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

var createM365BrokerSession = msauth.CreateBrokerSession

type AuthM365Cmd struct {
	LoginLink AuthM365LoginLinkCmd `cmd:"" name:"login-link" help:"Create a one-click Microsoft 365 read-only login link"`
}

type AuthM365LoginLinkCmd struct {
	Email        string        `arg:"" name:"email" help:"Expected Microsoft 365 account email"`
	BaseURL      string        `name:"base-url" help:"Public HTTPS broker base URL, e.g. https://login.workit.ai"`
	CallbackURL  string        `name:"callback-url" help:"Public HTTPS Microsoft OAuth callback URL"`
	TTL          time.Duration `name:"ttl" help:"Login link validity duration" default:"10m"`
	ForceConsent bool          `name:"force-consent" help:"Force Microsoft consent screen"`
}

func (c *AuthM365LoginLinkCmd) Run(ctx context.Context, _ *RootFlags) error {
	email := strings.TrimSpace(c.Email)
	if email == "" {
		return usage("empty email")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return usage("m365 login-link requires --base-url")
	}
	if strings.TrimSpace(c.CallbackURL) == "" {
		return usage("m365 login-link requires --callback-url")
	}

	session, err := createM365BrokerSession(ctx, msauth.BrokerSessionOptions{
		ExpectedEmail: email,
		BaseURL:       c.BaseURL,
		CallbackURL:   c.CallbackURL,
		Readonly:      true,
		ForceConsent:  c.ForceConsent,
		TTL:           c.TTL,
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
