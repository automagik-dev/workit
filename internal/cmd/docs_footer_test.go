package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// --- extractFooterText unit tests ---

func TestExtractFooterText_WithContent(t *testing.T) {
	footer := docs.Footer{
		FooterId: "ftr1",
		Content: []*docs.StructuralElement{
			{
				Paragraph: &docs.Paragraph{
					Elements: []*docs.ParagraphElement{
						{TextRun: &docs.TextRun{Content: "Page {PAGE}\n"}},
					},
				},
			},
		},
	}
	text := extractFooterText(&footer)
	if text != "Page {PAGE}" {
		t.Errorf("expected 'Page {PAGE}', got %q", text)
	}
}

func TestExtractFooterText_Empty(t *testing.T) {
	footer := docs.Footer{
		FooterId: "ftr1",
		Content:  []*docs.StructuralElement{},
	}
	text := extractFooterText(&footer)
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestExtractFooterText_Nil(t *testing.T) {
	text := extractFooterText(nil)
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

// --- DocsFooterCmd GET tests (mock HTTP) ---

func TestDocsFooter_Get_JSON_WithFooter(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/documents/") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
				"documentStyle": map[string]any{
					"defaultFooterId": "ftr1",
				},
				"footers": map[string]any{
					"ftr1": map[string]any{
						"footerId": "ftr1",
						"content": []any{
							map[string]any{
								"paragraph": map[string]any{
									"elements": []any{
										map[string]any{
											"textRun": map[string]any{
												"content": "Page {PAGE}\n",
											},
										},
									},
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

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	out := captureStdout(t, func() {
		cmd := &DocsFooterCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("footer get: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	if result["footerId"] != "ftr1" {
		t.Errorf("expected footerId 'ftr1', got %v", result["footerId"])
	}
	if result["text"] != "Page {PAGE}" {
		t.Errorf("expected text 'Page {PAGE}', got %v", result["text"])
	}
}

func TestDocsFooter_Get_JSON_NoFooter(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/documents/") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
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

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	out := captureStdout(t, func() {
		cmd := &DocsFooterCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("footer get (no footer): %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	if result["text"] != "no footer" {
		t.Errorf("expected text 'no footer', got %v", result["text"])
	}
}

func TestDocsFooter_Get_Text_WithFooter(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/documents/") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
				"documentStyle": map[string]any{
					"defaultFooterId": "ftr1",
				},
				"footers": map[string]any{
					"ftr1": map[string]any{
						"footerId": "ftr1",
						"content": []any{
							map[string]any{
								"paragraph": map[string]any{
									"elements": []any{
										map[string]any{
											"textRun": map[string]any{
												"content": "Page {PAGE}\n",
											},
										},
									},
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

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	flags := &RootFlags{Account: "a@b.com"}

	out := captureStdout(t, func() {
		u, _ := ui.New(ui.Options{Stdout: os.Stdout, Stderr: io.Discard, Color: "never"})
		ctx := ui.WithUI(context.Background(), u)
		cmd := &DocsFooterCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("footer get text: %v", err)
		}
	})

	if !strings.Contains(out, "Page {PAGE}") {
		t.Errorf("expected 'Page {PAGE}' in output, got: %q", out)
	}
}

// --- DocsFooterCmd SET test ---

func TestDocsFooter_Set_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	var batchUpdateCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/v1/documents/doc1") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
			})
			return
		}
		if strings.HasSuffix(path, ":batchUpdate") && r.Method == http.MethodPost {
			batchUpdateCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"replies": []any{
					map[string]any{
						"createFooter": map[string]any{
							"footerId": "ftr_new",
						},
					},
					map[string]any{},
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

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	out := captureStdout(t, func() {
		cmd := &DocsFooterCmd{}
		if err := runKong(t, cmd, []string{"doc1", "--set", "Page {PAGE}"}, ctx, flags); err != nil {
			t.Fatalf("footer set: %v", err)
		}
	})

	if !batchUpdateCalled {
		t.Fatal("expected batchUpdate to be called")
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	if result["success"] != true {
		t.Errorf("expected success true, got %v", result["success"])
	}
	if result["docId"] != "doc1" {
		t.Errorf("expected docId 'doc1', got %v", result["docId"])
	}
}

func TestDocsFooter_EmptyDocID(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)

	cmd := &DocsFooterCmd{}
	err := runKong(t, cmd, []string{""}, ctx, flags)
	if err == nil || !strings.Contains(err.Error(), "empty docId") {
		t.Fatalf("expected empty docId error, got: %v", err)
	}
}

// --- DocsFooterCmd SET with existing footer test ---

func TestDocsFooter_Set_ExistingFooter_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	var batchUpdateCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/v1/documents/doc1") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
				"documentStyle": map[string]any{
					"defaultFooterId": "ftr_existing",
				},
				"footers": map[string]any{
					"ftr_existing": map[string]any{
						"footerId": "ftr_existing",
						"content": []any{
							map[string]any{
								"startIndex": 0,
								"endIndex":   11,
								"paragraph": map[string]any{
									"elements": []any{
										map[string]any{
											"textRun": map[string]any{
												"content": "Old Footer\n",
											},
										},
									},
								},
							},
						},
					},
				},
			})
			return
		}
		if strings.HasSuffix(path, ":batchUpdate") && r.Method == http.MethodPost {
			batchUpdateCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"replies":    []any{map[string]any{}, map[string]any{}},
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

	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

	out := captureStdout(t, func() {
		cmd := &DocsFooterCmd{}
		if err := runKong(t, cmd, []string{"doc1", "--set", "New Footer"}, ctx, flags); err != nil {
			t.Fatalf("footer set existing: %v", err)
		}
	})

	if !batchUpdateCalled {
		t.Fatal("expected batchUpdate to be called")
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	if result["success"] != true {
		t.Errorf("expected success true, got %v", result["success"])
	}
}
