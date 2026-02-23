package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/namastexlabs/gog-cli/internal/outfmt"
	"github.com/namastexlabs/gog-cli/internal/ui"
)

func TestDriveCheckPublicCmd_PublicFile_JSON(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/file123/permissions") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"permissions": []map[string]any{
					{"id": "perm1", "type": "user", "role": "owner", "emailAddress": "owner@example.com"},
					{"id": "perm2", "type": "anyone", "role": "reader"},
				},
			})
			return
		}
		http.NotFound(w, r)
	})
	defer srv.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "test@example.com"}
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	u, err := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx = ui.WithUI(ctx, u)

	out := captureStdout(t, func() {
		cmd := &DriveCheckPublicCmd{}
		if execErr := runKong(t, cmd, []string{"file123"}, ctx, flags); execErr != nil {
			t.Fatalf("execute: %v", execErr)
		}
	})

	var parsed struct {
		Public           bool              `json:"public"`
		DomainShared     bool              `json:"domain_shared"`
		Permission       *drive.Permission `json:"permission"`
		DomainPermission *drive.Permission `json:"domain_permission"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if !parsed.Public {
		t.Fatalf("expected public=true, got false")
	}
	if parsed.Permission == nil {
		t.Fatalf("expected permission to be set")
	}
	if parsed.Permission.Type != "anyone" {
		t.Fatalf("expected permission type=anyone, got %q", parsed.Permission.Type)
	}
	if parsed.Permission.Role != "reader" {
		t.Fatalf("expected permission role=reader, got %q", parsed.Permission.Role)
	}
	if parsed.DomainShared {
		t.Fatalf("expected domain_shared=false, got true")
	}
	if parsed.DomainPermission != nil {
		t.Fatalf("expected domain_permission to be nil, got %+v", parsed.DomainPermission)
	}
}

func TestDriveCheckPublicCmd_PrivateFile_JSON(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/file456/permissions") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"permissions": []map[string]any{
					{"id": "perm1", "type": "user", "role": "owner", "emailAddress": "owner@example.com"},
					{"id": "perm2", "type": "user", "role": "reader", "emailAddress": "reader@example.com"},
				},
			})
			return
		}
		http.NotFound(w, r)
	})
	defer srv.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "test@example.com"}
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	u, err := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx = ui.WithUI(ctx, u)

	out := captureStdout(t, func() {
		cmd := &DriveCheckPublicCmd{}
		if execErr := runKong(t, cmd, []string{"file456"}, ctx, flags); execErr != nil {
			t.Fatalf("execute: %v", execErr)
		}
	})

	var parsed struct {
		Public           bool              `json:"public"`
		DomainShared     bool              `json:"domain_shared"`
		Permission       *drive.Permission `json:"permission"`
		DomainPermission *drive.Permission `json:"domain_permission"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if parsed.Public {
		t.Fatalf("expected public=false, got true")
	}
	if parsed.Permission != nil {
		t.Fatalf("expected permission to be nil for private file, got %+v", parsed.Permission)
	}
	if parsed.DomainShared {
		t.Fatalf("expected domain_shared=false, got true")
	}
	if parsed.DomainPermission != nil {
		t.Fatalf("expected domain_permission to be nil, got %+v", parsed.DomainPermission)
	}
}

