package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/namastexlabs/workit/internal/outfmt"
	"github.com/namastexlabs/workit/internal/ui"
)

func TestSheetsBatchUpdateCmd_Stdin(t *testing.T) {
	origNew := newSheetsService
	t.Cleanup(func() { newSheetsService = origNew })

	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/sheets/v4")
		path = strings.TrimPrefix(path, "/v4")

		if strings.Contains(path, "/spreadsheets/s1:batchUpdate") && r.Method == http.MethodPost {
			defer r.Body.Close()
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &capturedBody)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spreadsheetId": "s1",
				"replies": []map[string]any{
					{
						"addSheet": map[string]any{
							"properties": map[string]any{
								"sheetId": 42,
								"title":   "NewSheet",
							},
						},
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "a@b.com"}
	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	stdinPayload := `{"requests":[{"addSheet":{"properties":{"title":"NewSheet"}}}]}`

	var jsonOut string
	withStdin(t, stdinPayload, func() {
		jsonOut = captureStdout(t, func() {
			cmd := &SheetsBatchUpdateCmd{}
			if err := runKong(t, cmd, []string{"s1"}, ctx, flags); err != nil {
				t.Fatalf("batch-update stdin: %v", err)
			}
		})
	})

	// Verify JSON output contains spreadsheetId and replies
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("json parse: %v\nraw output: %q", err, jsonOut)
	}
	if parsed["spreadsheetId"] != "s1" {
		t.Fatalf("expected spreadsheetId=s1, got: %v", parsed["spreadsheetId"])
	}
	replies, ok := parsed["replies"].([]any)
	if !ok || len(replies) == 0 {
		t.Fatalf("expected replies array, got: %v", parsed["replies"])
	}

	// Verify the request body was forwarded correctly
	if capturedBody == nil {
		t.Fatal("no request body captured")
	}
	reqs, ok := capturedBody["requests"].([]any)
	if !ok || len(reqs) == 0 {
		t.Fatalf("expected requests array in body, got: %v", capturedBody)
	}
}

func TestSheetsBatchUpdateCmd_File(t *testing.T) {
	origNew := newSheetsService
	t.Cleanup(func() { newSheetsService = origNew })

	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/sheets/v4")
		path = strings.TrimPrefix(path, "/v4")

		if strings.Contains(path, "/spreadsheets/s2:batchUpdate") && r.Method == http.MethodPost {
			defer r.Body.Close()
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &capturedBody)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spreadsheetId": "s2",
				"replies": []map[string]any{
					{
						"updateCells": map[string]any{},
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	// Write a temp file with the request payload
	tmpFile, err := os.CreateTemp(t.TempDir(), "batch-update-*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	payload := `{"requests":[{"updateCells":{}}]}`
	if _, err := tmpFile.WriteString(payload); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	tmpFile.Close()

	flags := &RootFlags{Account: "a@b.com"}
	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	jsonOut := captureStdout(t, func() {
		cmd := &SheetsBatchUpdateCmd{}
		if err := runKong(t, cmd, []string{"s2", "--file", tmpFile.Name()}, ctx, flags); err != nil {
			t.Fatalf("batch-update file: %v", err)
		}
	})

	// Verify JSON output
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("json parse: %v\nraw output: %q", err, jsonOut)
	}
	if parsed["spreadsheetId"] != "s2" {
		t.Fatalf("expected spreadsheetId=s2, got: %v", parsed["spreadsheetId"])
	}

	// Verify the request body was forwarded correctly
	if capturedBody == nil {
		t.Fatal("no request body captured")
	}
	reqs, ok := capturedBody["requests"].([]any)
	if !ok || len(reqs) == 0 {
		t.Fatalf("expected requests array in body, got: %v", capturedBody)
	}
}

func TestSheetsBatchUpdateCmd_Text(t *testing.T) {
	origNew := newSheetsService
	t.Cleanup(func() { newSheetsService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/sheets/v4")
		path = strings.TrimPrefix(path, "/v4")

		if strings.Contains(path, "/spreadsheets/s1:batchUpdate") && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spreadsheetId": "s1",
				"replies": []map[string]any{
					{
						"addSheet": map[string]any{
							"properties": map[string]any{
								"sheetId": 42,
								"title":   "NewSheet",
							},
						},
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "a@b.com"}
	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{}) // text mode

	stdinPayload := `{"requests":[{"addSheet":{"properties":{"title":"NewSheet"}}}]}`

	withStdin(t, stdinPayload, func() {
		// In text mode, output goes to UI (stderr), stdout should have plain summary
		cmd := &SheetsBatchUpdateCmd{}
		if err := runKong(t, cmd, []string{"s1"}, ctx, flags); err != nil {
			t.Fatalf("batch-update text: %v", err)
		}
	})
}

func TestSheetsBatchUpdateCmd_NoFileTTYStdin(t *testing.T) {
	origNew := newSheetsService
	t.Cleanup(func() { newSheetsService = origNew })
	errUnexpectedCall := errors.New("unexpected sheets service call")

	newSheetsService = func(context.Context, string) (*sheets.Service, error) {
		t.Fatal("should not call Sheets API when no input is provided")
		return nil, errUnexpectedCall
	}

	oldStdin := os.Stdin
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open os.DevNull: %v", err)
	}
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = devNull.Close()
	})
	os.Stdin = devNull

	stat, err := os.Stdin.Stat()
	if err != nil {
		t.Fatalf("stdin stat: %v", err)
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		t.Skip("os.DevNull is not a character device on this platform")
	}

	flags := &RootFlags{Account: "a@b.com"}
	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)

	cmd := &SheetsBatchUpdateCmd{}
	runErr := runKong(t, cmd, []string{"s1"}, ctx, flags)
	if runErr == nil || !strings.Contains(runErr.Error(), "no input provided") {
		t.Fatalf("expected no-input error, got: %v", runErr)
	}
}
