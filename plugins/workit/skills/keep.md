# keep.md

Google Keep (`wk keep`) — Workspace only.

Use `--read-only` for exploration and `--json`/`--plain` for automation.
Use `--dry-run` when available on mutating paths.

## Top-level commands (from `wk keep --help`)
- `list`
- `get <noteId>`
- `search <query>`
- `attachment <attachmentName>`

## Service-account flags
- `--service-account <json-key>`
- `--impersonate <user@domain>`

## Examples
```bash
wk keep list --read-only --json
wk keep get <noteId> --read-only --plain
```