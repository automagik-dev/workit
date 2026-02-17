package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/steipete/gogcli/internal/outfmt"
)

func TestHelpTopics_ListTopics_NoArg(t *testing.T) {
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})

	out := captureStdout(t, func() {
		cmd := &AgentHelpCmd{}
		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	// Should list all topics
	for _, topic := range []string{"auth", "output", "agent", "pagination", "errors"} {
		if !strings.Contains(out, topic) {
			t.Errorf("expected topic %q in output, got:\n%s", topic, out)
		}
	}
}

func TestHelpTopics_ListTopics_TopicsKeyword(t *testing.T) {
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})

	out := captureStdout(t, func() {
		cmd := &AgentHelpCmd{Topic: "topics"}
		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	// "topics" keyword should behave same as no arg
	for _, topic := range []string{"auth", "output", "agent", "pagination", "errors"} {
		if !strings.Contains(out, topic) {
			t.Errorf("expected topic %q in output, got:\n%s", topic, out)
		}
	}
}

func TestHelpTopics_GetSpecificTopic(t *testing.T) {
	topics := []string{"auth", "output", "agent", "pagination", "errors"}

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})

			out := captureStdout(t, func() {
				cmd := &AgentHelpCmd{Topic: topic}
				if err := cmd.Run(ctx); err != nil {
					t.Fatalf("Run(%s): %v", topic, err)
				}
			})

			if out == "" {
				t.Fatalf("expected non-empty output for topic %q", topic)
			}
			// Each topic should have meaningful content (at least 50 chars)
			if len(out) < 50 {
				t.Errorf("topic %q content too short (%d chars): %q", topic, len(out), out)
			}
		})
	}
}

func TestHelpTopics_UnknownTopic(t *testing.T) {
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})

	cmd := &AgentHelpCmd{Topic: "nonexistent"}
	err := cmd.Run(ctx)
	if err == nil {
		t.Fatal("expected error for unknown topic, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the unknown topic name, got: %v", err)
	}
	// Should suggest available topics
	if !strings.Contains(err.Error(), "auth") {
		t.Errorf("error should list available topics, got: %v", err)
	}
}

func TestHelpTopics_UnknownTopic_ExitCode2(t *testing.T) {
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})

	cmd := &AgentHelpCmd{Topic: "nonexistent"}
	err := cmd.Run(ctx)
	if err == nil {
		t.Fatal("expected error for unknown topic, got nil")
	}
	if code := ExitCode(err); code != 2 {
		t.Fatalf("expected exit code 2 for unknown help topic, got %d", code)
	}
}

func TestHelpTopics_FuzzyMatch(t *testing.T) {
	tests := []struct {
		input       string
		wantSuggest string
	}{
		{"auht", "auth"},            // typo: transposed letters
		{"autth", "auth"},           // typo: extra letter
		{"aut", "auth"},             // truncated
		{"agen", "agent"},           // truncated
		{"outpu", "output"},         // truncated
		{"erors", "errors"},         // missing letter
		{"pagnation", "pagination"}, // missing letter
		{"agent-auth", "auth"},      // common prefix stripping
		{"agent-output", "output"},  // common prefix stripping
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})
			cmd := &AgentHelpCmd{Topic: tc.input}
			err := cmd.Run(ctx)
			if err == nil {
				t.Fatal("expected error for unknown topic, got nil")
			}
			if !strings.Contains(err.Error(), "Did you mean") {
				t.Errorf("expected 'Did you mean' suggestion for %q, got: %v", tc.input, err)
			}
			if !strings.Contains(err.Error(), tc.wantSuggest) {
				t.Errorf("expected suggestion %q for input %q, got: %v", tc.wantSuggest, tc.input, err)
			}
		})
	}
}

func TestHelpTopics_FuzzyMatch_NoSuggestionForDistantInput(t *testing.T) {
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})
	cmd := &AgentHelpCmd{Topic: "zzzzzzz"}
	err := cmd.Run(ctx)
	if err == nil {
		t.Fatal("expected error for unknown topic, got nil")
	}
	if strings.Contains(err.Error(), "Did you mean") {
		t.Errorf("should NOT suggest for very distant input, got: %v", err)
	}
	// Should still list available topics
	if !strings.Contains(err.Error(), "Available topics:") {
		t.Errorf("should list available topics, got: %v", err)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "b", 1},
		{"auth", "auth", 0},
		{"auth", "auht", 2},  // swap
		{"auth", "aut", 1},   // deletion
		{"auth", "autth", 1}, // insertion
		{"kitten", "sitting", 3},
	}
	for _, tc := range tests {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			got := levenshtein(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestHelpTopics_JSON_TopicList(t *testing.T) {
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{})

	out := captureStdout(t, func() {
		cmd := &AgentHelpCmd{}
		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	var doc struct {
		Topics []struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			Summary string `json:"summary"`
		} `json:"topics"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("unmarshal: %v (out=%q)", err, out)
	}
	if len(doc.Topics) < 5 {
		t.Fatalf("expected at least 5 topics, got %d", len(doc.Topics))
	}

	// Check that required topics are present
	topicNames := make(map[string]bool)
	for _, tp := range doc.Topics {
		topicNames[tp.Name] = true
		if tp.Title == "" {
			t.Errorf("topic %q has empty title", tp.Name)
		}
		if tp.Summary == "" {
			t.Errorf("topic %q has empty summary", tp.Name)
		}
	}
	for _, name := range []string{"auth", "output", "agent", "pagination", "errors"} {
		if !topicNames[name] {
			t.Errorf("missing topic %q in JSON output", name)
		}
	}
}

func TestHelpTopics_JSON_SpecificTopic(t *testing.T) {
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{})

	out := captureStdout(t, func() {
		cmd := &AgentHelpCmd{Topic: "auth"}
		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	var doc struct {
		Topic   string `json:"topic"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("unmarshal: %v (out=%q)", err, out)
	}
	if doc.Topic != "auth" {
		t.Errorf("expected topic=auth, got %q", doc.Topic)
	}
	if doc.Title == "" {
		t.Error("expected non-empty title")
	}
	if doc.Content == "" {
		t.Error("expected non-empty content")
	}
}
