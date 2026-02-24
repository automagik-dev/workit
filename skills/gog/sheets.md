# sheets.md

Google Sheets (`gog sheets` / alias `sheet`).

Use `--read-only` for non-mutating checks, `--dry-run` before writes, and `--json`/`--plain` for scripts.

## Top-level commands (from `gog sheets --help`)
- `get <spreadsheetId> <range>`
- `update <spreadsheetId> <range> [values...]`
- `append <spreadsheetId> <range> [values...]`
- `clear <spreadsheetId> <range>`
- `format <spreadsheetId> <range>`
- `notes <spreadsheetId> <range>`
- `metadata <spreadsheetId>`
- `create <title>`
- `copy <spreadsheetId> <title>`
- `export <spreadsheetId>`
- `add-tab <spreadsheet-id> <tab-name>`
- `batch-update <spreadsheetId>`

## Examples
```bash
gog sheets get <sheetId> 'Summary!A1:C20' --read-only --json
gog sheets update <sheetId> 'Summary!A1:C1' 'Week' 'Revenue' 'Delta' --dry-run
```