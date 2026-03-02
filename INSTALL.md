# workit — Installation Guide

> Give your OpenClaw agent full access to Google Workspace (Gmail, Calendar, Drive, Docs, Sheets, Contacts, Tasks, and more) via CLI.

## 1. Quick Install (Recommended)

Install the latest release binary and Claude Code skill in one command:

```bash
curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh
```

This downloads the pre-built binary for your platform from GitHub Releases and installs it to `~/.local/bin/wk`.

### Installer Flags

Pass flags after a `--` separator when piping to `sh`:

```bash
# Force overwrite an existing installation
curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh -s -- --force

# Install a specific version
curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh -s -- --version v0.5.0

# Show installer help
curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh -s -- --help
```

| Flag | Description |
|------|-------------|
| `--force` | Overwrite existing installation without prompting |
| `--version VERSION` | Install a specific release tag (e.g. `v0.5.0`) |
| `--help` | Show installer usage |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `WK_RELEASE_URL` | Override the download URL (useful for offline/air-gapped testing) |

### Update

```bash
wk update
```

---

## 2. Configure Credentials

**Option A: Pre-built binary (recommended for internal use)**

If the binary was built with `make build-internal`, it already has OAuth credentials baked in. Skip to step 3.

**Option B: Environment variables**

```bash
export WK_CLIENT_ID="your-client-id"
export WK_CLIENT_SECRET="your-client-secret"
export WK_CALLBACK_SERVER="https://auth.example.com"
```

**Option C: Credentials file (standard workit way)**

```bash
wk auth credentials ~/path/to/client_secret.json
```

---

## 3. Authenticate a Google Account

### For agents (headless — no browser needed):

```bash
# Start headless auth flow
wk auth add you@gmail.com --headless --services=user

# Output:
#   Visit this URL to authorize:
#   https://accounts.google.com/o/oauth2/v2/auth?...
#   Waiting for authorization...
```

The agent sends this URL to the user (via WhatsApp, Telegram, etc). User taps the link on their phone, logs in, and the CLI automatically picks up the token.

### For interactive use (has browser):

```bash
wk auth add you@gmail.com --services=user
# Opens browser, complete login, done.
```

---

## 4. Set Up Keyring (Headless Environments)

On servers without a desktop keychain:

```bash
# Use file-based keyring
wk auth keyring file

# Set password via env (for non-interactive use)
export WK_KEYRING_PASSWORD="your-secure-password"
```

---

## 5. Verify It Works

```bash
# Check auth status
wk auth list --check

# Test some commands
wk gmail labels list --account you@gmail.com
wk drive list --account you@gmail.com
wk calendar events list --account you@gmail.com
```

---

## 6. Add to OpenClaw Config

Add `wk` to your agent's workspace. In your agent's `TOOLS.md`:

```markdown
## Google Workspace (workit)

Access Gmail, Calendar, Drive, Docs, Sheets, Contacts, Tasks via `wk` CLI.

### Auth
- `wk auth list --check` — Check authenticated accounts
- `wk auth add EMAIL --headless --services=user` — Auth a new account (sends URL for mobile login)
- `wk auth status` — Show current auth state

### Gmail
- `wk gmail search "newer_than:1d" --account EMAIL` — Search recent emails
- `wk gmail send --to X --subject Y --body Z --account EMAIL` — Send email
- `wk gmail labels list --account EMAIL` — List labels

### Calendar
- `wk calendar events list --account EMAIL` — List upcoming events
- `wk calendar events create --title "Meeting" --start "2025-01-15T10:00" --account EMAIL`

### Drive
- `wk drive list --account EMAIL` — List files
- `wk drive upload FILE --account EMAIL` — Upload a file
- `wk drive download FILE_ID --account EMAIL` — Download a file
- `wk drive search "name contains 'report'" --account EMAIL`

### Drive Sync (bidirectional)
- `wk sync init ~/drive-folder --drive-folder "My Folder" --account EMAIL`
- `wk sync start ~/drive-folder --daemon --account EMAIL`
- `wk sync status`
- `wk sync stop`

### Docs & Sheets
- `wk docs export DOC_ID --format pdf --account EMAIL`
- `wk sheets read SHEET_ID --range "A1:D10" --account EMAIL`
- `wk sheets write SHEET_ID --range "A1" --values '["hello","world"]' --account EMAIL`

### Contacts & Tasks
- `wk contacts search "John" --account EMAIL`
- `wk tasks list --account EMAIL`
- `wk tasks add "Buy milk" --account EMAIL`

### Tips
- Add `--json` to any command for machine-parseable output
- Use `--account EMAIL` or set `WK_ACCOUNT=EMAIL` globally
- Multiple accounts supported: `wk auth add second@gmail.com`
```

