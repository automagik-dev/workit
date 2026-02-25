# Infrastructure Setup

This guide covers setting up the infrastructure needed for workit's headless OAuth and sync features.

## Overview

workit requires:

1. **Google Cloud Project** with OAuth credentials
2. **Callback Server** for headless authentication

```
┌────────────────────────────────────────────────────────────────┐
│                         Your Infrastructure                     │
├────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐      ┌──────────────────┐                      │
│  │  workit    │─────▶│  Callback Server │                      │
│  │  (agents)   │      │  (auth.x.io)     │                      │
│  └─────────────┘      └──────────────────┘                      │
│         │                      │                                 │
│         │                      │                                 │
│         ▼                      ▼                                 │
│  ┌─────────────────────────────────────────┐                    │
│  │            Google Cloud Project          │                    │
│  │  - OAuth 2.0 Credentials                │                    │
│  │  - Drive API enabled                    │                    │
│  │  - Gmail API enabled (if needed)        │                    │
│  └─────────────────────────────────────────┘                    │
│                                                                  │
└────────────────────────────────────────────────────────────────┘
```

## 1. Google Cloud Project Setup

### Create Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Note the **Project ID**

### Enable APIs

Enable the APIs you need:

```bash
# Using gcloud CLI
gcloud services enable drive.googleapis.com
gcloud services enable gmail.googleapis.com
gcloud services enable calendar-json.googleapis.com
gcloud services enable people.googleapis.com
```

Or enable in Console:
- [Drive API](https://console.cloud.google.com/apis/library/drive.googleapis.com)
- [Gmail API](https://console.cloud.google.com/apis/library/gmail.googleapis.com)
- [Calendar API](https://console.cloud.google.com/apis/library/calendar-json.googleapis.com)

### Configure OAuth Consent Screen

1. Go to **APIs & Services → OAuth consent screen**
2. Choose **External** (or Internal for Workspace)
3. Fill in:
   - App name: Your app name
   - User support email: Your email
   - Developer contact: Your email
4. Add scopes (optional for consent screen)
5. Add test users (if in Testing mode)

### Create OAuth Credentials

1. Go to **APIs & Services → Credentials**
2. Click **Create Credentials → OAuth client ID**
3. Choose **Web application**
4. Add authorized redirect URIs:
   ```
   https://auth.yourdomain.com/callback
   http://localhost:8080/callback  (for development)
   ```
5. Download the JSON credentials

### Publish App (Optional)

For production use with external users:
1. Go to OAuth consent screen
2. Click **Publish App**
3. Complete verification if needed

## 2. Callback Server Setup

The callback server handles OAuth redirects for headless authentication.

### Option A: Docker Deployment

```bash
# Build the image
cd auth-server
docker build -t wk-auth-server .

# Run
docker run -d \
  --name wk-auth \
  -p 8080:8080 \
  -e WK_CLIENT_ID=your-client-id \
  -e WK_CLIENT_SECRET=your-client-secret \
  -e WK_REDIRECT_URL=https://auth.yourdomain.com/callback \
  wk-auth-server
```

### Option B: Direct Binary

```bash
# Build
cd auth-server
go build -o auth-server .

# Run
./auth-server \
  --port=8080 \
  --client-id=your-client-id \
  --client-secret=your-client-secret \
  --redirect-url=https://auth.yourdomain.com/callback
```

### Option C: Systemd Service

```ini
# /etc/systemd/system/wk-auth-server.service
[Unit]
Description=wk OAuth Callback Server
After=network.target

[Service]
Type=simple
User=www-data
Environment=WK_CLIENT_ID=your-client-id
Environment=WK_CLIENT_SECRET=your-client-secret
Environment=WK_REDIRECT_URL=https://auth.yourdomain.com/callback
ExecStart=/usr/local/bin/wk-auth-server --port=8080
Restart=always

[Install]
WantedBy=multi-user.target
```

### Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name auth.yourdomain.com;

    ssl_certificate /etc/ssl/certs/auth.yourdomain.com.crt;
    ssl_certificate_key /etc/ssl/private/auth.yourdomain.com.key;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 3. VPN Considerations

For private/development deployments, the callback server should be behind VPN:

### Why VPN?

- Limits access to authorized users
- No need for public OAuth verification
- Simpler security model

### WireGuard Example

```ini
# Server config
[Interface]
Address = 10.0.0.1/24
ListenPort = 51820
PrivateKey = <server-private-key>

[Peer]
PublicKey = <client-public-key>
AllowedIPs = 10.0.0.2/32
```

The callback server then binds to VPN interface only:
```bash
./auth-server --port=8080  # Binds to 10.0.0.1:8080
```

## 4. Building CLI with Defaults

Compile the CLI with your infrastructure defaults:

```bash
go build -ldflags "\
  -X 'github.com/namastexlabs/workit/internal/config.DefaultClientID=your-client-id' \
  -X 'github.com/namastexlabs/workit/internal/config.DefaultClientSecret=your-client-secret' \
  -X 'github.com/namastexlabs/workit/internal/config.DefaultCallbackServer=https://auth.yourdomain.com'" \
  -o wk ./cmd/wk
```

Users of this binary won't need to configure credentials.

## 5. Environment Variables Reference

### CLI

| Variable | Description |
|----------|-------------|
| `WK_CLIENT_ID` | OAuth client ID |
| `WK_CLIENT_SECRET` | OAuth client secret |
| `WK_CALLBACK_SERVER` | Callback server URL |
| `WK_KEYRING_BACKEND` | Token storage backend |
| `WK_KEYRING_PASSWORD` | Password for file backend |

### Callback Server

| Variable | Description |
|----------|-------------|
| `WK_CLIENT_ID` | OAuth client ID |
| `WK_CLIENT_SECRET` | OAuth client secret |
| `WK_REDIRECT_URL` | OAuth redirect URL |

## 6. Testing the Setup

### Test Callback Server

```bash
# Health check
curl https://auth.yourdomain.com/health
# {"status": "ok"}

# Start auth flow
wk auth add test@gmail.com --headless --callback-server=https://auth.yourdomain.com

# Check token status (replace STATE with actual state)
curl https://auth.yourdomain.com/status/STATE
```

### Test Full Flow

```bash
# 1. Start headless auth
wk auth add you@gmail.com --headless --services=drive

# 2. Complete OAuth in browser (use URL from output)

# 3. Verify token stored
wk auth list --check
```

## Security Checklist

- [ ] OAuth credentials not committed to git
- [ ] Callback server uses HTTPS
- [ ] Callback server behind VPN (or rate-limited)
- [ ] OAuth app in Testing mode initially
- [ ] Test users added for development
- [ ] Tokens stored in secure keyring
