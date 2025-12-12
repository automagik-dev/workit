package googleauth

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/steipete/gogcli/internal/config"
)

type AuthorizeOptions struct {
	Services     []Service
	Scopes       []string
	Manual       bool
	ForceConsent bool
	Timeout      time.Duration
}

func Authorize(ctx context.Context, opts AuthorizeOptions) (string, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}
	if len(opts.Scopes) == 0 {
		return "", errors.New("missing scopes")
	}
	creds, err := config.ReadClientCredentials()
	if err != nil {
		return "", err
	}

	state, err := randomState()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if opts.Manual {
		redirectURI := "http://localhost:1"
		cfg := oauth2.Config{
			ClientID:     creds.ClientID,
			ClientSecret: creds.ClientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  redirectURI,
			Scopes:       opts.Scopes,
		}
		authURL := cfg.AuthCodeURL(state, authURLParams(opts.ForceConsent)...)
		fmt.Fprintln(os.Stderr, "Visit this URL to authorize:")
		fmt.Fprintln(os.Stderr, authURL)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "After authorizing, you'll be redirected to a localhost URL that won't load.")
		fmt.Fprintln(os.Stderr, "Copy the URL from your browser's address bar and paste it here.")
		fmt.Fprintln(os.Stderr)

		fmt.Fprint(os.Stderr, "Paste redirect URL: ")
		line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		if readErr != nil && !errors.Is(readErr, os.ErrClosed) {
			return "", readErr
		}
		line = strings.TrimSpace(line)
		code, gotState, parseErr := extractCodeAndState(line)
		if parseErr != nil {
			return "", parseErr
		}
		if gotState != "" && gotState != state {
			return "", errors.New("state mismatch")
		}
		tok, exchangeErr := cfg.Exchange(ctx, code)
		if exchangeErr != nil {
			return "", exchangeErr
		}
		if tok.RefreshToken == "" {
			return "", errors.New("no refresh token received; try again with --force-consent")
		}
		return tok.RefreshToken, nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/oauth2/callback", port)

	cfg := oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
		Scopes:       opts.Scopes,
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth2/callback" {
				http.NotFound(w, r)
				return
			}
			q := r.URL.Query()
			if q.Get("error") != "" {
				select {
				case errCh <- fmt.Errorf("authorization error: %s", q.Get("error")):
				default:
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Authorization cancelled. You can close this window."))
				return
			}
			if q.Get("state") != state {
				select {
				case errCh <- errors.New("state mismatch"):
				default:
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("State mismatch. You can close this window."))
				return
			}
			code := q.Get("code")
			if code == "" {
				select {
				case errCh <- errors.New("missing code"):
				default:
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("Missing code. You can close this window."))
				return
			}
			select {
			case codeCh <- code:
			default:
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Success! You can close this window."))
		}),
	}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	authURL := cfg.AuthCodeURL(state, authURLParams(opts.ForceConsent)...)
	fmt.Fprintln(os.Stderr, "Opening browser for authorization…")
	fmt.Fprintln(os.Stderr, "If the browser doesn't open, visit this URL:")
	fmt.Fprintln(os.Stderr, authURL)
	_ = openBrowser(authURL)

	select {
	case code := <-codeCh:
		_ = srv.Close()
		tok, exchangeErr := cfg.Exchange(ctx, code)
		if exchangeErr != nil {
			return "", exchangeErr
		}
		if tok.RefreshToken == "" {
			return "", errors.New("no refresh token received; try again with --force-consent")
		}
		return tok.RefreshToken, nil
	case err := <-errCh:
		_ = srv.Close()
		return "", err
	case <-ctx.Done():
		_ = srv.Close()
		return "", ctx.Err()
	}
}

func authURLParams(forceConsent bool) []oauth2.AuthCodeOption {
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	}
	if forceConsent {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "consent"))
	}
	return opts
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func extractCodeAndState(rawURL string) (code string, state string, err error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}
	q := parsed.Query()
	code = q.Get("code")
	if code == "" {
		return "", "", errors.New("no code found in URL")
	}
	return code, q.Get("state"), nil
}
