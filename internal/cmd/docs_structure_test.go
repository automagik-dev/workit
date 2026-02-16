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
	"unicode/utf8"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

func TestExtractDocStructure_Paragraphs(t *testing.T) {
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					StartIndex: 0,
					EndIndex:   1,
					SectionBreak: &docs.SectionBreak{
						SectionStyle: &docs.SectionStyle{},
					},
				},
				{
					StartIndex: 1,
					EndIndex:   14,
					Paragraph: &docs.Paragraph{
						ParagraphStyle: &docs.ParagraphStyle{
							NamedStyleType: "HEADING_1",
						},
						Elements: []*docs.ParagraphElement{
							{TextRun: &docs.TextRun{Content: "Introduction\n"}},
						},
					},
				},
				{
					StartIndex: 14,
					EndIndex:   40,
					Paragraph: &docs.Paragraph{
						ParagraphStyle: &docs.ParagraphStyle{
							NamedStyleType: "NORMAL_TEXT",
						},
						Elements: []*docs.ParagraphElement{
							{TextRun: &docs.TextRun{Content: "This is a normal paragraph\n"}},
						},
					},
				},
			},
		},
	}

	elements := extractDocStructure(doc)

	if len(elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elements))
	}

	// First element: section break
	if elements[0].Type != "sectionBreak" {
		t.Errorf("element 0: expected type sectionBreak, got %q", elements[0].Type)
	}
	if elements[0].StartIndex != 0 || elements[0].EndIndex != 1 {
		t.Errorf("element 0: unexpected indices %d-%d", elements[0].StartIndex, elements[0].EndIndex)
	}

	// Second element: heading paragraph
	if elements[1].Type != "paragraph" {
		t.Errorf("element 1: expected type paragraph, got %q", elements[1].Type)
	}
	if elements[1].Style != "HEADING_1" {
		t.Errorf("element 1: expected style HEADING_1, got %q", elements[1].Style)
	}
	if elements[1].StartIndex != 1 || elements[1].EndIndex != 14 {
		t.Errorf("element 1: unexpected indices %d-%d", elements[1].StartIndex, elements[1].EndIndex)
	}
	if !strings.Contains(elements[1].ContentSummary, "Introduction") {
		t.Errorf("element 1: expected content summary to contain 'Introduction', got %q", elements[1].ContentSummary)
	}

	// Third element: normal paragraph
	if elements[2].Type != "paragraph" {
		t.Errorf("element 2: expected type paragraph, got %q", elements[2].Type)
	}
	if elements[2].Style != "NORMAL_TEXT" {
		t.Errorf("element 2: expected style NORMAL_TEXT, got %q", elements[2].Style)
	}
}

func TestExtractDocStructure_Table(t *testing.T) {
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					StartIndex: 1,
					EndIndex:   50,
					Table: &docs.Table{
						Rows:    3,
						Columns: 4,
						TableRows: []*docs.TableRow{
							{TableCells: []*docs.TableCell{{}, {}, {}, {}}},
							{TableCells: []*docs.TableCell{{}, {}, {}, {}}},
							{TableCells: []*docs.TableCell{{}, {}, {}, {}}},
						},
					},
				},
			},
		},
	}

	elements := extractDocStructure(doc)

	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elements))
	}
	if elements[0].Type != "table" {
		t.Errorf("expected type table, got %q", elements[0].Type)
	}
	if elements[0].ContentSummary != "Table 3x4" {
		t.Errorf("expected 'Table 3x4', got %q", elements[0].ContentSummary)
	}
}

func TestExtractDocStructure_HeadingStyles(t *testing.T) {
	styles := []string{"TITLE", "SUBTITLE", "HEADING_1", "HEADING_2", "HEADING_3", "HEADING_4", "HEADING_5", "HEADING_6"}

	for _, style := range styles {
		doc := &docs.Document{
			Body: &docs.Body{
				Content: []*docs.StructuralElement{
					{
						StartIndex: 0,
						EndIndex:   10,
						Paragraph: &docs.Paragraph{
							ParagraphStyle: &docs.ParagraphStyle{
								NamedStyleType: style,
							},
							Elements: []*docs.ParagraphElement{
								{TextRun: &docs.TextRun{Content: "text\n"}},
							},
						},
					},
				},
			},
		}

		elements := extractDocStructure(doc)
		if len(elements) != 1 {
			t.Fatalf("style %s: expected 1 element, got %d", style, len(elements))
		}
		if elements[0].Style != style {
			t.Errorf("style %s: expected style %q, got %q", style, style, elements[0].Style)
		}
	}
}

