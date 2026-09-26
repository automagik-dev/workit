package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/automagik-dev/workit/internal/errfmt"
	"github.com/automagik-dev/workit/internal/outfmt"
)

// Error kinds emitted in the --json error envelope. Each kind is the name
// `wk exit-codes` prints for the same stable exit code (see
// agent_exit_codes.go), so machine callers can branch on either the process
// exit status or the "kind" field with one vocabulary.
const (
	errorKindError       = "error"
	errorKindUsage       = "usage"
	errorKindEmpty       = "empty_results"
	errorKindAuth        = "auth_required"
	errorKindNotFound    = "not_found"
	errorKindPerm        = "permission_denied"
	errorKindRateLimited = "rate_limited"
	errorKindRetryable   = "retryable"
	errorKindConfig      = "config"
	errorKindCancelled   = "cancelled"
)

type jsonErrorBody struct {
	Exit    int    `json:"exit"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type jsonErrorEnvelope struct {
	Error jsonErrorBody `json:"error"`
}

// errorKindForExitCode maps a stable exit code to its machine-readable kind.
// Unknown codes fall back to the generic "error" kind.
func errorKindForExitCode(code int) string {
	switch code {
	case 2:
		return errorKindUsage
	case emptyResultsExitCode:
		return errorKindEmpty
	case exitCodeAuthRequired:
		return errorKindAuth
	case exitCodeNotFound:
		return errorKindNotFound
	case exitCodePermissionDenied:
		return errorKindPerm
	case exitCodeRateLimited:
		return errorKindRateLimited
	case exitCodeRetryable:
		return errorKindRetryable
	case exitCodeConfig:
		return errorKindConfig
	case exitCodeCancelled:
		return errorKindCancelled
	default:
		return errorKindError
	}
}

func jsonErrorFor(err error) jsonErrorEnvelope {
	code := ExitCode(err)
	kind := errorKindForExitCode(code)

	msg := strings.TrimSpace(errfmt.Format(err))
	if msg == "" {
		if kind == errorKindEmpty {
			msg = "no results"
		} else {
			msg = kind
		}
	}

	return jsonErrorEnvelope{Error: jsonErrorBody{Exit: code, Kind: kind, Message: msg}}
}

// writeJSONError writes err as exactly one JSON line:
//
//	{"error":{"exit":N,"kind":"...","message":"..."}}
//
// Multi-line human messages are kept intact inside the JSON string, so the
// output is always a single line.
func writeJSONError(w io.Writer, err error) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(jsonErrorFor(err))
}

// printError reports a command failure on stderr: one JSON line when JSON
// output is requested, the existing human text otherwise.
func printError(jsonMode bool, err error) {
	if jsonMode {
		writeJSONError(os.Stderr, err)
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(err))
}

// autoJSON reports whether WK_AUTO_JSON asks for JSON on this process: the
// variable is set and stdout is not a terminal.
func autoJSON() bool {
	return envBool("WK_AUTO_JSON") && !term.IsTerminal(int(os.Stdout.Fd()))
}

// wantsJSON is the one JSON-mode rule shared by the output path (Execute) and
// the error path: --plain always wins, then an explicit JSON request (--json,
// --jq, WK_JSON), then WK_AUTO_JSON on a non-TTY stdout.
func wantsJSON(jsonMode, plain bool) bool {
	if plain {
		return false
	}
	return jsonMode || autoJSON()
}

// cliWantsJSON reports whether parsed flags ask for JSON output.
func cliWantsJSON(cli *CLI) bool {
	if cli == nil {
		return false
	}
	return wantsJSON(cli.JSON || cli.JQ != "", cli.Plain)
}

// argsWantJSON is the pre-parse variant of cliWantsJSON, used for errors that
// happen before kong has populated the flags (parse errors, --generate-input).
func argsWantJSON(args []string) bool {
	env := outfmt.FromEnv()
	jsonMode, plain := env.JSON, env.Plain

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}

		switch {
		case a == "--json" || a == "-j" || a == "--machine" || a == "--json=true":
			jsonMode = true
		case a == "--json=false":
			jsonMode = false
		case a == "--plain" || a == "-p" || a == "--tsv" || a == "--plain=true":
			plain = true
		case a == "--jq" || strings.HasPrefix(a, "--jq="):
			jsonMode = true
			if a == "--jq" {
				i++
			}
		case globalFlagTakesValue(a):
			i++
		}
	}

	return wantsJSON(jsonMode, plain)
}
