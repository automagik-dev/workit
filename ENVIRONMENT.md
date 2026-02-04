# ENVIRONMENT.md

## Machine
- **OS:** Linux (genie-os)
- **Shell:** zsh
- **Go:** Check with `go version`

## Paths
- **Workspace:** `/home/genie/repos/gog-cli`
- **Binary output:** `./bin/gog`
- **Config dir:** `~/.config/gog/`

## Git Remotes
- **origin:** `https://github.com/automagik-genie/gog-cli.git` (our fork)
- **upstream:** `https://github.com/steipete/gogcli.git` (original)

## Build Commands
```bash
make              # Build binary
make test         # Run tests
make lint         # Run linter
make ci           # Full CI check (fmt + lint + test)
make fmt          # Format code
```

## Key Directories
```
internal/
├── cmd/           # CLI command definitions
├── config/        # Configuration and credentials
├── googleauth/    # OAuth flow (modify for headless)
├── googleapi/     # API clients for Drive, Gmail, etc.
└── sync/          # NEW - Drive sync engine (to be created)

auth-server/       # NEW - OAuth callback server (to be created)
```

## Environment Variables (for testing)
```bash
export GOG_CLIENT_ID="your-client-id"
export GOG_CLIENT_SECRET="your-client-secret"
export GOG_CALLBACK_SERVER="https://your-callback.com"
export GOG_ACCOUNT="test@gmail.com"
```
