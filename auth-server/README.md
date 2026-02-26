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
| `/callback` | GET | OAuth callback, exchanges code for token |
| `/token/{state}` | GET | Retrieve token (consumes it) |
| `/status/{state}` | GET | Check token status without consuming |

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
| `WK_REDIRECT_URL` | OAuth redirect URL |

Command-line flags take precedence over environment variables.

## Building

### Local Build

```bash
cd auth-server
go build -o auth-server .
./auth-server --port 8089 --client-id "..." --client-secret "..."
```

### Docker Build

```bash
docker build -t auth-server .
docker run -p 8080:8080 \
  -e WK_CLIENT_ID="your-client-id" \
  -e WK_CLIENT_SECRET="your-client-secret" \
  -e WK_REDIRECT_URL="https://auth.example.com/callback" \
  auth-server
```

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
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
```

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
