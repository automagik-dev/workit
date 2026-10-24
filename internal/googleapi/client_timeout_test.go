package googleapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/automagik-dev/workit/internal/config"
)

// A body that streams for longer than the header timeout must not be cut off
// (issue #59), while a server that stalls before headers still times out.
func TestHTTPClientForScopes_SlowBodyOutlivesHeaderTimeout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))
	t.Setenv(httpTimeoutEnv, "100ms")

	saPath, err := config.ServiceAccountPath("a@b.com")
	if err != nil {
		t.Fatalf("ServiceAccountPath: %v", err)
	}
	if _, err := config.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if err := os.WriteFile(saPath, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("write sa: %v", err)
	}

	origSA := newServiceAccountTokenSource
	t.Cleanup(func() { newServiceAccountTokenSource = origSA })
	newServiceAccountTokenSource = func(context.Context, []byte, string, []string) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "t"}), nil
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stall" {
			time.Sleep(300 * time.Millisecond)
			return
		}
		w.WriteHeader(http.StatusOK)
		for range 4 {
			_, _ = io.WriteString(w, "chunk")
			w.(http.Flusher).Flush()
			time.Sleep(100 * time.Millisecond)
		}
	}))
	defer srv.Close()

	c, err := httpClientForScopes(context.Background(), "drive", "a@b.com", []string{"s1"})
	if err != nil {
		t.Fatalf("httpClientForScopes: %v", err)
	}
	if c.Timeout != 0 {
		t.Fatalf("http.Client.Timeout must be unset (it bounds body reads), got %v", c.Timeout)
	}

	resp, err := c.Get(srv.URL + "/slow")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("slow body read failed: %v", err)
	}
	if got := string(body); got != strings.Repeat("chunk", 4) {
		t.Fatalf("unexpected body %q", got)
	}

	resp, err = c.Get(srv.URL + "/stall")
	if err == nil {
		_ = resp.Body.Close()
		t.Fatalf("expected response header timeout")
	}
}
