# 🧭 workit — Google in your terminal.

![GitHub Repo Banner](https://ghrb.waren.build/banner?header=workit%F0%9F%A7%AD&subheader=Google+in+your+terminal&bg=f3f4f6&color=1f2937&support=true)
<!-- Created with GitHub Repo Banner by Waren Gonzaga: https://ghrb.waren.build -->

Fast, script-friendly CLI for Gmail, Calendar, Chat, Classroom, Drive, Docs, Slides, Sheets, Forms, Apps Script, Contacts, Tasks, People, Groups (Workspace), and Keep (Workspace-only). JSON-first output, multiple accounts, and least-privilege auth built in.

## Features

- **Gmail** - search threads and messages, send emails, view attachments, manage labels/drafts/filters/delegation/vacation settings, history, and watch (Pub/Sub push)
- **Email tracking** - track opens for `wk gmail send --track` with a small Cloudflare Worker backend
- **Calendar** - list/create/update events, detect conflicts, manage invitations, check free/busy status, team calendars, propose new times, focus/OOO/working-location events, recurrence + reminders
- **Classroom** - manage courses, roster, coursework/materials, submissions, announcements, topics, invitations, guardians, profiles
- **Chat** - list/find/create spaces, list messages/threads (filter by thread/unread), send messages and DMs (Workspace-only)
- **Drive** - list/search/upload/download files, manage permissions/comments, organize folders, list shared drives
- **Drive Sync** - bidirectional folder sync like Google Drive for Desktop, with daemon mode and conflict resolution
- **Contacts** - search/create/update contacts, access Workspace directory/other contacts
- **Tasks** - manage tasklists and tasks: get/create/add/update/done/undo/delete/clear, repeat schedules
- **Sheets** - read/write/update spreadsheets, format cells, create new sheets (and export via Drive)
- **Forms** - create/get forms and inspect responses
- **Apps Script** - create/get projects, inspect content, and run functions
- **Docs/Slides** - export to PDF/DOCX/PPTX via Drive (plus create/copy, docs-to-text)
- **People** - access profile information
- **Keep (Workspace only)** - list/get/search notes and download attachments (service account + domain-wide delegation)
- **Groups** - list groups you belong to, view group members (Google Workspace)
- **Local time** - quick local/UTC time display for scripts and agents
- **Multiple accounts** - manage multiple Google accounts simultaneously (with aliases)
- **Agent safety** - `--read-only`, `--command-tier`, and `--enable-commands` restrict what an AI agent can do
- **Headless OAuth** - authenticate users who complete OAuth on mobile/web, ideal for AI agents
- **Secure credential storage** using OS keyring or encrypted on-disk keyring (configurable)
- **Auto-refreshing tokens** - authenticate once, use indefinitely
- **Least-privilege auth** - `--readonly` and `--drive-scope` to request fewer scopes
- **Workspace service accounts** - domain-wide delegation auth (preferred when configured)
- **Parseable output** - JSON mode for scripting and automation (Calendar adds day-of-week fields)

## Installation

### Homebrew

```bash
brew install steipete/tap/workit
```
### Arch User Repository

```bash
yay -S workit
```

### Build from Source

```bash
git clone https://github.com/namastexlabs/workit.git
cd workit
make
```

Run:

```bash
./bin/wk --help
```

Help:

- `wk --help` shows top-level command groups.
- Drill down with `wk <group> --help` (and deeper subcommands).
- For the full expanded command list: `WK_HELP=full wk --help`.
- Make shortcut: `make wk -- --help` (or `make wk -- gmail --help`).
- `make wk-help` shows CLI help (note: `make wk --help` is Make’s own help; use `--`).
- Version: `wk --version` or `wk version`.

### Version artifact contract

`wk --version` and `wk version --json` expose build metadata used by CI/release automation.

Contract (`wk version --json`):

```json
{
  "version": "v0.12.0",
  "branch": "main",
  "commit": "abc123def456",
  "date": "2026-02-18T20:03:00Z"
}
```

- `version`: semantic version tag for releases, or dev auto-version for non-main builds.
- `branch`: git branch used for the build.
- `commit`: short commit SHA (12 chars).
- `date`: UTC build timestamp (RFC3339).

The `version` workflow (`.github/workflows/version.yml`) publishes this JSON as the `version-contract` artifact and validates the schema.

## Quick Start

### 1. Get OAuth2 Credentials

Before adding an account, create OAuth2 credentials from Google Cloud Console:

1. Open the Google Cloud Console credentials page: https://console.cloud.google.com/apis/credentials
1. Create a project: https://console.cloud.google.com/projectcreate
2. Enable the APIs you need:
   - Gmail API: https://console.cloud.google.com/apis/api/gmail.googleapis.com
   - Google Calendar API: https://console.cloud.google.com/apis/api/calendar-json.googleapis.com
   - Google Chat API: https://console.cloud.google.com/apis/api/chat.googleapis.com
   - Google Drive API: https://console.cloud.google.com/apis/api/drive.googleapis.com
   - Google Classroom API: https://console.cloud.google.com/apis/api/classroom.googleapis.com
   - People API (Contacts): https://console.cloud.google.com/apis/api/people.googleapis.com
   - Google Tasks API: https://console.cloud.google.com/apis/api/tasks.googleapis.com
   - Google Sheets API: https://console.cloud.google.com/apis/api/sheets.googleapis.com
   - Google Forms API: https://console.cloud.google.com/apis/api/forms.googleapis.com
   - Apps Script API: https://console.cloud.google.com/apis/api/script.googleapis.com
   - Cloud Identity API (Groups): https://console.cloud.google.com/apis/api/cloudidentity.googleapis.com
3. Configure OAuth consent screen: https://console.cloud.google.com/auth/branding
4. If your app is in "Testing", add test users: https://console.cloud.google.com/auth/audience
5. Create OAuth client:
   - Go to https://console.cloud.google.com/auth/clients
   - Click "Create Client"
   - Application type: "Desktop app"
   - Download the JSON file (usually named `client_secret_....apps.googleusercontent.com.json`)

### 2. Store Credentials

```bash
wk auth credentials ~/Downloads/client_secret_....json
```

For multiple OAuth clients/projects:

```bash
wk --client work auth credentials ~/Downloads/work-client.json
wk auth credentials list
```

### 3. Authorize Your Account

```bash
wk auth add you@gmail.com
```

This will open a browser window for OAuth authorization. The refresh token is stored securely in your system keychain.

Headless / remote server flows (no browser on the server):

Manual interactive flow (recommended):

```bash
wk auth add you@gmail.com --services user --manual
```

- The CLI prints an auth URL. Open it in a local browser.
- After approval, copy the full loopback redirect URL from the browser address bar.
- Paste that URL back into the terminal when prompted.

Split remote flow (`--remote`, useful for two-step/scripted handoff):

```bash
# Step 1: print auth URL (open it locally in a browser)
wk auth add you@gmail.com --services user --remote --step 1

# Step 2: paste the full redirect URL from your browser address bar
wk auth add you@gmail.com --services user --remote --step 2 --auth-url 'http://127.0.0.1:<port>/oauth2/callback?code=...&state=...'
```

- The `state` is cached on disk for a short time (about 10 minutes). If it expires, rerun step 1.
- Remote step 2 requires a redirect URL that includes `state` (state check mandatory).

### 4. Test Authentication

```bash
export WK_ACCOUNT=you@gmail.com
wk gmail labels list
```

## Authentication & Secrets

### Accounts and tokens

`wk` stores your OAuth refresh tokens in a “keyring” backend. Default is `auto` (best available backend for your OS/environment).

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

### Multiple OAuth clients

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

1) `--client` / `WK_CLIENT`
2) `account_clients` config (email -> client)
3) `client_domains` config (domain -> client)
4) Credentials file named after the email domain (`credentials-example.com.json`)
5) `default`

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

