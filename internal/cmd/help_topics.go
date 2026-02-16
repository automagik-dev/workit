package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/steipete/gogcli/internal/outfmt"
)

// AgentHelpCmd displays help topics for agent integration.
type AgentHelpCmd struct {
	Topic string `arg:"" optional:"" name:"topic" help:"Help topic name (auth, output, agent, pagination, errors)"`
}

type helpTopic struct {
	Name    string
	Title   string
	Summary string
	Content string
}

var helpTopics = []helpTopic{
	{
		Name:    "auth",
		Title:   "Authentication",
		Summary: "OAuth setup, token storage, headless auth",
		Content: topicAuth,
	},
	{
		Name:    "output",
		Title:   "Output Modes",
		Summary: "JSON, plain text, field selection, exit codes",
		Content: topicOutput,
	},
	{
		Name:    "agent",
		Title:   "Agent Integration",
		Summary: "Zero-shot patterns, recommended flags, error handling",
		Content: topicAgent,
	},
	{
		Name:    "pagination",
		Title:   "Pagination",
		Summary: "Page sizes, --all flag, nextPageToken in JSON output",
		Content: topicPagination,
	},
	{
		Name:    "errors",
		Title:   "Error Handling",
		Summary: "Error format, exit codes, retry guidance",
		Content: topicErrors,
	},
}

func findHelpTopic(name string) *helpTopic {
	name = strings.ToLower(strings.TrimSpace(name))
	for i := range helpTopics {
		if helpTopics[i].Name == name {
			return &helpTopics[i]
		}
	}
	return nil
}

func availableTopicNames() []string {
	names := make([]string, len(helpTopics))
	for i, t := range helpTopics {
		names[i] = t.Name
	}
	return names
}

func (c *AgentHelpCmd) Run(ctx context.Context) error {
	// Always emit untransformed JSON, even if the caller enabled global JSON transforms.
	ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{})

	topic := strings.TrimSpace(c.Topic)

	// No topic or "topics" keyword: list all topics.
	if topic == "" || topic == "topics" {
		return c.listTopics(ctx)
	}

	// Look up specific topic.
	t := findHelpTopic(topic)
	if t == nil {
		return fmt.Errorf("unknown help topic %q. Available topics: %s",
			topic, strings.Join(availableTopicNames(), ", "))
	}

	return c.showTopic(ctx, t)
}

func (c *AgentHelpCmd) listTopics(ctx context.Context) error {
	if outfmt.IsJSON(ctx) {
		type topicEntry struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			Summary string `json:"summary"`
		}
		entries := make([]topicEntry, len(helpTopics))
		for i, t := range helpTopics {
			entries[i] = topicEntry{
				Name:    t.Name,
				Title:   t.Title,
				Summary: t.Summary,
			}
		}
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"topics": entries})
	}

	// Plain/human output.
	fmt.Fprintln(os.Stdout, "Available help topics:")
	fmt.Fprintln(os.Stdout)
	for _, t := range helpTopics {
		fmt.Fprintf(os.Stdout, "  %-12s  %s\n", t.Name, t.Summary)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Usage: gog agent help <topic>")

	return nil
}

func (c *AgentHelpCmd) showTopic(ctx context.Context, t *helpTopic) error {
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"topic":   t.Name,
			"title":   t.Title,
			"content": t.Content,
		})
	}

	// Plain/human output.
	fmt.Fprintf(os.Stdout, "%s\n", t.Title)
	fmt.Fprintf(os.Stdout, "%s\n\n", strings.Repeat("=", len(t.Title)))
	fmt.Fprintln(os.Stdout, t.Content)

	return nil
}

// ---------------------------------------------------------------------------
// Topic content constants
// ---------------------------------------------------------------------------

const topicAuth = `Authentication in gog uses OAuth 2.0 to access Google Workspace APIs
on behalf of a user.

Setup:
  1. Configure OAuth client credentials (client ID + secret).
     Store them via: gog auth credentials set --client-id=ID --client-secret=SECRET
     Or set environment variables: GOG_CLIENT_ID and GOG_CLIENT_SECRET

  2. Authorize an account:
     gog auth add --account user@gmail.com
     This opens a browser for OAuth consent. The refresh token is stored
     in your OS keychain.

  3. Verify auth status:
     gog auth status
     gog auth status --account user@gmail.com

Using accounts:
  Most API commands require --account (or -a) to specify which Google
  account to use:
    gog drive ls --account user@gmail.com
    gog gmail labels --account user@gmail.com

  Set GOG_ACCOUNT to avoid passing --account on every call:
    export GOG_ACCOUNT=user@gmail.com

Headless auth:
  For environments without a browser (CI, containers, agents), use the
  device code flow or a callback server. See: gog auth add --help

Token storage:
  Tokens are stored in the OS keychain by default. For headless
  environments, set:
    export GOG_KEYRING_BACKEND=file
    export GOG_KEYRING_PASSWORD=your-password

Environment variables:
  GOG_CLIENT_ID       - OAuth client ID
  GOG_CLIENT_SECRET   - OAuth client secret
  GOG_ACCOUNT         - Default account email
  GOG_CLIENT          - Default OAuth client name
  GOG_KEYRING_BACKEND - Keyring backend (auto, file, keychain, kwallet, wincred)
  GOG_KEYRING_PASSWORD - Password for file-based keyring`