---

## 7. Drive Sync Setup (optional)

To sync a Google Drive folder to a local directory:

```bash
# Initialize sync for a folder
wk sync init ~/my-drive --drive-folder "Work Documents" --account you@gmail.com

# Start sync daemon (runs in background)
wk sync start ~/my-drive --daemon --account you@gmail.com

# Check status
wk sync status

# Stop sync
wk sync stop

# Add more folders
wk sync init ~/another-folder --drive-folder "Photos" --account you@gmail.com
wk sync start ~/another-folder --daemon --account you@gmail.com
```

Changes sync bidirectionally:
- Save a local file → uploaded to Drive within seconds
- Edit on Drive web → downloaded locally within 5 seconds

---

## 8. Build from Source (Developer / Contributor)

If you want to contribute to workit or build a custom binary, you need Go 1.21+ and the repo cloned locally.

### Requirements

- Go 1.21 or newer — [go.dev/doc/install](https://go.dev/doc/install)
- GNU Make

### Quick Build

```bash
git clone https://github.com/automagik-dev/workit.git
cd workit
make build
./bin/wk --help
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build binary to `bin/wk` (community build, no baked-in credentials) |
| `make build-automagik` | Build with Automagik OAuth credentials (requires internal secrets) |
| `make build-internal` | Build with internal-use baked-in credentials |
| `make tools` | Install dev tools (`gofumpt`, `goimports`, `golangci-lint`) to `.tools/` |
| `make test` | Run test suite |
| `make lint` | Run linter |
| `make install` | Install `bin/wk` to `~/.local/bin` |

### Developer Credential Setup

When building from source you need to supply your own OAuth credentials:

```bash
# Option A: Environment variables
export WK_CLIENT_ID="your-client-id"
export WK_CLIENT_SECRET="your-client-secret"
export WK_CALLBACK_SERVER="https://auth.example.com"

# Option B: Credentials file
mkdir -p ~/.config/workit && chmod 700 ~/.config/workit
cat > ~/.config/workit/credentials.env << 'CRED'
WK_CLIENT_ID=your-client-id
WK_CLIENT_SECRET=your-client-secret
WK_CALLBACK_SERVER=https://your-callback-server.example.com
CRED
```

Then build and install:

```bash
make build
make install   # copies bin/wk to ~/.local/bin
```

After building, proceed with [Configure Credentials](#2-configure-credentials) above.

---

## TL;DR for Agents

```bash
# Install
curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh

# Update
wk update

# Configure (if not using pre-built binary)
export WK_CLIENT_ID="xxx"
export WK_CLIENT_SECRET="xxx"
export WK_CALLBACK_SERVER="https://auth.example.com"
export WK_KEYRING_BACKEND=file
export WK_KEYRING_PASSWORD="secure-password"

# Auth
wk auth add user@gmail.com --headless --services=user
# → Send the URL to user → they login → done

# Use
wk gmail search "is:unread" --account user@gmail.com --json
wk drive list --account user@gmail.com --json
wk calendar events list --account user@gmail.com --json
```
