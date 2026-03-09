# keep.md

Google Keep (`wk keep`) — Workspace only.

Use `--read-only` for exploration and `--json`/`--plain` for automation.
Use `--dry-run` when available on mutating paths.

## Top-level commands (from `wk keep --help`)
- `list`
- `get <noteId>`
- `search <query>`
- `attachment <attachmentName>`
- `create --title <title> [--body <text> | --list-items <a,b,c>]`
- `delete <noteId> [--force]`
- `permissions add <noteId> --email <email> [--role WRITER]`
- `permissions remove <noteId> --email <email> [--force]`

## Service-account flags
- `--service-account <json-key>`
- `--impersonate <user@domain>`

## Examples
```bash
wk keep list --read-only --json
wk keep get <noteId> --read-only --plain
wk keep create --title "Meeting Notes" --body "Agenda items here" --json
wk keep create --title "Shopping List" --list-items "Milk,Eggs,Bread"
wk keep delete <noteId> --force
wk keep permissions add <noteId> --email user@example.com --role WRITER
wk keep permissions remove <noteId> --email user@example.com --force
```