func TestDriveCheckPublicCmd_DomainShared_JSON(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/file789/permissions") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"permissions": []map[string]any{
					{"id": "perm1", "type": "user", "role": "owner", "emailAddress": "owner@example.com"},
					{"id": "perm2", "type": "domain", "role": "reader", "domain": "example.com"},
				},
			})
			return
		}
		http.NotFound(w, r)
	})
	defer srv.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "test@example.com"}
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	u, err := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx = ui.WithUI(ctx, u)

	out := captureStdout(t, func() {
		cmd := &DriveCheckPublicCmd{}
		if execErr := runKong(t, cmd, []string{"file789"}, ctx, flags); execErr != nil {
			t.Fatalf("execute: %v", execErr)
		}
	})

	var parsed struct {
		Public           bool              `json:"public"`
		DomainShared     bool              `json:"domain_shared"`
		Permission       *drive.Permission `json:"permission"`
		DomainPermission *drive.Permission `json:"domain_permission"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if parsed.Public {
		t.Fatalf("expected public=false for domain-shared file, got true")
	}
	if !parsed.DomainShared {
		t.Fatalf("expected domain_shared=true for domain-shared file, got false")
	}
	if parsed.Permission != nil {
		t.Fatalf("expected permission to be nil for domain-shared file, got %+v", parsed.Permission)
	}
	if parsed.DomainPermission == nil {
		t.Fatalf("expected domain_permission to be set for domain-shared file")
	}
	if parsed.DomainPermission.Type != "domain" {
		t.Fatalf("expected domain_permission type=domain, got %q", parsed.DomainPermission.Type)
	}
}

func TestDriveCheckPublicCmd_TextOutput(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/filetext/permissions") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"permissions": []map[string]any{
					{"id": "perm1", "type": "user", "role": "owner", "emailAddress": "owner@example.com"},
					{"id": "perm2", "type": "anyone", "role": "reader"},
				},
			})
			return
		}
		http.NotFound(w, r)
	})
	defer srv.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "test@example.com"}
	var outBuf bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})
	u, err := ui.New(ui.Options{Stdout: &outBuf, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx = ui.WithUI(ctx, u)

	cmd := &DriveCheckPublicCmd{}
	if execErr := runKong(t, cmd, []string{"filetext"}, ctx, flags); execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}

	out := outBuf.String()
	if !strings.Contains(out, "true") {
		t.Fatalf("expected text output to contain 'true', got: %q", out)
	}
	if !strings.Contains(out, "anyone") {
		t.Fatalf("expected text output to contain 'anyone', got: %q", out)
	}
}

func TestDriveCheckPublicCmd_EmptyFileID(t *testing.T) {
	cmd := &DriveCheckPublicCmd{}
	flags := &RootFlags{Account: "test@example.com"}
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})
	u, err := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx = ui.WithUI(ctx, u)

	execErr := runKong(t, cmd, []string{""}, ctx, flags)
	if execErr == nil {
		t.Fatalf("expected error for empty fileId")
	}
	if !strings.Contains(execErr.Error(), "empty fileId") {
		t.Fatalf("expected 'empty fileId' error, got: %v", execErr)
	}
}

func TestDriveCheckPublicCmd_PaginatedPermissions_JSON(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/filepag/permissions") {
			w.Header().Set("Content-Type", "application/json")
			pageToken := r.URL.Query().Get("pageToken")
			switch pageToken {
			case "":
				// First page: only user permissions, with a nextPageToken.
				_ = json.NewEncoder(w).Encode(map[string]any{
					"permissions": []map[string]any{
						{"id": "perm1", "type": "user", "role": "owner", "emailAddress": "owner@example.com"},
					},
					"nextPageToken": "page2",
				})
			case "page2":
				// Second page: contains an "anyone" permission.
				_ = json.NewEncoder(w).Encode(map[string]any{
					"permissions": []map[string]any{
						{"id": "perm2", "type": "anyone", "role": "reader"},
					},
				})
			default:
				http.NotFound(w, r)
			}
			return
		}
		http.NotFound(w, r)
	})
	defer srv.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "test@example.com"}
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	u, err := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx = ui.WithUI(ctx, u)

	out := captureStdout(t, func() {
		cmd := &DriveCheckPublicCmd{}
		if execErr := runKong(t, cmd, []string{"filepag"}, ctx, flags); execErr != nil {
			t.Fatalf("execute: %v", execErr)
		}
	})

	var parsed struct {
		Public           bool              `json:"public"`
		DomainShared     bool              `json:"domain_shared"`
		Permission       *drive.Permission `json:"permission"`
		DomainPermission *drive.Permission `json:"domain_permission"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if !parsed.Public {
		t.Fatalf("expected public=true after pagination, got false")
	}
	if parsed.Permission == nil {
		t.Fatalf("expected permission to be set")
	}
	if parsed.Permission.Type != "anyone" {
		t.Fatalf("expected permission type=anyone, got %q", parsed.Permission.Type)
	}
	if parsed.DomainShared {
		t.Fatalf("expected domain_shared=false, got true")
	}
	if parsed.DomainPermission != nil {
		t.Fatalf("expected domain_permission to be nil, got %+v", parsed.DomainPermission)
	}
}

// newTestServer creates an httptest.Server from a handler function.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}
