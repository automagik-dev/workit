# TOOLS.md

## Available Tools

### Build System
- `make` / `make build` - Build binary to `./bin/gog`
- `make test` - Run unit tests
- `make lint` - Run golangci-lint
- `make ci` - Full CI check (always run before push)
- `make fmt` - Format code (goimports + gofumpt)

### Git
- `origin` - Our fork (automagik-genie/gog-cli)
- `upstream` - Original repo (steipete/gogcli)
- Sync upstream: `git fetch upstream && git merge upstream/main`

### Testing
```bash
# Unit tests
go test ./...

# Specific package
go test ./internal/googleauth/...

# Integration (needs real credentials)
GOG_IT_ACCOUNT=test@gmail.com go test -tags=integration ./internal/integration
```

### Debugging
```bash
# Run with verbose output
./bin/gog --debug gmail labels list

# Check auth status
./bin/gog auth status

# List accounts
./bin/gog auth list --check
```

## Tool Notes
*(Add notes as you discover things)*

- Kong CLI framework - commands are structs with `Run(ctx)` methods
- OAuth flow in `internal/googleauth/oauth_flow.go`
- Credentials loaded via `internal/config/credentials.go`