See `docs/auth-clients.md` for the full client selection and mapping rules.

### Keyring backend: Keychain vs encrypted file

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

## Configuration

### Account Selection

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

List configured accounts:

```bash
wk auth list
```

### Output

- Default: human-friendly tables on stdout.
- `--plain`: stable TSV on stdout (tabs preserved; best for piping to tools that expect `\t`).
- `--json`: JSON on stdout (best for scripting).
- Human-facing hints/progress go to stderr.
- Colors are enabled only in rich TTY output and are disabled automatically for `--json` and `--plain`.

### Service Scopes

By default, `wk auth add` requests access to the **user** services (see `wk auth services` for the current list and scopes).

To request fewer scopes:

```bash
wk auth add you@gmail.com --services drive,calendar
```

To request read-only scopes (write operations will fail with 403 insufficient scopes):

```bash
wk auth add you@gmail.com --services drive,calendar --readonly
```

To control Drive’s scope (default: `full`):

```bash
wk auth add you@gmail.com --services drive --drive-scope full
wk auth add you@gmail.com --services drive --drive-scope readonly
wk auth add you@gmail.com --services drive --drive-scope file
```

Notes:

- `--drive-scope readonly` is enough for listing/downloading/exporting via Drive (write operations will 403).
- `--drive-scope file` is write-capable (limited to files created/opened by this app) and can’t be combined with `--readonly`.

If you need to add services later and Google doesn't return a refresh token, re-run with `--force-consent`:

```bash
wk auth add you@gmail.com --services user --force-consent
# Or add just Sheets
wk auth add you@gmail.com --services sheets --force-consent
```

`--services all` is accepted as an alias for `user` for backwards compatibility.

Docs commands are implemented via the Drive API, and `docs` requests both Drive and Docs API scopes.

Service scope matrix (auto-generated; run `go run scripts/gen-auth-services-md.go`):

<!-- auth-services:start -->
| Service | User | APIs | Scopes | Notes |
| --- | --- | --- | --- | --- |
| gmail | yes | Gmail API | `https://www.googleapis.com/auth/gmail.modify`<br>`https://www.googleapis.com/auth/gmail.settings.basic`<br>`https://www.googleapis.com/auth/gmail.settings.sharing` |  |
| calendar | yes | Calendar API | `https://www.googleapis.com/auth/calendar` |  |
| chat | yes | Chat API | `https://www.googleapis.com/auth/chat.spaces`<br>`https://www.googleapis.com/auth/chat.messages`<br>`https://www.googleapis.com/auth/chat.memberships`<br>`https://www.googleapis.com/auth/chat.users.readstate.readonly` |  |
| classroom | yes | Classroom API | `https://www.googleapis.com/auth/classroom.courses`<br>`https://www.googleapis.com/auth/classroom.rosters`<br>`https://www.googleapis.com/auth/classroom.coursework.students`<br>`https://www.googleapis.com/auth/classroom.coursework.me`<br>`https://www.googleapis.com/auth/classroom.courseworkmaterials`<br>`https://www.googleapis.com/auth/classroom.announcements`<br>`https://www.googleapis.com/auth/classroom.topics`<br>`https://www.googleapis.com/auth/classroom.guardianlinks.students`<br>`https://www.googleapis.com/auth/classroom.profile.emails`<br>`https://www.googleapis.com/auth/classroom.profile.photos` |  |
| drive | yes | Drive API | `https://www.googleapis.com/auth/drive` |  |
| docs | yes | Docs API, Drive API | `https://www.googleapis.com/auth/drive`<br>`https://www.googleapis.com/auth/documents` | Export/copy/create via Drive |
| slides | yes | Slides API, Drive API | `https://www.googleapis.com/auth/drive`<br>`https://www.googleapis.com/auth/presentations` | Create/edit presentations |
| contacts | yes | People API | `https://www.googleapis.com/auth/contacts`<br>`https://www.googleapis.com/auth/contacts.other.readonly`<br>`https://www.googleapis.com/auth/directory.readonly` | Contacts + other contacts + directory |
| tasks | yes | Tasks API | `https://www.googleapis.com/auth/tasks` |  |
| sheets | yes | Sheets API, Drive API | `https://www.googleapis.com/auth/drive`<br>`https://www.googleapis.com/auth/spreadsheets` | Export via Drive |
| people | yes | People API | `profile` | OIDC profile scope |
| forms | yes | Forms API | `https://www.googleapis.com/auth/forms.body`<br>`https://www.googleapis.com/auth/forms.responses.readonly` |  |
| appscript | yes | Apps Script API | `https://www.googleapis.com/auth/script.projects`<br>`https://www.googleapis.com/auth/script.deployments`<br>`https://www.googleapis.com/auth/script.processes` |  |
| groups | no | Cloud Identity API | `https://www.googleapis.com/auth/cloud-identity.groups.readonly` | Workspace only |
| keep | no | Keep API | `https://www.googleapis.com/auth/keep.readonly` | Workspace only; service account (domain-wide delegation) |
<!-- auth-services:end -->

### Service Accounts (Workspace only)

A service account is a non-human Google identity that belongs to a Google Cloud project. In Google Workspace, a service account can impersonate a user via **domain-wide delegation** (admin-controlled) and access APIs like Gmail/Calendar/Drive as that user.

In `wk`, service accounts are an **optional auth method** that can be configured per account email. If a service account key is configured for an account, it takes precedence over OAuth refresh tokens (see `wk auth list`).

#### 1) Create a Service Account (Google Cloud)

1. Create (or pick) a Google Cloud project.
2. Enable the APIs you’ll use (e.g. Gmail, Calendar, Drive, Sheets, Docs, People, Tasks, Cloud Identity).
3. Go to **IAM & Admin → Service Accounts** and create a service account.
4. In the service account details, enable **Domain-wide delegation**.
5. Create a key (**Keys → Add key → Create new key → JSON**) and download the JSON key file.

