# docs.md

Google Docs: create/read/update text, structure operations, comments, and template generation.

## CRUD + content operations
- `gog docs create <title>`
- `gog docs info <docId>`
- `gog docs cat <docId>`
- `gog docs write <docId> [content] [--replace] [--markdown] [-f file]`
- `gog docs insert <docId> [content] [--index <n>] [-f file]`
- `gog docs delete <docId> --start <n> --end <n>`
- `gog docs find-replace <docId> <find> <replace>`
- `gog docs update <docId> ...`
- `gog docs structure <docId>`
- `gog docs list-tabs <docId>`
- `gog docs header <docId> [--set ...]`
- `gog docs footer <docId> [--set ...]`

## File-level and comments
- `gog docs export <docId> --format pdf|docx|txt`
- `gog docs copy <docId> <title>`
- `gog docs comments list <docId>`
- `gog docs comments get <docId> <commentId>`
- `gog docs comments add <docId> <content> [--quoted ...]`
- `gog docs comments reply <docId> <commentId> <content>`
- `gog docs comments resolve <docId> <commentId>`
- `gog docs comments delete <docId> <commentId>`

## Batch-like workflows
- Use `find-replace` + `insert` + `update` in sequence.
- For API-native bulk request payloads, prefer scripted invocation with `--json` outputs.

## Template creation
- `gog docs generate --template <templateDocId> --data @vars.json [--title ...]`

## Example
```bash
gog docs generate --template 1AbC... --data @proposal-data.json --title 'Proposal - ACME' --dry-run
gog docs find-replace <docId> '{{CLIENT_NAME}}' 'ACME Corp' --dry-run
```
