# Google Drive Sync

workit provides bidirectional sync between local folders and Google Drive, similar to Google Drive for Desktop but for Linux and headless environments.

## Quick Start

```bash
# 1. Authenticate
wk auth add you@gmail.com --services=drive

# 2. Initialize sync folder
wk sync init ~/Documents/work --drive-folder="Work Files"

# 3. Start sync (foreground)
wk sync start ~/Documents/work --account=you@gmail.com

# Or start as daemon
wk sync start ~/Documents/work --daemon --account=you@gmail.com
```

## Commands

### Initialize Sync

```bash
wk sync init <local-path> --drive-folder=<name-or-id> [--drive-id=<shared-drive-id>]
```

Creates a sync configuration linking a local folder to a Google Drive folder.

**Examples:**

```bash
# Sync with My Drive folder
wk sync init ~/projects/docs --drive-folder="Project Docs"

# Sync with a Shared Drive folder
wk sync init ~/team/assets --drive-folder="Assets" --drive-id=0AGrKFSfP...

# Using folder ID directly
wk sync init ~/backup --drive-folder=1a2b3c4d5e...
```

### List Configurations

```bash
wk sync list
```

Shows all configured sync folders:

```
ID  LOCAL PATH           DRIVE FOLDER    CREATED              LAST SYNC
1   /home/user/projects  Project Docs    2024-01-15T10:30:00  2024-01-15T14:22:00
2   /home/user/team      Assets          2024-01-10T09:00:00  -
```

### Start Sync

```bash
wk sync start <local-path> --account=<email> [--daemon] [--conflict=<strategy>]
```

Starts the sync engine for a configured folder.

**Flags:**
- `--daemon, -d`: Run in background
- `--conflict`: Conflict resolution strategy (see below)

**Examples:**

```bash
# Foreground (Ctrl+C to stop)
wk sync start ~/projects --account=you@gmail.com

# Background daemon
wk sync start ~/projects --daemon --account=you@gmail.com

# With conflict strategy
wk sync start ~/projects --daemon --account=you@gmail.com --conflict=local-wins
```

### Stop Sync

```bash
wk sync stop
```

Stops the running sync daemon.

### Check Status

```bash
wk sync status
```

Shows sync status for all configurations:

```
Daemon running (PID 12345)

ID  LOCAL PATH  TOTAL  SYNCED  PENDING  CONFLICT  ERROR  LAST SYNC
1   ~/projects  150    148     2        0         0      2024-01-15T14:22:00
```

### Remove Configuration

```bash
wk sync remove <local-path>
```

Removes a sync configuration (does not delete files).

## Conflict Resolution

When both local and remote files are modified between syncs, a conflict occurs. workit supports three resolution strategies:

### Rename (Default)

Keeps both versions by renaming the local file:

```
report.docx           → report.conflict-2024-01-15-143022.docx (local version)
report.docx           → downloaded from Drive (remote version)
```

```bash
wk sync start ~/docs --account=you@gmail.com --conflict=rename
```

### Local Wins

Uploads local version, overwrites remote:

```bash
wk sync start ~/docs --account=you@gmail.com --conflict=local-wins
```

### Remote Wins

Downloads remote version, overwrites local:

```bash
wk sync start ~/docs --account=you@gmail.com --conflict=remote-wins
```

## How Sync Works

### Local Changes → Drive

1. **fsnotify** watches the local folder for changes
2. Events are debounced (500ms) to batch rapid changes
3. Files are uploaded/updated/deleted on Drive
4. MD5 checksums verify integrity

### Drive Changes → Local

1. **Drive Changes API** is polled every 5 seconds
2. Changed files are downloaded
3. Deleted files are removed locally
4. MD5 checksums verify integrity

### What's Synced

- Regular files (documents, images, code, etc.)
- Folders (created/deleted)

### What's Ignored

- `.git` directories
- `node_modules` directories
- `__pycache__` directories
- Hidden files (starting with `.`)
- Temp files (ending with `~`)
- `.DS_Store` files

### Google Docs/Sheets/Slides

Native Google formats (Docs, Sheets, Slides) are **not synced** as they don't have binary content. Use `wk drive download --export-as=docx` for exports.

## Daemon Management

### PID File

When running as daemon, the PID is stored at:
```
~/.config/workit/sync.pid
```

### Log File

Daemon logs are written to:
```
~/.config/workit/sync.log
```

### Auto-restart

For production use, consider using systemd to auto-restart the daemon:

```ini
[Unit]
Description=wk sync daemon
After=network.target

[Service]
Type=simple
User=youruser
ExecStart=/usr/local/bin/wk sync start /path/to/folder --account=you@gmail.com
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## JSON Output

All commands support `--json` for machine-readable output:

```bash
wk sync status --json
```

```json
{
  "count": 1,
  "running": true,
  "pid": 12345,
  "statuses": [
    {
      "config": {
        "id": 1,
        "local_path": "/home/user/projects",
        "drive_folder_id": "1abc..."
      },
      "total_items": 150,
      "synced_items": 148,
      "pending_items": 2,
      "conflict_items": 0,
      "error_items": 0
    }
  ]
}
```

## Database

Sync state is stored in SQLite at:
```
~/.config/workit/sync.db
```

Tables:
- `sync_configs`: Configured sync folders
- `sync_items`: Individual file sync states
- `sync_log`: Sync activity log

## Limitations

- One daemon instance globally (not per-folder)
- No real-time push from Drive (polling every 5s)
- Native Google Docs formats not synced
- Symbolic links not followed
- Large files may take time to transfer

## Environment Variables

| Variable | Description |
|----------|-------------|
| `WK_SYNC_POLL_INTERVAL` | Drive polling interval (default: 5s) |
| `WK_KEYRING_BACKEND` | Keyring backend for tokens |

## Troubleshooting

### "sync config not found"

Initialize the folder first:
```bash
wk sync init <path> --drive-folder=<name>
```

### "daemon already running"

Stop the existing daemon:
```bash
wk sync stop
```

### "account flag is required"

Specify which account to use:
```bash
wk sync start ~/folder --account=you@gmail.com
```

### Files not syncing

1. Check the sync log: `cat ~/.config/workit/sync.log`
2. Verify the path isn't ignored (not hidden, not in node_modules, etc.)
3. Ensure Drive API access with: `wk drive list --account=you@gmail.com`
