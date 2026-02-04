# gog-cli Agent Instructions

You are the maintainer of **gog-cli**, a fork of [steipete/gogcli](https://github.com/steipete/gogcli) with enhancements for agent-driven Google Workspace access.

## Project Overview

**gog-cli** extends gogcli with two key features:

1. **Headless OAuth Flow** - Enables agents to authenticate users via mobile-friendly OAuth URLs (WhatsApp, Telegram, etc.)
2. **Real-time Folder Sync** - Bidirectional Google Drive sync like Google Drive for Desktop

This tool runs on agent infrastructure and is operated by AI agents on behalf of users who authorize via their mobile devices.

---

## Repository Setup

This is a fork. Keep upstream compatibility for easy PR contribution.

```bash
# Initial setup (if not done)
git clone https://github.com/steipete/gogcli.git .
git remote add upstream https://github.com/steipete/gogcli.git
git remote set-url origin git@github.com:YOURCOMPANY/gog-cli.git

# Keep in sync with upstream
git fetch upstream
git merge upstream/main
```

---

## Feature 1: Headless OAuth Flow

### Goal

Allow agents to initiate OAuth for users who will complete authorization on their mobile devices.

### Current Flow (gogcli)
```
CLI runs on user machine → Opens local browser → Redirects to localhost:8080/callback
```

### Target Flow (gog-cli)
```
CLI runs on agent infra → Returns OAuth URL → Agent sends URL to user via chat
→ User taps on mobile → Logs in → Redirects to auth.yourcompany.com/callback
→ Server stores token → CLI polls and retrieves token
```

### Implementation Plan

#### 1.1 Auth Callback Server

Create a minimal OAuth callback server. Location: `auth-server/`

```go
// auth-server/main.go
// Endpoints:
// - GET  /callback          - Google redirects here with ?code=...&state=...
// - GET  /token/{state}     - CLI polls this to retrieve token (returns 202 if pending)
// - GET  /status/{state}    - Check auth status without consuming token

// Storage: Redis or in-memory with TTL (5 min)
// Security: state parameter prevents CSRF, tokens are one-time use
```

**Requirements:**
- Single binary, easy to deploy
- Redis optional (can use in-memory for dev)
- Configurable OAuth client credentials via env vars
- HTTPS required in production

#### 1.2 CLI Headless Mode

Modify `internal/googleauth/oauth_flow.go` and `internal/cmd/auth.go` to support headless flow.

**New flags for `gog auth add`:**
```
--headless          Don't open browser, output auth URL instead
--callback-server   URL of callback server (default from build or env)
--poll-timeout      How long to wait for user to complete auth (default: 5m)
--json              Output machine-readable JSON
```

**New commands:**
```bash
gog auth add user@gmail.com --headless --json
# Output:
# {
#   "auth_url": "https://accounts.google.com/o/oauth2/auth?...",
#   "state": "xK9mP2...",
#   "poll_url": "https://auth.yourcompany.com/token/xK9mP2...",
#   "expires_in": 300
# }

gog auth poll <state>
# Polls until token received or timeout

gog auth status <email> --json
# Check if account is authenticated and token validity
```

#### 1.3 Build-time Credential Injection

**Do not commit credentials.** Inject at build time.

Create `internal/config/defaults.go`:
```go
package config

// Injected at build time via -ldflags
var (
    DefaultClientID       string // -X 'github.com/steipete/gogcli/internal/config.DefaultClientID=...'
    DefaultClientSecret   string // -X 'github.com/steipete/gogcli/internal/config.DefaultClientSecret=...'
    DefaultCallbackServer string // -X 'github.com/steipete/gogcli/internal/config.DefaultCallbackServer=...'
)
```

Modify `internal/config/credentials.go` `ReadClientCredentialsFor()`:
```go
func ReadClientCredentialsFor(client string) (ClientCredentials, error) {
    // Try file-based credentials first (existing behavior)
    path, err := ClientCredentialsPathFor(client)
    if err != nil {
        return ClientCredentials{}, err
    }
    if _, statErr := os.Stat(path); statErr == nil {
        // File exists, use it (existing logic)
        return readFromFile(path)
    }
    
    // Fall back to compiled-in defaults
    if DefaultClientID != "" && DefaultClientSecret != "" {
        return ClientCredentials{
            ClientID:     envOr("GOG_CLIENT_ID", DefaultClientID),
            ClientSecret: envOr("GOG_CLIENT_SECRET", DefaultClientSecret),
        }, nil
    }
    
    // Fall back to env vars alone
    if id, secret := os.Getenv("GOG_CLIENT_ID"), os.Getenv("GOG_CLIENT_SECRET"); id != "" && secret != "" {
        return ClientCredentials{ClientID: id, ClientSecret: secret}, nil
    }
    
    return ClientCredentials{}, errMissingCredentials
}
```

Add to `Makefile`:
```makefile
# Internal build with embedded credentials
build-internal:
	go build -ldflags "\
		-X 'github.com/steipete/gogcli/internal/config.DefaultClientID=$(GOG_CLIENT_ID)' \
		-X 'github.com/steipete/gogcli/internal/config.DefaultClientSecret=$(GOG_CLIENT_SECRET)' \
		-X 'github.com/steipete/gogcli/internal/config.DefaultCallbackServer=$(GOG_CALLBACK_SERVER)'" \
		-o bin/gog ./cmd/gog
```

---

## Feature 2: Real-time Folder Sync

### Goal

Provide bidirectional sync between a local folder and a Google Drive folder, with real-time change detection on both sides.

### Architecture

```
┌──────────────┐    ┌──────────────┐    ┌──────────────────┐
│   Local FS   │    │  Sync Engine │    │  Google Drive    │
│   Watcher    │◄──►│              │◄──►│  Changes API     │
│  (fsnotify)  │    │  (conflict   │    │  (polling or     │
└──────────────┘    │   resolver)  │    │   push)          │
                    └──────────────┘    └──────────────────┘
                           │
                    ┌──────▼──────┐
                    │   State DB  │
                    │  (sqlite)   │
                    └─────────────┘
```

### Implementation Plan

#### 2.1 New Commands

```bash
# Initialize sync configuration
gog sync init <local-path> --drive-folder "<folder-name-or-id>" [--drive-id <shared-drive-id>]

# Start sync daemon
gog sync start [--daemon] [--conflict=rename|local-wins|remote-wins]

# Check sync status
gog sync status [--json]

# Force full resync
gog sync rescan

# Stop daemon
gog sync stop

# List all sync configurations
gog sync list

# Remove sync configuration
gog sync remove <local-path>
```

#### 2.2 State Database Schema

Location: `~/.config/gog/sync.db` (SQLite)

```sql
CREATE TABLE sync_configs (
    id INTEGER PRIMARY KEY,
    local_path TEXT UNIQUE NOT NULL,
    drive_folder_id TEXT NOT NULL,
    drive_id TEXT,  -- for shared drives
    created_at INTEGER,
    last_sync_at INTEGER,
    change_token TEXT  -- Drive API startPageToken
);

CREATE TABLE sync_items (
    id INTEGER PRIMARY KEY,
    config_id INTEGER REFERENCES sync_configs(id),
    local_path TEXT NOT NULL,
    drive_id TEXT,
    local_md5 TEXT,
    remote_md5 TEXT,
    local_mtime INTEGER,
    remote_mtime INTEGER,
    sync_state TEXT DEFAULT 'unknown',  -- synced, local_modified, remote_modified, conflict, deleted_local, deleted_remote
    UNIQUE(config_id, local_path)
);

CREATE TABLE sync_log (
    id INTEGER PRIMARY KEY,
    config_id INTEGER,
    action TEXT,  -- upload, download, delete_local, delete_remote, conflict_resolved
    path TEXT,
    timestamp INTEGER,
    details TEXT  -- JSON with extra info
);
```

#### 2.3 Sync Engine Components

**Local Watcher (`internal/sync/watcher.go`):**
- Use `github.com/fsnotify/fsnotify`
- Debounce rapid changes (500ms window)
- Handle: CREATE, WRITE, DELETE, RENAME
- Ignore: `.gog-sync/`, `.git/`, temp files

**Remote Watcher (`internal/sync/drive_changes.go`):**
- Poll `changes.list` API with saved pageToken
- Default poll interval: 30 seconds (configurable)
- Future: Implement `changes.watch` for push notifications

**Sync Logic (`internal/sync/engine.go`):**
```go
type SyncEngine struct {
    config     *SyncConfig
    db         *StateDB
    driveClient *drive.Service
    watcher    *fsnotify.Watcher
}

func (e *SyncEngine) Run(ctx context.Context) error {
    // 1. Initial scan if needed
    // 2. Start local watcher
    // 3. Start remote poller
    // 4. Process changes as they come
}

func (e *SyncEngine) handleLocalChange(event fsnotify.Event) {
    // Debounce, then:
    // - Calculate MD5
    // - Compare with DB state
    // - Upload if changed
    // - Update DB
}

func (e *SyncEngine) handleRemoteChange(change *drive.Change) {
    // - Compare with DB state
    // - Download if remote is newer
    // - Handle conflicts
    // - Update DB
}
```

**Conflict Resolution (`internal/sync/conflict.go`):**
```go
type ConflictStrategy string

const (
    ConflictRename     ConflictStrategy = "rename"      // file.conflict-2024-01-15.ext
    ConflictLocalWins  ConflictStrategy = "local-wins"
    ConflictRemoteWins ConflictStrategy = "remote-wins"
)
```

#### 2.4 Daemon Mode

```bash
gog sync start --daemon
# Writes PID to ~/.config/gog/sync.pid
# Logs to ~/.config/gog/sync.log

gog sync stop
# Reads PID file, sends SIGTERM
```

---

## File Structure (New/Modified)

```
gog-cli/
├── auth-server/                    # NEW: OAuth callback server
│   ├── main.go
│   ├── handlers.go
│   ├── storage.go                  # Redis/memory token store
│   └── Dockerfile
├── cmd/
│   └── gog/
│       └── main.go                 # MODIFY: Add sync subcommand registration
├── internal/
│   ├── cmd/
│   │   ├── auth.go                 # MODIFY: Add --headless flag handling
│   │   └── sync.go                 # NEW: Sync CLI commands
│   ├── config/
│   │   ├── credentials.go          # MODIFY: Add fallback to defaults
│   │   └── defaults.go             # NEW: Build-time injected defaults
│   ├── googleauth/
│   │   ├── oauth_flow.go           # MODIFY: Add headless flow path
│   │   └── headless.go             # NEW: Headless-specific logic
│   └── sync/                       # NEW: Entire directory
│       ├── config.go               # Sync configuration
│       ├── db.go                   # SQLite state management
│       ├── watcher.go              # Local filesystem watcher
│       ├── drive_changes.go        # Remote change detection
│       ├── engine.go               # Main sync logic
│       ├── conflict.go             # Conflict resolution
│       └── daemon.go               # Background daemon management
├── Makefile                        # MODIFY: Add build-internal target
└── docs/
    ├── headless-auth.md            # NEW: Documentation
    └── sync.md                     # NEW: Documentation
```

---

## Development Workflow

### Testing Headless Auth

1. Start callback server locally:
   ```bash
   cd auth-server && go run . --port 8089
   ```

2. Set up ngrok for mobile testing:
   ```bash
   ngrok http 8089
   # Note the https URL
   ```

3. Test headless flow:
   ```bash
   GOG_CALLBACK_SERVER=https://xxxx.ngrok.io gog auth add test@gmail.com --headless
   # Open URL on phone, complete auth
   ```

### Testing Sync

1. Create test folder structure
2. Initialize sync with a test Drive folder
3. Test scenarios:
   - Local file create → appears in Drive
   - Remote file create → appears locally
   - Simultaneous edit → conflict resolution
   - Delete on either side
   - Rename/move operations

---

## Dependencies to Add

```go
// go.mod additions
require (
    github.com/fsnotify/fsnotify v1.7+    // Filesystem watching
    github.com/mattn/go-sqlite3 v1.14+    // State database
    github.com/redis/go-redis/v9 v9.0+    // Callback server storage (optional)
)
```

---

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `GOG_CLIENT_ID` | OAuth client ID | Yes (or build-time) |
| `GOG_CLIENT_SECRET` | OAuth client secret | Yes (or build-time) |
| `GOG_CALLBACK_SERVER` | Callback server URL | Yes (or build-time) |
| `GOG_ACCOUNT` | Default account email | No |
| `GOG_SYNC_POLL_INTERVAL` | Remote poll interval (default: 30s) | No |
| `GOG_SYNC_DEBOUNCE` | Local change debounce (default: 500ms) | No |

---

## PR Strategy for Upstream

Keep changes modular for potential upstream contribution:

1. **PR 1: Headless OAuth** - Generic, useful for any server/agent use case
2. **PR 2: Sync Foundation** - Core sync engine without company-specific bits
3. **Keep separate**: Build-time credential injection (company-specific)

When preparing PRs:
- Follow upstream code style
- Add tests
- Update README
- Don't include company-specific configurations

---

## Testing Checklist

### Headless Auth
- [ ] Generate auth URL with correct parameters
- [ ] State parameter is cryptographically random
- [ ] Polling respects timeout
- [ ] Token stored correctly in keyring
- [ ] Works with multiple accounts
- [ ] Handles user declining auth
- [ ] Handles expired states

### Sync
- [ ] Initial sync downloads all files
- [ ] Local create uploads to Drive
- [ ] Remote create downloads to local
- [ ] Local modify uploads new version
- [ ] Remote modify downloads new version
- [ ] Local delete removes from Drive
- [ ] Remote delete removes locally
- [ ] Conflict detection works
- [ ] All three conflict strategies work
- [ ] Daemon starts/stops correctly
- [ ] Survives network interruptions
- [ ] Handles large files (chunked upload)
- [ ] Respects .gogignore patterns

---

## Agent Usage Examples

Once complete, agents will use gog-cli like this:

```python
# Agent authenticating a new user
def setup_google_access(user_phone):
    result = exec("gog auth add --headless --json")
    auth_data = json.loads(result)
    
    send_whatsapp(user_phone, f"Please authorize Google access:\n{auth_data['auth_url']}")
    
    # Wait for completion (CLI is polling)
    exec(f"gog auth poll {auth_data['state']} --timeout 5m")
    
    return "Google access authorized!"

# Agent syncing user's documents
def setup_drive_sync(local_path, drive_folder):
    exec(f"gog sync init {local_path} --drive-folder '{drive_folder}'")
    exec("gog sync start --daemon")
    return f"Syncing {drive_folder} to {local_path}"

# Agent reading/writing files (sync handles the rest)
def save_report(content):
    with open("/user-drive/reports/weekly.md", "w") as f:
        f.write(content)
    # Sync daemon automatically uploads
```

---

## References

- Upstream repo: https://github.com/steipete/gogcli
- Google Drive API: https://developers.google.com/drive/api/v3/reference
- Changes API: https://developers.google.com/drive/api/v3/reference/changes
- fsnotify: https://github.com/fsnotify/fsnotify
