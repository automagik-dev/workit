# Authentication & Secrets

> Back to [README](../README.md)

## OAuth Credentials Setup

Before adding an account, create OAuth2 credentials from Google Cloud Console:

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

## Store Credentials

```bash
wk auth credentials ~/Downloads/client_secret_....json
```

For multiple OAuth clients/projects:

```bash
wk --client work auth credentials ~/Downloads/work-client.json
wk auth credentials list
```

## Authorize Your Account

```bash
wk auth add you@gmail.com
```

This opens a browser window for OAuth authorization. The refresh token is stored securely in your system keychain.

## Accounts and Tokens

`wk` stores your OAuth refresh tokens in a "keyring" backend. Default is `auto` (best available backend for your OS/environment).

Before you can run `wk auth add`, you must store OAuth client credentials once via `wk auth credentials <credentials.json>` (download a Desktop app OAuth client JSON from the Cloud Console). For multiple clients, use `wk --client <name> auth credentials ...`; tokens are isolated per client.

List accounts:

```bash
wk auth list
```

Verify tokens are usable (helps spot revoked/expired tokens):

```bash
wk auth list --check
```

Accounts can be authorized either via OAuth refresh tokens or Workspace service accounts (domain-wide delegation). If a service account key is configured for an account, it takes precedence over OAuth refresh tokens (see `wk auth list`).

Show current auth state/services for the active account:

```bash
wk auth status
```

## Account Selection

Specify the account using either a flag or environment variable:

```bash
# Via flag
wk gmail search 'newer_than:7d' --account you@gmail.com

# Via alias
wk auth alias set work work@company.com
wk gmail search 'newer_than:7d' --account work

# Via environment
export WK_ACCOUNT=you@gmail.com
wk gmail search 'newer_than:7d'

# Auto-select (default account or the single stored token)
wk gmail labels list --account auto
```

## Account Aliases

```bash
wk auth alias set work work@company.com
wk auth alias list
wk auth alias unset work
```

Aliases work anywhere you pass `--account` or `WK_ACCOUNT` (reserved: `auto`, `default`).

## Multiple OAuth Clients

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

## Keyring Backends

Backends:

- `auto` (default): picks the best backend for the platform.
- `keychain`: macOS Keychain (recommended on macOS; avoids password management).
- `file`: encrypted on-disk keyring (requires a password).

Set backend via command (writes `keyring_backend` into `config.json`):

```bash
wk auth keyring file
wk auth keyring keychain
wk auth keyring auto
```

Show current backend + source (env/config/default) and config path:

```bash
wk auth keyring
```

Non-interactive runs (CI/ssh): file backend requires `WK_KEYRING_PASSWORD`.

```bash
export WK_KEYRING_PASSWORD='...'
wk --no-input auth status
```

Force backend via env (overrides config):

```bash
export WK_KEYRING_BACKEND=file
```

Precedence: `WK_KEYRING_BACKEND` env var overrides `config.json`.

### Keychain Prompts (macOS)

macOS Keychain may prompt more than you'd expect when the "app identity" keeps changing (different binary path, `go run` temp builds, rebuilding to new `./bin/wk`, multiple copies). Keychain treats those as different apps, so it asks again.

Options:

- **Default (recommended):** keep using Keychain (secure) and run a stable `wk` binary path to reduce repeat prompts.
- **Force Keychain:** `WK_KEYRING_BACKEND=keychain` (disables any file-backend fallback).
- **Avoid Keychain prompts entirely:** `WK_KEYRING_BACKEND=file` (stores encrypted entries on disk under your config dir).
  - To avoid password prompts too (CI/non-interactive): set `WK_KEYRING_PASSWORD=...` (tradeoff: secret in env).

### Credential Storage

OAuth credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

