# keep.md

Google Keep (`gog keep`) — Workspace only.

Use `--read-only` for exploration and `--json`/`--plain` for automation.
Use `--dry-run` when available on mutating paths.

## Top-level commands (from `gog keep --help`)
- `list`
- `get <noteId>`
- `search <query>`
- `attachment <attachmentName>`

## Service-account flags
- `--service-account <json-key>`
- `--impersonate <user@domain>`

## Examples
```bash
gog keep list --read-only --json
gog keep get <noteId> --read-only --plain
```