# Google Drive Sync

gog-cli provides bidirectional sync between local folders and Google Drive, similar to Google Drive for Desktop but for Linux and headless environments.

## Quick Start

```bash
# 1. Authenticate
gog auth add you@gmail.com --services=drive

# 2. Initialize sync folder
gog sync init ~/Documents/work --drive-folder="Work Files"

# 3. Start sync (foreground)
gog sync start ~/Documents/work --account=you@gmail.com

# Or start as daemon
gog sync start ~/Documents/work --daemon --account=you@gmail.com
```

## Commands

### Initialize Sync

```bash
gog sync init <local-path> --drive-folder=<name-or-id> [--drive-id=<shared-drive-id>]
```

Creates a sync configuration linking a local folder to a Google Drive folder.

**Examples:**

```bash
# Sync with My Drive folder
gog sync init ~/projects/docs --drive-folder="Project Docs"

# Sync with a Shared Drive folder
gog sync init ~/team/assets --drive-folder="Assets" --drive-id=0AGrKFSfP...

# Using folder ID directly
gog sync init ~/backup --drive-folder=1a2b3c4d5e...
```

### List Configurations

```bash
gog sync list
```

Shows all configured sync folders:

```
ID  LOCAL PATH           DRIVE FOLDER    CREATED              LAST SYNC
1   /home/user/projects  Project Docs    2024-01-15T10:30:00  2024-01-15T14:22:00
2   /home/user/team      Assets          2024-01-10T09:00:00  -
```

### Start Sync

```bash
gog sync start <local-path> --account=<email> [--daemon] [--conflict=<strategy>]
```

Starts the sync engine for a configured folder.

**Flags:**
- `--daemon, -d`: Run in background
- `--conflict`: Conflict resolution strategy (see below)

**Examples:**

```bash
# Foreground (Ctrl+C to stop)
gog sync start ~/projects --account=you@gmail.com

# Background daemon
gog sync start ~/projects --daemon --account=you@gmail.com

# With conflict strategy
gog sync start ~/projects --daemon --account=you@gmail.com --conflict=local-wins
```

### Stop Sync

```bash
gog sync stop
```

Stops the running sync daemon.

### Check Status

```bash
gog sync status
```

Shows sync status for all configurations:

```
Daemon running (PID 12345)

ID  LOCAL PATH  TOTAL  SYNCED  PENDING  CONFLICT  ERROR  LAST SYNC
1   ~/projects  150    148     2        0         0      2024-01-15T14:22:00
```

### Remove Configuration

```bash
gog sync remove <local-path>
```

Removes a sync configuration (does not delete files).

## Conflict Resolution

When both local and remote files are modified between syncs, a conflict occurs. gog-cli supports three resolution strategies:

### Rename (Default)

Keeps both versions by renaming the local file:

```
report.docx           → report.conflict-2024-01-15-143022.docx (local version)
report.docx           → downloaded from Drive (remote version)
```

```bash
gog sync start ~/docs --account=you@gmail.com --conflict=rename
```

### Local Wins

Uploads local version, overwrites remote:

```bash
gog sync start ~/docs --account=you@gmail.com --conflict=local-wins
```

### Remote Wins

Downloads remote version, overwrites local:

```bash
gog sync start ~/docs --account=you@gmail.com --conflict=remote-wins
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

Native Google formats (Docs, Sheets, Slides) are **not synced** as they don't have binary content. Use `gog drive download --export-as=docx` for exports.

## Daemon Management

### PID File

When running as daemon, the PID is stored at:
```
~/.config/gog/sync.pid
```

### Log File

Daemon logs are written to:
```
~/.config/gog/sync.log
```

### Auto-restart

For production use, consider using systemd to auto-restart the daemon:

```ini
[Unit]
Description=gog sync daemon
After=network.target

[Service]
Type=simple
User=youruser
ExecStart=/usr/local/bin/gog sync start /path/to/folder --account=you@gmail.com
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## JSON Output

All commands support `--json` for machine-readable output:

```bash
gog sync status --json
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
~/.config/gog/sync.db
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
| `GOG_SYNC_POLL_INTERVAL` | Drive polling interval (default: 5s) |
| `GOG_KEYRING_BACKEND` | Keyring backend for tokens |

## Troubleshooting

### "sync config not found"

Initialize the folder first:
```bash
gog sync init <path> --drive-folder=<name>
```

### "daemon already running"

Stop the existing daemon:
```bash
gog sync stop
```

### "account flag is required"

Specify which account to use:
```bash
gog sync start ~/folder --account=you@gmail.com
```

### Files not syncing

1. Check the sync log: `cat ~/.config/gog/sync.log`
2. Verify the path isn't ignored (not hidden, not in node_modules, etc.)
3. Ensure Drive API access with: `gog drive list --account=you@gmail.com`
