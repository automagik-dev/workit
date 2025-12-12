# gogcli

Minimal Google (Gmail/Calendar/Drive/Contacts) CLI in Go.

## Setup

1. Create OAuth credentials in Google Cloud Console (Desktop app) and enable the APIs you need.
2. Store credentials:
   - `gog auth credentials ~/path/to/credentials.json`
3. Authorize (stores a refresh token in Keychain via keyring):
   - `gog auth add you@gmail.com` (default: all services)
   - or: `gog auth add you@gmail.com --services drive,calendar`

Most API commands require `--account you@gmail.com`.

### Output

- `--output=text` writes plain text to stdout (designed to be script-friendly).
- `--output=json` writes JSON to stdout (best for scripting).
- Human-facing hints/progress go to stderr.

### Environment

- `GOG_ACCOUNT=you@gmail.com` (used if `--account` is omitted)
- `GOG_COLOR=auto|always|never` (default `auto`)
- `GOG_OUTPUT=text|json` (default `text`)

### Integration tests (local only)

Run smoke tests against real APIs (not in CI):

- `GOG_IT_ACCOUNT=you@gmail.com go test -tags=integration ./internal/integration`

## Development

- Format: `make fmt`
- Lint: `make lint`
- Test: `make test`
