# workit -- your office, from the terminal.

<p align="center">
  <a href="https://github.com/automagik-dev/workit/blob/main/LICENSE"><img src="https://img.shields.io/github/license/automagik-dev/workit" alt="License"></a>
  <a href="https://github.com/automagik-dev/workit/releases"><img src="https://img.shields.io/github/v/release/automagik-dev/workit" alt="Release"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/automagik-dev/workit" alt="Go"></a>
</p>

<p align="center">
  Agent-native CLI for Google Workspace. 18+ services, headless OAuth, local DOCX editing, JSON-first output — one binary.
</p>

<p align="center">
  <a href="#install">Install</a> &middot;
  <a href="#quick-start">Quick Start</a> &middot;
  <a href="#services">Services</a> &middot;
  <a href="#agent-features">Agent Features</a> &middot;
  <a href="#docx-editing">DOCX Editing</a> &middot;
  <a href="#skills--plugin">Skills</a> &middot;
  <a href="#documentation">Docs</a>
</p>

---

## Why workit?

| | |
|---|---|
| **Agent-native** | JSON-first output, field discovery (`--select ""`), input templates (`--generate-input`), predictable exit codes |
| **No GCP setup** | Auth relay at `auth.automagik.dev` — works out of the box, no Google Cloud Console required |
| **Headless OAuth** | Authorize on any device, run on any server — no tunnels, no browser required on the server |
| **18+ services** | Gmail, Calendar, Drive, Docs, Sheets, Slides, Chat, DOCX editing and more — one binary |
| **Agent safety** | `--read-only`, `--dry-run`, `--command-tier`, `--enable-commands` sandboxing |
| **Multi-account** | Named aliases, multiple OAuth clients, per-domain configuration |
| **Least-privilege** | Per-service scopes, runtime read-only flag, file-only Drive scope |

---

## Install

### One Command (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh
```

This installs:
- The `wk` binary to `~/.local/bin/wk`
- The Claude Code plugin (28 skill files) to `~/.workit/plugin/` with a symlink at `~/.claude/plugins/workit`

The release also includes `gog`, a git operations companion tool, installed alongside `wk`.

### Update

```bash
wk update
```

See [INSTALL.md](INSTALL.md) for detailed install options.

<details>
<summary>Build from source</summary>

Requires Go 1.21+.

```bash
git clone https://github.com/automagik-dev/workit.git
cd workit
make build
./bin/wk --help
```

</details>

---

## Quick Start

No Google Cloud Console setup needed. The auth relay handles OAuth for you.

### 1. Check auth status

```bash
wk auth status
```

### 2. Authorize your account

```bash
wk auth manage
```

This opens the account manager UI. On headless or remote servers it prints a URL — open it on any device to complete authorization. See [docs/headless-auth.md](docs/headless-auth.md).

For fully non-interactive agent environments:

```bash
wk auth add user@example.com --headless --no-input
# Prints an authorization URL, then polls until the token arrives
```

### 3. First query

```bash
export WK_ACCOUNT=you@gmail.com

# Search recent emails
wk gmail search 'newer_than:7d' --json --select "id,snippet"

# List Drive files
wk drive ls --json --select "name,id,mimeType"

# Edit a local DOCX (no auth needed)
wk docx cat report.docx
wk docx replace report.docx "DRAFT" "FINAL" -o report.docx
```

---

<details>
<summary>Quick Start -- BYO GCP (bring your own OAuth client)</summary>

For users who want to use their own Google Cloud project and OAuth client instead of the shared relay:

### 1. Create OAuth credentials

In [Google Cloud Console](https://console.cloud.google.com/apis/credentials), create a Desktop app OAuth client and download the JSON file.

### 2. Store credentials

```bash
wk auth credentials ~/Downloads/client_secret_....json
```

### 3. Authorize your account

```bash
wk auth manage
```

### 4. First query

```bash
export WK_ACCOUNT=you@gmail.com
wk gmail search 'newer_than:7d' --json
```

Full details on OAuth clients and per-domain configuration: [docs/auth-clients.md](docs/auth-clients.md)

</details>

---

## Services

| Service | Highlights |
|---------|-----------|
| **Gmail** | Search, send, labels, drafts, filters, delegation, open-tracking, Pub/Sub watch |
| **Calendar** | Events, conflicts, free/busy, team calendars, focus/OOO, recurrence |
| **Chat** | Spaces, messages, threads, DMs (Workspace) |
| **Classroom** | Courses, roster, coursework, submissions, announcements, guardians |
| **Drive** | List, search, upload, download, permissions, comments, shared drives |
| **Drive Sync** | Bidirectional folder sync with daemon mode and conflict resolution |
| **Docs** | Export PDF/DOCX, create, copy, full-text extraction |
| **DOCX** | Local editing: find/replace, insert, tracked changes, comments, tables, rewrite from markdown |
| **Slides** | Create, export PPTX/PDF, speaker notes, markdown decks |
| **Sheets** | Read/write ranges, format cells, create sheets, export via Drive |
| **Forms** | Create/get forms, inspect responses |
| **Apps Script** | Create/get projects, inspect content, run functions |
| **Contacts** | Search, create, update, Workspace directory, other contacts |
| **Tasks** | Tasklists, create/update/complete/undo, repeat schedules |
| **People** | Profile information, directory |
| **Keep** | List/get/search notes, download attachments (Workspace, service account) |
| **Groups** | List groups, view members (Workspace) |
| **Templates** | Manage reusable DOCX templates, inspect placeholders, fill from JSON |
| **Time** | Local/UTC time display for scripts and agents |

---

## Agent Features

Designed for AI agents that need structured, predictable output and safety guardrails.

### Structured output

```bash
# JSON output with field selection
wk gmail search 'from:boss' --json --select "id,snippet,date"

