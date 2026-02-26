# Brainstorm: Auto-Detect Headless Auth + Callback Server Deployment

## Problem
The `gog auth add` command requires `--headless --callback-server URL --force-consent` for agent use. Should just be `gog auth add email --services all`. The callback server at auth.example.com is down (502). Local browser auth uses random ports and `127.0.0.1`, both broken with Web application OAuth clients.

## Scope
- **IN**: config keys (callback_server, auth_mode), auto-detect logic, CallbackServerURL precedence, fixed local auth port (8085), localhost instead of 127.0.0.1, auth-server --credentials-file flag, PM2 deployment to /opt, auth status output, tests
- **OUT**: Changing OAuth client type, new OAuth flows beyond existing ones

## Decisions

### D1: Deploy target
Auth-server binary deployed to `/opt/gog-auth-server/`. PM2 ecosystem.config.js lives there. Decoupled from repo.

### D2: Credentials flow
Auth-server gets `--credentials-file` flag that reads `~/.config/gogcli/credentials.json` (same file gog manages). Zero duplication.

### D3: Auto-detect logic (layered)
1. Explicit flags (`--headless`, `--manual`) always win
2. Config `auth_mode` wins next (browser/headless/manual)
3. Fallback `auto`: if unset AND `callback_server` configured AND no TTY → headless
4. `--force-consent` auto-applied in headless mode

### D4: PR strategy
Single PR — everything ships together.

### D5: Fixed local auth port
Port `8085` hardcoded as `DefaultLocalAuthPort`. Change `127.0.0.1:0` → `localhost:8085` in:
- `internal/googleauth/oauth_flow.go:114`
- `internal/googleauth/accounts_server.go:102`
- All redirect URI formatting from `127.0.0.1:{port}` → `localhost:8085`

### D6: Google Console redirect URIs
- `https://auth.example.com/callback` ← headless (already registered)
- `http://localhost:8085/oauth2/callback` ← local browser auth (adding now)

## Risks

### R1: Port 8085 conflict
If something else is on 8085, local auth fails. Mitigation: fall back to random port with warning, or error with "port 8085 in use" message.

### R2: Auth-server in-memory storage
Tokens in-memory with 15min TTL. Server restart = lost flow. Acceptable (auth flows are short-lived).

### R3: Health check latency
3-second /health ping in auto-detect. Only when no TTY, so interactive users unaffected.

## Acceptance Criteria

1. `gog config set callback_server https://auth.example.com` → saves to config
2. `gog config set auth_mode headless` → saves to config
3. Auth-server running via PM2 at /opt/gog-auth-server/ → `curl localhost:8089/health` returns 200
4. `gog auth add user@example.com --services all` → auto-detects headless → prints URL → polls → stores token
5. `gog people me --json` → returns profile data using new token
6. Local browser auth uses `http://localhost:8085/oauth2/callback` (fixed port, localhost not 127.0.0.1)

## Files to Modify

| File | Change |
|------|--------|
| `internal/config/config.go` | Add CallbackServer, AuthMode fields |
| `internal/config/keys.go` | Add KeyCallbackServer, KeyAuthMode with specs |
| `internal/googleauth/headless.go` | Update CallbackServerURL() precedence (add config) |
| `internal/googleauth/detect.go` | **NEW** — ResolveAuthMode() + callbackServerReachable() |
| `internal/googleauth/oauth_flow.go` | `127.0.0.1:0` → `localhost:8085`, format URIs with localhost |
| `internal/googleauth/accounts_server.go` | Same: `127.0.0.1:0` → `localhost:8085` |
| `internal/cmd/auth.go` | Wire ResolveAuthMode, auto --force-consent, add hint on failure |
| `auth-server/main.go` | Add --credentials-file flag |
| `auth-server/ecosystem.config.js` | **NEW** — PM2 config |
| `auth-server/setup-pm2.sh` | **NEW** — build + deploy to /opt + PM2 start |
| Tests | detect_test.go, keys_test.go, headless_test.go, auth_add_test.go updates |
