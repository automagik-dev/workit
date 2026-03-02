# Authentication & Secrets

> Back to [README](../README.md)

## Quick Start (Relay Auth — No GCP Setup)

`workit` ships with a built-in auth relay at `auth.automagik.dev`. You do **not** need a Google Cloud project, OAuth credentials, or any API keys to get started.

Just run:

```bash
wk auth manage
```

This opens an account manager UI. It binds to `0.0.0.0:8085`, detects and displays your outbound IP, and auto-closes after authentication completes — no localhost tunnels or manual port-forwarding required.

### How the relay flow works

1. `wk auth manage` starts a local HTTP server and opens the browser (or prints a URL)
2. You log in with your Google account
3. Google redirects to `auth.automagik.dev` (the default callback server)
4. The callback server holds your token for up to 15 minutes
5. The CLI polls, retrieves the token, stores it in your system keychain, and closes

```
┌─────────┐     ┌──────────┐     ┌─────────────────────┐     ┌────────┐
│   CLI   │────▶│  Google  │────▶│  auth.automagik.dev  │────▶│  CLI   │
│ (agent) │     │  OAuth   │     │  (callback server)   │     │ (poll) │
└─────────┘     └──────────┘     └─────────────────────┘     └────────┘
```

That's it. Your refresh token is stored securely in your system keychain and ready to use.

## Account Management

The recommended entry point for all auth operations:

```bash
wk auth manage
```

Other individual commands:

```bash
# Add a single account directly
wk auth add you@gmail.com

# Remove an account
wk auth remove you@gmail.com

# List stored accounts
wk auth list

# Verify tokens are usable (spots revoked/expired tokens)
wk auth list --check

# Show auth state and enabled services for the active account
wk auth status

# List available services and their scopes
wk auth services
```

Accounts can be authorized either via OAuth refresh tokens or Workspace service accounts (domain-wide delegation). If a service account key is configured for an account, it takes precedence over OAuth refresh tokens (see `wk auth list`).

## Multi-Account Usage

### Account Selection

Specify the account using a flag or environment variable:

```bash
# Via flag
wk gmail search 'newer_than:7d' --account you@gmail.com

# Via alias
wk auth alias set work work@company.com
wk gmail search 'newer_than:7d' --account work

# Via environment variable
export WK_ACCOUNT=you@gmail.com
wk gmail search 'newer_than:7d'

# Auto-select (default account or the single stored token)
wk gmail labels list --account auto
```

### Account Aliases

```bash
wk auth alias set work work@company.com
wk auth alias list
wk auth alias unset work
```

Aliases work anywhere you pass `--account` or `WK_ACCOUNT` (reserved names: `auto`, `default`).

## Headless / Agent Auth

For servers, CI, or AI agents where no browser is available:

```bash
# Recommended: headless interactive flow (binds to 0.0.0.0, shows outbound IP)
wk auth manage

# Agent-driven: emit the URL as JSON for programmatic handling
wk auth manage --print-url
# Output: {"url":"http://203.0.113.42:8085","port":8085}
```

### Direct headless add (non-interactive)

```bash
# Start headless auth — polls until complete
wk auth add you@gmail.com --headless --no-input

# Start headless auth and print JSON (url + state + poll_url)
wk auth add you@gmail.com --headless --json

# Start headless auth without polling (async — poll later)
wk auth add you@gmail.com --headless --no-poll
wk auth poll abc123xyz

# Extend poll timeout
wk auth add you@gmail.com --headless --poll-timeout=10m
```

### Manual fallback (paste-URL flow)

```bash
wk auth add you@gmail.com --services user --manual
```

The CLI prints an auth URL. Open it in a local browser, then paste the full loopback redirect URL back when prompted.

### Split remote flow

Useful for two-step/scripted handoff:

```bash
# Step 1: print auth URL (open locally in a browser)
wk auth add you@gmail.com --services user --remote --step 1

# Step 2: paste the full redirect URL from your browser address bar
wk auth add you@gmail.com --services user --remote --step 2 --auth-url 'http://127.0.0.1:<port>/oauth2/callback?code=...&state=...'
```

The `state` is cached on disk for ~10 minutes. If it expires, rerun step 1.

See [docs/headless-auth.md](headless-auth.md) for full details.

## Token Management

```bash
# List stored tokens
wk auth tokens list

# Export tokens (for backup or migration)
wk auth tokens export

# Import tokens
wk auth tokens import

# Delete a token
wk auth tokens delete you@gmail.com
```

## Keyring Backends

`wk` stores OAuth refresh tokens in a "keyring" backend. The default is `auto` (best available for your OS/environment).

Available backends:

