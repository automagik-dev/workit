# Headless OAuth Authentication

workit supports a headless OAuth flow designed for AI agents and automation. Users authenticate on any device (like a mobile phone), and the CLI retrieves the token automatically.

## How It Works

1. **CLI generates auth URL** → User opens URL on any device (phone, tablet, etc.)
2. **User completes OAuth** → Google redirects to callback server
3. **Callback server stores token** → Token held temporarily (15-minute TTL)
4. **CLI polls for token** → Retrieves and stores in keychain

```
┌─────────┐     ┌──────────┐     ┌───────────────┐     ┌────────┐
│   CLI   │────▶│  Google  │────▶│ Callback Srv  │────▶│  CLI   │
│ (agent) │     │  OAuth   │     │ (auth.x.io)   │     │ (poll) │
└─────────┘     └──────────┘     └───────────────┘     └────────┘
     │               │                  │                   │
     │   auth URL    │                  │                   │
     └──────────────▶│   user login     │                   │
                     │─────────────────▶│   store token     │
                     │                  │◀──────────────────│
                     │                  │   GET /token/xxx  │
                     │                  │──────────────────▶│
                     │                  │   {refresh_token} │
                     │                  │◀──────────────────│
```

## Usage

### Interactive Mode (Default)

```bash
# Standard OAuth with browser popup
wk auth add you@gmail.com --services=user
```

### Headless Mode (For Agents)

```bash
# Generate auth URL without opening browser
wk auth add you@gmail.com --headless --services=user

# Output:
# Visit this URL to authorize:
# https://accounts.google.com/o/oauth2/v2/auth?...
#
# State: abc123xyz
# Poll URL: https://auth.automagik.dev/token/abc123xyz
# Expires in: 300 seconds
#
# Waiting for authorization...
```

### Headless Mode with No Polling

```bash
# Just output the URL, don't poll (for async workflows)
wk auth add you@gmail.com --headless --no-poll --services=user

# Later, poll manually:
wk auth poll abc123xyz
```

### JSON Output

```bash
wk auth add you@gmail.com --headless --json

# Output:
{
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?...",
  "state": "abc123xyz",
  "poll_url": "https://auth.automagik.dev/token/abc123xyz",
  "expires_in": 300
}
```

## Configuration

### Callback Server URL

The callback server URL can be configured in order of precedence:

1. **Flag**: `--callback-server=https://auth.example.com`
2. **Environment**: `WK_CALLBACK_SERVER=https://auth.example.com`
3. **Build-time default**: Compiled into binary with `-ldflags`

### OAuth Credentials

For headless mode, OAuth credentials are resolved in order:

1. **File**: `~/.config/workit/credentials.json` (standard flow)
2. **Build-time defaults**: Compiled into binary with `-ldflags`
3. **Environment**: `WK_CLIENT_ID` and `WK_CLIENT_SECRET`

## Troubleshooting

### "callback server URL required for headless auth"

Set the callback server URL:
```bash
export WK_CALLBACK_SERVER=https://auth.automagik.dev
```

### "timeout waiting for token"

The user didn't complete authentication within the timeout (default 5 minutes). Try again with a longer timeout:
```bash
wk auth add you@gmail.com --headless --poll-timeout=10m
```

### "token has already been retrieved"

The token was already polled and consumed. Each auth flow produces a single-use token. Start a new auth flow.

### "authorized as X, expected Y"

The user signed in with a different email than specified. Ensure the correct Google account is used.

## Security Considerations

- **State parameter**: Prevents CSRF attacks by binding the auth flow to a specific session
- **Token TTL**: Tokens expire from the callback server after 15 minutes
- **Single use**: Tokens are consumed on first retrieval
- **HTTPS**: Always use HTTPS for the callback server in production
- **VPN**: For development/private use, the callback server should be behind VPN

## For Operators

See [infrastructure.md](infrastructure.md) for setting up your own callback server.
