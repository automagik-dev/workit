package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/namastexlabs/gog-cli/internal/outfmt"
	"github.com/namastexlabs/gog-cli/internal/ui"
)

func TestSheetsAddTabCmd_JSON(t *testing.T) {
	origNew := newSheetsService
	t.Cleanup(func() { newSheetsService = origNew })

	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/sheets/v4")
		path = strings.TrimPrefix(path, "/v4")

		// batchUpdate endpoint: POST /spreadsheets/{id}:batchUpdate
		if strings.Contains(path, "/spreadsheets/s1:batchUpdate") && r.Method == http.MethodPost {
			// Capture request body for verification
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
								"sheetId":   42,
								"title":     "NewTab",
								"index":     1,
								"sheetType": "GRID",
								"gridProperties": map[string]any{
									"rowCount":    1000,
									"columnCount": 26,
								},
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

	jsonOut := captureStdout(t, func() {
		cmd := &SheetsAddTabCmd{}
		if err := runKong(t, cmd, []string{"s1", "NewTab"}, ctx, flags); err != nil {
			t.Fatalf("add-tab: %v", err)
		}
	})

	// Verify JSON output contains sheet ID and properties
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("json parse: %v\nraw output: %q", err, jsonOut)
	}
	if parsed["sheetId"] == nil {
		t.Fatalf("expected sheetId in output, got: %v", parsed)
	}
	if parsed["title"] != "NewTab" {
		t.Fatalf("expected title=NewTab, got: %v", parsed["title"])
	}

	// Verify request body contained AddSheet request
	if capturedBody == nil {
		t.Fatal("no request body captured")
	}
	reqs, ok := capturedBody["requests"].([]any)
	if !ok || len(reqs) == 0 {
		t.Fatalf("expected requests array, got: %v", capturedBody)
	}
	firstReq, ok := reqs[0].(map[string]any)
	if !ok {
		t.Fatalf("expected request object, got: %T", reqs[0])
	}
	addSheet, ok := firstReq["addSheet"]
	if !ok {
		t.Fatalf("expected addSheet in request, got: %v", firstReq)
	}
	addSheetMap, ok := addSheet.(map[string]any)
	if !ok {
		t.Fatalf("expected addSheet object, got: %T", addSheet)
	}
	props, ok := addSheetMap["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties in addSheet, got: %v", addSheetMap)
	}
	if props["title"] != "NewTab" {
		t.Fatalf("expected title=NewTab in request, got: %v", props["title"])
	}
}

func TestSheetsAddTabCmd_WithIndex(t *testing.T) {
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
								"sheetId":   99,
								"title":     "AtIndex",
								"index":     2,
								"sheetType": "GRID",
								"gridProperties": map[string]any{
									"rowCount":    1000,
									"columnCount": 26,
								},
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

	jsonOut := captureStdout(t, func() {
		cmd := &SheetsAddTabCmd{}
		if err := runKong(t, cmd, []string{"s1", "AtIndex", "--index", "2"}, ctx, flags); err != nil {
			t.Fatalf("add-tab with index: %v", err)
		}
	})

	// Verify JSON output
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("json parse: %v\nraw output: %q", err, jsonOut)
	}
	if parsed["title"] != "AtIndex" {
		t.Fatalf("expected title=AtIndex, got: %v", parsed["title"])
	}

	// Verify index was set in request
	if capturedBody == nil {
		t.Fatal("no request body captured")
	}
	reqs := capturedBody["requests"].([]any)
	firstReq := reqs[0].(map[string]any)
	addSheet := firstReq["addSheet"].(map[string]any)
	props := addSheet["properties"].(map[string]any)
	if props["index"] == nil {
		t.Fatal("expected index in addSheet properties")
	}
	idx, ok := props["index"].(float64)
	if !ok || int(idx) != 2 {
		t.Fatalf("expected index=2 in request, got: %v", props["index"])
	}
}

func TestSheetsAddTabCmd_Text(t *testing.T) {
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
								"sheetId":   42,
								"title":     "NewTab",
								"index":     1,
								"sheetType": "GRID",
								"gridProperties": map[string]any{
									"rowCount":    1000,
									"columnCount": 26,
								},
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
	ctx = outfmt.WithMode(ctx, outfmt.Mode{})

	out := captureStdout(t, func() {
		u2, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
		if uiErr != nil {
			t.Fatalf("ui.New: %v", uiErr)
		}
		ctx2 := ui.WithUI(ctx, u2)
		ctx2 = outfmt.WithMode(ctx2, outfmt.Mode{})

		cmd := &SheetsAddTabCmd{}
		if err := runKong(t, cmd, []string{"s1", "NewTab"}, ctx2, flags); err != nil {
			t.Fatalf("add-tab text: %v", err)
		}
	})

	// In text mode the output should go to the UI (stderr), not stdout
	// so stdout may be empty, which is fine.
	_ = out
}
