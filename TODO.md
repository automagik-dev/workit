# TODO: gog-cli Setup & Infrastructure

This document tracks the infrastructure and configuration tasks needed before development can proceed.

---

## 1. Repository Transfer (Optional)

The repo was created under `automagik-genie`. To transfer to your org:

```bash
# Via GitHub UI: Settings → Danger Zone → Transfer ownership
# Or via API:
gh api repos/automagik-genie/gog-cli/transfer -f new_owner=YOUR_ORG
```

---

## 2. Google Cloud OAuth App Setup

### 2.1 Create GCP Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/projectcreate)
2. Create project: `gog-cli-agent` (or similar)
3. Note the Project ID

### 2.2 Enable Required APIs

Enable these APIs in the project:

| API | Console Link |
|-----|--------------|
| Gmail API | https://console.cloud.google.com/apis/api/gmail.googleapis.com |
| Google Calendar API | https://console.cloud.google.com/apis/api/calendar-json.googleapis.com |
| Google Drive API | https://console.cloud.google.com/apis/api/drive.googleapis.com |
| People API | https://console.cloud.google.com/apis/api/people.googleapis.com |
| Tasks API | https://console.cloud.google.com/apis/api/tasks.googleapis.com |
| Sheets API | https://console.cloud.google.com/apis/api/sheets.googleapis.com |
| Docs API | https://console.cloud.google.com/apis/api/docs.googleapis.com |

### 2.3 Configure OAuth Consent Screen

