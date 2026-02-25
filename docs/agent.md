# Agent UX Features

> Back to [README](../README.md)

## Field Discovery (`--select ""`)

Pass an explicit empty string to `--select` to list available JSON fields:

```bash
wk drive ls --json --select ""
```

- Output goes to **stderr** (stdout stays clean for piping).
- Exit code **0**.
- A usage hint is printed: `wk drive ls --json --select "name,id,size"`.
- Works for every command that supports `--json` output.
- **Note:** The command still executes (including auth and API calls) because field names are discovered via reflection on the result struct at output time. The normal JSON payload is suppressed from stdout. A future optimization may short-circuit execution for commands whose output type can be determined statically.

## Input Templates (`--generate-input` / `--gen-input`)

Print a JSON template showing all flags for any command:

```bash
wk gmail send --generate-input
```

- Required fields are prefixed with `(required)` in the value.
- Types, defaults, and enum values are included.
- Exit code **0** -- the command is not executed.
- Excludes Kong built-ins (`--help`, `--version`) and hidden flags.
- Includes both global `RootFlags` and command-specific flags.

## Global Pagination (`--max-results`, `--page-token`)

Control pagination across all services with global flags:

```bash
wk drive ls --max-results 5 --json
wk drive ls --max-results 5 --page-token TOKEN --json
```

- Maps to the correct API parameter per service (`pageSize` or `maxResults`).
- **Precedence:** Global `--max-results` takes priority over per-command defaults when set. Per-command `--max`/`--limit` flags use compile-time defaults that cannot be distinguished from explicit user values by the framework, so the global flag always wins when non-zero.
- `--all` overrides `--max-results` (fetches all pages).
- `--results-only` strips `nextPageToken` from output; avoid it when paginating across multiple pages.

## Help Topics

Concept-level documentation for agent integration:

```bash
wk agent help topics          # list all topics
wk agent help auth            # authentication guide
wk agent help output          # output modes, --json, --select, exit codes
wk agent help agent           # zero-shot patterns, recommended flags
wk agent help pagination      # pagination control
wk agent help errors          # error handling, exit codes, retry guidance
```

- JSON output with `--json` flag.
- Unknown topics suggest the closest match.

Available topics:

| Topic | Description |
|---|---|
| `auth` | OAuth setup, token storage, headless auth |
| `output` | JSON, plain text, field selection, exit codes |
| `agent` | Zero-shot patterns, recommended flags, error handling |
| `pagination` | Page sizes, --all flag, nextPageToken in JSON output |
| `errors` | Error format, exit codes, retry guidance |

## Agent Safety

When an AI agent drives `wk` on behalf of a user, you can restrict what it is allowed to do. Three independent mechanisms are available and they stack -- all filters are evaluated, and the most restrictive combination wins.

### `--read-only`

Block every write operation (send, upload, delete, create, etc.) and, when used with `auth add --readonly`, request read-only OAuth scopes so the token itself cannot perform mutations.

```bash
wk --read-only drive ls        # OK -- listing is read-only
wk --read-only gmail send ...  # BLOCKED
```

### `--command-tier core|extended|complete`

Limit which subcommands are visible. The three tiers are cumulative:

| Tier | What it includes |
|---|---|
| **core** | Read-only essentials -- list, search, get, download, export, cat |
| **extended** | Common mutations -- create, update, delete, send, upload |
| **complete** | Everything (default) -- batch ops, permissions, settings, advanced |

Commands not assigned to a tier default to **complete**. Utility commands (`auth`, `config`, `agent`, `version`, etc.) are always available regardless of tier.

```bash
wk --command-tier core calendar ls      # OK
wk --command-tier core calendar create  # BLOCKED (create requires "extended")
```

### `--enable-commands <csv>`

Allowlist specific top-level commands. Only the listed service groups are accessible; everything else is rejected.

```bash
wk --enable-commands calendar,tasks calendar ls   # OK
wk --enable-commands calendar,tasks gmail search   # BLOCKED
```

### Composing Filters

All three mechanisms are independent and evaluated in order. A command must pass every active filter to execute. For example:

```bash
wk --read-only --command-tier core --enable-commands drive,calendar \
    drive ls
```

This allows only read-only, core-tier subcommands of `drive` and `calendar`.

### Environment Variables for Agent Sandboxing

Set these before launching an agent session so every invocation is automatically restricted:

```bash
export WK_READ_ONLY=true
export WK_COMMAND_TIER=core
export WK_ENABLE_COMMANDS=drive,calendar
```

## Version Artifact Contract

`wk --version` and `wk version --json` expose build metadata used by CI/release automation.

Contract (`wk version --json`):

```json
{
  "version": "v0.12.0",
  "branch": "main",
  "commit": "abc123def456",
  "date": "2026-02-18T20:03:00Z"
}
```

- `version`: semantic version tag for releases, or dev auto-version for non-main builds.
- `branch`: git branch used for the build.
- `commit`: short commit SHA (12 chars).
- `date`: UTC build timestamp (RFC3339).

The `version` workflow (`.github/workflows/version.yml`) publishes this JSON as the `version-contract` artifact and validates the schema.

## Exit Codes

Stable exit codes for automation and agent integration:

| Code | Name | Description |
|---|---|---|
| 0 | `ok` | Success |
| 1 | `error` | General error |
| 2 | `usage` | Invalid usage / bad arguments |
| 3 | `empty_results` | Query returned no results |
| 4 | `auth_required` | Authentication required or token expired |
| 5 | `not_found` | Requested resource not found |
| 6 | `permission_denied` | Insufficient permissions |
| 7 | `rate_limited` | API rate limit exceeded |
| 8 | `retryable` | Transient error (retry may succeed) |
| 10 | `config` | Configuration error |
| 130 | `cancelled` | Operation cancelled (e.g. Ctrl+C) |

Run `wk agent exit-codes` (or `wk exit-codes`) to print these in your preferred output format.

## Global Flags

All commands support these flags:

| Flag | Description |
|---|---|
| `--account <email\|alias\|auto>` | Account to use (overrides `WK_ACCOUNT`) |
| `--client <name>` | OAuth client name (selects stored credentials + token bucket) |
| `--command-tier <core\|extended\|complete>` | Command visibility tier (default: complete; env: `WK_COMMAND_TIER`) |
| `--enable-commands <csv>` | Allowlist top-level commands (env: `WK_ENABLE_COMMANDS`) |
| `--read-only` | Hide write commands and request read-only OAuth scopes (env: `WK_READ_ONLY`) |
| `--json` / `-j` | Output JSON to stdout (best for scripting) |
| `--plain` / `-p` | Output stable, parseable text to stdout (TSV; no colors) |
| `--results-only` | In JSON mode, emit only the primary result (drops `nextPageToken`) |
| `--select <fields>` | In JSON mode, select comma-separated fields |
| `--jq <expr>` | Apply jq expression to JSON output |
| `--max-results <n>` | Maximum number of results to return |
| `--page-token <token>` | Page token for pagination |
| `--generate-input` | Print JSON input template for the command and exit |
| `--dry-run` / `-n` | Do not make changes; print intended actions |
| `--force` / `-y` | Skip confirmations for destructive commands |
| `--no-input` | Never prompt; fail instead (useful for CI) |
| `--color <mode>` | Color mode: `auto`, `always`, or `never` |
| `--verbose` / `-v` | Enable verbose logging |
| `--version` | Print version and exit |