The CLI uses [github.com/99designs/keyring](https://github.com/99designs/keyring) for secure storage.

If no OS keychain backend is available (e.g., Linux/WSL/container), keyring can fall back to an encrypted on-disk store and may prompt for a password; for non-interactive runs set `WK_KEYRING_PASSWORD`.

## Headless / Remote Auth Flows

For servers without a browser:

### Manual Interactive Flow (recommended)

```bash
wk auth add you@gmail.com --services user --manual
```

- The CLI prints an auth URL. Open it in a local browser.
- After approval, copy the full loopback redirect URL from the browser address bar.
- Paste that URL back into the terminal when prompted.

### Split Remote Flow

Useful for two-step/scripted handoff:

```bash
# Step 1: print auth URL (open it locally in a browser)
wk auth add you@gmail.com --services user --remote --step 1

# Step 2: paste the full redirect URL from your browser address bar
wk auth add you@gmail.com --services user --remote --step 2 --auth-url 'http://127.0.0.1:<port>/oauth2/callback?code=...&state=...'
```

- The `state` is cached on disk for a short time (about 10 minutes). If it expires, rerun step 1.
- Remote step 2 requires a redirect URL that includes `state` (state check mandatory).

See [docs/headless-auth.md](headless-auth.md) for more details.

## Service Accounts (Workspace only)

A service account is a non-human Google identity that belongs to a Google Cloud project. In Google Workspace, a service account can impersonate a user via **domain-wide delegation** (admin-controlled) and access APIs like Gmail/Calendar/Drive as that user.

In `wk`, service accounts are an **optional auth method** that can be configured per account email. If a service account key is configured for an account, it takes precedence over OAuth refresh tokens (see `wk auth list`).

### 1) Create a Service Account (Google Cloud)

1. Create (or pick) a Google Cloud project.
2. Enable the APIs you'll use (e.g. Gmail, Calendar, Drive, Sheets, Docs, People, Tasks, Cloud Identity).
3. Go to **IAM & Admin -> Service Accounts** and create a service account.
4. In the service account details, enable **Domain-wide delegation**.
5. Create a key (**Keys -> Add key -> Create new key -> JSON**) and download the JSON key file.

### 2) Allowlist scopes (Google Workspace Admin Console)

Domain-wide delegation is enforced by Workspace admin settings.

1. Open **Admin console -> Security -> API controls -> Domain-wide delegation**.
2. Add a new API client:
   - Client ID: use the service account's "Client ID" from Google Cloud.
   - OAuth scopes: comma-separated list of scopes you want to allow (copy from `wk auth services` and/or your `wk auth add --services ...` usage).

If a scope is missing from the allowlist, service-account token minting can fail (or API calls will 403 with insufficient permissions).

### 3) Configure `wk` to use the service account

Store the key for the user you want to impersonate:

```bash
wk auth service-account set you@yourdomain.com --key ~/Downloads/service-account.json
```

Verify `wk` is preferring the service account for that account:

```bash
wk --account you@yourdomain.com auth status
wk auth list
```

## Google Keep (Workspace only)

Keep requires Workspace + domain-wide delegation. You can configure it via the generic service-account command above (recommended), or the legacy Keep helper:

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

See also [docs/configuration.md](configuration.md) for the full environment variables reference.

## Best Practices

- **Never commit OAuth client credentials** to version control.
- Store client credentials outside your project directory.
- Use different OAuth clients for development and production.
- Re-authorize with `--force-consent` if you suspect token compromise.
- Remove unused accounts with `wk auth remove <email>`.

## OAuth Client IDs in Open Source

Some open source Google CLIs ship a pre-configured OAuth client ID/secret copied from other desktop apps to avoid OAuth consent verification, testing-user limits, or quota issues. This makes the consent screen/security emails show the other app's name and can stop working at any time.

`workit` does not do this. Supported auth:

- Your own OAuth Desktop client JSON via `wk auth credentials ...` + `wk auth add ...`
- Google Workspace service accounts with domain-wide delegation (Workspace only)