func TestExtractDocStructure_ContentSummaryTruncation(t *testing.T) {
	longText := strings.Repeat("A", 200) + "\n"
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					StartIndex: 0,
					EndIndex:   201,
					Paragraph: &docs.Paragraph{
						ParagraphStyle: &docs.ParagraphStyle{
							NamedStyleType: "NORMAL_TEXT",
						},
						Elements: []*docs.ParagraphElement{
							{TextRun: &docs.TextRun{Content: longText}},
						},
					},
				},
			},
		},
	}

	elements := extractDocStructure(doc)
	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elements))
	}
	if len(elements[0].ContentSummary) > 83 {
		t.Errorf("content summary should be truncated to ~80 chars + '...', got %d chars: %q",
			len(elements[0].ContentSummary), elements[0].ContentSummary)
	}
	if !strings.HasSuffix(elements[0].ContentSummary, "...") {
		t.Errorf("truncated content summary should end with '...', got %q", elements[0].ContentSummary)
	}
}

func TestExtractDocStructure_ContentSummaryTruncation_UTF8Safe(t *testing.T) {
	longText := strings.Repeat("界", 100) + "\n"
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					StartIndex: 0,
					EndIndex:   201,
					Paragraph: &docs.Paragraph{
						ParagraphStyle: &docs.ParagraphStyle{
							NamedStyleType: "NORMAL_TEXT",
						},
						Elements: []*docs.ParagraphElement{
							{TextRun: &docs.TextRun{Content: longText}},
						},
					},
				},
			},
		},
	}

	elements := extractDocStructure(doc)
	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elements))
	}

	summary := elements[0].ContentSummary
	if !utf8.ValidString(summary) {
		t.Fatalf("summary must be valid UTF-8, got %q", summary)
	}
	if !strings.HasSuffix(summary, "...") {
		t.Fatalf("summary should end with ellipsis when truncated, got %q", summary)
	}

	trimmed := strings.TrimSuffix(summary, "...")
	if got := len([]rune(trimmed)); got != 80 {
		t.Fatalf("expected 80 runes before ellipsis, got %d (%q)", got, trimmed)
	}
}

func TestExtractDocStructure_NilAndEmpty(t *testing.T) {
	// nil document
	elements := extractDocStructure(nil)
	if len(elements) != 0 {
		t.Errorf("nil doc: expected 0 elements, got %d", len(elements))
	}

	// nil body
	elements = extractDocStructure(&docs.Document{})
	if len(elements) != 0 {
		t.Errorf("nil body: expected 0 elements, got %d", len(elements))
	}

	// empty content
	elements = extractDocStructure(&docs.Document{Body: &docs.Body{}})
	if len(elements) != 0 {
		t.Errorf("empty content: expected 0 elements, got %d", len(elements))
	}
}

