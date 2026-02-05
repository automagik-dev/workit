# gog-cli — Quick Install for OpenClaw Agents

> Give your OpenClaw agent full access to Google Workspace (Gmail, Calendar, Drive, Docs, Sheets, Contacts, Tasks, and more) via CLI.

## 1. Install the Binary

```bash
# Clone and build
git clone https://github.com/namastexlabs/gog-cli.git
cd gog-cli
make

# Move to PATH
sudo cp bin/gog /usr/local/bin/gog
```

Or, if you have a pre-built binary with Namastex credentials baked in:
```bash
# Just copy the binary to your agent's PATH
cp gog /usr/local/bin/gog
```

## 2. Configure Credentials

**Option A: Pre-built binary (recommended for internal use)**

If the binary was built with `make build-internal`, it already has OAuth credentials. Skip to step 3.

**Option B: Environment variables**

```bash
export GOG_CLIENT_ID="your-client-id"
export GOG_CLIENT_SECRET="your-client-secret"
export GOG_CALLBACK_SERVER="https://auth.namastex.io"
```

**Option C: Credentials file (standard gogcli way)**

```bash
gog auth credentials ~/path/to/client_secret.json
```

## 3. Authenticate a Google Account

### For agents (headless — no browser needed):

```bash
# Start headless auth flow
gog auth add you@gmail.com --headless --services=user

# Output:
#   Visit this URL to authorize:
#   https://accounts.google.com/o/oauth2/v2/auth?...
#   Waiting for authorization...
```

The agent sends this URL to the user (via WhatsApp, Telegram, etc). User taps the link on their phone, logs in, and the CLI automatically picks up the token.

### For interactive use (has browser):

```bash
gog auth add you@gmail.com --services=user
# Opens browser, complete login, done.
```

## 4. Set Up Keyring (Headless Environments)

On servers without a desktop keychain:

```bash
# Use file-based keyring
gog auth keyring file

# Set password via env (for non-interactive use)
export GOG_KEYRING_PASSWORD="your-secure-password"
```

## 5. Verify It Works

```bash
# Check auth status
gog auth list --check

# Test some commands
gog gmail labels list --account you@gmail.com
gog drive list --account you@gmail.com
gog calendar events list --account you@gmail.com
```

## 6. Add to OpenClaw Config

Add `gog` to your agent's workspace. In your agent's `TOOLS.md`:

```markdown
## Google Workspace (gog-cli)

Access Gmail, Calendar, Drive, Docs, Sheets, Contacts, Tasks via `gog` CLI.

### Auth
- `gog auth list --check` — Check authenticated accounts
- `gog auth add EMAIL --headless --services=user` — Auth a new account (sends URL for mobile login)
- `gog auth status` — Show current auth state

### Gmail
- `gog gmail search "newer_than:1d" --account EMAIL` — Search recent emails
- `gog gmail send --to X --subject Y --body Z --account EMAIL` — Send email
- `gog gmail labels list --account EMAIL` — List labels

### Calendar
- `gog calendar events list --account EMAIL` — List upcoming events
- `gog calendar events create --title "Meeting" --start "2025-01-15T10:00" --account EMAIL`

### Drive
- `gog drive list --account EMAIL` — List files
- `gog drive upload FILE --account EMAIL` — Upload a file
- `gog drive download FILE_ID --account EMAIL` — Download a file
- `gog drive search "name contains 'report'" --account EMAIL`

### Drive Sync (bidirectional)
- `gog sync init ~/drive-folder --drive-folder "My Folder" --account EMAIL`
- `gog sync start ~/drive-folder --daemon --account EMAIL`
- `gog sync status`
- `gog sync stop`

### Docs & Sheets
- `gog docs export DOC_ID --format pdf --account EMAIL`
- `gog sheets read SHEET_ID --range "A1:D10" --account EMAIL`
- `gog sheets write SHEET_ID --range "A1" --values '["hello","world"]' --account EMAIL`

### Contacts & Tasks
- `gog contacts search "John" --account EMAIL`
- `gog tasks list --account EMAIL`
- `gog tasks add "Buy milk" --account EMAIL`

### Tips
- Add `--json` to any command for machine-parseable output
- Use `--account EMAIL` or set `GOG_ACCOUNT=EMAIL` globally
- Multiple accounts supported: `gog auth add second@gmail.com`
```

## 7. Drive Sync Setup (optional)

To sync a Google Drive folder to a local directory:

```bash
# Initialize sync for a folder
gog sync init ~/my-drive --drive-folder "Work Documents" --account you@gmail.com

# Start sync daemon (runs in background)
gog sync start ~/my-drive --daemon --account you@gmail.com

# Check status
gog sync status

# Stop sync
gog sync stop

# Add more folders
gog sync init ~/another-folder --drive-folder "Photos" --account you@gmail.com
gog sync start ~/another-folder --daemon --account you@gmail.com
```

Changes sync bidirectionally:
- Save a local file → uploaded to Drive within seconds
- Edit on Drive web → downloaded locally within 5 seconds

---

## TL;DR for Agents

```bash
# Install
git clone https://github.com/namastexlabs/gog-cli.git && cd gog-cli && make && sudo cp bin/gog /usr/local/bin/

# Configure (if not using pre-built binary)
export GOG_CLIENT_ID="xxx"
export GOG_CLIENT_SECRET="xxx" 
export GOG_CALLBACK_SERVER="https://auth.namastex.io"
export GOG_KEYRING_BACKEND=file
export GOG_KEYRING_PASSWORD="secure-password"

# Auth
gog auth add user@gmail.com --headless --services=user
# → Send the URL to user → they login → done

# Use
gog gmail search "is:unread" --account user@gmail.com --json
gog drive list --account user@gmail.com --json
gog calendar events list --account user@gmail.com --json
```
