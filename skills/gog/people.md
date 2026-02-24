# people.md

Google People (`gog people` / alias `person`).

Use `--read-only` for discovery and `--json`/`--plain` for scripts.

## Top-level commands (from `gog people --help`)
- `me`
- `get <userId>`
- `search <query...>`
- `relations [userId]`

## Examples
```bash
gog people me --read-only --json
gog people search 'product manager sao paulo' --read-only --plain
```