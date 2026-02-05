# Wish: Headless OAuth & Drive Sync for gog-cli

## Status
**IN_PROGRESS** | Created: 2025-02-04

## Slug
`headless-oauth-drive-sync`

---

## Summary

Transform gog-cli into a zero-friction Google Workspace CLI for Linux users. Add headless OAuth (users authenticate via mobile, no GCP setup needed) and real-time Drive sync (bidirectional folder sync like Google Drive for Desktop). This enables Namastex to provide a hosted OAuth service at `auth.namastex.io` where users authenticate once and access Gmail, Calendar, Drive, Docs, Sheets, and all other Workspace products forever.

**Why:** Linux lacks an official Google Drive client. Users struggle with OAuth setup (creating GCP projects, downloading credentials.json). By providing hosted auth infrastructure, we eliminate friction entirely.

---

## Scope

### IN Scope

1. **Auth Callback Server** (`auth-server/`)
   - Go HTTP server deployable at `auth.namastex.io`
   - `/callback` - receives OAuth redirect from Google
   - `/token/{state}` - CLI polls to retrieve token
   - `/status/{state}` - check status without consuming
   - In-memory storage with 15-minute TTL
   - Health check endpoint

2. **Headless OAuth in CLI**
   - `--headless` flag for `gog auth add`
   - `--callback-server` flag/env var
   - `gog auth poll <state>` command
   - Build-time credential injection via `-ldflags`
   - Fallback chain: file → build-time defaults → env vars

3. **Drive Sync Foundation**
   - `gog sync init <local-path> --drive-folder <name-or-id>`
   - `gog sync start [--daemon]`
   - `gog sync stop`
   - `gog sync status`
   - `gog sync list` / `gog sync remove`
   - SQLite state database

4. **Sync Engine**
   - Local filesystem watching (fsnotify)
   - Remote change detection (5-second polling)
   - Bidirectional sync with conflict resolution
   - Daemon mode with PID/log management

5. **Infrastructure Documentation**
   - GCP project setup guide
   - Callback server deployment guide
   - VPN configuration notes

### OUT of Scope

- **Push notifications (changes.watch)** - Phase 2 optimization
- **Public callback server** - VPN-only for Phase 1
- **Shared Drive support** - Personal Drive first
- **Selective sync / ignore patterns** - Basic sync first
- **Google Slides API** - Already supported upstream
- **New Google API integrations** - Upstream already covers Gmail, Calendar, Chat, Classroom, Drive, Docs, Contacts, Tasks, People, Sheets, Groups, Keep
- **Mobile apps** - CLI only
- **Windows/macOS Drive sync** - Linux priority (they have official clients)

---

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Callback server tech | Simple Go server | Same language as CLI, single binary, full control |
| Handoff storage | In-memory + TTL | Simple, tokens consumed once then deleted, no persistence needed |
| Handoff TTL | 15 minutes | Enough time to complete auth, short enough for security |
| Security model | State parameter only | VPN-only access, low risk of interception |
| Remote change detection | 5-second polling | Feels instant, simple to implement, well within quota |
| Local change detection | fsnotify | Industry standard, instant detection |
| State database | SQLite | No external dependencies, persistent, good for sync state |
| Upstream strategy | Generic code + Namastex defaults via ldflags | PR-able core, zero-friction for Namastex users |
| Conflict strategy | Rename by default | Safe, no data loss (file.conflict-2024-01-15.ext) |

---

## Success Criteria

### Headless OAuth
- [ ] User can run `gog auth add user@gmail.com --headless` and receive an OAuth URL
- [ ] User can complete OAuth on mobile device (VPN required)
- [ ] CLI automatically retrieves token after user completes auth
- [ ] Token is stored in local keyring, works for all gog commands
- [ ] No credentials.json needed when using Namastex-built binary
- [ ] Works with multiple Google accounts

### Drive Sync
- [ ] User can initialize sync between local folder and Drive folder
- [ ] Local file changes appear in Drive within 5 seconds
- [ ] Drive file changes appear locally within 10 seconds (5s poll + download)
- [ ] Sync survives daemon restart (state persisted in SQLite)
- [ ] Conflicts are detected and resolved (rename strategy)
- [ ] Daemon runs in background, logs to file

