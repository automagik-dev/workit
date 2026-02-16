package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

func TestDocsGenerate_JSON_Basic(t *testing.T) {
	origDocs := newDocsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newDocsService = origDocs
		newDriveService = origDrive
	})

	var copyFile *drive.File
	var batchUpdateBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Drive Files.Copy: POST .../files/{id}/copy
		if strings.HasSuffix(r.URL.Path, "/copy") && r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&copyFile)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "new-doc-id",
				"name":        "Generated Doc",
				"mimeType":    "application/vnd.google-apps.document",
				"webViewLink": "https://docs.google.com/document/d/new-doc-id/edit",
			})
			return
		}

		// Docs batchUpdate
		if strings.HasSuffix(r.URL.Path, ":batchUpdate") && r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&batchUpdateBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "new-doc-id",
				"replies": []any{
					map[string]any{
						"replaceAllText": map[string]any{
							"occurrencesChanged": 1,
						},
					},
					map[string]any{
						"replaceAllText": map[string]any{
							"occurrencesChanged": 1,
						},
					},
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDriveService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	// Create a temp data file
	dataFile := filepath.Join(t.TempDir(), "data.json")
	dataContent := `{"name": "Acme Corp", "date": "2026-01-15"}`
	if err := os.WriteFile(dataFile, []byte(dataContent), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	out := captureStdout(t, func() {
		cmd := &DocsGenerateCmd{}
		if err := runKong(t, cmd, []string{"--template", "template-doc-id", "--data", dataFile}, ctx, flags); err != nil {
			t.Fatalf("generate: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	if result["documentId"] != "new-doc-id" {
		t.Errorf("expected documentId 'new-doc-id', got %v", result["documentId"])
	}
	if result["url"] != "https://docs.google.com/document/d/new-doc-id/edit" {
		t.Errorf("expected url with new-doc-id, got %v", result["url"])
	}

	// Verify batchUpdate had ReplaceAllText requests
	if batchUpdateBody == nil {
		t.Fatal("expected batchUpdate to be called")
	}
	requests, ok := batchUpdateBody["requests"].([]any)
	if !ok {
		t.Fatalf("expected requests array in batchUpdate body, got: %v", batchUpdateBody)
	}
	if len(requests) != 2 {
		t.Errorf("expected 2 replace requests (name and date), got %d", len(requests))
	}
}

func TestDocsGenerate_WithFolder(t *testing.T) {
	origDocs := newDocsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newDocsService = origDocs
		newDriveService = origDrive
	})

	var capturedCopyBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Drive Files.Copy
		if strings.HasSuffix(r.URL.Path, "/copy") && r.Method == http.MethodPost {
			capturedCopyBody, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "new-doc-id",
				"name":        "Generated Doc",
				"mimeType":    "application/vnd.google-apps.document",
				"webViewLink": "https://docs.google.com/document/d/new-doc-id/edit",
			})
			return
		}

		// Docs batchUpdate
		if strings.HasSuffix(r.URL.Path, ":batchUpdate") && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "new-doc-id",
				"replies":    []any{},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDriveService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	dataFile := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(dataFile, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	captureStdout(t, func() {
		cmd := &DocsGenerateCmd{}
		if err := runKong(t, cmd, []string{
			"--template", "tmpl-id",
			"--data", dataFile,
			"--folder", "folder-123",
		}, ctx, flags); err != nil {
			t.Fatalf("generate with folder: %v", err)
		}
	})

	// Verify folder was included in copy request
	if capturedCopyBody == nil {
		t.Fatal("expected copy call to be made")
	}
	var copyReq map[string]any
	if err := json.Unmarshal(capturedCopyBody, &copyReq); err != nil {
		t.Fatalf("parse copy body: %v\nraw: %s", err, capturedCopyBody)
	}
	parents, ok := copyReq["parents"].([]any)
	if !ok || len(parents) == 0 {
		t.Fatalf("expected parents to contain folder ID, got: %v", copyReq["parents"])
	}
	if parents[0] != "folder-123" {
		t.Errorf("expected parent 'folder-123', got %v", parents[0])
	}
}

func TestDocsGenerate_WithTitle(t *testing.T) {
	origDocs := newDocsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newDocsService = origDocs
		newDriveService = origDrive
	})

	var capturedCopyBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/copy") && r.Method == http.MethodPost {
			capturedCopyBody, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "new-doc-id",
				"name":        "Custom Title",
				"mimeType":    "application/vnd.google-apps.document",
				"webViewLink": "https://docs.google.com/document/d/new-doc-id/edit",
			})
			return
		}

		if strings.HasSuffix(r.URL.Path, ":batchUpdate") && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "new-doc-id",
				"replies":    []any{},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDriveService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	dataFile := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(dataFile, []byte(`{"x": "y"}`), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	captureStdout(t, func() {
		cmd := &DocsGenerateCmd{}
		if err := runKong(t, cmd, []string{
			"--template", "tmpl-id",
			"--data", dataFile,
			"--title", "Custom Title",
		}, ctx, flags); err != nil {
			t.Fatalf("generate with title: %v", err)
		}
	})

	if capturedCopyBody == nil {
		t.Fatal("expected copy call to be made")
	}
	var copyReq map[string]any
	if err := json.Unmarshal(capturedCopyBody, &copyReq); err != nil {
		t.Fatalf("parse copy body: %v", err)
	}
	if copyReq["name"] != "Custom Title" {
		t.Errorf("expected name 'Custom Title', got %v", copyReq["name"])
	}
}

