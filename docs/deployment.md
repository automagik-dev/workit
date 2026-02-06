# gog-cli Deployment Guide

## Quick Install (Namastex Servers)

```bash
# One-liner install with embedded credentials
curl -sL https://raw.githubusercontent.com/automagik-genie/gog-cli/main/scripts/install.sh | bash
```

Or manually:

```bash
git clone https://github.com/automagik-genie/gog-cli.git
cd gog-cli
./scripts/setup-credentials.sh  # Creates ~/.config/gog/credentials.env
make build-namastex             # Builds with embedded OAuth credentials
cp bin/gog ~/.local/bin/
```

## Configuration Files

| File | Purpose |
|------|---------|
| `~/.config/gog/credentials.env` | OAuth client credentials (sourced by scripts) |
| `~/.config/gog/` | Token storage (when using file keyring) |

## OAuth Callback Server

The callback server handles headless OAuth for mobile users.

### Running with systemd

```bash
# Install service
sudo cp auth-server/gog-auth-server.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now gog-auth-server

# Check status
sudo systemctl status gog-auth-server
```

### Running manually

```bash
cd auth-server
source ~/.config/gog/credentials.env
./start.sh
```

### Architecture

```
┌─────────────────┐     ┌──────────────────────────┐     ┌─────────────────┐
│  Agent (gog)    │────▶│  gogoauth.namastex.io    │◀────│  User's Phone   │
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
gog auth add you@gmail.com --headless

# The CLI will print a URL - send to user's phone
# After they complete OAuth, token is stored automatically
```

### Sync Google Drive

```bash
# Initialize sync folder
gog sync init ~/GoogleDrive

# Start sync daemon
gog sync start --daemon

# Check status
gog sync status
```

### Gmail/Calendar/etc

```bash
gog gmail labels list
gog gmail send --to user@example.com --subject "Test" --body "Hello"
gog calendar events list
gog drive list
```

## Build Variants

| Command | Description |
|---------|-------------|
| `make build` | Standard build (credentials from env vars) |
| `make build-namastex` | Build with Namastex credentials embedded |
| `make build-internal` | Build with custom credentials (pass via args) |

## Credentials

**Project:** felipe-bot (felipe-bot-485616)  
**OAuth Client:** gog-cli-headless  
**Callback URL:** https://gogoauth.namastex.io/callback  
**Audience:** Internal (namastex.ai domain only)
