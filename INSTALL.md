# workit — Installation Guide

> Give your agent full access to Google Workspace (Gmail, Calendar, Drive, Docs, Sheets, Contacts, Tasks, and more) via CLI.

---

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

## 2. What Gets Installed

The installer places the following on your system:

| Path | Description |
|------|-------------|
| `~/.local/bin/wk` | The `wk` CLI binary |
| `~/.local/bin/gog` | The `gog` companion binary |
| `~/.workit/plugin/` | Plugin directory containing 28 skills |
| `~/.claude/plugins/workit` | Symlink registering the plugin with Claude Code |

The Claude Code symlink means Claude Code automatically picks up the `wk` skills — no manual configuration required.

---

## 3. Authentication

Authentication uses the **Automagik relay** at `auth.automagik.dev`. No GCP project setup is needed by default.

### Interactive (default)

```bash
wk auth manage
```

This opens a browser-based flow. Log in with your Google account and the token is stored locally.

### Headless (agents and servers without a browser)

```bash
wk auth add user@example.com --headless --no-input
```

The command prints an authorization URL. Send that URL to the user (via chat, email, etc.). When they complete the login in their browser, the CLI automatically receives the token.

### Check Auth Status

```bash
# Show auth state for all accounts
wk auth status

# List authenticated accounts
wk auth list
```

---

## 4. Advanced: BYO GCP Credentials (Optional)

> This section is for users who want to use their own Google Cloud OAuth client instead of the Automagik relay. Most users do not need this.

### Steps

1. Go to [Google Cloud Console](https://console.cloud.google.com/) and create an OAuth 2.0 client ID (Desktop app type).
2. Download the `client_secret_*.json` file.
3. Register it with `wk`:

```bash
wk auth credentials ~/path/to/client_secret.json
```

For environments with multiple OAuth clients:

```bash
# Register and name a specific client
wk auth credentials ~/path/to/client_secret.json --name my-project

# List registered credential sets
wk auth credentials list

# Select which credential set to use
wk auth credentials use my-project
```

After registering credentials, run `wk auth manage` or `wk auth add` as normal.

---

## 5. Keyring Configuration

`wk` stores tokens in the system keyring. The default backend is chosen automatically:

| Platform | Default Backend |
|----------|----------------|
| macOS | macOS Keychain |
| Linux (desktop) | GNOME Keyring |
| Windows | Windows Credential Manager |
| Linux (headless) | File backend (see below) |

### File Backend (Headless / Server Environments)

On servers without a desktop keyring, use the file backend:

```bash
export WK_KEYRING_BACKEND=file
export WK_KEYRING_PASSWORD="your-secure-passphrase"
```

Add these to your shell profile or systemd environment to persist across sessions.

### Service Account

For server-to-server automation without interactive login:

```bash
wk auth service-account set --key /path/to/key.json impersonate@company.com
```

---

## 6. Build from Source (Developer / Contributor)

If you want to contribute to workit or build a custom binary, you need Go 1.21+ and GNU Make.

### Requirements

- Go 1.21 or newer — [go.dev/doc/install](https://go.dev/doc/install)
- GNU Make

### Steps

```bash
git clone https://github.com/automagik-dev/workit.git
cd workit
make build
./bin/wk --help
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build binary to `bin/wk` |
| `make test` | Run test suite |
| `make lint` | Run linter |
| `make install` | Install `bin/wk` to `~/.local/bin` |

After building from source, authenticate using `wk auth manage` or the headless flow described in [Authentication](#3-authentication).

---

## TL;DR for Agents

Copy-paste block to get up and running immediately. Uses relay auth — no GCP setup or credential env vars needed.

```bash
# Install
curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh

# Ensure wk is on PATH (add to shell profile if needed)
export PATH="$HOME/.local/bin:$PATH"

# Check auth status
wk auth status

# Authenticate interactively (opens browser)
wk auth manage

# Authenticate headlessly (prints URL for user to open)
# wk auth add user@example.com --headless --no-input

# First queries
wk gmail search "is:unread" --account user@example.com --json
wk drive ls --account user@example.com --json
wk calendar events --account user@example.com --json
```
