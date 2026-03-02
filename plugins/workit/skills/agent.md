# agent.md

Agent-friendly helpers for automation, introspection, and machine-readable output.

## Help topics
- `wk agent help topics` — list all available help topics
- `wk agent help <topic>` — display a help topic (auth, output, agent, pagination, errors)

## Exit codes
- `wk agent exit-codes` — print stable exit codes for automation (aliases: `exitcodes`, `exit-code`)

## Machine-readable schema
- `wk schema [<command> ...]` — full CLI schema as JSON (alias: `help-json`)
- `wk schema <service> <subcommand> --json` — schema for a specific command
- `wk schema --include-hidden` — include hidden commands and flags

## Global agent-oriented flags
These flags work across all commands:
- `--json` + `--select <fields>` — structured JSON output with field selection
- `--select ""` — pass empty string to discover available fields (output on stderr)
- `--generate-input` / `--gen-input` — print JSON input template showing all flags for a command
- `--read-only` — hide write commands and request read-only OAuth scopes (safe exploration)
- `--dry-run` — preview changes without executing writes
- `--results-only` — strip envelope fields (like `nextPageToken`) from JSON output
- `--jq <expr>` — apply jq expression to JSON output
- `--plain` — stable TSV output for piping
- `--command-tier core|extended|complete` — control command visibility (default: complete)
- `--enable-commands <list>` — comma-separated allowlist of enabled top-level commands
- `--no-input` — never prompt; fail instead (useful for CI)
- `--max-results <N>` + `--page-token <token>` — global pagination control

## Safety
- All commands in this section are read-only. Use `--read-only` for extra safety.
- `--generate-input` and `schema` exit immediately without executing the target command.

## Examples
```bash
# List all help topics
wk agent help topics
wk agent help topics --json

# Read specific help topics
wk agent help auth
wk agent help output
wk agent help pagination
wk agent help errors
wk agent help agent

# Print exit codes for automation
wk agent exit-codes
wk agent exit-codes --json

# Machine-readable schema for the full CLI
wk schema --json

# Schema for a specific command
wk schema gmail send --json
wk schema drive ls --json

# Discover available JSON fields for a command
wk drive ls --json --select "" --read-only

# Get input template for sending email
wk gmail send --generate-input

# Get input template for creating a calendar event
wk calendar create --generate-input

# Restrict CLI to specific commands
wk drive ls --enable-commands "drive,config" --read-only

# Use core tier only
wk --command-tier core --help
```

## Command index
`agent help` `agent exit-codes` `schema`
