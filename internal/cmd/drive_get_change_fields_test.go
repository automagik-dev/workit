package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// driveGetChangeFieldsServer serves one file for GET /files/<id> and records
// the `fields` query parameter of each request.
func driveGetChangeFieldsServer(t *testing.T, payload map[string]any) (*sync.Mutex, *[]string) {
	t.Helper()

	var mu sync.Mutex
	var fields []string

	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	svc, closeSrv := newDriveTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.Contains(r.URL.Path, "/files/id1") {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		fields = append(fields, r.URL.Query().Get("fields"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(closeSrv)
	newDriveService = stubDriveService(svc)

	return &mu, &fields
}

func assertDriveGetRequestedChangeFields(t *testing.T, mu *sync.Mutex, fields *[]string) {
	t.Helper()

	mu.Lock()
	defer mu.Unlock()
	if len(*fields) != 1 {
		t.Fatalf("expected exactly one files.get request, got %d", len(*fields))
	}
	requested := map[string]bool{}
	for _, f := range strings.Split((*fields)[0], ",") {
		requested[strings.TrimSpace(f)] = true
	}
	for _, want := range []string{"md5Checksum", "version", "headRevisionId", "modifiedTime", "mimeType"} {
		if !requested[want] {
			t.Errorf("files.get fields %q does not request %q", (*fields)[0], want)
		}
	}
}

func TestExecute_DriveGet_ChangeFields_JSON(t *testing.T) {
	tests := []struct {
		name         string
		payload      map[string]any
		wantVersion  string
		wantMD5      string
		wantRevision string
	}{
		{
			name: "binary file",
			payload: map[string]any{
				"id": "id1", "name": "report.pdf", "mimeType": "application/pdf",
				"modifiedTime": "2025-12-12T14:37:47Z",
				// The Drive API sends version as a string-encoded int64.
				"version":        "42",
				"md5Checksum":    "0cc175b9c0f1b6a831c399e269772661",
				"headRevisionId": "rev-7",
			},
			wantVersion:  "42",
			wantMD5:      "0cc175b9c0f1b6a831c399e269772661",
			wantRevision: "rev-7",
		},
		{
			name: "google-native sheet has version only",
			payload: map[string]any{
				"id": "id1", "name": "Budget", "mimeType": "application/vnd.google-apps.spreadsheet",
				"modifiedTime": "2025-12-12T14:37:47Z",
				"version":      "1234",
			},
			wantVersion: "1234",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mu, fields := driveGetChangeFieldsServer(t, tc.payload)

			out := captureStdout(t, func() {
				_ = captureStderr(t, func() {
					if err := Execute([]string{"--json", "--account", "a@b.com", "drive", "get", "id1"}); err != nil {
						t.Fatalf("Execute: %v", err)
					}
				})
			})

			assertDriveGetRequestedChangeFields(t, mu, fields)

			var parsed struct {
				File map[string]any `json:"file"`
			}
			if err := json.Unmarshal([]byte(out), &parsed); err != nil {
				t.Fatalf("json parse: %v\nout=%q", err, out)
			}
			if got, _ := parsed.File["version"].(string); got != tc.wantVersion {
				t.Errorf("version=%#v, want %q (out=%s)", parsed.File["version"], tc.wantVersion, out)
			}
			checkOptional := func(key, want string) {
				got, present := parsed.File[key]
				if want == "" {
					if present {
						t.Errorf("%s should be omitted when Drive does not return it, got %#v", key, got)
					}
					return
				}
				if s, _ := got.(string); s != want {
					t.Errorf("%s=%#v, want %q", key, got, want)
				}
			}
			checkOptional("md5Checksum", tc.wantMD5)
			checkOptional("headRevisionId", tc.wantRevision)
		})
	}
}

func TestExecute_DriveGet_ChangeFields_Text(t *testing.T) {
	mu, fields := driveGetChangeFieldsServer(t, map[string]any{
		"id": "id1", "name": "report.pdf", "mimeType": "application/pdf",
		"size": "1024", "modifiedTime": "2025-12-12T14:37:47Z",
		"version": "42", "md5Checksum": "0cc175b9c0f1b6a831c399e269772661", "headRevisionId": "rev-7",
	})

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--account", "a@b.com", "drive", "get", "id1"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	assertDriveGetRequestedChangeFields(t, mu, fields)
	for _, want := range []string{"version\t42", "md5\t0cc175b9c0f1b6a831c399e269772661", "head_revision\trev-7"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}
