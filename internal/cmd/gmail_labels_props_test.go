package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

func TestValidateLabelColor(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"#16a765", "#16a765", true},
		{"#16A765", "#16a765", true},
		{"ffffff", "#ffffff", true},
		{" #000000 ", "#000000", true},
		{"#123456", "", false},
		{"red", "", false},
	}
	for _, tc := range cases {
		got, err := validateLabelColor("--text-color", tc.in)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Fatalf("validateLabelColor(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("validateLabelColor(%q): expected error", tc.in)
		}
		if !strings.Contains(err.Error(), "#16a765") || !strings.Contains(err.Error(), "--text-color") {
			t.Fatalf("error should name the flag and list the palette: %v", err)
		}
	}
	if len(gmailLabelColorPalette) != 102 {
		t.Fatalf("palette size = %d, want 102", len(gmailLabelColorPalette))
	}
}

func TestBuildLabelRequest(t *testing.T) {
	l, err := buildLabelRequest(" alias/alpha ", GmailLabelPropsFlags{
		BackgroundColor:       "#16A765",
		TextColor:             "#ffffff",
		LabelListVisibility:   "labelshowifunread",
		MessageListVisibility: "HIDE",
	})
	if err != nil {
		t.Fatalf("buildLabelRequest: %v", err)
	}
	if l.Name != "alias/alpha" || l.Color == nil || l.Color.BackgroundColor != "#16a765" || l.Color.TextColor != "#ffffff" ||
		l.LabelListVisibility != "labelShowIfUnread" || l.MessageListVisibility != "hide" {
		t.Fatalf("unexpected label: %+v color=%+v", l, l.Color)
	}

	l, err = buildLabelRequest("x", GmailLabelPropsFlags{})
	if err != nil || l.Color != nil || l.LabelListVisibility != "" || l.MessageListVisibility != "" {
		t.Fatalf("empty flags should leave fields unset: %+v, %v", l, err)
	}

	bad := []GmailLabelPropsFlags{
		{BackgroundColor: "#16a765"},
		{TextColor: "#ffffff"},
		{BackgroundColor: "#16a765", TextColor: "#abcdef"},
		{LabelListVisibility: "visible"},
		{MessageListVisibility: "labelShow"},
	}
	for _, p := range bad {
		if _, err := buildLabelRequest("x", p); err == nil {
			t.Fatalf("expected error for %+v", p)
		}
	}
}

func TestGmailLabelsCreateCmd_WithColorAndVisibility(t *testing.T) {
	var got map[string]any
	srv := newLabelsServer(t, []map[string]any{}, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "Label_9", "name": got["name"]})
	})
	defer srv.Close()
	stubGmailService(t, srv)

	_ = captureStdout(t, func() {
		u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
		ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})
		args := []string{"alias/alpha", "--background-color", "#16a765", "--text-color", "#ffffff", "--message-list-visibility", "hide"}
		if err := runKong(t, &GmailLabelsCreateCmd{}, args, ctx, &RootFlags{Account: "a@b.com"}); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	color, _ := got["color"].(map[string]any)
	if color["backgroundColor"] != "#16a765" || color["textColor"] != "#ffffff" {
		t.Fatalf("unexpected color in request: %v", got)
	}
	if got["labelListVisibility"] != "labelShow" || got["messageListVisibility"] != "hide" {
		t.Fatalf("unexpected visibility in request: %v", got)
	}
}

func TestGmailLabelsModifyLabelCmd_PatchByName(t *testing.T) {
	var patched map[string]any
	var patchPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/me/labels") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"labels": []map[string]any{{"id": "Label_1", "name": "Custom", "type": "user"}}})
		case strings.HasSuffix(r.URL.Path, "/users/me/labels/Label_1") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "Label_1", "name": "Custom", "type": "user"})
		case strings.HasSuffix(r.URL.Path, "/users/me/labels/Label_1") && r.Method == http.MethodPatch:
			patchPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&patched)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "Label_1", "name": "Custom"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	stubGmailService(t, srv)

	out := captureStdout(t, func() {
		u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
		ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})
		args := []string{"custom", "--background-color", "#fb4c2f", "--text-color", "#000000"}
		if err := runKong(t, &GmailLabelsModifyLabelCmd{}, args, ctx, &RootFlags{Account: "a@b.com"}); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if patchPath == "" {
		t.Fatalf("expected PATCH request; out=%q", out)
	}
	if _, ok := patched["name"]; ok {
		t.Fatalf("name must not be sent when --name is not set: %v", patched)
	}
	color, _ := patched["color"].(map[string]any)
	if color["backgroundColor"] != "#fb4c2f" || color["textColor"] != "#000000" {
		t.Fatalf("unexpected patch body: %v", patched)
	}
	if _, ok := patched["labelListVisibility"]; ok {
		t.Fatalf("unset visibility must not be sent: %v", patched)
	}
}

func TestGmailLabelsModifyLabelCmd_NothingToUpdate(t *testing.T) {
	u, _ := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	ctx := ui.WithUI(context.Background(), u)
	err := runKong(t, &GmailLabelsModifyLabelCmd{}, []string{"Label_1"}, ctx, &RootFlags{Account: "a@b.com"})
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("expected usage error, got %v", err)
	}
}
