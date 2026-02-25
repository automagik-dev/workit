# workit Deployment Guide

## Quick Install (Automagik Servers)

```bash
# One-liner install with embedded credentials
curl -sL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/install.sh | bash
```

Or manually:

```bash
git clone https://github.com/automagik-dev/workit.git
cd workit
./scripts/setup-credentials.sh  # Creates ~/.config/workit/credentials.env
make build-automagik            # Builds with embedded OAuth credentials
cp bin/wk ~/.local/bin/
```

## Configuration Files

| File | Purpose |
|------|---------|
| `~/.config/workit/credentials.env` | OAuth client credentials (sourced by scripts) |
| `~/.config/workit/` | Token storage (when using file keyring) |

## OAuth Callback Server

The callback server handles headless OAuth for mobile users.

### Running with systemd

```bash
# Install service
sudo cp auth-server/wk-auth-server.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now wk-auth-server

# Check status
sudo systemctl status wk-auth-server
```

### Running manually

```bash
cd auth-server
source ~/.config/workit/credentials.env
./start.sh
```

### Architecture

```
┌─────────────────┐     ┌──────────────────────────┐     ┌─────────────────┐
│  Agent (wk)    │────▶│  auth.automagik.dev    │◀────│  User's Phone   │
│  --headless     │     │  (callback server)       │     │  (OAuth login)  │
└─────────────────┘     └──────────────────────────┘     └─────────────────┘
        │                         │
        └─────────────────────────┘
              Polls for token
```

## Usage

### Add Account (Headless)

```bash
# Generates auth URL for mobile login
wk auth add you@gmail.com --headless

# The CLI will print a URL - send to user's phone
# After they complete OAuth, token is stored automatically
```

### Sync Google Drive

```bash
# Initialize sync folder
wk sync init ~/GoogleDrive

# Start sync daemon
wk sync start --daemon

# Check status
wk sync status
```

### Gmail/Calendar/etc

```bash
wk gmail labels list
wk gmail send --to user@example.com --subject "Test" --body "Hello"
wk calendar events list
wk drive list
```

## Build Variants

| Command | Description |
|---------|-------------|
| `make build` | Standard build (credentials from env vars) |
| `make build-automagik` | Build with internal credentials embedded |
| `make build-internal` | Build with custom credentials (pass via args) |

## Credentials

**Project:** felipe-bot (felipe-bot-485616)  
**OAuth Client:** workit-headless  
**Callback URL:** https://auth.automagik.dev/callback  
**Audience:** Internal deployments
