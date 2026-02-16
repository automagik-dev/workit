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
