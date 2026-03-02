# gmail.md

Gmail (`wk gmail` / alias `mail,email`).

Use safe defaults while exploring:
- Add `--read-only` for discovery/read tasks
- Add `--dry-run` before write actions
- Prefer `--json` or `--plain` for scripts

## Top-level commands (from `wk gmail --help`)

### Read
- `wk gmail search <query...>` — search threads
- `wk gmail messages <command>` — message operations
- `wk gmail get <messageId>` — get one message
- `wk gmail attachment <messageId> <attachmentId>` — download one attachment
- `wk gmail url <threadId...>` — print Gmail web URLs
- `wk gmail history` — Gmail history

### Organize
- `wk gmail thread <command>` — thread operations
- `wk gmail labels <command>` — label operations
- `wk gmail batch <command>` — batch operations

### Write / admin
- `wk gmail send` — send an email
- `wk gmail track <command>` — open-tracking commands
- `wk gmail drafts <command>` — draft operations
- `wk gmail settings <command>` — settings/admin

## Examples
```bash
# Search safely
wk gmail search 'subject:(invoice OR receipt) newer_than:30d' --read-only --json

# Preview a send
wk gmail send --to ops@acme.com --subject "Daily report" --body "Attached" --dry-run
```