### Infrastructure
- [ ] Callback server deployable via Docker
- [ ] Health check endpoint works
- [ ] Documentation covers GCP setup end-to-end

---

## Upstream gogcli Analysis

**Already supported (no changes needed):**
- Gmail, Calendar, Chat, Classroom, Drive, Docs, Slides, Contacts, Tasks, People, Sheets, Groups, Keep
- Multi-account support (tokens keyed by client:email)
- Keyring storage (macOS Keychain, Linux Secret Service, file backend)
- JSON output for scripting
- Service account support (Workspace)

**Services defined in upstream:**
```go
ServiceGmail, ServiceCalendar, ServiceChat, ServiceClassroom,
ServiceDrive, ServiceDocs, ServiceContacts, ServiceTasks,
ServicePeople, ServiceSheets, ServiceGroups, ServiceKeep
```

**What we're adding:**
- Headless OAuth flow (alternative to browser-based)
- Drive sync daemon (new feature)
- Build-time credential injection (fork-specific)

---

## API Quota Analysis (100 users)

| API | Operation | Frequency | Calls/min | Daily Estimate |
|-----|-----------|-----------|-----------|----------------|
| Drive | changes.list (sync) | Every 5s/user | 1,200 | 1.7M |
| Drive | files.get/create/update | On change | Variable | ~50K |
| Gmail | Various | On-demand | Variable | ~10K |
| Calendar | Various | On-demand | Variable | ~10K |

**Default quotas:**
- 12,000 queries per 100 seconds (project-wide) = 7,200/min
- 1,000,000,000 queries per day

**Assessment:** Sync polling (1,200/min) is well within limits. May need quota increase request if usage grows significantly.

---

## Execution Groups

### Group 1: Auth Callback Server
**Goal:** Deployable callback server at auth.namastex.io

**Deliverables:**
- `auth-server/main.go` - Entry point
- `auth-server/handlers.go` - HTTP handlers (/callback, /token/{state}, /status/{state}, /health)
- `auth-server/storage.go` - In-memory store with TTL cleanup
- `auth-server/Dockerfile` - Container build
- `auth-server/README.md` - Deployment instructions

**Acceptance Criteria:**
- [ ] Server starts and listens on configurable port
- [ ] `/health` returns 200 OK
- [ ] `/callback?code=X&state=Y` stores token, returns success page
- [ ] `/token/{state}` returns token if ready, 202 if pending, 404 if expired
- [ ] Tokens auto-expire after 15 minutes
- [ ] Concurrent requests handled safely (mutex or sync.Map)

**Validation:**
```bash
cd auth-server && go build . && ./auth-server --port 8089 &
curl http://localhost:8089/health  # expect 200
# Manual OAuth flow test with ngrok
```

**Files:**
- `auth-server/main.go` (new)
- `auth-server/handlers.go` (new)
- `auth-server/storage.go` (new)
- `auth-server/Dockerfile` (new)
- `auth-server/README.md` (new)

---

### Group 2: CLI Headless OAuth
**Goal:** `gog auth add --headless` works end-to-end

**Deliverables:**
- Modify `internal/cmd/auth.go` - Add --headless, --callback-server flags
- Create `internal/googleauth/headless.go` - Headless flow logic
- Create `internal/config/defaults.go` - Build-time injected defaults
- Modify `internal/config/credentials.go` - Fallback to defaults
- Add `gog auth poll <state>` command
- Update Makefile with `build-internal` target

**Acceptance Criteria:**
- [ ] `gog auth add user@gmail.com --headless --json` outputs auth_url, state, poll_url
- [ ] `gog auth poll <state>` waits for token and stores in keyring
- [ ] `--callback-server` flag overrides default
- [ ] `GOG_CALLBACK_SERVER` env var works
- [ ] Build with `-ldflags` injects defaults correctly
- [ ] Fallback chain: file → defaults → env vars
- [ ] Existing browser-based flow still works

**Validation:**
```bash
# Build with defaults
make build-internal GOG_CLIENT_ID=test GOG_CLIENT_SECRET=test GOG_CALLBACK_SERVER=http://localhost:8089

# Test headless flow (requires callback server running)
./bin/gog auth add test@gmail.com --headless --json
# Expect JSON with auth_url
```