- `auto` (default): picks the best backend for the platform.
- `keychain`: macOS Keychain (recommended on macOS; avoids password management).
- `file`: encrypted on-disk keyring (requires a password).

```bash
# Set backend
wk auth keyring file
wk auth keyring keychain
wk auth keyring auto

# Show current backend + source and config path
wk auth keyring
```

Force backend via env (overrides config):

```bash
export WK_KEYRING_BACKEND=file
```

### Linux Headless / CI Auto-Setup

On Linux headless environments (servers, containers, WSL without a desktop), `wk` automatically configures the `file` keyring backend and sets up an encryption password — no manual `WK_KEYRING_PASSWORD` required for typical use:

```bash
wk auth manage
```

For fully non-interactive CI runs where you want to supply the password explicitly:

```bash
export WK_KEYRING_PASSWORD='...'
wk --no-input auth status
```

### Keychain Prompts (macOS)

macOS Keychain may prompt more than expected when the app identity keeps changing (different binary path, `go run` temp builds, rebuilding to a new `./bin/wk`, multiple copies). Keychain treats those as different apps.

Options:

- **Default (recommended):** keep using Keychain and run a stable `wk` binary path to reduce repeat prompts.
- **Force Keychain:** `WK_KEYRING_BACKEND=keychain` (disables file-backend fallback).
- **Avoid Keychain prompts entirely:** `WK_KEYRING_BACKEND=file` (stores encrypted entries on disk under your config dir). For non-interactive/CI use: `WK_KEYRING_PASSWORD=...` (tradeoff: secret in env).

### Credential Storage Locations

OAuth credentials are stored securely in your system's keychain:

- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

