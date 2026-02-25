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

	"github.com/namastexlabs/workit/internal/outfmt"
	"github.com/namastexlabs/workit/internal/ui"
)

// --- extractHeaderText unit tests ---

func TestExtractHeaderText_WithContent(t *testing.T) {
	header := docs.Header{
		HeaderId: "hdr1",
		Content: []*docs.StructuralElement{
			{
				Paragraph: &docs.Paragraph{
					Elements: []*docs.ParagraphElement{
						{TextRun: &docs.TextRun{Content: "Company Report\n"}},
					},
				},
			},
		},
	}
	text := extractHeaderText(&header)
	if text != "Company Report" {
		t.Errorf("expected 'Company Report', got %q", text)
	}
}

func TestExtractHeaderText_Empty(t *testing.T) {
	header := docs.Header{
		HeaderId: "hdr1",
		Content:  []*docs.StructuralElement{},
	}
	text := extractHeaderText(&header)
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestExtractHeaderText_Nil(t *testing.T) {
	text := extractHeaderText(nil)
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestExtractHeaderText_MultipleElements(t *testing.T) {
	header := docs.Header{
		HeaderId: "hdr1",
		Content: []*docs.StructuralElement{
			{
				Paragraph: &docs.Paragraph{
					Elements: []*docs.ParagraphElement{
						{TextRun: &docs.TextRun{Content: "Line 1\n"}},
					},
				},
			},
			{
				Paragraph: &docs.Paragraph{
					Elements: []*docs.ParagraphElement{
						{TextRun: &docs.TextRun{Content: "Line 2\n"}},
					},
				},
			},
		},
	}
	text := extractHeaderText(&header)
	if !strings.Contains(text, "Line 1") {
		t.Errorf("expected text to contain 'Line 1', got %q", text)
	}
	if !strings.Contains(text, "Line 2") {
		t.Errorf("expected text to contain 'Line 2', got %q", text)
	}
}

// --- DocsHeaderCmd GET tests (mock HTTP) ---

func TestDocsHeader_Get_JSON_WithHeader(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/documents/") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
				"documentStyle": map[string]any{
					"defaultHeaderId": "hdr1",
				},
				"headers": map[string]any{
					"hdr1": map[string]any{
						"headerId": "hdr1",
						"content": []any{
							map[string]any{
								"paragraph": map[string]any{
									"elements": []any{
										map[string]any{
											"textRun": map[string]any{
												"content": "Company Report\n",
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
		cmd := &DocsHeaderCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("header get: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	if result["headerId"] != "hdr1" {
		t.Errorf("expected headerId 'hdr1', got %v", result["headerId"])
	}
	if result["text"] != "Company Report" {
		t.Errorf("expected text 'Company Report', got %v", result["text"])
	}
}

func TestDocsHeader_Get_JSON_NoHeader(t *testing.T) {
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
		cmd := &DocsHeaderCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("header get (no header): %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	if result["text"] != "no header" {
		t.Errorf("expected text 'no header', got %v", result["text"])
	}
}

func TestDocsHeader_Get_Text_WithHeader(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/documents/") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
				"documentStyle": map[string]any{
					"defaultHeaderId": "hdr1",
				},
				"headers": map[string]any{
					"hdr1": map[string]any{
						"headerId": "hdr1",
						"content": []any{
							map[string]any{
								"paragraph": map[string]any{
									"elements": []any{
										map[string]any{
											"textRun": map[string]any{
												"content": "Company Report\n",
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
		cmd := &DocsHeaderCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("header get text: %v", err)
		}
	})

	if !strings.Contains(out, "Company Report") {
		t.Errorf("expected 'Company Report' in output, got: %q", out)
	}
}

// --- DocsHeaderCmd SET test ---

func TestDocsHeader_Set_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	var batchUpdateCalled bool
	var batchBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// GET document
		if strings.HasPrefix(path, "/v1/documents/doc1") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
			})
			return
		}
		// batchUpdate
		if strings.HasSuffix(path, ":batchUpdate") && r.Method == http.MethodPost {
			batchUpdateCalled = true
			_ = json.NewDecoder(r.Body).Decode(&batchBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"replies": []any{
					map[string]any{
						"createHeader": map[string]any{
							"headerId": "hdr_new",
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
		cmd := &DocsHeaderCmd{}
		if err := runKong(t, cmd, []string{"doc1", "--set", "Company Report"}, ctx, flags); err != nil {
			t.Fatalf("header set: %v", err)
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

func TestDocsHeader_EmptyDocID(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)

	cmd := &DocsHeaderCmd{}
	err := runKong(t, cmd, []string{""}, ctx, flags)
	if err == nil || !strings.Contains(err.Error(), "empty docId") {
		t.Fatalf("expected empty docId error, got: %v", err)
	}
}

// --- DocsHeaderCmd SET with existing header test ---

func TestDocsHeader_Set_ExistingHeader_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	var batchUpdateCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// GET document -- has existing header
		if strings.HasPrefix(path, "/v1/documents/doc1") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "Test Doc",
				"documentStyle": map[string]any{
					"defaultHeaderId": "hdr_existing",
				},
				"headers": map[string]any{
					"hdr_existing": map[string]any{
						"headerId": "hdr_existing",
						"content": []any{
							map[string]any{
								"startIndex": 0,
								"endIndex":   11,
								"paragraph": map[string]any{
									"elements": []any{
										map[string]any{
											"textRun": map[string]any{
												"content": "Old Header\n",
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
		// batchUpdate
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
		cmd := &DocsHeaderCmd{}
		if err := runKong(t, cmd, []string{"doc1", "--set", "New Header"}, ctx, flags); err != nil {
			t.Fatalf("header set existing: %v", err)
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
