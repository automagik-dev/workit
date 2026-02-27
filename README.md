# workit -- your office, from the terminal.

<p align="center">
  <a href="https://github.com/automagik-dev/workit/blob/main/LICENSE"><img src="https://img.shields.io/github/license/automagik-dev/workit" alt="License"></a>
  <a href="https://github.com/automagik-dev/workit/releases"><img src="https://img.shields.io/github/v/release/automagik-dev/workit" alt="Release"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/automagik-dev/workit" alt="Go"></a>
</p>

<p align="center">
  Fast, agent-native CLI for Google Workspace. 18+ services, local DOCX editing, JSON-first output, least-privilege auth.
</p>

<p align="center">
  <a href="#install">Install</a> &middot;
  <a href="#quick-start">Quick Start</a> &middot;
  <a href="#agent-features">Agent Features</a> &middot;
  <a href="#services">Services</a> &middot;
  <a href="#skills">Skills</a> &middot;
  <a href="#documentation">Docs</a>
</p>

---

## Why workit?

| | |
|---|---|
| **Agent-native** | JSON-first output, field discovery (`--select ""`), input templates (`--generate-input`) |
| **Agent safety** | `--read-only`, `--command-tier`, `--enable-commands` sandboxing |
| **18+ services** | Gmail, Calendar, Drive, Docs, Sheets, DOCX editing, and more -- one binary |
| **Headless OAuth** | Auth on mobile/web, run on server. Perfect for agents |
| **Multi-account** | Switch accounts, OAuth clients per domain, aliases |
| **Least-privilege** | Per-service scopes, `--readonly`, file-only Drive scope |

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
| **Templates** | Manage reusable DOCX templates, inspect `{{PLACEHOLDER}}` patterns, fill from JSON |
| **Time** | Local/UTC time display for scripts and agents |

---

## Install

### One Command (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/install.sh | bash
```

This installs the latest release binary and bootstraps local skills.

### Update

```bash
wk update
```

<details>
<summary>Other methods</summary>

#### Arch User Repository

```bash
yay -S workit
```

#### Build from source

```bash
git clone https://github.com/automagik-dev/workit.git
cd workit
make
./bin/wk --help
```

</details>

---

## Quick Start

### 1. Store OAuth credentials

Download a Desktop app OAuth client JSON from [Google Cloud Console](https://console.cloud.google.com/apis/credentials), then:

```bash
wk auth credentials ~/Downloads/client_secret_....json
```

### 2. Authorize your account

```bash
wk auth manage
```

This opens the account manager UI. On headless/remote servers it binds to `0.0.0.0`, shows your outbound IP, and auto-closes after auth — no tunnels needed. See [docs/headless-auth.md](docs/headless-auth.md).

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
```

### Safety controls

```bash
# Read-only mode -- blocks all write operations
wk --read-only drive ls

# Restrict to specific services
wk --enable-commands calendar,tasks calendar events --today

# Tier-based access control
wk --command-tier core calendar ls       # OK
wk --command-tier core calendar create   # BLOCKED

# Combine for maximum restriction
wk --read-only --command-tier core --enable-commands drive,calendar drive ls
```

### Built-in help topics

```bash
wk agent help topics      # list all topics
wk agent help auth        # authentication guide
wk agent help output      # output modes, --json, --select, exit codes
wk agent help agent       # zero-shot patterns, recommended flags
```

Full details: [docs/agent.md](docs/agent.md)

---

## Skills

workit ships as a [Claude Code skill](https://docs.anthropic.com/en/docs/claude-code/skills) with 25+ playbooks covering all Google services and DOCX editing.

### What the skills provide

- Intent-based routing: describe what you want, the router loads the right playbook
- Safety defaults: `--read-only` for reads, `--dry-run` preview before writes
- Full DOCX coverage: read, edit, create, tracked changes, tables, PDF export
- Per-service reference: flags, examples, and gotchas for each Google API

### Install

The skills are distributed via the [automagik-dev marketplace](https://github.com/automagik-dev). To install:

```bash
# If you already have the automagik-dev marketplace, skills are available automatically.
# Otherwise, add the marketplace to your Claude Code plugins:
# ~/.claude/plugins/marketplaces/automagik-dev/
```

Once installed, invoke with `/wk` in Claude Code. The router auto-loads the right playbook based on intent.

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

## Security

Credentials are stored in your OS keychain (macOS Keychain, GNOME Keyring, Windows Credential Manager) via [keyring](https://github.com/99designs/keyring). For headless environments, use the encrypted file backend with `WK_KEYRING_BACKEND=file` and `WK_KEYRING_PASSWORD`.

Never commit OAuth client credential JSON files. Use `--readonly` scopes when exploring. Use `--read-only` runtime flag to prevent accidental writes.

Full details: [docs/auth.md](docs/auth.md)

---

<p align="center">
  MIT License &middot; <a href="https://github.com/automagik-dev/workit">github.com/automagik-dev/workit</a>
</p>