The CLI uses [github.com/99designs/keyring](https://github.com/99designs/keyring) for secure storage.

If no OS keychain backend is available (e.g., Linux/WSL/container), keyring falls back to an encrypted on-disk store and may prompt for a password; for non-interactive runs set `WK_KEYRING_PASSWORD`.

## Advanced: BYO GCP Credentials (Optional)

> This section is for advanced users who want to use their own Google Cloud project and OAuth client instead of the default relay. Most users do not need this.

### Create OAuth Credentials (Google Cloud Console)

1. Open the [Google Cloud Console credentials page](https://console.cloud.google.com/apis/credentials)
2. [Create a project](https://console.cloud.google.com/projectcreate)
3. Enable the APIs you need:
   - [Gmail API](https://console.cloud.google.com/apis/api/gmail.googleapis.com)
   - [Google Calendar API](https://console.cloud.google.com/apis/api/calendar-json.googleapis.com)
   - [Google Chat API](https://console.cloud.google.com/apis/api/chat.googleapis.com)
   - [Google Drive API](https://console.cloud.google.com/apis/api/drive.googleapis.com)
   - [Google Classroom API](https://console.cloud.google.com/apis/api/classroom.googleapis.com)
   - [People API (Contacts)](https://console.cloud.google.com/apis/api/people.googleapis.com)
   - [Google Tasks API](https://console.cloud.google.com/apis/api/tasks.googleapis.com)
   - [Google Sheets API](https://console.cloud.google.com/apis/api/sheets.googleapis.com)
   - [Google Forms API](https://console.cloud.google.com/apis/api/forms.googleapis.com)
   - [Apps Script API](https://console.cloud.google.com/apis/api/script.googleapis.com)
   - [Cloud Identity API (Groups)](https://console.cloud.google.com/apis/api/cloudidentity.googleapis.com)
4. [Configure OAuth consent screen](https://console.cloud.google.com/auth/branding)
5. If your app is in "Testing", [add test users](https://console.cloud.google.com/auth/audience)
6. Create OAuth client:
   - Go to https://console.cloud.google.com/auth/clients
   - Click "Create Client"
   - Application type: "Desktop app"
   - Download the JSON file (usually named `client_secret_....apps.googleusercontent.com.json`)

### Store Your Credentials

```bash
wk auth credentials ~/Downloads/client_secret_....json
```

For multiple OAuth clients/projects:

```bash
wk --client work auth credentials ~/Downloads/work-client.json
wk auth credentials list
```

### Multiple OAuth Clients

Use `--client` (or `WK_CLIENT`) to select a named OAuth client:

```bash
wk --client work auth credentials ~/Downloads/work.json
wk --client work auth add you@company.com
```

Optional domain mapping for auto-selection:

```bash
wk --client work auth credentials ~/Downloads/work.json --domain example.com
```

How it works:

- Default client is `default` (stored in `credentials.json`).
- Named clients are stored as `credentials-<client>.json`.
- Tokens are isolated per client (`token:<client>:<email>`); defaults are per client too.

Client selection order (when `--client` is not set):

1. `--client` / `WK_CLIENT`
2. `account_clients` config (email -> client)
3. `client_domains` config (domain -> client)
4. Credentials file named after the email domain (`credentials-example.com.json`)
5. `default`

Config example (JSON5):

```json5
{
  account_clients: { "you@company.com": "work" },
  client_domains: { "example.com": "work" },
}
```

List stored credentials:

```bash
wk auth credentials list
```

See [docs/auth-clients.md](auth-clients.md) for the full client selection and mapping rules.

### Overriding the Callback Server

The callback server URL (`auth.automagik.dev`) can be overridden in order of precedence:

1. **Flag**: `--callback-server=https://your-server.example.com`
2. **Environment**: `WK_CALLBACK_SERVER=https://your-server.example.com`
3. **Build-time default**: Compiled into binary with `-ldflags`

## Service Accounts (Workspace Only)

A service account is a non-human Google identity that belongs to a Google Cloud project. In Google Workspace, a service account can impersonate a user via **domain-wide delegation** (admin-controlled) and access APIs like Gmail/Calendar/Drive as that user.

In `wk`, service accounts are an **optional auth method** that can be configured per account email. If a service account key is configured for an account, it takes precedence over OAuth refresh tokens (see `wk auth list`).

### 1) Create a Service Account (Google Cloud)

1. Create (or pick) a Google Cloud project.
2. Enable the APIs you'll use (e.g. Gmail, Calendar, Drive, Sheets, Docs, People, Tasks, Cloud Identity).
3. Go to **IAM & Admin -> Service Accounts** and create a service account.
4. In the service account details, enable **Domain-wide delegation**.
5. Create a key (**Keys -> Add key -> Create new key -> JSON**) and download the JSON key file.

### 2) Allowlist Scopes (Google Workspace Admin Console)

Domain-wide delegation is enforced by Workspace admin settings.

1. Open **Admin console -> Security -> API controls -> Domain-wide delegation**.
2. Add a new API client:
   - Client ID: use the service account's "Client ID" from Google Cloud.
   - OAuth scopes: comma-separated list of scopes you want to allow (copy from `wk auth services` and/or your `wk auth add --services ...` usage).

If a scope is missing from the allowlist, service-account token minting can fail (or API calls will 403 with insufficient permissions).

### 3) Configure `wk` to Use the Service Account

Store the key for the user you want to impersonate:

```bash
wk auth service-account set you@yourdomain.com --key ~/Downloads/service-account.json
```

Verify `wk` is preferring the service account for that account:

```bash
wk --account you@yourdomain.com auth status
wk auth list
```

Remove the service account config:

```bash
wk auth service-account unset you@yourdomain.com

# Check current status
wk auth service-account status you@yourdomain.com
```

## Google Keep (Workspace Only)

Keep requires Workspace + domain-wide delegation. Configure it via the service-account command:

```bash
wk auth service-account set you@yourdomain.com --key ~/Downloads/service-account.json
wk keep list --account you@yourdomain.com
wk keep get <noteId> --account you@yourdomain.com
```

## Environment Variables

| Variable | Description |
|---|---|
| `WK_ACCOUNT` | Default account email or alias to use (avoids repeating `--account`; otherwise uses keyring default or a single stored token) |
| `WK_CLIENT` | OAuth client name (selects stored credentials + token bucket) |
| `WK_KEYRING_BACKEND` | Force keyring backend: `auto`, `keychain`, or `file` (overrides config) |
| `WK_KEYRING_PASSWORD` | Password for the encrypted on-disk keyring (file backend; avoids interactive prompt) |
| `WK_CALLBACK_SERVER` | Override the relay callback server URL (default: `https://auth.automagik.dev`) |

See also [docs/configuration.md](configuration.md) for the full environment variables reference.

## Best Practices

- **Never commit OAuth client credentials** to version control.
- Store client credentials outside your project directory.
- Use different OAuth clients for development and production.
- Re-authorize with `--force-consent` if you suspect token compromise.
- Remove unused accounts with `wk auth remove <email>`.

## Note on OAuth Client IDs in Open Source

Some open source Google CLIs ship a pre-configured OAuth client ID/secret copied from other desktop apps to avoid OAuth consent verification, testing-user limits, or quota issues. This makes the consent screen/security emails show the other app's name and can stop working at any time.

`workit` does not do this. Supported auth methods:

- Default relay via `auth.automagik.dev` (no GCP setup required — just run `wk auth manage`)
- Your own OAuth Desktop client JSON via `wk auth credentials ...` + `wk auth add ...`
- Google Workspace service accounts with domain-wide delegation (Workspace only)