**Files:**
- `internal/cmd/auth.go` (modify)
- `internal/googleauth/headless.go` (new)
- `internal/googleauth/oauth_flow.go` (modify)
- `internal/config/defaults.go` (new)
- `internal/config/credentials.go` (modify)
- `Makefile` (modify)

---

### Group 3: Sync Foundation (Database & Commands)
**Goal:** Sync commands work, state persisted

**Deliverables:**
- Create `internal/sync/` package
- `internal/sync/db.go` - SQLite state management
- `internal/sync/config.go` - Sync configuration types
- `internal/cmd/sync.go` - CLI commands
- Register sync command in root

**Acceptance Criteria:**
- [ ] `gog sync init ~/Drive --drive-folder "My Folder"` creates sync config in DB
- [ ] `gog sync list` shows configured syncs
- [ ] `gog sync remove ~/Drive` removes sync config
- [ ] `gog sync status` shows sync state (even if not running)
- [ ] SQLite DB created at `~/.config/gog/sync.db`
- [ ] Schema includes sync_configs, sync_items, sync_log tables

**Validation:**
```bash
./bin/gog sync init /tmp/test-sync --drive-folder "Test"
./bin/gog sync list --json  # expect config in output
./bin/gog sync remove /tmp/test-sync
./bin/gog sync list --json  # expect empty
```

**Files:**
- `internal/sync/db.go` (new)
- `internal/sync/config.go` (new)
- `internal/cmd/sync.go` (new)
- `internal/cmd/root.go` (modify - add SyncCmd)
- `go.mod` (modify - add sqlite3)

---

### Group 4: Local Filesystem Watcher
**Goal:** Detect local file changes instantly

**Deliverables:**
- `internal/sync/watcher.go` - fsnotify wrapper
- Debouncing logic (500ms window)
- Ignore patterns (.git, .gog-sync, temp files)
- Event channel for sync engine

**Acceptance Criteria:**
- [ ] Watcher detects CREATE, WRITE, DELETE, RENAME events
- [ ] Rapid changes debounced (multiple saves → one event)
- [ ] .git and temp files ignored
- [ ] Watcher can be started/stopped cleanly
- [ ] Events delivered via channel

**Validation:**
```bash
# Unit tests
go test ./internal/sync/... -run TestWatcher -v
```

**Files:**
- `internal/sync/watcher.go` (new)
- `internal/sync/watcher_test.go` (new)
- `go.mod` (modify - add fsnotify)

---

### Group 5: Remote Change Detection
**Goal:** Detect Drive changes within 5 seconds

**Deliverables:**
- `internal/sync/drive_changes.go` - Changes API poller
- Page token management (stored in DB)
- Configurable poll interval (default 5s)

**Acceptance Criteria:**
- [ ] Poller calls changes.list every 5 seconds
- [ ] Page token persisted across restarts
- [ ] Pagination handled for large change sets
- [ ] Changes delivered via channel
- [ ] Poller can be started/stopped cleanly

**Validation:**
```bash
# Integration test (requires auth)
GOG_ACCOUNT=test@gmail.com go test ./internal/sync/... -run TestDriveChanges -v
```

**Files:**
- `internal/sync/drive_changes.go` (new)
- `internal/sync/drive_changes_test.go` (new)

---

### Group 6: Sync Engine
**Goal:** Bidirectional sync works

**Deliverables:**
- `internal/sync/engine.go` - Main sync loop
- Upload logic (local → Drive)
- Download logic (Drive → local)
- MD5 comparison for change detection
- Sync state tracking in DB

**Acceptance Criteria:**
- [ ] Local file create → uploaded to Drive
- [ ] Local file modify → new version uploaded
- [ ] Local file delete → removed from Drive
- [ ] Remote file create → downloaded locally
- [ ] Remote file modify → downloaded locally
- [ ] Remote file delete → removed locally
- [ ] Sync state tracked in sync_items table
- [ ] `gog sync start` runs sync loop (blocking)

**Validation:**
```bash
# Manual integration test
./bin/gog sync init /tmp/sync-test --drive-folder "Sync Test"
./bin/gog sync start &
echo "test" > /tmp/sync-test/hello.txt
# Verify file appears in Drive within 5 seconds
```

**Files:**
- `internal/sync/engine.go` (new)
- `internal/sync/upload.go` (new)
- `internal/sync/download.go` (new)

---

