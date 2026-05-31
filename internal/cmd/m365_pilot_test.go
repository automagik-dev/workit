package cmd

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestM365ReadOnlyPilotCommandsAreExposed(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "outlook search",
			args: []string{"--json", "--read-only", "m365", "outlook", "search", "--query", "from:felipe"},
			want: "m365.outlook.search",
		},
		{
			name: "outlook message get",
			args: []string{"--json", "--read-only", "m365", "outlook", "message", "get", "AAMk-message-id"},
			want: "m365.outlook.message.get",
		},
		{
			name: "calendar events",
			args: []string{"--json", "--read-only", "m365", "calendar", "events", "--from", "2026-05-31T00:00:00Z", "--to", "2026-06-01T00:00:00Z"},
			want: "m365.calendar.events",
		},
		{
			name: "calendar freebusy",
			args: []string{"--json", "--read-only", "m365", "calendar", "freebusy", "--users", "bernardo@example.com,felipe@example.com"},
			want: "m365.calendar.freebusy",
		},
		{
			name: "calendar freebusy without users",
			args: []string{"--json", "--read-only", "m365", "calendar", "freebusy"},
			want: "m365.calendar.freebusy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				_ = captureStderr(t, func() {
					if err := Execute(tt.args); err != nil {
						t.Fatalf("Execute(%v): %v", tt.args, err)
					}
				})
			})

			var got map[string]any
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("json output: %v\n%s", err, out)
			}
			if got["operation"] != tt.want {
				t.Fatalf("operation = %v, want %s; output=%s", got["operation"], tt.want, out)
			}
			if got["provider"] != "microsoft_graph" {
				t.Fatalf("provider = %v, want microsoft_graph; output=%s", got["provider"], out)
			}
			if got["mode"] != "read_only_pilot" {
				t.Fatalf("mode = %v, want read_only_pilot; output=%s", got["mode"], out)
			}
			if tt.name == "calendar freebusy without users" {
				request, ok := got["request"].(map[string]any)
				if !ok {
					t.Fatalf("request has type %T, want object; output=%s", got["request"], out)
				}
				users, ok := request["users"].([]any)
				if !ok {
					t.Fatalf("request.users has type %T, want empty array; output=%s", request["users"], out)
				}
				if len(users) != 0 {
					t.Fatalf("request.users = %#v, want empty array", users)
				}
			}
		})
	}
}

func TestM365PilotCommandsRequireExplicitReadOnlyFlag(t *testing.T) {
	_ = captureStderr(t, func() {
		err := Execute([]string{"--json", "m365", "outlook", "search", "--query", "from:felipe"})
		if err == nil {
			t.Fatal("expected m365 pilot command without --read-only to fail closed")
		}
		if !strings.Contains(err.Error(), "--read-only") {
			t.Fatalf("expected --read-only error, got: %v", err)
		}
	})
}

func TestAuthServicesJSONIncludesM365PilotReadOnlyScopes(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "auth", "services"}); err != nil {
				t.Fatalf("auth services: %v", err)
			}
		})
	})

	var payload struct {
		Services []struct {
			Service string   `json:"service"`
			Scopes  []string `json:"scopes"`
		} `json:"services"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json output: %v\n%s", err, out)
	}

	var scopes []string
	for _, service := range payload.Services {
		if service.Service == "m365" {
			scopes = service.Scopes
			break
		}
	}
	if len(scopes) == 0 {
		t.Fatalf("auth services missing m365 service: %s", out)
	}
	for _, scope := range []string{"User.Read", "Mail.Read", "Calendars.Read"} {
		if !slices.Contains(scopes, scope) {
			t.Fatalf("m365 auth services missing %s: %#v", scope, scopes)
		}
	}
	for _, forbidden := range []string{"Mail.Send", "Calendars.ReadWrite"} {
		if slices.Contains(scopes, forbidden) {
			t.Fatalf("m365 auth services exposed write scope %s: %#v", forbidden, scopes)
		}
	}
}
