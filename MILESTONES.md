# gog-cli Development Milestones

## Milestone 1: Repository Setup ✅
- [x] Create repo structure
- [x] Write AGENT.md (implementation spec)
- [x] Write TODO.md (infrastructure guide)
- [x] Fork upstream gogcli
- [x] Set up OpenClaw agent structure (SOUL, ROLE, IDENTITY, MEMORY, etc.)
- [ ] Set up CI (GitHub Actions)
- [ ] Verify build works

## Milestone 2: Auth Callback Server
- [ ] Create `auth-server/` directory
- [ ] Implement `/callback` handler (receives OAuth code)
- [ ] Implement `/token/{state}` endpoint (CLI polls this)
- [ ] Implement `/status/{state}` endpoint (check without consuming)
- [ ] Add Redis storage (with in-memory fallback)
- [ ] Add Dockerfile
- [ ] Add health check endpoint
- [ ] Test with ngrok on mobile
- [ ] Deploy to staging

## Milestone 3: Headless OAuth in CLI
- [ ] Add `--headless` flag to `AuthAddCmd` in `internal/cmd/auth.go`
- [ ] Add `--callback-server` flag/env var
- [ ] Modify `Authorize()` in `internal/googleauth/oauth_flow.go` for headless path
- [ ] Create `internal/googleauth/headless.go` for callback server polling
- [ ] Create `internal/config/defaults.go` for build-time injected values
- [ ] Modify `ReadClientCredentialsFor()` in `internal/config/credentials.go` to fallback to defaults
- [ ] Add `build-internal` target to Makefile
- [ ] Implement `gog auth poll <state>` command
- [ ] Update `gog auth status` to show token expiry
- [ ] Test full flow: CLI → WhatsApp → mobile → callback → token
- [ ] Document in `docs/headless-auth.md`

## Milestone 4: Sync Foundation
- [ ] Create `internal/sync/` package
- [ ] Implement SQLite state DB (`internal/sync/db.go`)
- [ ] Implement sync config management (`internal/sync/config.go`)
- [ ] Add `SyncCmd` struct to `internal/cmd/sync.go`
- [ ] Register sync command in `cmd/gog/main.go`
- [ ] Add `gog sync init` command
- [ ] Add `gog sync list` command
- [ ] Add `gog sync remove` command
- [ ] Add fsnotify and go-sqlite3 to go.mod
- [ ] Test DB operations

## Milestone 5: Local Filesystem Watching
- [ ] Add fsnotify dependency
- [ ] Implement `watcher.go`
- [ ] Add debouncing logic
- [ ] Add ignore patterns (.git, .gog-sync, temp files)
- [ ] Handle CREATE, WRITE, DELETE, RENAME events
- [ ] Test with rapid file changes

## Milestone 6: Remote Change Detection
- [ ] Implement `drive_changes.go`
- [ ] Use changes.list API with startPageToken
- [ ] Implement configurable poll interval
- [ ] Handle pagination
- [ ] Track change tokens in DB
- [ ] Test with remote changes

## Milestone 7: Sync Engine
- [ ] Implement `engine.go` main loop
- [ ] Implement upload logic (local → Drive)
- [ ] Implement download logic (Drive → local)
- [ ] Implement delete propagation (both directions)
- [ ] Calculate and compare MD5 hashes
- [ ] Handle folder creation
- [ ] Test bidirectional sync

## Milestone 8: Conflict Resolution
- [ ] Implement `conflict.go`
- [ ] Implement "rename" strategy (file.conflict-date.ext)
- [ ] Implement "local-wins" strategy
- [ ] Implement "remote-wins" strategy
- [ ] Add `--conflict` flag to sync start
- [ ] Log conflicts to sync_log table
- [ ] Test concurrent edits

## Milestone 9: Daemon Mode
- [ ] Implement `daemon.go`
- [ ] Implement PID file management
- [ ] Implement log file rotation
- [ ] Add `gog sync start --daemon`
- [ ] Add `gog sync stop`
- [ ] Add `gog sync status` (daemon status + sync stats)
- [ ] Handle graceful shutdown (SIGTERM)
- [ ] Test daemon lifecycle

## Milestone 10: Polish & Documentation
- [ ] Write `docs/sync.md`
- [ ] Update main README
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
