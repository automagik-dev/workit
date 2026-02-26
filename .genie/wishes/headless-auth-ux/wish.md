# Wish: Auto-Detect Headless Auth + Callback Server Deployment

- **Status:** ready
- **Slug:** headless-auth-ux
- **Date:** 2026-02-17
- **Design:** `.genie/brainstorms/headless-auth-ux/design.md`

## Problem

The `gog auth add` command requires `--headless --callback-server URL --force-consent` for agent use — 3 extra flags just to authenticate. For an agent-first CLI, `gog auth add email --services all` should auto-detect headless mode when running without a TTY and a callback server is configured.

Additionally, the callback server at `auth.example.com` is down (502) and local browser auth is broken because it uses random ports with `127.0.0.1` (incompatible with Web application OAuth clients).

## Scope

### IN
- Two new config keys: `callback_server` (URL), `auth_mode` (enum)
- `ResolveAuthMode()` auto-detect: explicit flags > config auth_mode > TTY fallback
- `CallbackServerURL()` precedence: flag > env > **config** > build-default
- Fixed local auth port `8085` (`DefaultLocalAuthPort` constant)
- `localhost` instead of `127.0.0.1` in all redirect URIs
- Auth-server `--credentials-file` flag for loading OAuth creds from disk
- PM2 deployment to `/opt/gog-auth-server/` with `ecosystem.config.js`
- `auth status` output shows `auth_mode` + `callback_server`
- Unit tests for all new code paths

### OUT
- Changing OAuth client type (stays Web application)
- New OAuth flows beyond existing headless/browser/manual
- Modifying the auth-server token store (stays in-memory)

## Key Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Deploy to `/opt/gog-auth-server/` via PM2 | PM2 is the official service deployment standard; systemd only used to keep PM2 reboot-resilient; better health/log management |
| D2 | `--credentials-file` reads `~/.config/gogcli/credentials.json` | Zero duplication — same file gog manages; also works with env vars for PM2 `EnvironmentFile` |
| D3 | Layered auto-detect: flags > config > TTY fallback | Backward-compatible, explicit always wins |
| D4 | Single PR | All changes ship together |
| D5 | Fixed port `8085` as `DefaultLocalAuthPort` | Google OAuth Web clients need exact URI match for non-localhost |
| D6 | `localhost` not `127.0.0.1` | Google treats `localhost` specially (wildcard ports for native apps) |

## Success Criteria

- [ ] `gog config set callback_server https://auth.example.com` saves to config
- [ ] `gog config set auth_mode headless` saves to config; rejects invalid values
- [ ] `gog config list --json` shows `callback_server` and `auth_mode` keys
- [ ] Auth-server accepts `--credentials-file` flag and reads creds from JSON
- [ ] Auth-server running via PM2 at `/opt/gog-auth-server/` — `curl localhost:8089/health` returns 200
- [ ] `pm2 list` shows `gog-auth-server` as `online`
- [ ] `curl -s https://auth.example.com/health | jq -r .status` returns `ok` (full reverse proxy chain)
- [ ] `gog auth add email --services all` auto-detects headless when no TTY + callback server configured
- [ ] `--force-consent` auto-applied in headless mode (no manual flag needed)
- [ ] Local browser auth uses `http://localhost:8085/oauth2/callback` (fixed port, localhost)
- [ ] `gog auth status --json` includes `auth_mode` and `callback_server` fields
- [ ] `make ci` passes (fmt + lint + test)

---

## Execution Groups

### Group 1: Config Keys (`callback_server`, `auth_mode`)

**Deliverables:**
1. Add `CallbackServer string` and `AuthMode string` fields to `File` struct in `internal/config/config.go`
2. Add `KeyCallbackServer` and `KeyAuthMode` constants + `KeySpec` entries in `internal/config/keys.go`
   - `KeyCallbackServer`: validate URL starts with `http://` or `https://`, `EmptyHint: "(not set)"`
   - `KeyAuthMode`: validate value is one of `auto`, `browser`, `headless`, `manual`, `EmptyHint: "(not set, using auto)"`
3. Add both to `keyOrder` slice (after `KeyKeyringBackend`)
4. Add unit tests in `internal/config/keys_test.go` for validation (valid values, invalid values, get/set/unset)

**Acceptance Criteria:**
- `gog config set callback_server https://auth.example.com` → writes to config.json
- `gog config set callback_server ftp://bad` → error: must start with http/https
- `gog config set auth_mode headless` → writes to config.json
- `gog config set auth_mode invalid` → error: must be auto/browser/headless/manual
- `gog config list` → shows both keys with values or hints

**Validation:**
```bash
cd /home/genie/workspace/repos/gog-cli && go test ./internal/config/...
```

---

### Group 2: CallbackServerURL Precedence + ResolveAuthMode

**Deliverables:**
1. Update `CallbackServerURL()` in `internal/googleauth/headless.go` — insert config read between env and build-default:
   ```
   flag → env GOG_CALLBACK_SERVER → config.ReadConfig().CallbackServer → build-time DefaultCallbackServer → error
   ```
2. Create `internal/googleauth/detect.go` with:
   - `AuthModeResult` struct: `Mode string` (browser/headless/manual), `Source string` (flag/config/auto), `CallbackServer string`
   - `ResolveAuthMode(explicitHeadless, explicitManual bool, callbackServerFlag string) AuthModeResult`:
     1. `explicitHeadless=true` → headless
     2. `explicitManual=true` → manual
     3. Config `auth_mode=browser` → browser
     4. Config `auth_mode=headless` → headless (if callback server resolvable)
     5. Config `auth_mode=manual` → manual
     6. Default (auto): no TTY (`!term.IsTerminal(int(os.Stdin.Fd()))`) AND callback server resolvable AND `callbackServerReachable()` → headless; else browser
   - `callbackServerReachable(url string) bool`: HTTP GET `{url}/health` with 3-second timeout, returns true on 200