#### 2) Allowlist scopes (Google Workspace Admin Console)

Domain-wide delegation is enforced by Workspace admin settings.

1. Open **Admin console → Security → API controls → Domain-wide delegation**.
2. Add a new API client:
   - Client ID: use the service account’s “Client ID” from Google Cloud.
   - OAuth scopes: comma-separated list of scopes you want to allow (copy from `wk auth services` and/or your `wk auth add --services ...` usage).

If a scope is missing from the allowlist, service-account token minting can fail (or API calls will 403 with insufficient permissions).

#### 3) Configure `wk` to use the service account

Store the key for the user you want to impersonate:

```bash
wk auth service-account set you@yourdomain.com --key ~/Downloads/service-account.json
```

Verify `wk` is preferring the service account for that account:

```bash
wk --account you@yourdomain.com auth status
wk auth list
```

### Google Keep (Workspace only)

Keep requires Workspace + domain-wide delegation. You can configure it via the generic service-account command above (recommended), or the legacy Keep helper:

```bash
wk auth service-account set you@yourdomain.com --key ~/Downloads/service-account.json
wk keep list --account you@yourdomain.com
wk keep get <noteId> --account you@yourdomain.com
```

### Environment Variables

- `WK_ACCOUNT` - Default account email or alias to use (avoids repeating `--account`; otherwise uses keyring default or a single stored token)
- `WK_CLIENT` - OAuth client name (selects stored credentials + token bucket)
- `WK_JSON` - Default JSON output
- `WK_PLAIN` - Default plain output
- `WK_COLOR` - Color mode: `auto` (default), `always`, or `never`
- `WK_TIMEZONE` - Default output timezone for Calendar/Gmail (IANA name, `UTC`, or `local`)
- `WK_ENABLE_COMMANDS` - Comma-separated allowlist of top-level commands (e.g., `calendar,tasks`)

### Config File (JSON5)

Find the actual config path in `wk --help` or `wk auth keyring`.

Typical paths:

- macOS: `~/Library/Application Support/workit/config.json`
- Linux: `~/.config/workit/config.json` (or `$XDG_CONFIG_HOME/workit/config.json`)
- Windows: `%AppData%\\workit\\config.json`

Example (JSON5 supports comments and trailing commas):

```json5
{
  // Avoid macOS Keychain prompts
  keyring_backend: "file",
  // Default output timezone for Calendar/Gmail (IANA, UTC, or local)
  default_timezone: "UTC",
  // Optional account aliases
  account_aliases: {
    work: "work@company.com",
    personal: "me@gmail.com",
  },
  // Optional per-account OAuth client selection
  account_clients: {
    "work@company.com": "work",
  },
  // Optional domain -> client mapping
  client_domains: {
    "example.com": "work",
  },
}
```

### Config Commands

```bash
wk config path
wk config list
wk config keys
wk config get default_timezone
wk config set default_timezone UTC
wk config unset default_timezone
```

### Account Aliases

```bash
wk auth alias set work work@company.com
wk auth alias list
wk auth alias unset work
```

Aliases work anywhere you pass `--account` or `WK_ACCOUNT` (reserved: `auto`, `default`).

### Command Allowlist (Sandboxing)

```bash
# Only allow calendar + tasks commands for an agent
wk --enable-commands calendar,tasks calendar events --today

# Same via env
export WK_ENABLE_COMMANDS=calendar,tasks
wk tasks list <tasklistId>
```
 
## Security

### Credential Storage

OAuth credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

