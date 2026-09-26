package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/automagik-dev/workit/internal/config"
	gogapi "github.com/automagik-dev/workit/internal/googleapi"
)

func parseJSONErrorLine(t *testing.T, stderr string) jsonErrorBody {
	t.Helper()

	trimmed := strings.TrimRight(stderr, "\n")
	if trimmed == "" {
		t.Fatalf("expected a JSON error line on stderr, got nothing")
	}
	if strings.Contains(trimmed, "\n") {
		t.Fatalf("expected exactly one stderr line, got %q", stderr)
	}

	var env struct {
		Error *jsonErrorBody `json:"error"`
	}
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		t.Fatalf("stderr is not JSON: %v\nstderr=%q", err, stderr)
	}
	if env.Error == nil {
		t.Fatalf("missing error object: %q", stderr)
	}
	return *env.Error
}

func TestErrorKindForExitCode_CoversAgentExitCodes(t *testing.T) {
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "exit-codes"}); err != nil {
			t.Fatalf("Execute exit-codes: %v", err)
		}
	})

	var parsed struct {
		ExitCodes map[string]int `json:"exit_codes"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse exit-codes: %v\nout=%q", err, out)
	}
	if len(parsed.ExitCodes) == 0 {
		t.Fatalf("no exit codes in %q", out)
	}

	for name, code := range parsed.ExitCodes {
		if name == "ok" {
			continue
		}
		// The JSON error kind is the exit-code name itself, so callers use one
		// vocabulary for both. A new stable exit code must be classified in
		// errorKindForExitCode, or this fails.
		if got := errorKindForExitCode(code); got != name {
			t.Errorf("exit code %q (%d): kind=%q, want the exit-code name %q", name, code, got, name)
		}
	}
}

func TestErrorKindForExitCode_UnknownFallsBackToError(t *testing.T) {
	for _, code := range []int{1, 9, 42, 255} {
		if got := errorKindForExitCode(code); got != errorKindError {
			t.Fatalf("code %d: kind=%q, want %q", code, got, errorKindError)
		}
	}
}

func TestJSONErrorFor_StableErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantExit int
		wantKind string
	}{
		{"generic", errors.New("boom"), 1, errorKindError},
		{"usage", usage("bad flag"), 2, errorKindUsage},
		{"empty", failEmptyExit(true), emptyResultsExitCode, errorKindEmpty},
		{"auth", stableExitCode(&gogapi.AuthRequiredError{Service: "drive", Email: "a@b.com", Cause: keyring.ErrKeyNotFound}), exitCodeAuthRequired, errorKindAuth},
		{"keyring", stableExitCode(keyring.ErrKeyNotFound), exitCodeAuthRequired, errorKindAuth},
		{"config", stableExitCode(&config.CredentialsMissingError{Path: "/x/credentials.json", Cause: errors.New("missing")}), exitCodeConfig, errorKindConfig},
		{"cancelled", stableExitCode(context.Canceled), exitCodeCancelled, errorKindCancelled},
		{"deadline", stableExitCode(context.DeadlineExceeded), exitCodeRetryable, errorKindRetryable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := jsonErrorFor(tc.err).Error
			if got.Exit != tc.wantExit || got.Kind != tc.wantKind {
				t.Fatalf("got exit=%d kind=%q, want exit=%d kind=%q", got.Exit, got.Kind, tc.wantExit, tc.wantKind)
			}
			if strings.TrimSpace(got.Message) == "" {
				t.Fatalf("expected a non-empty message")
			}
		})
	}
}

func TestWriteJSONError_MultiLineMessageStaysOneLine(t *testing.T) {
	err := stableExitCode(&gogapi.AuthRequiredError{Service: "drive", Email: "a@b.com", Cause: keyring.ErrKeyNotFound})

	var b strings.Builder
	writeJSONError(&b, err)

	got := parseJSONErrorLine(t, b.String())
	if got.Exit != exitCodeAuthRequired || got.Kind != errorKindAuth {
		t.Fatalf("unexpected body: %#v", got)
	}
	if !strings.Contains(got.Message, "\n") || !strings.Contains(got.Message, "wk auth add a@b.com") {
		t.Fatalf("expected the multi-line human message inside the JSON string, got %q", got.Message)
	}
}

func TestWriteJSONError_EmptyResultsDefaultMessage(t *testing.T) {
	var b strings.Builder
	writeJSONError(&b, failEmptyExit(true))

	got := parseJSONErrorLine(t, b.String())
	if got.Exit != emptyResultsExitCode || got.Kind != errorKindEmpty || got.Message != "no results" {
		t.Fatalf("unexpected body: %#v", got)
	}
}

func googleAPIErrorHandler(status int, reason string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    status,
				"message": "fake " + reason,
				"errors":  []map[string]any{{"reason": reason, "message": "fake " + reason}},
			},
		})
	})
}

func TestExecute_JSONErrors_GoogleAPIStatusToKind(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		reason   string
		wantExit int
		wantKind string
	}{
		{"unauthorized", http.StatusUnauthorized, "authError", exitCodeAuthRequired, errorKindAuth},
		{"forbidden", http.StatusForbidden, "insufficientPermissions", exitCodePermissionDenied, errorKindPerm},
		{"forbidden rate limit", http.StatusForbidden, "rateLimitExceeded", exitCodeRateLimited, errorKindRateLimited},
		{"not found", http.StatusNotFound, "notFound", exitCodeNotFound, errorKindNotFound},
		{"too many requests", http.StatusTooManyRequests, "rateLimitExceeded", exitCodeRateLimited, errorKindRateLimited},
		{"server error", http.StatusServiceUnavailable, "backendError", exitCodeRetryable, errorKindRetryable},
		{"bad request", http.StatusBadRequest, "invalid", 1, errorKindError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origNew := newDriveService
			t.Cleanup(func() { newDriveService = origNew })

			svc, closeSrv := newDriveTestService(t, googleAPIErrorHandler(tc.status, tc.reason))
			defer closeSrv()
			newDriveService = stubDriveService(svc)

			var execErr error
			stderr := captureStderr(t, func() {
				_ = captureStdout(t, func() {
					execErr = Execute([]string{"--json", "--account", "a@b.com", "drive", "get", "id1"})
				})
			})

			if got := ExitCode(execErr); got != tc.wantExit {
				t.Fatalf("exit=%d, want %d (err=%v)", got, tc.wantExit, execErr)
			}
			body := parseJSONErrorLine(t, stderr)
			if body.Exit != tc.wantExit || body.Kind != tc.wantKind {
				t.Fatalf("body=%#v, want exit=%d kind=%q", body, tc.wantExit, tc.wantKind)
			}
			if !strings.Contains(body.Message, "fake "+tc.reason) {
				t.Fatalf("message lost the API detail: %q", body.Message)
			}
		})
	}
}

func TestExecute_PlainErrorsUnchangedWithoutJSON(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	svc, closeSrv := newDriveTestService(t, googleAPIErrorHandler(http.StatusNotFound, "notFound"))
	defer closeSrv()
	newDriveService = stubDriveService(svc)

	var execErr error
	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			execErr = Execute([]string{"--color", "never", "--account", "a@b.com", "drive", "get", "id1"})
		})
	})

	if got := ExitCode(execErr); got != exitCodeNotFound {
		t.Fatalf("exit=%d, want %d", got, exitCodeNotFound)
	}
	if strings.HasPrefix(strings.TrimSpace(stderr), "{") {
		t.Fatalf("plain mode must not emit JSON errors, got %q", stderr)
	}
	if !strings.Contains(stderr, "Google API error (404 notFound)") {
		t.Fatalf("expected human error text, got %q", stderr)
	}
}

func TestExecute_JSONErrors_EmptyResults(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	svc, closeSrv := newDriveTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"drives": []any{}})
	}))
	defer closeSrv()
	newDriveService = stubDriveService(svc)

	var execErr error
	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			execErr = Execute([]string{"--json", "--account", "a@b.com", "drive", "drives", "--fail-empty"})
		})
	})

	if got := ExitCode(execErr); got != emptyResultsExitCode {
		t.Fatalf("exit=%d, want %d", got, emptyResultsExitCode)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("stdout should still carry the (empty) JSON result, got %q", stdout)
	}
	body := parseJSONErrorLine(t, stderr)
	if body.Exit != emptyResultsExitCode || body.Kind != errorKindEmpty {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestExecute_JSONErrors_UsageBeforeRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"parse error", []string{"--json", "drive", "get"}},
		{"unknown command", []string{"--json", "no-such-command"}},
		{"read-only block", []string{"--json", "--read-only", "--account", "a@b.com", "drive", "delete", "id1"}},
		{"enable-commands block", []string{"--json", "--enable-commands", "calendar", "drive", "get", "id1"}},
		{"generate-input unknown", []string{"--json", "--generate-input", "no-such-command"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var execErr error
			stderr := captureStderr(t, func() {
				_ = captureStdout(t, func() {
					execErr = Execute(tc.args)
				})
			})
			if execErr == nil {
				t.Fatalf("expected an error")
			}
			body := parseJSONErrorLine(t, stderr)
			if body.Exit != ExitCode(execErr) {
				t.Fatalf("body exit=%d, process exit=%d", body.Exit, ExitCode(execErr))
			}
			if body.Kind != errorKindForExitCode(body.Exit) {
				t.Fatalf("body kind=%q does not match exit %d", body.Kind, body.Exit)
			}
		})
	}
}

func TestArgsWantJSON(t *testing.T) {
	t.Setenv("WK_JSON", "")
	t.Setenv("WK_PLAIN", "")
	t.Setenv("WK_AUTO_JSON", "")

	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"drive", "get"}, false},
		{[]string{"--json", "drive", "get"}, true},
		{[]string{"-j", "drive", "get"}, true},
		{[]string{"--machine", "drive", "get"}, true},
		{[]string{"drive", "get", "--json"}, true},
		{[]string{"--jq", ".file", "drive", "get"}, true},
		{[]string{"--jq=.file", "drive", "get"}, true},
		{[]string{"--json", "--plain", "drive", "get"}, false},
		{[]string{"--account", "--json", "drive", "get"}, false},
		{[]string{"drive", "get", "--", "--json"}, false},
	}
	for _, tc := range tests {
		if got := argsWantJSON(tc.args); got != tc.want {
			t.Errorf("argsWantJSON(%q)=%v, want %v", tc.args, got, tc.want)
		}
	}

	t.Setenv("WK_JSON", "1")
	if !argsWantJSON([]string{"drive", "get"}) {
		t.Errorf("WK_JSON=1 should enable JSON errors")
	}
}

// WK_AUTO_JSON on a non-TTY stdout selects JSON for normal output, so it must
// select JSON for errors too, including errors raised before kong has parsed
// the flags. captureStdout swaps stdout for a pipe, which is the non-TTY case
// the auto-json output tests use.
func TestArgsWantJSON_AutoJSONNonTTY(t *testing.T) {
	t.Setenv("WK_JSON", "")
	t.Setenv("WK_PLAIN", "")
	t.Setenv("WK_AUTO_JSON", "1")

	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"nosuch"}, true},
		{[]string{"drive", "get"}, true},
		{[]string{"--generate-input", "nosuch"}, true},
		{[]string{"--plain", "nosuch"}, false},
		{[]string{"--json=false", "--plain", "drive", "get"}, false},
	}
	_ = captureStdout(t, func() {
		for _, tc := range tests {
			if got := argsWantJSON(tc.args); got != tc.want {
				t.Errorf("WK_AUTO_JSON=1 argsWantJSON(%q)=%v, want %v", tc.args, got, tc.want)
			}
		}
		if !cliWantsJSON(&CLI{}) {
			t.Errorf("WK_AUTO_JSON=1 cliWantsJSON(no flags)=false, want true")
		}
		if cliWantsJSON(&CLI{RootFlags: RootFlags{Plain: true}}) {
			t.Errorf("WK_AUTO_JSON=1 cliWantsJSON(--plain)=true, want false")
		}
	})
}

func TestExecute_JSONErrors_AutoJSONNonTTY(t *testing.T) {
	t.Setenv("WK_JSON", "")
	t.Setenv("WK_PLAIN", "")
	t.Setenv("WK_AUTO_JSON", "1")

	tests := []struct {
		name string
		args []string
	}{
		{"unknown command", []string{"nosuch"}},
		{"missing argument", []string{"drive", "get"}},
		{"generate-input unknown", []string{"--generate-input", "nosuch"}},
		{"read-only block", []string{"--read-only", "--account", "a@b.com", "drive", "delete", "id1"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var execErr error
			stderr := captureStderr(t, func() {
				_ = captureStdout(t, func() {
					execErr = Execute(tc.args)
				})
			})
			if execErr == nil {
				t.Fatalf("expected an error")
			}
			body := parseJSONErrorLine(t, stderr)
			if body.Exit != ExitCode(execErr) || body.Kind != errorKindForExitCode(body.Exit) {
				t.Fatalf("body=%#v, process exit=%d", body, ExitCode(execErr))
			}
		})
	}

	t.Run("plain wins", func(t *testing.T) {
		stderr := captureStderr(t, func() {
			_ = captureStdout(t, func() {
				_ = Execute([]string{"--plain", "nosuch"})
			})
		})
		if strings.HasPrefix(strings.TrimSpace(stderr), "{") {
			t.Fatalf("--plain should keep human error text, got %q", stderr)
		}
	})
}

// The human "No data found" / "No files" lines are text-mode only: in JSON mode
// an empty result is an empty JSON payload on stdout, exit 0, and nothing on
// stderr (exit 3 with kind "empty_results" stays opt-in via --fail-empty).
func TestExecute_JSONEmptyResults_NoHumanTextOnStderr(t *testing.T) {
	origDrive := newDriveService
	origSheets := newSheetsService
	t.Cleanup(func() {
		newDriveService = origDrive
		newSheetsService = origSheets
	})

	driveSvc, closeDrive := newDriveTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []any{}})
	}))
	defer closeDrive()
	newDriveService = stubDriveService(driveSvc)

	sheetsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"range": "Sheet1!A1:B2"})
	}))
	defer sheetsSrv.Close()
	sheetsSvc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(sheetsSrv.Client()),
		option.WithEndpoint(sheetsSrv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("sheets.NewService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return sheetsSvc, nil }

	tests := []struct {
		name string
		args []string
		text string
	}{
		{"drive ls", []string{"--json", "--account", "a@b.com", "drive", "ls"}, "No files"},
		{"sheets get", []string{"--json", "--account", "a@b.com", "sheets", "get", "sid", "Sheet1!A1:B2"}, "No data found"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var execErr error
			var stdout string
			stderr := captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					execErr = Execute(tc.args)
				})
			})
			if execErr != nil {
				t.Fatalf("unexpected error: %v", execErr)
			}
			if !json.Valid([]byte(stdout)) {
				t.Fatalf("stdout should be the empty JSON result, got %q", stdout)
			}
			if strings.Contains(stderr, tc.text) || strings.TrimSpace(stderr) != "" {
				t.Fatalf("JSON mode must not print human text on stderr, got %q", stderr)
			}
		})
	}
}