func TestExtractDocStructure_TableOfContents(t *testing.T) {
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					StartIndex: 1,
					EndIndex:   20,
					TableOfContents: &docs.TableOfContents{
						Content: []*docs.StructuralElement{
							{
								Paragraph: &docs.Paragraph{
									Elements: []*docs.ParagraphElement{
										{TextRun: &docs.TextRun{Content: "Heading 1"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	elements := extractDocStructure(doc)
	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elements))
	}
	if elements[0].Type != "tableOfContents" {
		t.Errorf("expected type tableOfContents, got %q", elements[0].Type)
	}
}

func TestDocsStructure_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/v1/documents/") && r.Method == http.MethodGet {
			id := strings.TrimPrefix(path, "/v1/documents/")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": id,
				"title":      "Test Doc",
				"body": map[string]any{
					"content": []any{
						map[string]any{
							"startIndex": 0,
							"endIndex":   1,
							"sectionBreak": map[string]any{
								"sectionStyle": map[string]any{},
							},
						},
						map[string]any{
							"startIndex": 1,
							"endIndex":   14,
							"paragraph": map[string]any{
								"paragraphStyle": map[string]any{
									"namedStyleType": "HEADING_1",
								},
								"elements": []any{
									map[string]any{
										"textRun": map[string]any{
											"content": "Introduction\n",
										},
									},
								},
							},
						},
						map[string]any{
							"startIndex": 14,
							"endIndex":   40,
							"paragraph": map[string]any{
								"paragraphStyle": map[string]any{
									"namedStyleType": "NORMAL_TEXT",
								},
								"elements": []any{
									map[string]any{
										"textRun": map[string]any{
											"content": "Some body text here.\n",
										},
									},
								},
							},
						},
						map[string]any{
							"startIndex": 40,
							"endIndex":   80,
							"table": map[string]any{
								"rows":    2,
								"columns": 3,
								"tableRows": []any{
									map[string]any{
										"tableCells": []any{
											map[string]any{"content": []any{}},
											map[string]any{"content": []any{}},
											map[string]any{"content": []any{}},
										},
									},
									map[string]any{
										"tableCells": []any{
											map[string]any{"content": []any{}},
											map[string]any{"content": []any{}},
											map[string]any{"content": []any{}},
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
		cmd := &DocsStructureCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("structure: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %q", err, out)
	}

	elems, ok := result["elements"].([]any)
	if !ok {
		t.Fatalf("expected 'elements' array in JSON, got: %v", result)
	}

	if len(elems) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(elems))
	}

	// Check heading element
	heading := elems[1].(map[string]any)
	if heading["type"] != "paragraph" {
		t.Errorf("expected paragraph type, got %v", heading["type"])
	}
	if heading["style"] != "HEADING_1" {
		t.Errorf("expected HEADING_1 style, got %v", heading["style"])
	}
	if !strings.Contains(heading["contentSummary"].(string), "Introduction") {
		t.Errorf("expected content summary containing 'Introduction', got %v", heading["contentSummary"])
	}

	// Check table element
	table := elems[3].(map[string]any)
	if table["type"] != "table" {
		t.Errorf("expected table type, got %v", table["type"])
	}
	if table["contentSummary"] != "Table 2x3" {
		t.Errorf("expected 'Table 2x3', got %v", table["contentSummary"])
	}
}

func TestDocsStructure_Text(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/v1/documents/") && r.Method == http.MethodGet {
			id := strings.TrimPrefix(path, "/v1/documents/")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": id,
				"title":      "Test Doc",
				"body": map[string]any{
					"content": []any{
						map[string]any{
							"startIndex": 1,
							"endIndex":   14,
							"paragraph": map[string]any{
								"paragraphStyle": map[string]any{
									"namedStyleType": "HEADING_1",
								},
								"elements": []any{
									map[string]any{
										"textRun": map[string]any{
											"content": "Introduction\n",
										},
									},
								},
							},
						},
						map[string]any{
							"startIndex": 14,
							"endIndex":   40,
							"paragraph": map[string]any{
								"paragraphStyle": map[string]any{
									"namedStyleType": "NORMAL_TEXT",
								},
								"elements": []any{
									map[string]any{
										"textRun": map[string]any{
											"content": "Body text.\n",
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
		cmd := &DocsStructureCmd{}
		if err := runKong(t, cmd, []string{"doc1"}, ctx, flags); err != nil {
			t.Fatalf("structure: %v", err)
		}
	})

	// Verify plain text output contains expected content
	if !strings.Contains(out, "HEADING_1") {
		t.Errorf("expected HEADING_1 in text output, got: %q", out)
	}
	if !strings.Contains(out, "Introduction") {
		t.Errorf("expected 'Introduction' in text output, got: %q", out)
	}
	if !strings.Contains(out, "paragraph") {
		t.Errorf("expected 'paragraph' in text output, got: %q", out)
	}
}

func TestDocsStructure_EmptyDocID(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com"}
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)

	cmd := &DocsStructureCmd{}
	err := runKong(t, cmd, []string{""}, ctx, flags)
	if err == nil || !strings.Contains(err.Error(), "empty docId") {
		t.Fatalf("expected empty docId error, got: %v", err)
	}
}
