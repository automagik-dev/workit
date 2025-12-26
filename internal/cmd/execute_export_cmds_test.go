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

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestExecute_SheetsExport_DefaultXLSX(t *testing.T) {
	origNew := newDriveService
	origExport := driveExportDownload
	t.Cleanup(func() {
		newDriveService = origNew
		driveExportDownload = origExport
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "sheet1",
			"name":     "My Sheet",
			"mimeType": "application/vnd.google-apps.spreadsheet",
		})
	}))
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

	var gotMime string
	driveExportDownload = func(_ context.Context, _ *drive.Service, fileID string, mimeType string) (*http.Response, error) {
		gotMime = mimeType
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("XLSX-FAKE")),
		}, nil
	}

	dest := filepath.Join(t.TempDir(), "out")
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "--account", "a@b.com", "sheets", "export", "sheet1", "--out", dest}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	if gotMime != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("mimeType=%q", gotMime)
	}

	var parsed struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if !strings.HasSuffix(parsed.Path, ".xlsx") {
		t.Fatalf("expected .xlsx path, got %q", parsed.Path)
	}
	if b, err := os.ReadFile(parsed.Path); err != nil || string(b) != "XLSX-FAKE" {
		t.Fatalf("file mismatch: err=%v body=%q", err, string(b))
	}
}

func TestExecute_DocsExport_DOCX(t *testing.T) {
	origNew := newDriveService
	origExport := driveExportDownload
	t.Cleanup(func() {
		newDriveService = origNew
		driveExportDownload = origExport
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "doc1",
			"name":     "My Doc",
			"mimeType": "application/vnd.google-apps.document",
		})
	}))
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

	var gotMime string
	driveExportDownload = func(_ context.Context, _ *drive.Service, fileID string, mimeType string) (*http.Response, error) {
		gotMime = mimeType
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("DOCX-FAKE")),
		}, nil
	}

	dest := filepath.Join(t.TempDir(), "out")
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "--account", "a@b.com", "docs", "export", "doc1", "--format", "docx", "--out", dest}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	if gotMime != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Fatalf("mimeType=%q", gotMime)
	}

	var parsed struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if !strings.HasSuffix(parsed.Path, ".docx") {
		t.Fatalf("expected .docx path, got %q", parsed.Path)
	}
}

func TestExecute_SlidesExport_DefaultPPTX(t *testing.T) {
	origNew := newDriveService
	origExport := driveExportDownload
	t.Cleanup(func() {
		newDriveService = origNew
		driveExportDownload = origExport
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "slides1",
			"name":     "My Slides",
			"mimeType": "application/vnd.google-apps.presentation",
		})
	}))
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

	var gotMime string
	driveExportDownload = func(_ context.Context, _ *drive.Service, fileID string, mimeType string) (*http.Response, error) {
		gotMime = mimeType
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("PPTX-FAKE")),
		}, nil
	}

	dest := filepath.Join(t.TempDir(), "out")
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "--account", "a@b.com", "slides", "export", "slides1", "--out", dest}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	if gotMime != "application/vnd.openxmlformats-officedocument.presentationml.presentation" {
		t.Fatalf("mimeType=%q", gotMime)
	}

	var parsed struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if !strings.HasSuffix(parsed.Path, ".pptx") {
		t.Fatalf("expected .pptx path, got %q", parsed.Path)
	}
}

func TestExecute_DocsExport_RejectsNonDoc(t *testing.T) {
	origNew := newDriveService
	origExport := driveExportDownload
	t.Cleanup(func() {
		newDriveService = origNew
		driveExportDownload = origExport
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "x",
			"name":     "Not a Doc",
			"mimeType": "application/vnd.google-apps.spreadsheet",
		})
	}))
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

	called := false
	driveExportDownload = func(context.Context, *drive.Service, string, string) (*http.Response, error) {
		called = true
		return nil, nil
	}

	err = Execute([]string{"--json", "--account", "a@b.com", "docs", "export", "x", "--out", filepath.Join(t.TempDir(), "out")})
	if err == nil || !strings.Contains(err.Error(), "not a Google Doc") {
		t.Fatalf("unexpected err=%v", err)
	}
	if called {
		t.Fatalf("export should not be called")
	}
}