const topicOutput = `gog supports multiple output modes to suit both human and machine consumers.

Flags:
  --json, -j         Output JSON to stdout (best for scripting and agents)
  --plain, -p        Output stable, parseable text (TSV; no colors)
  (default)          Human-friendly output with colors when on a TTY

JSON envelope:
  JSON output wraps results in an envelope with metadata:
    {
      "files": [...],
      "nextPageToken": "..."
    }

  Use --results-only to strip the envelope and emit only the primary
  result array or object. Note: this also strips nextPageToken, so
  avoid --results-only when paginating across multiple pages.

Field selection:
  --select FIELDS     Project JSON output to only the specified fields.
                      Comma-separated; supports dot paths.
    gog drive ls --json --select "name,id,mimeType"
    gog gmail messages --json --select "id,snippet,payload.headers"

  Aliases: --pick, --project, --fields (except in calendar events)

Exit codes:
  0   Success
  1   General error
  2   Usage / parse error
  3   Empty results (no matches found)
  4   Auth required (token expired or missing)
  5   Not found
  6   Permission denied
  7   Rate limited
  8   Retryable server error
  10  Configuration error
  130 Cancelled (Ctrl-C)

  Use: gog agent exit-codes --json for the full machine-readable list.

Auto-JSON mode:
  Set GOG_AUTO_JSON=1 to default to JSON output when stdout is piped
  (non-TTY). --plain can still override.`

const topicAgent = `Agent Integration Guide

gog is designed for zero-shot LLM agent use. Agents should always pass
--json for machine-parseable output and --account to specify the user.

Recommended invocation pattern:
  gog <command> --json --account user@gmail.com [flags...]

Discovery:
  gog agent help <topic>         Concept-level documentation
  gog agent exit-codes --json    Stable exit codes for automation
  gog schema --json              Full command/flag schema (machine-readable)
  gog <command> --help           Per-command usage help

Error handling:
  1. Check exit code first (see: gog agent help errors)
  2. If exit code is 4 (auth_required), run: gog auth add --account user@
  3. If exit code is 7 (rate_limited) or 8 (retryable), wait and retry
  4. If exit code is 3 (empty_results), the query succeeded but found nothing
  5. Parse stderr for human-readable error messages

Flags for agents:
  --json              Always use this. Output is a JSON envelope to stdout.
  --account EMAIL     Specify the Google account.
  --results-only      Strip envelope, emit only the primary result.
  --select FIELDS     Project to specific fields (comma-separated).
  --no-input          Never prompt; fail instead (for CI / agent use).
  --force, -y         Skip destructive-action confirmations.
  --dry-run, -n       Preview changes without executing.
  --verbose, -v       Enable debug logging to stderr.

Stdin input:
  Commands that accept structured input (e.g., gmail send) can read
  from stdin. Pipe JSON directly:
    echo '{"to":"a@b.com","subject":"Hi"}' | gog gmail send --json

Services available:
  gmail, calendar, drive, docs, slides, sheets, forms, contacts,
  people, tasks, chat, classroom, keep, groups, appscript`

const topicPagination = `Pagination

Google APIs return results in pages. gog provides flags to control
pagination behavior.

Flags:
  --all               Fetch all pages and combine results.
                      Warning: may be slow for large result sets.

  Individual commands may define their own page size flags
  (e.g., --max, --limit). Check per-command --help for details.

How it works:
  When --json is used, paginated responses include a nextPageToken field:
    {
      "files": [...],
      "nextPageToken": "TOKEN_STRING"
    }

  To fetch the next page, pass the token back:
    gog drive ls --json --page-token TOKEN_STRING

  Repeat until nextPageToken is absent from the response.

Per-service page size defaults:
  Most Google APIs default to 10-100 results per page. The exact default
  depends on the service and endpoint. Use per-command flags to control
  the page size.

Tips for agents:
  - Use --all only when you need every result and the total is bounded.
  - For large collections, paginate manually using nextPageToken.
  - --results-only strips nextPageToken from output. Avoid using it
    when you need to paginate across multiple pages.
  - Check exit code 3 (empty_results) to detect "no matches found".`

const topicErrors = `Error Handling

gog uses structured exit codes so agents can branch on failure type
without parsing error text.

Exit codes:
  0   Success
  1   General error (unexpected failure)
  2   Usage / parse error (bad flags, missing required args)
  3   Empty results (query succeeded, but nothing matched)
  4   Auth required (token expired, missing, or revoked)
  5   Not found (resource does not exist)
  6   Permission denied (insufficient scopes or access)
  7   Rate limited (API quota exceeded)
  8   Retryable server error (5xx, timeout, network)
  10  Configuration error (missing credentials file, bad config)
  130 Cancelled (SIGINT / Ctrl-C)

  Machine-readable list: gog agent exit-codes --json

Error output:
  - Error messages are written to stderr.
  - stdout remains clean for JSON parsing.
  - In --json mode, errors are NOT written to stdout; always check
    the exit code and stderr.

Common errors and remedies:

  Exit 4 (auth_required):
    Token has expired or was revoked. Re-authenticate:
      gog auth add --account user@gmail.com

  Exit 6 (permission_denied):
    The OAuth token lacks required scopes. Re-add with broader scopes:
      gog auth add --account user@gmail.com --services gmail,drive

  Exit 7 (rate_limited):
    Google API quota exceeded. Wait and retry with exponential backoff.
    Typical wait: 30-60 seconds. Check Google Cloud Console for quotas.

  Exit 8 (retryable):
    Transient server error (5xx) or network issue. Retry after a short
    delay (1-5 seconds). Usually resolves on retry.

  API enablement errors:
    If you see "API not enabled" in stderr, the Google API needs to be
    enabled in the Google Cloud Console for your project. Common APIs:
      - Gmail API
      - Google Drive API
      - Google Calendar API
      - People API (for contacts/people commands)

Retry guidance:
  - Transient (exit 7, 8): retry with exponential backoff (1s, 2s, 4s...)
  - Permanent (exit 4, 5, 6, 10): do NOT retry; fix the root cause
  - Usage (exit 2): do NOT retry; fix the command invocation`
