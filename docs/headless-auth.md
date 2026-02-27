# Headless OAuth Authentication

workit supports a headless OAuth flow designed for AI agents and automation. The CLI binds to `0.0.0.0`, detects and displays your outbound IP, and auto-closes after authentication completes — no localhost tunnels or manual port-forwarding required.

## How It Works

1. **`wk auth manage`** → binds local HTTP server to `0.0.0.0:8085`, prints your outbound IP so you can open the URL from any device
2. **User completes OAuth** → Google redirects to `auth.automagik.dev` (the default callback server — no GCP client needed for most users)
3. **Callback server stores token** → Token held temporarily (15-minute TTL)
4. **CLI polls for token** → Retrieves and stores in keychain, then auto-closes the local server

```
┌─────────┐     ┌──────────┐     ┌─────────────────────┐     ┌────────┐
│   CLI   │────▶│  Google  │────▶│  auth.automagik.dev  │────▶│  CLI   │
│ (agent) │     │  OAuth   │     │  (callback server)   │     │ (poll) │
└─────────┘     └──────────┘     └─────────────────────┘     └────────┘
     │               │                     │                      │
     │   auth URL    │                     │                      │
     └──────────────▶│   user login        │                      │
                     │────────────────────▶│   store token        │
                     │                     │◀─────────────────────│
                     │                     │   GET /token/xxx     │
                     │                     │─────────────────────▶│
                     │                     │   {refresh_token}    │
                     │                     │◀─────────────────────│
```

## Usage

### Recommended: Account Manager UI

```bash
wk auth manage
```

This is the recommended entry point for all auth flows. It:
- Binds to `0.0.0.0:8085` (accessible from any device on the same network or via public IP)
- Displays your outbound IP so you know which URL to open
- Uses `auth.automagik.dev` as the default callback server (no GCP client needed for most users)
- Auto-closes the local server once authentication completes

### Agent / Headless Mode (`--print-url`)

For fully automated or agent-driven environments where you want to capture the auth URL programmatically:

```bash
wk auth manage --print-url
```

Output:
```json
{"url":"http://203.0.113.42:8085","port":8085}
```

Your agent can open this URL or present it to the user, then poll for completion.

### Legacy: Direct Account Add

```bash
# Standard OAuth — opens browser automatically (interactive environments)
wk auth add you@gmail.com --services=user

# Headless: generate auth URL without opening browser
wk auth add you@gmail.com --headless --services=user
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

`auth.automagik.dev` is the default callback server — most users do not need to configure anything. The callback server URL can be overridden in order of precedence:

1. **Flag**: `--callback-server=https://your-server.example.com`
2. **Environment**: `WK_CALLBACK_SERVER=https://your-server.example.com`
3. **Build-time default**: Compiled into binary with `-ldflags`

### OAuth Credentials

For headless mode, OAuth credentials are resolved in order:

1. **File**: `~/.config/workit/credentials.json` (standard flow)
2. **Build-time defaults**: Compiled into binary with `-ldflags`
3. **Environment**: `WK_CLIENT_ID` and `WK_CLIENT_SECRET`

## Troubleshooting

### "callback server URL required for headless auth"

Set the callback server URL or rely on the default:
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
- **Outbound IP binding**: `wk auth manage` binds to `0.0.0.0` and shows your outbound IP — ensure firewall rules are appropriate for your environment

## For Operators

See [infrastructure.md](infrastructure.md) for setting up your own callback server.
