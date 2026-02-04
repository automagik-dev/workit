# CLAUDE.md - gog-cli

## Project Summary

**gog-cli** is a fork of [steipete/gogcli](https://github.com/steipete/gogcli) enhanced for AI agent use cases. It provides CLI access to Google Workspace (Gmail, Drive, Calendar, etc.) with two key additions:

1. **Headless OAuth** - Mobile-friendly auth flow for remote users
2. **Real-time Folder Sync** - Bidirectional Drive sync like Google Drive for Desktop

## Quick Context

- **Language**: Go
- **Upstream**: github.com/steipete/gogcli (keep in sync)
- **Target users**: AI agents operating on behalf of mobile users
- **Auth flow**: Agent infra → OAuth URL → User on mobile → Callback server → Token back to CLI

## Key Files

| Path | Purpose |
|------|---------|
| `cmd/gog/main.go` | CLI entrypoint |
| `internal/cmd/auth.go` | Auth subcommands (modify for --headless) |
| `internal/googleauth/oauth_flow.go` | OAuth flow logic (modify for headless) |
| `internal/config/credentials.go` | Credential loading (add defaults fallback) |
| `internal/sync/` | NEW - Sync engine |
| `auth-server/` | NEW - OAuth callback server |
| `AGENT.md` | Full implementation spec |
| `MILESTONES.md` | Progress tracking |

## Build Commands

```bash
# Standard build (same as upstream)
make

# Internal build with embedded credentials
make build-internal \
  GOG_CLIENT_ID=xxx \
  GOG_CLIENT_SECRET=xxx \
  GOG_CALLBACK_SERVER=https://auth.example.com

# Run tests
make test

# Lint
make lint

# Full CI check
make ci
```

## Current State

Check git log and compare with MILESTONES.md to see what's done.

## Code Style

Follow upstream conventions (see AGENTS.md):
- `make fmt` before committing (goimports + gofumpt)
- Conventional Commits: `feat(auth): add headless flow`
- stdout = parseable output, stderr = human hints
- JSON mode for all commands

## Key Patterns in Codebase

### OAuth Flow (existing)
Located in `internal/googleauth/oauth_flow.go`:
- `Authorize()` function handles full OAuth
- Already has `--manual` mode (user pastes redirect URL)
- Starts local HTTP server on random port for callback

### Credentials (existing)  
Located in `internal/config/credentials.go`:
- `ReadClientCredentialsFor(client)` loads from file
- Need to add fallback to compiled defaults + env vars

### CLI Commands (existing)
Located in `internal/cmd/`:
- Kong-style CLI with struct tags
- Each command is a struct with `Run(ctx)` method

## Headless OAuth Flow

```bash
gog auth add user@email.com --headless --json
# Returns auth_url for user to visit
# CLI polls callback server until token received
```

## Sync Commands

```bash
gog sync init ~/folder --drive-folder "Folder Name"
gog sync start --daemon
gog sync status --json
```

## Dependencies to Add

```go
// go.mod
require (
    github.com/fsnotify/fsnotify v1.7+
    github.com/mattn/go-sqlite3 v1.14+
    github.com/redis/go-redis/v9 v9.0+  // auth-server only
)
```

## Testing

```bash
# Unit tests
go test ./...

# Integration (needs credentials)
GOG_IT_ACCOUNT=test@gmail.com go test -tags=integration ./internal/integration
```

## Upstream Sync

```bash
git fetch upstream
git merge upstream/main
# Resolve conflicts, keeping our additions
```

## PR to Upstream

Keep changes modular. See AGENT.md "PR Strategy" section.
Headless auth is generic enough to upstream; sync may be too opinionated.