3. Create `internal/googleauth/detect_test.go` — test all branches:
   - Explicit flags override everything
   - Config auth_mode values
   - Auto-detect with/without TTY (mock `os.Stdin.Fd()` or use test helper)
   - callbackServerReachable with httptest server
4. Update `internal/googleauth/headless_test.go` — test new config precedence in CallbackServerURL

**Acceptance Criteria:**
- `CallbackServerURL("")` with config `callback_server` set → returns config value
- `CallbackServerURL("")` with env set → env wins over config
- `ResolveAuthMode(true, false, "")` → headless (flag wins)
- `ResolveAuthMode(false, false, "")` with config `auth_mode=headless` → headless
- `ResolveAuthMode(false, false, "")` with no config, no TTY, reachable callback → headless (auto-detect)
- `ResolveAuthMode(false, false, "")` with TTY → browser

**Validation:**
```bash
cd /home/genie/workspace/repos/gog-cli && go test ./internal/googleauth/...
```

---

### Group 3: Wire Auth Mode into CLI + Fixed Local Port

**Deliverables:**
1. Modify `internal/cmd/auth.go` line ~545:
   - Replace `if c.Headless { ... }` with `ResolveAuthMode()` call
   - When resolved mode is headless: auto-set `c.ForceConsent = true`, call `runHeadless()`
   - When resolved mode is manual: fall through to existing manual flow
   - When resolved mode is browser: fall through to existing browser flow
   - On browser auth failure + callback server configured: print stderr hint
2. Modify `internal/googleauth/oauth_flow.go` line 114:
   - Change `"127.0.0.1:0"` → `"localhost:8085"`
   - Change redirect URI format from `http://127.0.0.1:%d/oauth2/callback` → `"http://localhost:8085/oauth2/callback"` (constant, no port formatting)
   - Add `DefaultLocalAuthPort = 8085` constant
3. Modify `internal/googleauth/accounts_server.go` line 102:
   - Change `"127.0.0.1:0"` → `"localhost:8085"`
   - Update redirect URI similarly
4. Update `AuthStatusCmd.Run()` in `internal/cmd/auth.go` to include `auth_mode` and `callback_server` in JSON output

**Acceptance Criteria:**
- `gog auth add email --services all` in TTY → browser flow (unchanged behavior)
- `gog auth add email --services all --headless --callback-server URL` → headless (explicit flag, unchanged)
- With config `auth_mode=headless` + `callback_server` set: `gog auth add email --services all` → headless + force-consent auto-applied
- Local browser auth starts on port 8085 with `localhost` hostname
- `gog auth status --json` includes `auth_mode` and `callback_server`

**Validation:**
```bash
cd /home/genie/workspace/repos/gog-cli && go test ./internal/cmd/... && go test ./internal/googleauth/...
```

---

### Group 4: Auth-Server `--credentials-file` + PM2 Deployment

**Deliverables:**
1. Modify `auth-server/main.go`:
   - Add `--credentials-file` flag (string, default empty)
   - When set: read JSON file, extract `installed.client_id` and `installed.client_secret` (or `web.client_id`/`web.client_secret`)
   - Credentials file values override `--client-id`/`--client-secret` flags and env vars
   - Log: `"Loaded credentials from {path}"`
2. Create `auth-server/ecosystem.config.js`:
   ```js
   module.exports = {
     apps: [{
       name: 'gog-auth-server',
       script: './auth-server',
       args: '--port 8089 --credentials-file ~/.config/workit/credentials.json --redirect-url https://auth.example.com/callback',
       cwd: '/opt/gog-auth-server',
       interpreter: 'none',
       autorestart: true,
       max_memory_restart: '100M',
     }]
   };
   ```
3. Create `auth-server/deploy.sh`:
   - Build: `cd auth-server && CGO_ENABLED=0 go build -ldflags="-w -s" -o auth-server .`
   - Deploy: `sudo mkdir -p /opt/gog-auth-server && sudo cp auth-server ecosystem.config.js /opt/gog-auth-server/`
   - Start: `cd /opt/gog-auth-server && pm2 start ecosystem.config.js && pm2 save`
   - Health check: `curl -sf http://localhost:8089/health`

**Acceptance Criteria:**
- `./auth-server --credentials-file /path/to/credentials.json --port 8089` starts and serves `/health`
- `pm2 list` shows `gog-auth-server` as `online`
- `curl localhost:8089/health` returns 200
- `curl -s https://auth.example.com/health | jq -r .status` returns `ok` (full reverse proxy chain)
- Server survives restart (`pm2 restart gog-auth-server`)

**Validation:**
```bash
cd /home/genie/workspace/repos/gog-cli/auth-server && go build -o /tmp/auth-server-test . && echo "build ok"
```

**Rollback:**
```bash
pm2 stop gog-auth-server && pm2 delete gog-auth-server && pm2 save
sudo rm -rf /opt/gog-auth-server
```
No data at risk — auth-server is stateless (in-memory token store).

---

### Group 5: Full CI Gate + Integration Smoke

**Deliverables:**
1. Run `make ci` — all existing + new tests pass, no lint errors
2. Verify `gog config set callback_server https://auth.example.com` works
3. Verify `gog config set auth_mode headless` works
4. Verify `gog config list --json` shows new keys
5. Verify `gog auth status --json` shows new fields

**Depends-on:** Groups 1-4

**Acceptance Criteria:**
- `make ci` exits 0
- Config keys read/write correctly end-to-end
- Auth status reflects new configuration

**Validation:**
```bash
cd /home/genie/workspace/repos/gog-cli && make ci
```
