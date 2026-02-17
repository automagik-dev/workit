# Design: Auto-Detect Headless Auth + Callback Server Deployment

> Crystallized from brainstorm at WRS 100/100

## Problem

The `gog auth add` command requires `--headless --callback-server URL --force-consent` for agent use. Should just be `gog auth add email --services all`. The callback server at gogoauth.namastex.io is down (502). Local browser auth uses random ports and `127.0.0.1`, both broken with Web application OAuth clients.

## Scope

**IN:**
- Config keys (`callback_server`, `auth_mode`) with validation
- Auto-detect logic: explicit flags → config `auth_mode` → TTY fallback
- `CallbackServerURL()` precedence update (flag → env → config → build-default)
- Fixed local auth port `8085` as `DefaultLocalAuthPort`
- `localhost` instead of `127.0.0.1` for redirect URIs
- Auth-server `--credentials-file` flag
- PM2 deployment to `/opt/gog-auth-server/`
- Auth status output showing auth_mode + callback_server
- Unit tests for all new code paths

**OUT:**
- Changing OAuth client type
- New OAuth flows beyond existing ones

## Architecture

### Auto-Detect Flow (D3)

```
ResolveAuthMode(explicitHeadless, explicitManual bool, callbackServerFlag string) → AuthModeResult

1. Explicit --headless flag     → headless
2. Explicit --manual flag       → manual
3. Config auth_mode=browser     → browser
4. Config auth_mode=headless    → headless (if callback server resolvable)
5. Config auth_mode=manual      → manual
6. Config auth_mode=auto (default when unset):
   - No TTY AND callback_server configured AND /health responds → headless
   - Otherwise → browser
```

When headless mode is resolved, `--force-consent` is auto-applied (headless always needs a refresh token).

### CallbackServerURL Precedence (updated)

```
flag → env GOG_CALLBACK_SERVER → config callback_server → build-time DefaultCallbackServer → error
```

### Local Auth Port (D5)

- Fixed port `8085` as `DefaultLocalAuthPort` constant
- `127.0.0.1:0` → `localhost:8085` in oauth_flow.go and accounts_server.go
- All redirect URIs: `http://localhost:8085/oauth2/callback`

### Google Console Redirect URIs (D6)

- `https://gogoauth.namastex.io/callback` — headless (already registered)
- `http://localhost:8085/oauth2/callback` — local browser auth (adding now)

### Auth-Server Deployment (D1, D2)

- Binary deployed to `/opt/gog-auth-server/` via PM2
- PM2 `ecosystem.config.js` with `interpreter: 'none'` for Go binary
- `--credentials-file` flag reads `~/.config/gogcli/credentials.json`
- Listens on port `8089`; Caddy on CT 118 proxies `gogoauth.namastex.io` → CT 111:8089

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| R1: Port 8085 conflict | Local auth fails | Error with "port 8085 in use" message, suggest killing the process |
| R2: Auth-server in-memory storage | Server restart loses in-flight auth flows | Acceptable — auth flows are short-lived (< 5 min) |
| R3: Health check latency | 3-second /health ping adds latency | Only triggers when no TTY; interactive users unaffected |

## Acceptance Criteria

1. `gog config set callback_server https://gogoauth.namastex.io` → saves to config
2. `gog config set auth_mode headless` → saves to config
3. Auth-server running via PM2 at /opt/gog-auth-server/ → `curl localhost:8089/health` returns 200; `https://gogoauth.namastex.io/health` returns `ok`
4. `gog auth add felipe@namastex.ai --services all` → auto-detects headless → prints URL → polls → stores token
5. `gog people me --json` → returns profile data using new token
6. Local browser auth uses `http://localhost:8085/oauth2/callback` (fixed port, localhost not 127.0.0.1)

## Files to Create/Modify

| File | Action | Change |
|------|--------|--------|
| `internal/config/config.go` | Modify | Add `CallbackServer`, `AuthMode` fields to File struct |
| `internal/config/keys.go` | Modify | Add `KeyCallbackServer` (URL validation), `KeyAuthMode` (enum: auto/browser/headless/manual) |
| `internal/googleauth/headless.go` | Modify | Insert config read in `CallbackServerURL()` precedence chain |
| `internal/googleauth/detect.go` | **Create** | `ResolveAuthMode()` + `callbackServerReachable()` |
| `internal/googleauth/oauth_flow.go` | Modify | `127.0.0.1:0` → `localhost:8085`, fix redirect URI format |
| `internal/googleauth/accounts_server.go` | Modify | `127.0.0.1:0` → `localhost:8085` |
| `internal/cmd/auth.go` | Modify | Wire `ResolveAuthMode`, auto `--force-consent`, hint on failure |
| `auth-server/main.go` | Modify | Add `--credentials-file` flag |
| `auth-server/ecosystem.config.js` | **Create** | PM2 process config |
| `auth-server/deploy.sh` | **Create** | Build + deploy to /opt + PM2 start |
| `internal/googleauth/detect_test.go` | **Create** | Test all ResolveAuthMode branches |
| `internal/config/keys_test.go` | Modify | Test new key validation |
| `internal/googleauth/headless_test.go` | Modify | Test updated precedence chain |

## End-to-End UX After Implementation

```bash
# One-time setup:
gog config set callback_server https://gogoauth.namastex.io

# Then forever after, agents just run:
gog auth add felipe@namastex.ai --services all
# → auto-detects no TTY → pings callback server → uses headless → prints URL → polls → done

# Interactive users are unaffected:
gog auth add felipe@namastex.ai --services all
# → detects TTY → opens browser as before
```

## Verification

```bash
make ci                                    # all tests pass
gog config set callback_server https://gogoauth.namastex.io
gog config set auth_mode headless
gog config list --json                     # shows new keys
gog auth status --json                     # shows auth_mode + callback_server
curl http://localhost:8089/health           # auth server responds
# TTY: gog auth add email --services all   → browser flow (unchanged)
# No TTY: echo | gog auth add ...          → auto-headless
```