func TestDocsGenerate_PlainOutput(t *testing.T) {
	origDocs := newDocsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newDocsService = origDocs
		newDriveService = origDrive
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/copy") && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "new-doc-id",
				"name":        "Generated Doc",
				"mimeType":    "application/vnd.google-apps.document",
				"webViewLink": "https://docs.google.com/document/d/new-doc-id/edit",
			})
			return
		}

		if strings.HasSuffix(r.URL.Path, ":batchUpdate") && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "new-doc-id",
				"replies":    []any{},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDriveService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	dataFile := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(dataFile, []byte(`{"k": "v"}`), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	flags := &RootFlags{Account: "a@b.com"}

	out := captureStdout(t, func() {
		u, _ := ui.New(ui.Options{Stdout: os.Stdout, Stderr: io.Discard, Color: "never"})
		ctx := ui.WithUI(context.Background(), u)
		cmd := &DocsGenerateCmd{}
		if err := runKong(t, cmd, []string{"--template", "tmpl-id", "--data", dataFile}, ctx, flags); err != nil {
			t.Fatalf("generate plain: %v", err)
		}
	})

	if !strings.Contains(out, "new-doc-id") {
		t.Errorf("expected 'new-doc-id' in output, got: %q", out)
	}
	if !strings.Contains(out, "https://docs.google.com/document/d/new-doc-id/edit") {
		t.Errorf("expected doc URL in output, got: %q", out)
	}
}

func TestDocsGenerate_EmptyTemplate(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)

	dataFile := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(dataFile, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	cmd := &DocsGenerateCmd{}
	err := runKong(t, cmd, []string{"--template", "", "--data", dataFile}, ctx, flags)
	if err == nil || !strings.Contains(err.Error(), "empty template") {
		t.Fatalf("expected empty template error, got: %v", err)
	}
}

func TestDocsGenerate_InvalidDataJSON(t *testing.T) {
	origDocs := newDocsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newDocsService = origDocs
		newDriveService = origDrive
	})

	// No need for real server; parsing should fail before API calls
	newDocsService = func(context.Context, string) (*docs.Service, error) {
		t.Fatal("should not call newDocsService for invalid JSON")
		return nil, nil
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) {
		t.Fatal("should not call newDriveService for invalid JSON")
		return nil, nil
	}

	dataFile := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(dataFile, []byte(`not json`), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)

	cmd := &DocsGenerateCmd{}
	err := runKong(t, cmd, []string{"--template", "tmpl-id", "--data", dataFile}, ctx, flags)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("expected invalid JSON error, got: %v", err)
	}
}

func TestDocsGenerate_MissingDataFile(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)

	cmd := &DocsGenerateCmd{}
	err := runKong(t, cmd, []string{"--template", "tmpl-id", "--data", "/nonexistent/path.json"}, ctx, flags)
	if err == nil {
		t.Fatalf("expected error for missing data file, got nil")
	}
}

func TestDocsGenerate_EmptyDataMap(t *testing.T) {
	origDocs := newDocsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newDocsService = origDocs
		newDriveService = origDrive
	})

	var batchUpdateCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/copy") && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "new-doc-id",
				"name":        "Generated Doc",
				"mimeType":    "application/vnd.google-apps.document",
				"webViewLink": "https://docs.google.com/document/d/new-doc-id/edit",
			})
			return
		}

		if strings.HasSuffix(r.URL.Path, ":batchUpdate") && r.Method == http.MethodPost {
			batchUpdateCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "new-doc-id",
				"replies":    []any{},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDriveService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	dataFile := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(dataFile, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	captureStdout(t, func() {
		cmd := &DocsGenerateCmd{}
		if err := runKong(t, cmd, []string{"--template", "tmpl-id", "--data", dataFile}, ctx, flags); err != nil {
			t.Fatalf("generate with empty data: %v", err)
		}
	})

	// With empty data map, batchUpdate should not be called since there are no replacements
	if batchUpdateCalled {
		t.Error("expected batchUpdate NOT to be called with empty data map")
	}
}
