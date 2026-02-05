# gog-cli Development Milestones

> **Vision**: "Google Drive for Linux + frictionless Google Workspace CLI"
>
> Full CLI access to Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks, Chat, Classroom, Keep, Groups - with zero-friction OAuth setup.

## Milestone 1: Repository Setup ✅
- [x] Create repo structure
- [x] Write AGENT.md (implementation spec)
- [x] Write TODO.md (infrastructure guide)
- [x] Fork upstream gogcli
- [x] Set up OpenClaw agent structure (SOUL, ROLE, IDENTITY, MEMORY, etc.)
- [x] Verify build works (upstream CI already covers this)
- [x] Brainstorm and lock architecture decisions

## Milestone 2: Auth Callback Server ✅
**Location:** `auth.namastex.io` (VPN-only Phase 1)

- [x] Create `auth-server/` directory
- [x] Implement `/callback` handler (receives OAuth code, exchanges for token)
- [x] Implement `/token/{state}` endpoint (CLI polls this, 15-min TTL)
- [x] Implement `/status/{state}` endpoint (check without consuming)
- [x] In-memory storage with TTL (Redis later if needed)
- [x] Add Dockerfile
- [x] Add health check endpoint
- [ ] Test with ngrok on mobile
- [ ] Deploy to auth.namastex.io

## Milestone 3: Headless OAuth in CLI ✅
- [x] Add `--headless` flag to `AuthAddCmd` in `internal/cmd/auth.go`
- [x] Add `--callback-server` flag/env var
- [x] Create `internal/googleauth/headless.go` for callback server polling
- [x] Create `internal/config/defaults.go` for build-time injected values
- [x] Modify `ReadClientCredentialsFor()` in `internal/config/credentials.go` to fallback to defaults
- [x] Implement `gog auth poll <state>` command
- [x] Document in `docs/headless-auth.md`
- [ ] Add `build-internal` target to Makefile
- [ ] Update `gog auth status` to show token expiry
- [ ] Test full flow: CLI → WhatsApp → mobile → callback → token

## Milestone 4: Sync Foundation ✅
- [x] Create `internal/sync/` package
- [x] Implement SQLite state DB (`internal/sync/db.go`)
- [x] Implement sync config management (`internal/sync/config.go`)
- [x] Add `SyncCmd` struct to `internal/cmd/sync.go`
- [x] Register sync command in `cmd/gog/main.go`
- [x] Add `gog sync init` command
- [x] Add `gog sync list` command
- [x] Add `gog sync remove` command
- [x] Add fsnotify and go-sqlite3 to go.mod
- [x] Test DB operations

## Milestone 5: Local Filesystem Watching ✅
- [x] Add fsnotify dependency
- [x] Implement `watcher.go`
- [x] Add debouncing logic
- [x] Add ignore patterns (.git, .gog-sync, temp files)
- [x] Handle CREATE, WRITE, DELETE, RENAME events
- [x] Test with rapid file changes

## Milestone 6: Remote Change Detection ✅
- [x] Implement `drive_changes.go`
- [x] Use changes.list API with startPageToken
- [x] **5-second poll interval** (feels instant, within quota)
- [x] Handle pagination
- [x] Track change tokens in DB
- [x] Test with remote changes

## Milestone 7: Sync Engine ✅
- [x] Implement `engine.go` main loop
- [x] Implement upload logic (local → Drive)
- [x] Implement download logic (Drive → local)
- [x] Implement delete propagation (both directions)
- [x] Calculate and compare MD5 hashes
- [x] Handle folder creation
- [ ] Test bidirectional sync (requires integration testing)

## Milestone 8: Conflict Resolution ✅
- [x] Implement `conflict.go`
- [x] Implement "rename" strategy (file.conflict-date.ext)
- [x] Implement "local-wins" strategy
- [x] Implement "remote-wins" strategy
- [x] Add `--conflict` flag to sync start
- [x] Log conflicts to sync_log table
- [ ] Test concurrent edits (requires integration testing)

## Milestone 9: Daemon Mode ✅
- [x] Implement `daemon.go`
- [x] Implement PID file management
- [x] Add `gog sync start --daemon`
- [x] Add `gog sync stop`
- [x] Add `gog sync status` (daemon status + sync stats)
- [x] Handle graceful shutdown (SIGTERM)
- [ ] Implement log file rotation
- [ ] Test daemon lifecycle

## Milestone 10: Polish & Documentation ✅
- [x] Write `docs/sync.md`
- [x] Write `docs/headless-auth.md`
- [x] Write `docs/infrastructure.md`
- [x] Update main README
- [ ] Add integration tests
- [ ] Performance testing with large folders
- [ ] Error handling review
- [ ] Logging consistency

## Milestone 11: Upstream PR Preparation
- [ ] Separate headless auth into clean PR
- [ ] Separate sync foundation into clean PR
- [ ] Ensure no company-specific code in PRs
- [ ] Add tests for PR code
- [ ] Follow upstream code style
- [ ] Open PRs

---

## Notes

- Always keep upstream compatibility: `git fetch upstream && git merge upstream/main`
- Build-time credential injection stays in our fork only
- Test on Linux (agent infra) primarily, but maintain macOS/Windows compat for sync
