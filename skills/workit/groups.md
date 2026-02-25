# groups.md

Google Groups (`wk groups` / alias `group`). Workspace context required.

Use `--read-only` for inspection and `--json`/`--plain` for machine-friendly output.

## Top-level commands (from `wk groups --help`)
- `list`
- `members <groupEmail>`

## Examples
```bash
wk groups list --read-only --plain
wk groups members eng@acme.com --read-only --json
```