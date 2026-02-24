# groups.md

Google Groups (`gog groups` / alias `group`). Workspace context required.

Use `--read-only` for inspection and `--json`/`--plain` for machine-friendly output.

## Top-level commands (from `gog groups --help`)
- `list`
- `members <groupEmail>`

## Examples
```bash
gog groups list --read-only --plain
gog groups members eng@acme.com --read-only --json
```