# Auth Callback Server

OAuth callback server for headless authentication. This server receives OAuth callbacks from Google, exchanges authorization codes for tokens, and holds them temporarily for CLI retrieval.

## Overview

When users authenticate via the headless OAuth flow:
1. CLI generates an auth URL with a unique state parameter
2. User opens the URL on their mobile device and completes OAuth
3. Google redirects to this server's `/callback` endpoint
4. Server exchanges the auth code for tokens and stores them keyed by state
5. CLI polls `/token/{state}` to retrieve the token
6. Tokens auto-expire after 15 minutes

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check, returns `{"status": "ok"}` |
| `/callback` | GET | Google OAuth callback, exchanges code for token |
| `/token/{state}` | GET | Retrieve token (consumes it) |
| `/status/{state}` | GET | Check token status without consuming |
| `/m365/sessions` | POST | Create a one-click Microsoft 365 read-only login session (admin bearer token) |
| `/m365/start/{state}` | GET | Redirect user to Microsoft authorize URL |
| `/m365/callback` | GET | Microsoft OAuth callback with PKCE, email and tenant validation |

### Response Codes

**GET /token/{state}**
- `200 OK` - Token ready, returns JSON with access_token, refresh_token, token_type, expiry
- `202 Accepted` - Token pending (user hasn't completed OAuth yet)
- `404 Not Found` - State unknown or expired
- `410 Gone` - Token already consumed

**GET /status/{state}**
- Returns `{"status": "ready|pending|consumed|not_found"}`

## Configuration

### Command-Line Flags

```bash
./auth-server \
  --port 8080 \
  --client-id "your-client-id" \
  --client-secret "your-client-secret" \
  --redirect-url "https://auth.example.com/callback" \
  --ttl 15m
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `WK_CLIENT_ID` | OAuth client ID |
| `WK_CLIENT_SECRET` | OAuth client secret |
| `WK_REDIRECT_URL` | Google OAuth redirect URL |
| `WK_PUBLIC_BASE_URL` | Public HTTPS base URL for the deployed auth server; M365 derives `/m365/callback` from this |
| `WK_CALLBACK_SERVER` | Backward-compatible public base URL fallback |
| `WK_M365_CLIENT_ID` | Microsoft Entra application/client ID for M365 broker |
| `WK_M365_TENANT_ID` | Microsoft tenant ID or verified domain; required with the M365 broker (`organizations`/`common` are refused) |
| `WK_M365_BROKER_TOKEN` | Required bearer token for trusted callers creating `/m365/sessions` |

Command-line flags take precedence over environment variables.

## Microsoft 365 one-click login

For a user who cannot run the CLI: the operator creates a link, the user clicks it and logs into
Microsoft, and the refresh token lands in the operator's Workit keyring under the `m365` client.

1. Register an Entra app (single-tenant, delegated `User.Read`, `Mail.Read`, `Calendars.Read`) with
   redirect URI `https://<public-base-url>/m365/callback`.
2. Run this server behind HTTPS with `WK_PUBLIC_BASE_URL`, `WK_M365_CLIENT_ID`, `WK_M365_TENANT_ID`
   and `WK_M365_BROKER_TOKEN` set.
3. On the operator machine:

   ```bash
   WK_M365_BROKER_TOKEN=... wk --json auth m365 login-link bernardo@example.com --base-url https://<public-base-url>
   ```

   The first JSON document carries `login_url`; send it to the user. The command then waits until
   the login completes (or the link expires after 10 minutes) and prints the stored result.
   `--no-poll` prints the link and exits, so the token is never collected.

Guarantees: scopes are fixed to `offline_access User.Read Mail.Read Calendars.Read`; the state is
random, single-use and expires after 10 minutes; the PKCE verifier never leaves the server; the
callback fails closed when the state is unknown, expired or reused, or when Graph `/me`
(mail, else UPN) or `/organization` does not match the expected email and tenant. The token is
keyed by a separate `poll_key` returned only to the admin caller, so the public link cannot be
used to fetch it.

Audit lines are JSON on stderr: `m365.link_created`, `m365.callback_ok`, `m365.callback_failed`
(with `reason`) and `m365.state_expired`. They carry a short hash of the state, a masked email and
the tenant id, never tokens, the poll key or mail content.

## Building

### Local Build

```bash
cd auth-server
go build -o auth-server .
./auth-server --port 8089 --client-id "..." --client-secret "..."
```

### Docker Build

```bash
docker build -t workit-auth-server .
docker run -p 8080:8080 \
  -e WK_PUBLIC_BASE_URL="https://auth.hv.example" \
  -e WK_M365_CLIENT_ID="<hapvida-entra-app-client-id>" \
  -e WK_M365_TENANT_ID="<hapvida-tenant-id>" \
  -e WK_M365_BROKER_TOKEN="<strong-random-admin-token>" \
  workit-auth-server
```

For Google relay compatibility, also set `WK_CLIENT_ID`, `WK_CLIENT_SECRET`, and optionally `WK_REDIRECT_URL`.

## Deployment

### Docker Compose Example

```yaml
version: '3.8'
services:
  auth-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - WK_CLIENT_ID=${WK_CLIENT_ID}
      - WK_CLIENT_SECRET=${WK_CLIENT_SECRET}
      - WK_REDIRECT_URL=https://auth.example.com/callback
    restart: unless-stopped
```

The runtime image is distroless and has no shell utilities such as `wget`; configure Kubernetes/OCI HTTP probes against `/health` instead of a container-local shell healthcheck.

### Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name auth.example.com;

    ssl_certificate /etc/letsencrypt/live/auth.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/auth.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Security Considerations

1. **VPN-Only Access**: For Phase 1, the callback server should only be accessible via VPN
2. **Short TTL**: Tokens expire after 15 minutes by default
3. **Single Use**: Tokens can only be retrieved once via `/token/{state}`
4. **State Parameter**: Prevents CSRF attacks; each auth flow gets a unique state
5. **No Persistence**: Tokens are stored in memory only, lost on restart

## Development

### Testing Locally with ngrok

```bash
# Start the server
./auth-server --port 8089 --client-id "..." --client-secret "..."

# In another terminal, expose via ngrok
ngrok http 8089

# Use the ngrok URL as the redirect URL in your GCP OAuth configuration
# e.g., https://abc123.ngrok.io/callback
```

### Manual Testing

```bash
# Health check
curl http://localhost:8089/health

# Check status of a state (should be not_found)
curl http://localhost:8089/status/test-state

# After completing OAuth, retrieve token
curl http://localhost:8089/token/your-state
```
