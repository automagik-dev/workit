# workit — Quick Install for OpenClaw Agents

> Give your OpenClaw agent full access to Google Workspace (Gmail, Calendar, Drive, Docs, Sheets, Contacts, Tasks, and more) via CLI.

## 1. Install the Binary

```bash
# Clone and build
git clone https://github.com/namastexlabs/workit.git
cd workit
make

# Move to PATH
sudo cp bin/wk /usr/local/bin/wk
```

Or, if you have a pre-built binary with Namastex credentials baked in:
```bash
# Just copy the binary to your agent's PATH
cp wk /usr/local/bin/wk
```

## 2. Configure Credentials

**Option A: Pre-built binary (recommended for internal use)**

If the binary was built with `make build-internal`, it already has OAuth credentials. Skip to step 3.

**Option B: Environment variables**

```bash
export WK_CLIENT_ID="your-client-id"
export WK_CLIENT_SECRET="your-client-secret"
export WK_CALLBACK_SERVER="https://auth.namastex.io"
```

**Option C: Credentials file (standard workit way)**

```bash
wk auth credentials ~/path/to/client_secret.json
```

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

## 4. Set Up Keyring (Headless Environments)

On servers without a desktop keychain:

```bash
# Use file-based keyring
wk auth keyring file

# Set password via env (for non-interactive use)
export WK_KEYRING_PASSWORD="your-secure-password"
```

## 5. Verify It Works

```bash
# Check auth status
wk auth list --check

# Test some commands
wk gmail labels list --account you@gmail.com
wk drive list --account you@gmail.com
wk calendar events list --account you@gmail.com
```

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

## TL;DR for Agents

```bash
# Install
git clone https://github.com/namastexlabs/workit.git && cd workit && make && sudo cp bin/wk /usr/local/bin/

# Configure (if not using pre-built binary)
export WK_CLIENT_ID="xxx"
export WK_CLIENT_SECRET="xxx" 
export WK_CALLBACK_SERVER="https://auth.namastex.io"
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