# Discover available fields for any command
wk drive ls --json --select ""

# Generate input template for any command
wk gmail send --generate-input

# Filter output with jq expressions
wk drive ls --json --jq '.[].name'
```

### Safety controls

```bash
# Read-only mode -- blocks all write operations
wk --read-only drive ls

# Dry-run mode -- preview what would happen without executing
wk --dry-run gmail send --to user@example.com --subject "Test" --body "Hello"

# Restrict to specific services
wk --enable-commands calendar,tasks calendar events --today

# Tier-based access control
wk --command-tier core calendar ls       # OK
wk --command-tier core calendar create   # BLOCKED

# Non-interactive mode -- never prompt, fail instead
wk --no-input drive upload file.pdf

# Combine for maximum restriction
wk --read-only --command-tier core --enable-commands drive,calendar drive ls
```

| Flag | Effect |
|------|--------|
| `--read-only` | Blocks all write, delete, and send operations |
| `--dry-run` | Previews the operation without executing |
| `--command-tier core\|extended\|complete` | Restricts available commands by tier |
| `--enable-commands <list>` | Whitelist specific services or commands |
| `--no-input` | Fails instead of prompting for missing input |
| `--generate-input` | Prints a JSON input template for the command |

### Built-in help topics

```bash
wk agent help topics      # list all topics
wk agent help auth        # authentication guide
wk agent help output      # output modes, --json, --select, exit codes
wk agent help agent       # zero-shot patterns, recommended flags
```

Full details: [docs/agent.md](docs/agent.md)

---

## DOCX Editing

DOCX editing is fully local — no Google account or internet connection required.

```bash
# Read document content
wk docx cat report.docx

# Find and replace text
wk docx replace report.docx "DRAFT" "FINAL" -o report.docx

# Insert content at a bookmark
wk docx insert report.docx --bookmark SECTION_1 --text "New paragraph."

# Add tracked changes (review mode)
wk docx replace report.docx "old text" "new text" --tracked-changes

# Rewrite document from markdown
wk docx rewrite report.docx --from-markdown content.md

# Export to PDF
wk docx export report.docx --format pdf -o report.pdf
```

Key capabilities: find/replace, insert at bookmarks, tracked changes, comments, table manipulation, rewrite from markdown, PDF export. See [docs/commands.md](docs/commands.md) for full reference.

---

## Skills & Plugin

The `install.sh` script automatically installs the Claude Code plugin alongside the `wk` binary. No manual marketplace setup required.

**What gets installed:**
- 28 skill files covering all Google services and DOCX editing
- Installed to `~/.workit/plugin/` with a symlink at `~/.claude/plugins/workit`
- Loaded automatically by Claude Code on startup

**What the skills provide:**
- Intent-based routing: describe what you want, the router loads the right playbook
- Safety defaults: `--read-only` for reads, `--dry-run` preview before writes
- Full DOCX coverage: read, edit, create, tracked changes, tables, PDF export
- Per-service reference: flags, examples, and gotchas for each Google API

Once installed, invoke with `/wk` in Claude Code. The router auto-loads the right playbook based on intent.

---

## Configuration

### Multi-account

```bash
# List configured accounts
wk auth list

# Add a second account
wk auth add other@example.com

# Use a specific account for one command
wk --account other@example.com gmail search 'is:unread'

# Set a default account via environment variable
export WK_ACCOUNT=you@gmail.com
```

### Keyring backends

Credentials are stored in your OS keychain by default:
- macOS: Keychain
- Linux: GNOME Keyring / libsecret
- Windows: Credential Manager

For headless environments (servers, CI, agents):

```bash
export WK_KEYRING_BACKEND=file
export WK_KEYRING_PASSWORD=your-encryption-passphrase
```

### Headless auth

For environments where a browser cannot open:

```bash
# Interactive prompt with URL
wk auth manage

# Fully non-interactive: prints URL, polls for token
wk auth add user@example.com --headless --no-input
```

Full details: [docs/headless-auth.md](docs/headless-auth.md)

---

## Security

- Credentials are stored in your OS keychain via [keyring](https://github.com/99designs/keyring). For headless environments, use the encrypted file backend with `WK_KEYRING_BACKEND=file` and `WK_KEYRING_PASSWORD`.
- OAuth scopes are requested per-service — only the services you use request permissions.
- Use `--read-only` at runtime to prevent accidental writes during exploration or agent execution.
- Never commit OAuth client credential JSON files to version control.

Full details: [docs/auth.md](docs/auth.md)

---

## Documentation

| Topic | Link |
|-------|------|
| Authentication | [docs/auth.md](docs/auth.md) |
| OAuth Clients | [docs/auth-clients.md](docs/auth-clients.md) |
| Headless Auth | [docs/headless-auth.md](docs/headless-auth.md) |
| Command Reference | [docs/commands.md](docs/commands.md) |
| Service Scopes | [docs/services.md](docs/services.md) |
| Configuration | [docs/configuration.md](docs/configuration.md) |
| Agent Features | [docs/agent.md](docs/agent.md) |
| Drive Sync | [docs/sync.md](docs/sync.md) |
| Email Tracking | [docs/email-tracking.md](docs/email-tracking.md) |
| Development | [docs/development.md](docs/development.md) |

---

<p align="center">
  MIT License &middot; <a href="https://github.com/automagik-dev/workit">github.com/automagik-dev/workit</a>
</p>