The CLI uses [github.com/99designs/keyring](https://github.com/99designs/keyring) for secure storage.

If no OS keychain backend is available (e.g., Linux/WSL/container), keyring can fall back to an encrypted on-disk store and may prompt for a password; for non-interactive runs set `WK_KEYRING_PASSWORD`.

### Keychain Prompts (macOS)

macOS Keychain may prompt more than you’d expect when the “app identity” keeps changing (different binary path, `go run` temp builds, rebuilding to new `./bin/wk`, multiple copies). Keychain treats those as different apps, so it asks again.

Options:

- **Default (recommended):** keep using Keychain (secure) and run a stable `wk` binary path to reduce repeat prompts.
- **Force Keychain:** `WK_KEYRING_BACKEND=keychain` (disables any file-backend fallback).
- **Avoid Keychain prompts entirely:** `WK_KEYRING_BACKEND=file` (stores encrypted entries on disk under your config dir).
  - To avoid password prompts too (CI/non-interactive): set `WK_KEYRING_PASSWORD=...` (tradeoff: secret in env).

### Best Practices

- **Never commit OAuth client credentials** to version control
- Store client credentials outside your project directory
- Use different OAuth clients for development and production
- Re-authorize with `--force-consent` if you suspect token compromise
- Remove unused accounts with `wk auth remove <email>`

### OAuth Client IDs in Open Source

Some open source Google CLIs ship a pre-configured OAuth client ID/secret copied from other desktop apps to avoid OAuth consent verification, testing-user limits, or quota issues. This makes the consent screen/security emails show the other app’s name and can stop working at any time.

`workit` does not do this. Supported auth:

- Your own OAuth Desktop client JSON via `wk auth credentials ...` + `wk auth add ...`
- Google Workspace service accounts with domain-wide delegation (Workspace only)

## Commands

Flag aliases:
- `--out` also accepts `--output`.
- `--out-dir` also accepts `--output-dir` (Gmail thread attachment downloads).

### Authentication

```bash
wk auth credentials <path>           # Store OAuth client credentials
wk auth credentials list             # List stored OAuth client credentials
wk --client work auth credentials <path>  # Store named OAuth client credentials
wk auth add <email>                  # Authorize and store refresh token
wk auth service-account set <email> --key <path>  # Configure service account impersonation (Workspace only)
wk auth service-account status <email>            # Show service account status
wk auth service-account unset <email>             # Remove service account
wk auth keep <email> --key <path>                 # Legacy alias (Keep)
wk auth keyring [backend]            # Show/set keyring backend (auto|keychain|file)
wk auth status                       # Show current auth state/services
wk auth services                     # List available services and OAuth scopes
wk auth list                         # List stored accounts
wk auth list --check                 # Validate stored refresh tokens
wk auth remove <email>               # Remove a stored refresh token
wk auth manage                       # Open accounts manager in browser
wk auth tokens                       # Manage stored refresh tokens
```

### Keep (Workspace only)

```bash
wk keep list --account you@yourdomain.com
wk keep get <noteId> --account you@yourdomain.com
wk keep search <query> --account you@yourdomain.com
wk keep attachment <attachmentName> --account you@yourdomain.com --out ./attachment.bin
```

### Gmail

```bash
# Search and read
wk gmail search 'newer_than:7d' --max 10
wk gmail thread get <threadId>
wk gmail thread get <threadId> --download              # Download attachments to current dir
wk gmail thread get <threadId> --download --out-dir ./attachments
wk gmail get <messageId>
wk gmail get <messageId> --format metadata
wk gmail attachment <messageId> <attachmentId>
wk gmail attachment <messageId> <attachmentId> --out ./attachment.bin
wk gmail url <threadId>              # Print Gmail web URL
wk gmail thread modify <threadId> --add STARRED --remove INBOX

# Send and compose
wk gmail send --to a@b.com --subject "Hi" --body "Plain fallback"
wk gmail send --to a@b.com --subject "Hi" --body-file ./message.txt
wk gmail send --to a@b.com --subject "Hi" --body-file -   # Read body from stdin
wk gmail send --to a@b.com --subject "Hi" --body "Plain fallback" --body-html "<p>Hello</p>"
# Reply + include quoted original message (auto-generates HTML quote unless you pass --body-html)
wk gmail send --reply-to-message-id <messageId> --quote --to a@b.com --subject "Re: Hi" --body "My reply"
wk gmail drafts list
wk gmail drafts create --subject "Draft" --body "Body"
wk gmail drafts create --to a@b.com --subject "Draft" --body "Body"
wk gmail drafts update <draftId> --subject "Draft" --body "Body"
wk gmail drafts update <draftId> --to a@b.com --subject "Draft" --body "Body"
wk gmail drafts send <draftId>

# Labels
wk gmail labels list
wk gmail labels get INBOX --json  # Includes message counts
wk gmail labels create "My Label"
wk gmail labels modify <threadId> --add STARRED --remove INBOX
wk gmail labels delete <labelIdOrName>  # Deletes user label (guards system labels; confirm)

# Batch operations
wk gmail batch delete <messageId> <messageId>
wk gmail batch modify <messageId> <messageId> --add STARRED --remove INBOX

# Filters
wk gmail filters list
wk gmail filters create --from 'noreply@example.com' --add-label 'Notifications'
wk gmail filters delete <filterId>

# Settings
wk gmail autoforward get
wk gmail autoforward enable --email forward@example.com
wk gmail autoforward disable
wk gmail forwarding list
wk gmail forwarding add --email forward@example.com
wk gmail sendas list
wk gmail sendas create --email alias@example.com
wk gmail vacation get
wk gmail vacation enable --subject "Out of office" --message "..."
wk gmail vacation disable

# Delegation (G Suite/Workspace)
wk gmail delegates list
wk gmail delegates add --email delegate@example.com
wk gmail delegates remove --email delegate@example.com

# Watch (Pub/Sub push)
wk gmail watch start --topic projects/<p>/topics/<t> --label INBOX
wk gmail watch serve --bind 127.0.0.1 --token <shared> --hook-url http://127.0.0.1:18789/hooks/agent
wk gmail watch serve --bind 0.0.0.0 --verify-oidc --oidc-email <svc@...> --hook-url <url>
wk gmail watch serve --bind 127.0.0.1 --token <shared> --exclude-labels SPAM,TRASH --hook-url http://127.0.0.1:18789/hooks/agent
wk gmail history --since <historyId>
```

Gmail watch (Pub/Sub push):
- Create Pub/Sub topic + push subscription (OIDC preferred; shared token ok for dev).
- Full flow + payload details: `docs/watch.md`.
- `watch serve --exclude-labels` defaults to `SPAM,TRASH`; IDs are case-sensitive.

### Email Tracking

Track when recipients open your emails:

```bash
# Set up local tracking config (per-account; generates keys; follow printed deploy steps)
wk gmail track setup --worker-url https://wk-email-tracker.<acct>.workers.dev

# Send with tracking
wk gmail send --to recipient@example.com --subject "Hello" --body-html "<p>Hi!</p>" --track

# Check opens
wk gmail track opens <tracking_id>
wk gmail track opens --to recipient@example.com

# View status
wk gmail track status
```

Docs: `docs/email-tracking.md` (setup/deploy) + `docs/email-tracking-worker.md` (internals).

**Notes:** `--track` requires exactly 1 recipient (no cc/bcc) and an HTML body (`--body-html` or `--quote`). Use `--track-split` to send per-recipient messages with individual tracking ids. The tracking worker stores IP/user-agent + coarse geo by default.

### Calendar

```bash
# Calendars
wk calendar calendars
wk calendar acl <calendarId>         # List access control rules
wk calendar colors                   # List available event/calendar colors
wk calendar time --timezone America/New_York
wk calendar users                    # List workspace users (use email as calendar ID)

# Events (with timezone-aware time flags)
wk calendar events <calendarId> --today                    # Today's events
wk calendar events <calendarId> --tomorrow                 # Tomorrow's events
wk calendar events <calendarId> --week                     # This week (Mon-Sun by default; use --week-start)
wk calendar events <calendarId> --days 3                   # Next 3 days
wk calendar events <calendarId> --from today --to friday   # Relative dates
wk calendar events <calendarId> --from today --to friday --weekday   # Include weekday columns
wk calendar events <calendarId> --from 2025-01-01T00:00:00Z --to 2025-01-08T00:00:00Z
wk calendar events --all             # Fetch events from all calendars
wk calendar event <calendarId> <eventId>
wk calendar get <calendarId> <eventId>                     # Alias for event
wk calendar search "meeting" --today
wk calendar search "meeting" --tomorrow
wk calendar search "meeting" --days 365
wk calendar search "meeting" --from 2025-01-01T00:00:00Z --to 2025-01-31T00:00:00Z --max 50

# Search defaults to 30 days ago through 90 days ahead unless you set --from/--to/--today/--week/--days.
# Tip: set WK_CALENDAR_WEEKDAY=1 to default --weekday for calendar events output.

# JSON event output includes timezone and localized times (useful for agents).
wk calendar get <calendarId> <eventId> --json
# {
#   "event": {
#     "id": "...",
#     "summary": "...",
#     "startDayOfWeek": "Friday",
#     "endDayOfWeek": "Friday",
#     "timezone": "America/Los_Angeles",
#     "eventTimezone": "America/New_York",
#     "startLocal": "2026-01-23T20:45:00-08:00",
#     "endLocal": "2026-01-23T22:45:00-08:00",
#     "start": { "dateTime": "2026-01-23T23:45:00-05:00" },
#     "end": { "dateTime": "2026-01-24T01:45:00-05:00" }
#   }
# }

# Team calendars (requires Cloud Identity API for Google Workspace)
wk calendar team <group-email> --today           # Show team's events for today
wk calendar team <group-email> --week            # Show team's events for the week (use --week-start)
wk calendar team <group-email> --freebusy        # Show only busy/free blocks (faster)
wk calendar team <group-email> --query "standup" # Filter by event title

# Create and update
wk calendar create <calendarId> \
  --summary "Meeting" \
  --from 2025-01-15T10:00:00Z \
  --to 2025-01-15T11:00:00Z

wk calendar create <calendarId> \
  --summary "Team Sync" \
  --from 2025-01-15T14:00:00Z \
  --to 2025-01-15T15:00:00Z \
  --attendees "alice@example.com,bob@example.com" \
  --location "Zoom"

wk calendar update <calendarId> <eventId> \
  --summary "Updated Meeting" \
  --from 2025-01-15T11:00:00Z \
  --to 2025-01-15T12:00:00Z

# Send notifications when creating/updating
wk calendar create <calendarId> \
  --summary "Team Sync" \
  --from 2025-01-15T14:00:00Z \
  --to 2025-01-15T15:00:00Z \
  --send-updates all

wk calendar update <calendarId> <eventId> \
  --send-updates externalOnly

# Default: no attendee notifications unless you pass --send-updates.
wk calendar delete <calendarId> <eventId> \
  --send-updates all --force

# Recurrence + reminders
wk calendar create <calendarId> \
  --summary "Payment" \
  --from 2025-02-11T09:00:00-03:00 \
  --to 2025-02-11T09:15:00-03:00 \
  --rrule "RRULE:FREQ=MONTHLY;BYMONTHDAY=11" \
  --reminder "email:3d" \
  --reminder "popup:30m"

# Special event types via --event-type (focus-time/out-of-office/working-location)
wk calendar create primary \
  --event-type focus-time \
  --from 2025-01-15T13:00:00Z \
  --to 2025-01-15T14:00:00Z

wk calendar create primary \
  --event-type out-of-office \
  --from 2025-01-20 \
  --to 2025-01-21 \
  --all-day

wk calendar create primary \
  --event-type working-location \
  --working-location-type office \
  --working-office-label "HQ" \
  --from 2025-01-22 \
  --to 2025-01-23

# Dedicated shortcuts (same event types, more opinionated defaults)
wk calendar focus-time --from 2025-01-15T13:00:00Z --to 2025-01-15T14:00:00Z
wk calendar out-of-office --from 2025-01-20 --to 2025-01-21 --all-day
wk calendar working-location --type office --office-label "HQ" --from 2025-01-22 --to 2025-01-23
# Add attendees without replacing existing attendees/RSVP state
wk calendar update <calendarId> <eventId> \
  --add-attendee "alice@example.com,bob@example.com"

wk calendar delete <calendarId> <eventId>

# Invitations
wk calendar respond <calendarId> <eventId> --status accepted
wk calendar respond <calendarId> <eventId> --status declined
wk calendar respond <calendarId> <eventId> --status tentative
wk calendar respond <calendarId> <eventId> --status declined --send-updates externalOnly

# Propose a new time (browser-only flow; API limitation)
wk calendar propose-time <calendarId> <eventId>
wk calendar propose-time <calendarId> <eventId> --open
wk calendar propose-time <calendarId> <eventId> --decline --comment "Can we do 5pm?"

# Availability
wk calendar freebusy --calendars "primary,work@example.com" \
  --from 2025-01-15T00:00:00Z \
  --to 2025-01-16T00:00:00Z

wk calendar conflicts --calendars "primary,work@example.com" \
  --today                             # Today's conflicts
```

### Time

```bash
wk time now
wk time now --timezone UTC
```

### Drive

```bash
# List and search
wk drive ls --max 20
wk drive ls --parent <folderId> --max 20
wk drive ls --no-all-drives            # Only list from "My Drive"
wk drive search "invoice" --max 20
wk drive search "invoice" --no-all-drives
wk drive search "mimeType = 'application/pdf'" --raw-query
wk drive get <fileId>                # Get file metadata
wk drive url <fileId>                # Print Drive web URL
wk drive copy <fileId> "Copy Name"

# Upload and download
wk drive upload ./path/to/file --parent <folderId>
wk drive upload ./path/to/file --replace <fileId>  # Replace file content in-place (preserves shared link)
wk drive upload ./report.docx --convert
wk drive upload ./chart.png --convert-to sheet
wk drive upload ./report.docx --convert --name report.docx
wk drive download <fileId> --out ./downloaded.bin
wk drive download <fileId> --format pdf --out ./exported.pdf     # Google Workspace files only
wk drive download <fileId> --format docx --out ./doc.docx
wk drive download <fileId> --format pptx --out ./slides.pptx

# Organize
wk drive mkdir "New Folder"
wk drive mkdir "New Folder" --parent <parentFolderId>
wk drive rename <fileId> "New Name"
wk drive move <fileId> --parent <destinationFolderId>
wk drive delete <fileId>             # Move to trash
wk drive delete <fileId> --permanent # Permanently delete

# Permissions
wk drive permissions <fileId>
wk drive share <fileId> --to user --email user@example.com --role reader
wk drive share <fileId> --to user --email user@example.com --role writer
wk drive share <fileId> --to domain --domain example.com --role reader
wk drive unshare <fileId> --permission-id <permissionId>

# Shared drives (Team Drives)
wk drive drives --max 100
```

### Docs / Slides / Sheets

```bash
# Docs
wk docs info <docId>
wk docs cat <docId> --max-bytes 10000
wk docs create "My Doc"
wk docs create "My Doc" --file ./doc.md            # Import markdown
wk docs copy <docId> "My Doc Copy"
wk docs export <docId> --format pdf --out ./doc.pdf
wk docs list-tabs <docId>
wk docs cat <docId> --tab "Notes"
wk docs cat <docId> --all-tabs
wk docs update <docId> --format markdown --content-file ./doc.md
wk docs write <docId> --replace --markdown --file ./doc.md
wk docs find-replace <docId> "old" "new"

# Slides
wk slides info <presentationId>
wk slides create "My Deck"
wk slides create-from-markdown "My Deck" --content-file ./slides.md
wk slides copy <presentationId> "My Deck Copy"
wk slides export <presentationId> --format pdf --out ./deck.pdf
wk slides list-slides <presentationId>
wk slides add-slide <presentationId> ./slide.png --notes "Speaker notes"
wk slides update-notes <presentationId> <slideId> --notes "Updated notes"
wk slides replace-slide <presentationId> <slideId> ./new-slide.png --notes "New notes"

# Sheets
wk sheets copy <spreadsheetId> "My Sheet Copy"
wk sheets export <spreadsheetId> --format pdf --out ./sheet.pdf
wk sheets format <spreadsheetId> 'Sheet1!A1:B2' --format-json '{"textFormat":{"bold":true}}' --format-fields 'userEnteredFormat.textFormat.bold'
```

### Contacts

```bash
# Personal contacts
wk contacts list --max 50
wk contacts search "Ada" --max 50
wk contacts get people/<resourceName>
wk contacts get user@example.com     # Get by email

# Other contacts (people you've interacted with)
wk contacts other list --max 50
wk contacts other search "John" --max 50

# Create and update
wk contacts create \
  --given "John" \
  --family "Doe" \
  --email "john@example.com" \
  --phone "+1234567890"

wk contacts update people/<resourceName> \
  --given "Jane" \
  --email "jane@example.com" \
  --birthday "1990-05-12" \
  --notes "Met at WWDC"

# Update via JSON (see docs/contacts-json-update.md)
wk contacts get people/<resourceName> --json | \
  jq '(.contact.urls //= []) | (.contact.urls += [{"value":"obsidian://open?vault=notes&file=People/John%20Doe","type":"profile"}])' | \
  wk contacts update people/<resourceName> --from-file -

wk contacts delete people/<resourceName>

# Workspace directory (requires Google Workspace)
wk contacts directory list --max 50
wk contacts directory search "Jane" --max 50
```

### Tasks

```bash
# Task lists
wk tasks lists --max 50
wk tasks lists create <title>

# Tasks in a list
wk tasks list <tasklistId> --max 50
wk tasks get <tasklistId> <taskId>
wk tasks add <tasklistId> --title "Task title"
wk tasks add <tasklistId> --title "Weekly sync" --due 2025-02-01 --repeat weekly --repeat-count 4
wk tasks add <tasklistId> --title "Daily standup" --due 2025-02-01 --repeat daily --repeat-until 2025-02-05
wk tasks update <tasklistId> <taskId> --title "New title"
wk tasks done <tasklistId> <taskId>
wk tasks undo <tasklistId> <taskId>
wk tasks delete <tasklistId> <taskId>
wk tasks clear <tasklistId>

# Note: Google Tasks treats due dates as date-only; time components may be ignored.
# See docs/dates.md for all supported date/time input formats across commands.
```

### Sheets

```bash
# Read
wk sheets metadata <spreadsheetId>
wk sheets get <spreadsheetId> 'Sheet1!A1:B10'

# Export (via Drive)
wk sheets export <spreadsheetId> --format pdf --out ./sheet.pdf
wk sheets export <spreadsheetId> --format xlsx --out ./sheet.xlsx

# Write
wk sheets update <spreadsheetId> 'A1' 'val1|val2,val3|val4'
wk sheets update <spreadsheetId> 'A1' --values-json '[["a","b"],["c","d"]]'
wk sheets update <spreadsheetId> 'Sheet1!A1:C1' 'new|row|data' --copy-validation-from 'Sheet1!A2:C2'
wk sheets append <spreadsheetId> 'Sheet1!A:C' 'new|row|data'
wk sheets append <spreadsheetId> 'Sheet1!A:C' 'new|row|data' --copy-validation-from 'Sheet1!A2:C2'
wk sheets clear <spreadsheetId> 'Sheet1!A1:B10'

# Format
wk sheets format <spreadsheetId> 'Sheet1!A1:B2' --format-json '{"textFormat":{"bold":true}}' --format-fields 'userEnteredFormat.textFormat.bold'

# Create
wk sheets create "My New Spreadsheet" --sheets "Sheet1,Sheet2"
```

### Forms

```bash
# Forms
wk forms get <formId>
wk forms create --title "Weekly Check-in" --description "Friday async update"

# Responses
wk forms responses list <formId> --max 20
wk forms responses get <formId> <responseId>
```

### Apps Script

```bash
# Projects
wk appscript get <scriptId>
wk appscript content <scriptId>
wk appscript create --title "Automation Helpers"
wk appscript create --title "Bound Script" --parent-id <driveFileId>

# Execute functions
wk appscript run <scriptId> myFunction --params '["arg1", 123, true]'
wk appscript run <scriptId> myFunction --dev-mode
```

### People

```bash
# Profile
wk people me
wk people get people/<userId>

# Search the Workspace directory
wk people search "Ada Lovelace" --max 5

# Relations (defaults to people/me)
wk people relations
wk people relations people/<userId> --type manager
```

### Chat

```bash
# Spaces
wk chat spaces list
wk chat spaces find "Engineering"
wk chat spaces create "Engineering" --member alice@company.com --member bob@company.com

# Messages
wk chat messages list spaces/<spaceId> --max 5
wk chat messages list spaces/<spaceId> --thread <threadId>
wk chat messages list spaces/<spaceId> --unread
wk chat messages send spaces/<spaceId> --text "Build complete!" --thread spaces/<spaceId>/threads/<threadId>

# Threads
wk chat threads list spaces/<spaceId>

# Direct messages
wk chat dm space user@company.com
wk chat dm send user@company.com --text "ping"
```

Note: Chat commands require a Google Workspace account (consumer @gmail.com accounts are not supported).

### Groups (Google Workspace)

```bash
# List groups you belong to
wk groups list

# List members of a group
wk groups members engineering@company.com
```

Note: Groups commands require the Cloud Identity API and the `cloud-identity.groups.readonly` scope. If you get a permissions error, re-authenticate:

```bash
wk auth add your@email.com --services groups --force-consent
```

### Classroom (Google Workspace for Education)

```bash
# Courses
wk classroom courses list
wk classroom courses list --role teacher
wk classroom courses get <courseId>
wk classroom courses create --name "Math 101"
wk classroom courses update <courseId> --name "Math 102"
wk classroom courses archive <courseId>
wk classroom courses unarchive <courseId>
wk classroom courses url <courseId>

# Roster
wk classroom roster <courseId>
wk classroom roster <courseId> --students
wk classroom students add <courseId> <userId>
wk classroom teachers add <courseId> <userId>

# Coursework
wk classroom coursework list <courseId>
wk classroom coursework get <courseId> <courseworkId>
wk classroom coursework create <courseId> --title "Homework 1" --type ASSIGNMENT --state PUBLISHED
wk classroom coursework update <courseId> <courseworkId> --title "Updated"
wk classroom coursework assignees <courseId> <courseworkId> --mode INDIVIDUAL_STUDENTS --add-student <studentId>

# Materials
wk classroom materials list <courseId>
wk classroom materials create <courseId> --title "Syllabus" --state PUBLISHED

# Submissions
wk classroom submissions list <courseId> <courseworkId>
wk classroom submissions get <courseId> <courseworkId> <submissionId>
wk classroom submissions grade <courseId> <courseworkId> <submissionId> --grade 85
wk classroom submissions return <courseId> <courseworkId> <submissionId>
wk classroom submissions turn-in <courseId> <courseworkId> <submissionId>
wk classroom submissions reclaim <courseId> <courseworkId> <submissionId>

# Announcements
wk classroom announcements list <courseId>
wk classroom announcements create <courseId> --text "Welcome!"
wk classroom announcements update <courseId> <announcementId> --text "Updated"
wk classroom announcements assignees <courseId> <announcementId> --mode INDIVIDUAL_STUDENTS --add-student <studentId>

# Topics
wk classroom topics list <courseId>
wk classroom topics create <courseId> --name "Unit 1"
wk classroom topics update <courseId> <topicId> --name "Unit 2"

# Invitations
wk classroom invitations list
wk classroom invitations create <courseId> <userId> --role student
wk classroom invitations accept <invitationId>

# Guardians
wk classroom guardians list <studentId>
wk classroom guardians get <studentId> <guardianId>
wk classroom guardians delete <studentId> <guardianId>

# Guardian invitations
wk classroom guardian-invitations list <studentId>
wk classroom guardian-invitations create <studentId> --email parent@example.com

# Profiles
wk classroom profile get
wk classroom profile get <userId>
```

Note: Classroom commands require a Google Workspace for Education account. Personal Google accounts have limited Classroom functionality.

### Docs

```bash
# Export (via Drive)
wk docs export <docId> --format pdf --out ./doc.pdf
wk docs export <docId> --format docx --out ./doc.docx
wk docs export <docId> --format txt --out ./doc.txt
```

### Slides

```bash
# Export (via Drive)
wk slides export <presentationId> --format pptx --out ./deck.pptx
wk slides export <presentationId> --format pdf --out ./deck.pdf
```

## Output Formats

### Text

Human-readable output with colors (default):

```bash
$ wk gmail search 'newer_than:7d' --max 3
THREAD_ID           SUBJECT                           FROM                  DATE
18f1a2b3c4d5e6f7    Meeting notes                     alice@example.com     2025-01-10
17e1d2c3b4a5f6e7    Invoice #12345                    billing@vendor.com    2025-01-09
16d1c2b3a4e5f6d7    Project update                    bob@example.com       2025-01-08
```

Message-level search (one row per email; add `--include-body` to fetch/decode bodies):

```bash
$ wk gmail messages search 'newer_than:7d' --max 3
ID                  THREAD             SUBJECT                           FROM                  DATE
18f1a2b3c4d5e6f7    9e8d7c6b5a4f3e2d    Meeting notes                     alice@example.com     2025-01-10
17e1d2c3b4a5f6e7    9e8d7c6b5a4f3e2d    Invoice #12345                    billing@vendor.com    2025-01-09
16d1c2b3a4e5f6d7    7f6e5d4c3b2a1908    Project update                    bob@example.com       2025-01-08
```

### JSON

Machine-readable output for scripting and automation:

```bash
$ wk gmail search 'newer_than:7d' --max 3 --json
{
  "threads": [
    {
      "id": "18f1a2b3c4d5e6f7",
      "snippet": "Meeting notes from today...",
      "messages": [...]
    },
    ...
  ]
}
```

```bash
$ wk gmail messages search 'newer_than:7d' --max 3 --json
{
  "messages": [
    {
      "id": "18f1a2b3c4d5e6f7",
      "threadId": "9e8d7c6b5a4f3e2d",
      "subject": "Meeting notes",
      "from": "alice@example.com",
      "date": "2025-01-10"
    },
    ...
  ]
}
```

```bash
$ wk gmail messages search 'newer_than:7d' --max 1 --include-body --json
{
  "messages": [
    {
      "id": "18f1a2b3c4d5e6f7",
      "threadId": "9e8d7c6b5a4f3e2d",
      "subject": "Meeting notes",
      "from": "alice@example.com",
      "date": "2025-01-10",
      "body": "Hi team — meeting notes..."
    }
  ]
}
```

Data goes to stdout, errors and progress to stderr for clean piping:

```bash
wk --json drive ls --max 5 | jq '.files[] | select(.mimeType=="application/pdf")'
```

Useful pattern:

- `wk --json ... | jq .`

Calendar JSON convenience fields:

- `startDayOfWeek` / `endDayOfWeek` on event payloads (derived from start/end).

## Examples

### Search recent emails and download attachments

```bash
# Search for emails from the last week
wk gmail search 'newer_than:7d has:attachment' --max 10

# Get thread details and download attachments
wk gmail thread get <threadId> --download
```

### Modify labels on a thread

```bash
# Archive and star a thread
wk gmail thread modify <threadId> --remove INBOX --add STARRED
```

### Create a calendar event with attendees

```bash
# Find a free time slot
wk calendar freebusy --calendars "primary" \
  --from 2025-01-15T00:00:00Z \
  --to 2025-01-16T00:00:00Z

# Create the meeting
wk calendar create primary \
  --summary "Team Standup" \
  --from 2025-01-15T10:00:00Z \
  --to 2025-01-15T10:30:00Z \
  --attendees "alice@example.com,bob@example.com"
```

### Find and download files from Drive

```bash
# Search for PDFs
wk drive search "invoice filetype:pdf" --max 20 --json | \
  jq -r '.files[] | .id' | \
  while read fileId; do
    wk drive download "$fileId"
  done
```

### Manage multiple accounts

```bash
# Check personal Gmail
wk gmail search 'is:unread' --account personal@gmail.com

# Check work Gmail
wk gmail search 'is:unread' --account work@company.com

# Or set default
export WK_ACCOUNT=work@company.com
wk gmail search 'is:unread'
```

### Update a Google Sheet from a CSV

```bash
# Convert CSV to pipe-delimited format and update sheet
cat data.csv | tr ',' '|' | \
  wk sheets update <spreadsheetId> 'Sheet1!A1'
```

### Export Sheets / Docs / Slides

```bash
# Sheets
wk sheets export <spreadsheetId> --format pdf

# Docs
wk docs export <docId> --format docx

# Slides
wk slides export <presentationId> --format pptx
```

### Batch process Gmail threads

```bash
# Mark all emails from a sender as read
wk --json gmail search 'from:noreply@example.com' --max 200 | \
  jq -r '.threads[].id' | \
  xargs -n 50 wk gmail labels modify --remove UNREAD

# Archive old emails
wk --json gmail search 'older_than:1y' --max 200 | \
  jq -r '.threads[].id' | \
  xargs -n 50 wk gmail labels modify --remove INBOX

# Label important emails
wk --json gmail search 'from:boss@example.com' --max 200 | \
  jq -r '.threads[].id' | \
  xargs -n 50 wk gmail labels modify --add IMPORTANT
```

## Advanced Features

### Verbose Mode

Enable verbose logging for troubleshooting:

```bash
wk --verbose gmail search 'newer_than:7d'
# Shows API requests and responses
```

## Agent Safety

When an AI agent drives `wk` on behalf of a user, you can restrict what it is
allowed to do. Three independent mechanisms are available and they stack -- all
filters are evaluated, and the most restrictive combination wins.

### `--read-only`

Block every write operation (send, upload, delete, create, etc.) and, when used
with `auth add --readonly`, request read-only OAuth scopes so the token itself
cannot perform mutations.

```bash
wk --read-only drive ls        # OK -- listing is read-only
wk --read-only gmail send ...  # BLOCKED
```

### `--command-tier core|extended|complete`

Limit which subcommands are visible. The three tiers are cumulative:

| Tier | What it includes |
|------|-----------------|
| **core** | Read-only essentials -- list, search, get, download, export, cat |
| **extended** | Common mutations -- create, update, delete, send, upload |
| **complete** | Everything (default) -- batch ops, permissions, settings, advanced |

Commands not assigned to a tier default to **complete**. Utility commands
(`auth`, `config`, `agent`, `version`, etc.) are always available regardless of
tier.

```bash
wk --command-tier core calendar ls      # OK
wk --command-tier core calendar create  # BLOCKED (create requires "extended")
```

### `--enable-commands <csv>`

Allowlist specific top-level commands. Only the listed service groups are
accessible; everything else is rejected.

```bash
wk --enable-commands calendar,tasks calendar ls   # OK
wk --enable-commands calendar,tasks gmail search   # BLOCKED
```

### Composing filters

All three mechanisms are independent and evaluated in order. A command must pass
every active filter to execute. For example:

```bash
wk --read-only --command-tier core --enable-commands drive,calendar \
    drive ls
```

This allows only read-only, core-tier subcommands of `drive` and `calendar`.

### Environment variables for agent sandboxing

Set these before launching an agent session so every invocation is
automatically restricted:

```bash
export WK_READ_ONLY=true
export WK_COMMAND_TIER=core
export WK_ENABLE_COMMANDS=drive,calendar
```

## Global Flags

All commands support these flags:

- `--account <email|alias|auto>` - Account to use (overrides WK_ACCOUNT)
- `--command-tier <core|extended|complete>` - Command visibility tier (default: complete; env: `WK_COMMAND_TIER`)
- `--enable-commands <csv>` - Allowlist top-level commands (e.g., `calendar,tasks`; env: `WK_ENABLE_COMMANDS`)
- `--read-only` - Hide write commands and request read-only OAuth scopes (env: `WK_READ_ONLY`)
- `--json` - Output JSON to stdout (best for scripting)
- `--plain` - Output stable, parseable text to stdout (TSV; no colors)
- `--color <mode>` - Color mode: `auto`, `always`, or `never` (default: auto)
- `--force` - Skip confirmations for destructive commands
- `--no-input` - Never prompt; fail instead (useful for CI)
- `--verbose` - Enable verbose logging
- `--help` - Show help for any command

## Shell Completions

Generate shell completions for your preferred shell:

### Bash

```bash
# macOS (with Homebrew)
wk completion bash > $(brew --prefix)/etc/bash_completion.d/wk

# Linux
wk completion bash > /etc/bash_completion.d/wk

# Or load directly in your current session
source <(wk completion bash)
```

### Zsh

```zsh
# Generate completion file
wk completion zsh > "${fpath[1]}/_wk"

# Or add to .zshrc for automatic loading
echo 'eval "$(wk completion zsh)"' >> ~/.zshrc

# Enable completions if not already enabled
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

### Fish

```fish
wk completion fish > ~/.config/fish/completions/wk.fish
```

### PowerShell

```powershell
# Load for current session
wk completion powershell | Out-String | Invoke-Expression

# Or add to profile for all sessions
wk completion powershell >> $PROFILE
```

After installing completions, start a new shell session for changes to take effect.

## Development

After cloning, install tools:

```bash
make tools
```

Pinned tools (installed into `.tools/`):

- Format: `make fmt` (goimports + gofumpt)
- Lint: `make lint` (golangci-lint)
- Test: `make test`

CI runs format checks, tests, lint, deadcode, race, and coverage gates on push/PR.

Required checks for protected branches (`main`, `dev`) should include at least:

- `ci / test`
- `ci / worker`
- `ci / darwin-cgo-build`
- `version / version-artifact`

Branch protection recommendation:
- Require pull requests before merge.
- Require all required checks to pass.
- Restrict direct pushes to `main`.
- Use `dev` as the integration branch and merge `dev -> main` for release promotion.

### Integration Tests (Live Google APIs)

Opt-in tests that hit real Google APIs using your stored `wk` credentials/tokens.

```bash
# Optional: override which account to use
export WK_IT_ACCOUNT=you@gmail.com
export WK_CLIENT=work
go test -tags=integration ./...
```

Tip: if you want to avoid macOS Keychain prompts during these runs, set `WK_KEYRING_BACKEND=file` and `WK_KEYRING_PASSWORD=...` (uses encrypted on-disk keyring).

### Live Test Script (CLI)

Fast end-to-end smoke checks against live APIs:

```bash
scripts/live-test.sh --fast
scripts/live-test.sh --account you@gmail.com --skip groups,keep,calendar-enterprise
scripts/live-test.sh --client work --account you@company.com
```

Script toggles:

- `--auth all,groups` to re-auth before running
- `--client <name>` to select OAuth client credentials
- `--strict` to fail on optional features (groups/keep/enterprise)
- `--allow-nontest` to override the test-account guardrail

Go test wrapper (opt-in):

```bash
WK_LIVE=1 go test -tags=integration ./internal/integration -run Live
```

Optional env:
- `WK_LIVE_FAST=1`
- `WK_LIVE_SKIP=groups,keep`
- `WK_LIVE_AUTH=all,groups`
- `WK_LIVE_ALLOW_NONTEST=1`
- `WK_LIVE_EMAIL_TEST=steipete+gogtest@gmail.com`
- `WK_LIVE_GROUP_EMAIL=group@domain`
- `WK_LIVE_CLASSROOM_COURSE=<courseId>`
- `WK_LIVE_CLASSROOM_CREATE=1`
- `WK_LIVE_CLASSROOM_ALLOW_STATE=1`
- `WK_LIVE_TRACK=1`
- `WK_LIVE_GMAIL_BATCH_DELETE=1`
- `WK_LIVE_GMAIL_FILTERS=1`
- `WK_LIVE_GMAIL_WATCH_TOPIC=projects/.../topics/...`
- `WK_LIVE_CALENDAR_RESPOND=1`
- `WK_LIVE_CALENDAR_RECURRENCE=1`
- `WK_KEEP_SERVICE_ACCOUNT=/path/to/service-account.json`
- `WK_KEEP_IMPERSONATE=user@workspace-domain`

### Make Shortcut

Build and run:

```bash
make wk auth add you@gmail.com
```

For clean stdout when scripting:

- Use `--` when the first arg is a flag: `make wk -- --json gmail search "from:me" | jq .`

## License

MIT

## Links

- [GitHub Repository](https://github.com/namastexlabs/workit)
- [Gmail API Documentation](https://developers.google.com/gmail/api)
- [Google Calendar API Documentation](https://developers.google.com/calendar)
- [Google Drive API Documentation](https://developers.google.com/drive)
- [Google People API Documentation](https://developers.google.com/people)
- [Google Tasks API Documentation](https://developers.google.com/tasks)
- [Google Sheets API Documentation](https://developers.google.com/sheets)
- [Cloud Identity API Documentation](https://cloud.google.com/identity/docs/reference/rest)

## Credits

This project is inspired by Mario Zechner's original CLIs:

- [gmcli](https://github.com/badlogic/gmcli)
- [gccli](https://github.com/badlogic/gccli)
- [gdcli](https://github.com/badlogic/gdcli)