### Group 7: Conflict Resolution
**Goal:** Handle simultaneous edits safely

**Deliverables:**
- `internal/sync/conflict.go` - Conflict detection and resolution
- Rename strategy (default): file.conflict-YYYY-MM-DD.ext
- Local-wins and remote-wins strategies
- `--conflict` flag for sync start

**Acceptance Criteria:**
- [ ] Conflict detected when both local and remote modified since last sync
- [ ] Rename strategy creates .conflict file, keeps both versions
- [ ] Local-wins uploads local version, overwrites remote
- [ ] Remote-wins downloads remote version, overwrites local
- [ ] Conflicts logged to sync_log table

**Validation:**
```bash
# Conflict test
# 1. Create file, sync
# 2. Modify locally AND via Drive web simultaneously
# 3. Verify conflict file created
```

**Files:**
- `internal/sync/conflict.go` (new)
- `internal/sync/conflict_test.go` (new)

---

### Group 8: Daemon Mode
**Goal:** Sync runs in background

**Deliverables:**
- `internal/sync/daemon.go` - Background process management
- PID file at `~/.config/gog/sync.pid`
- Log file at `~/.config/gog/sync.log`
- `gog sync start --daemon`
- `gog sync stop`
- Graceful shutdown on SIGTERM

**Acceptance Criteria:**
- [ ] `gog sync start --daemon` backgrounds process, returns immediately
- [ ] PID file created, contains valid PID
- [ ] Logs written to sync.log
- [ ] `gog sync stop` sends SIGTERM, process exits cleanly
- [ ] `gog sync status` shows daemon running/stopped
- [ ] Daemon survives terminal close

**Validation:**
```bash
./bin/gog sync start --daemon
cat ~/.config/gog/sync.pid  # expect PID
./bin/gog sync status --json  # expect running: true
./bin/gog sync stop
./bin/gog sync status --json  # expect running: false
```

**Files:**
- `internal/sync/daemon.go` (new)
- `internal/cmd/sync.go` (modify - add --daemon flag)

---

### Group 9: Documentation & Polish
**Goal:** Ready for users

**Deliverables:**
- `docs/headless-auth.md` - Headless OAuth guide
- `docs/sync.md` - Drive sync guide
- `docs/infrastructure.md` - GCP + callback server setup
- Update README.md with new features
- Update MILESTONES.md checkboxes

**Acceptance Criteria:**
- [ ] Headless auth documented end-to-end
- [ ] Sync usage documented with examples
- [ ] GCP project setup documented step-by-step
- [ ] Callback server deployment documented
- [ ] README updated with feature overview

**Validation:**
```bash
# Docs exist and are non-empty
test -s docs/headless-auth.md
test -s docs/sync.md
test -s docs/infrastructure.md
```

**Files:**
- `docs/headless-auth.md` (new)
- `docs/sync.md` (new)
- `docs/infrastructure.md` (new)
- `README.md` (modify)
- `MILESTONES.md` (modify)

---

## Dependencies

```
Group 1 (Callback Server) ─────┐
                               ├──► Group 2 (CLI Headless OAuth)
                               │
Group 3 (Sync Foundation) ─────┼──► Group 4 (Watcher) ──┐
                               │                        │
                               └──► Group 5 (Remote) ───┼──► Group 6 (Engine) ──► Group 7 (Conflict) ──► Group 8 (Daemon)
                                                        │
                                                        └──► Group 9 (Docs) [can start after Group 2]
```

**Parallelizable:**
- Groups 1 and 3 can run in parallel
- Groups 4 and 5 can run in parallel (after Group 3)
- Group 9 can start after Group 2, run alongside sync work

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| OAuth callback timing issues | Medium | Medium | Generous TTL (15 min), clear error messages |
| fsnotify misses events | Low | High | Periodic full rescan as fallback |
| SQLite locking under load | Low | Medium | Use WAL mode, single writer |
| Quota exceeded | Low | High | Monitor usage, request increase proactively |
| VPN requirement limits testing | Medium | Low | Document ngrok workaround for dev |

---

## Notes

- Keep upstream compatibility: don't break existing `gog auth add` browser flow
- Build-time injection (`-ldflags`) is fork-specific, don't PR that part upstream
- Headless OAuth core is PR-able to upstream (generic, useful for others)
- Sync feature could be PR-able if made generic enough
