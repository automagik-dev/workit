# gmail.md

Gmail (`gog gmail` / alias `mail,email`).

Use safe defaults while exploring:
- Add `--read-only` for discovery/read tasks
- Add `--dry-run` before write actions
- Prefer `--json` or `--plain` for scripts

## Top-level commands (from `gog gmail --help`)

### Read
- `gog gmail search <query...>` — search threads
- `gog gmail messages <command>` — message operations
- `gog gmail get <messageId>` — get one message
- `gog gmail attachment <messageId> <attachmentId>` — download one attachment
- `gog gmail url <threadId...>` — print Gmail web URLs
- `gog gmail history` — Gmail history

### Organize
- `gog gmail thread <command>` — thread operations
- `gog gmail labels <command>` — label operations
- `gog gmail batch <command>` — batch operations

### Write / admin
- `gog gmail send` — send an email
- `gog gmail track <command>` — open-tracking commands
- `gog gmail drafts <command>` — draft operations
- `gog gmail settings <command>` — settings/admin

## Examples
```bash
# Search safely
gog gmail search 'subject:(invoice OR receipt) newer_than:30d' --read-only --json

# Preview a send
gog gmail send --to ops@acme.com --subject "Daily report" --body "Attached" --dry-run
```