1. Go to [OAuth Consent Screen](https://console.cloud.google.com/auth/branding)
2. Choose **External** (or Internal if Workspace-only)
3. Fill in:
   - App name: `gog-cli` (or your brand)
   - User support email: your email
   - Developer contact: your email
4. Add scopes (can be broad for internal use):
   - `https://www.googleapis.com/auth/gmail.modify`
   - `https://www.googleapis.com/auth/calendar`
   - `https://www.googleapis.com/auth/drive`
   - `https://www.googleapis.com/auth/contacts`
   - `https://www.googleapis.com/auth/tasks`
   - `https://www.googleapis.com/auth/spreadsheets`
   - `https://www.googleapis.com/auth/documents`
5. If in testing mode, add test users

### 2.4 Create OAuth Client (Web Application)

**Important**: Use "Web application" type, NOT "Desktop app" - because the callback goes to your server, not localhost.

1. Go to [Credentials](https://console.cloud.google.com/auth/clients)
2. Click **Create Credentials** → **OAuth client ID**
3. Application type: **Web application**
4. Name: `gog-cli-headless`
5. Authorized redirect URIs:
   - `https://auth.yourcompany.com/callback` (production)
   - `https://YOUR-NGROK-URL.ngrok.io/callback` (development)
6. Click **Create**
7. **Save the Client ID and Client Secret** securely

```bash
# Store these securely (e.g., in password manager or secrets manager)
GOG_CLIENT_ID="123456789-xxxxx.apps.googleusercontent.com"
GOG_CLIENT_SECRET="GOCSPX-xxxxxxxxxxxxxxxx"
```

---

## 3. OAuth Callback Server Setup

### 3.1 Choose a Domain

You need a publicly accessible domain for the OAuth callback. Options:

| Option | Pros | Cons |
|--------|------|------|
| Subdomain (auth.yourcompany.com) | Professional, permanent | Needs DNS setup |
| Cloudflare Tunnel | Easy, no DNS changes | Depends on CF |
| Ngrok (dev only) | Instant setup | URL changes, not for prod |

**Recommended**: `auth.yourcompany.com` or similar subdomain

### 3.2 DNS Configuration

Add an A record or CNAME pointing to your server:

```
auth.yourcompany.com → YOUR_SERVER_IP
# or
auth.yourcompany.com → CNAME your-server.example.com
```

### 3.3 SSL Certificate

Use Let's Encrypt (via certbot) or Cloudflare for HTTPS:

```bash
# With certbot
sudo certbot certonly --standalone -d auth.yourcompany.com

# Or use Cloudflare proxy for automatic SSL
```

### 3.4 Deploy Callback Server

Once the `auth-server/` is built (Milestone 2), deploy it:

```bash
# Build
cd auth-server && go build -o auth-server .

# Run (with env vars)
GOG_CLIENT_ID=xxx \
GOG_CLIENT_SECRET=xxx \
REDIS_URL=redis://localhost:6379 \
./auth-server --port 443 --cert /path/to/cert.pem --key /path/to/key.pem

# Or via Docker
docker run -d \
  -e GOG_CLIENT_ID=xxx \
  -e GOG_CLIENT_SECRET=xxx \
  -e REDIS_URL=redis://redis:6379 \
  -p 443:443 \
  your-registry/gog-auth-server
```

### 3.5 Update Google OAuth Redirect URI

Once your callback server is live, add its URL to the OAuth client's authorized redirect URIs:

```
https://auth.yourcompany.com/callback
```

---

## 4. Agent Infrastructure Configuration

### 4.1 Build Internal Binary

In your CI/CD pipeline, build gog-cli with embedded credentials:

```bash
make build-internal \
  GOG_CLIENT_ID=$GOG_CLIENT_ID \
  GOG_CLIENT_SECRET=$GOG_CLIENT_SECRET \
  GOG_CALLBACK_SERVER=https://auth.yourcompany.com
```

### 4.2 Distribute to Agent Runtime

Place the built binary where your agents can access it:

```bash
# Example: copy to agent container or shared volume
cp bin/gog /opt/agent-tools/gog

# Or publish to internal artifact registry
aws s3 cp bin/gog s3://your-internal-bucket/tools/gog-cli/gog-$(git rev-parse --short HEAD)
```

### 4.3 Agent Environment Variables (Alternative to Embedded)

If you prefer env vars over embedded credentials:

```bash
export GOG_CLIENT_ID="your-client-id"
export GOG_CLIENT_SECRET="your-client-secret"
export GOG_CALLBACK_SERVER="https://auth.yourcompany.com"
export GOG_KEYRING_BACKEND="file"
export GOG_KEYRING_PASSWORD="secure-random-password"  # For headless keyring
```

---

## 5. Test the Flow

### 5.1 Development Testing (with ngrok)

```bash
# Terminal 1: Start callback server locally
cd auth-server && go run . --port 8089

# Terminal 2: Expose via ngrok
ngrok http 8089
# Note the https URL, e.g., https://abc123.ngrok.io

# Terminal 3: Test headless auth
GOG_CALLBACK_SERVER=https://abc123.ngrok.io ./bin/gog auth add test@gmail.com --headless --json
# Copy the auth_url and open on your phone
```

### 5.2 Production Testing

```bash
# With production callback server
./bin/gog auth add user@yourcompany.com --headless --json

# Send the auth_url to a user via WhatsApp/Telegram
# Verify token is received after they complete auth

# Test authenticated commands
./bin/gog gmail labels list
./bin/gog drive list
```

---

## 6. Checklist Summary

### Prerequisites (Do First)
- [x] GCP Project created (felipe-bot / felipe-bot-485616)
- [x] Required APIs enabled (Drive, Gmail, Calendar, People, Tasks, Sheets, Docs)
- [x] OAuth consent screen configured (Internal, namastex.ai)
- [x] OAuth Web client created (gog-cli-headless)
- [x] Client ID and Secret saved securely

### Infrastructure (Before Production)
- [x] Callback server domain chosen (gogoauth.namastex.io)
- [x] DNS configured
- [x] SSL certificate obtained
- [ ] Redis instance available (or use in-memory for low-volume)

### After Milestone 2 Complete
- [x] Callback server deployed (auth-server/)
- [x] Google OAuth redirect URI updated
- [ ] End-to-end flow tested with mobile device

### After Milestone 3 Complete
- [ ] Internal binary built with credentials
- [ ] Binary distributed to agent infrastructure
- [ ] Agent successfully authenticates users

---

## 7. Security Considerations

### Secrets Management

| Secret | Where to Store |
|--------|----------------|
| `GOG_CLIENT_SECRET` | CI secrets, never in code |
| `GOG_KEYRING_PASSWORD` | Agent runtime secrets |
| Redis auth (if used) | Container secrets |

### Token Storage

- Tokens are stored in the keyring (file-based for headless)
- Each user session should have isolated keyring file
- Consider per-session `GOG_CONFIG_DIR` for isolation:
  ```bash
  GOG_CONFIG_DIR=/tmp/gog-session-$SESSION_ID ./bin/gog ...
  ```

### Rate Limiting

- Google API quotas apply per-project
- Consider requesting quota increases for production use
- Monitor usage in GCP Console → APIs & Services → Quotas

---

## 8. Useful Links

- GCP Console: https://console.cloud.google.com/
- OAuth Credentials: https://console.cloud.google.com/apis/credentials
- API Quotas: https://console.cloud.google.com/apis/dashboard
- Upstream gogcli: https://github.com/steipete/gogcli
- This repo: https://github.com/automagik-genie/gog-cli
