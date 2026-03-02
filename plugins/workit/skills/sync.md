# sync.md

Google Drive sync (`wk sync`).

Use `--dry-run` before `init`, and prefer `status --read-only` for checks.
Use `--json`/`--plain` when scripting.

## Top-level commands (from `wk sync --help`)
- `init --drive-folder <folderId|path> <local-path>`
- `list`
- `remove <local-path>`
- `status`
- `start <local-path>` *(placeholder daemon start)*
- `stop` *(placeholder daemon stop)*

## Safe pattern
```bash
# Preview init first
wk sync init --drive-folder <driveFolderId> ./local-folder --dry-run

# Apply
wk sync init --drive-folder <driveFolderId> ./local-folder

# Check state
wk sync status --read-only --json
```