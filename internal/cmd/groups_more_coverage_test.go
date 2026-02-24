package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/ui"
)

func TestCollectGroupMemberEmails_RecursiveAndPaging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "groups:lookup"):
			id := r.URL.Query().Get("groupKey.id")
			switch id {
			case "group-a@example.com":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"name": "groups/ga"})
				return
			case "group-b@example.com":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"name": "groups/gb"})
				return
			default:
				http.NotFound(w, r)
				return
			}
		case strings.Contains(r.URL.Path, "groups/ga/memberships"):
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Query().Get("pageToken") {
			case "":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"memberships": []any{
						nil,
						map[string]any{"preferredMemberKey": map[string]any{"id": "group-b@example.com"}, "type": "GROUP"},
						map[string]any{"preferredMemberKey": map[string]any{"id": "user1@example.com"}, "type": "USER"},
						map[string]any{"preferredMemberKey": map[string]any{"id": "notanemail"}, "type": "USER"},
						map[string]any{"preferredMemberKey": map[string]any{"id": ""}, "type": "USER"},
					},
					"nextPageToken": "next",
				})
				return
			case "next":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"memberships": []any{
						map[string]any{"preferredMemberKey": map[string]any{"id": "user2@example.com"}, "type": ""},
					},
				})
				return
			default:
				http.NotFound(w, r)
				return
			}
		case strings.Contains(r.URL.Path, "groups/gb/memberships"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"memberships": []any{
					map[string]any{"preferredMemberKey": map[string]any{"id": "group-a@example.com"}, "type": "GROUP"},
					map[string]any{"preferredMemberKey": map[string]any{"id": "user3@example.com"}, "type": "USER"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := cloudidentity.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	emails, err := collectGroupMemberEmails(context.Background(), svc, "group-a@example.com")
	if err != nil {
		t.Fatalf("collectGroupMemberEmails: %v", err)
	}
	want := []string{"user1@example.com", "user2@example.com", "user3@example.com"}
	if strings.Join(emails, ",") != strings.Join(want, ",") {
		t.Fatalf("emails=%v want %v", emails, want)
	}
}

func TestGroupsListCmd_QueryIncludesLabelFilter(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "searchTransitiveGroups") {
			capturedQuery = r.URL.Query().Get("query")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"memberships": []any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	origFactory := newCloudIdentityService
	newCloudIdentityService = func(ctx context.Context, account string) (*cloudidentity.Service, error) {
		return cloudidentity.NewService(ctx,
			option.WithoutAuthentication(),
			option.WithHTTPClient(srv.Client()),
			option.WithEndpoint(srv.URL+"/"),
		)
	}
	defer func() { newCloudIdentityService = origFactory }()

	cmd := &GroupsListCmd{Max: 10}
	u, err := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx := ui.WithUI(context.Background(), u)
	flags := &RootFlags{Account: "user@company.com"}
	_ = cmd.Run(ctx, flags)

	if !strings.Contains(capturedQuery, "member_key_id == 'user@company.com'") {
		t.Fatalf("query missing member_key_id: %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "cloudidentity.googleapis.com/groups.discussion_forum") {
		t.Fatalf("query missing label filter: %q", capturedQuery)
	}
}

func TestCollectGroupMemberEmails_LookupError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := cloudidentity.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if _, err := collectGroupMemberEmails(context.Background(), svc, "missing@example.com"); err == nil {
		t.Fatalf("expected error")
	}
}
