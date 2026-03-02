# docs.md

Google Docs: create/read/update text, structure operations, comments, and template generation.

## CRUD + content operations
- `wk docs create <title>`
- `wk docs info <docId>`
- `wk docs cat <docId>`
- `wk docs write <docId> [content] [--replace] [--markdown] [-f file]`
- `wk docs insert <docId> [content] [--index <n>] [-f file]`
- `wk docs delete <docId> --start <n> --end <n>`
- `wk docs find-replace <docId> <find> <replace>`
- `wk docs update <docId> ...`
- `wk docs structure <docId>`
- `wk docs list-tabs <docId>`
- `wk docs header <docId> [--set ...]`
- `wk docs footer <docId> [--set ...]`

## File-level and comments
- `wk docs export <docId> --format pdf|docx|txt`
- `wk docs copy <docId> <title>`
- `wk docs comments list <docId>`
- `wk docs comments get <docId> <commentId>`
- `wk docs comments add <docId> <content> [--quoted ...]`
- `wk docs comments reply <docId> <commentId> <content>`
- `wk docs comments resolve <docId> <commentId>`
- `wk docs comments delete <docId> <commentId>`

## Batch-like workflows
- Use `find-replace` + `insert` + `update` in sequence.
- For API-native bulk request payloads, prefer scripted invocation with `--json` outputs.

## Template creation
- `wk docs generate --template <templateDocId> --data @vars.json [--title ...]`

## Example
```bash
wk docs generate --template 1AbC... --data @proposal-data.json --title 'Proposal - ACME' --dry-run
wk docs find-replace <docId> '{{CLIENT_NAME}}' 'ACME Corp' --dry-run
```
