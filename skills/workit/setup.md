# wk setup and authentication

Use this file for account setup, token lifecycle, and Workspace service-account flows.

## 1) Inspect auth state
- `wk auth status`
- `wk auth list`
- `wk auth services`

## 2) Login (interactive OAuth)
- **Recommended entry point:** `wk auth manage` — opens account manager UI, works on desktop and headless/remote servers (binds to 0.0.0.0, shows outbound IP, auto-closes after auth)
- Add account directly: `wk auth add <email>`
- Remove account: `wk auth remove <email>`

## 3) Multi-account workflows
- List accounts: `wk auth list`
- Per-command account selection: `wk -a user@company.com drive ls`
- Use aliases:
  - `wk auth alias set work user@company.com`
  - `wk auth alias list`
  - `wk auth alias unset work`

## 4) Headless OAuth flow
- **Recommended:** `wk auth manage` — binds to 0.0.0.0, shows outbound IP for remote access, auto-closes after auth
- For agents/automation (prints JSON with URL): `wk auth manage --print-url`
- Legacy: `wk auth add user@company.com --headless --no-input`
- Poll completion manually: `wk auth poll <state>`
- No manual keyring setup needed on Linux headless — auto-configured automatically

## 5) Token management
- List token keys: `wk auth tokens list`
- Export token (sensitive): `wk auth tokens export <key> --out token.json`
- Import token (sensitive): `wk auth tokens import <inPath>`
- Delete token: `wk auth tokens delete <key>`

## 6) OAuth client credentials
- List clients: `wk auth credentials list`
- Set client from credentials.json: `wk auth credentials set <credentials-json-path> [--domain example.com]`
- Select client on commands: `wk --client <name> gmail search 'in:inbox'`

## 7) Keyring backend
- Show/set backend: `wk auth keyring [backend]`
- Verify with: `wk auth status`

## 8) Service account (Workspace only)
- Store key for domain-wide delegation:
  - `wk auth service-account set --key /path/key.json <impersonate@company.com>`
- Check status:
  - `wk auth service-account status`
- Remove key:
  - `wk auth service-account unset`

## 9) Keep-specific service account (Workspace only)
- `wk auth keep --key /path/key.json admin@company.com`

## 10) Recommended auth pattern in agents
1. `wk auth status`
2. choose account (`-a`) and optional `--client`
3. run read checks with `--read-only`
4. run writes with `--dry-run`, then execute after confirmation
