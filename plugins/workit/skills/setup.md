# wk setup and authentication

## Quick Start (no GCP setup needed)

The `wk` binary ships with a shared OAuth client via `auth.automagik.dev`.
**No GCP console, no credentials.json, no client secrets required.**

Install:
```bash
# Linux amd64
curl -sSL https://github.com/automagik-dev/workit/releases/latest/download/workit_linux_amd64.tar.gz | tar xz -C ~/.local/bin
# macOS arm64
curl -sSL https://github.com/automagik-dev/workit/releases/latest/download/workit_darwin_arm64.tar.gz | tar xz -C ~/.local/bin
```

Check: `wk version` and `wk auth status`

---

## Auth flows by environment

### Desktop / laptop
```bash
wk auth manage   # opens browser, auto-closes after login
```

### Remote server / VPS (SSH headless)
```bash
wk auth manage   # detects no TTY, prints URL with server outbound IP
# Open printed URL in your browser — auth completes automatically
```

### Agent / automation (fully unattended)
```bash
wk auth add user@example.com --headless --no-input
# Prints a Google login URL. User (or automation) opens it.
# CLI polls auth.automagik.dev until token arrives, then stores it.
```

### Get just the URL (for scripting)
```bash
wk auth manage --print-url   # prints JSON: {"url":"https://...","state":"..."}
```

**Linux headless keyring:** auto-configured. No manual setup or `source` needed after v2.260227.4+.

---

## 1) Inspect auth state
```bash
wk auth status          # overall state + keyring backend
wk auth list            # all stored accounts
wk auth services        # services enabled per account
```

## 2) Add / remove accounts
```bash
wk auth manage                              # recommended: interactive account manager
wk auth add user@example.com               # direct add (browser opens)
wk auth add user@example.com --headless    # headless: prints URL, polls until done
wk auth remove user@example.com
```

## 3) Multi-account
```bash
wk auth list
wk -a user@company.com drive ls            # per-command account
wk auth alias set work user@company.com
wk auth alias list
wk auth alias unset work
```

## 4) Token management
```bash
wk auth tokens list
wk auth tokens export <key> --out token.json   # sensitive
wk auth tokens import <path>                    # sensitive
wk auth tokens delete <key>
```

## 5) OAuth client credentials (BYO GCP)
```bash
wk auth credentials list
wk auth credentials set credentials.json [--domain example.com]
wk --client <name> gmail search 'in:inbox'
```

## 6) Keyring backend
```bash
wk auth keyring           # show current backend
wk auth keyring <backend> # set backend (secret-service, keychain, file, etc.)
wk auth status            # verify
```

## 7) Service account (Workspace domain-wide delegation)
```bash
wk auth service-account set --key /path/key.json impersonate@company.com
wk auth service-account status
wk auth service-account unset
```

## 8) Recommended pattern in agents
1. `wk auth status` — check if account already exists
2. If not: `wk auth add user@example.com --headless --no-input` and surface the URL
3. `wk auth services` — verify services are authorized
4. Read operations: add `--read-only`
5. Write operations: `--dry-run` first, then without after confirmation
