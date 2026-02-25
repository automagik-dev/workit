# Development

> Back to [README](../README.md)

## Build from Source

```bash
git clone https://github.com/automagik-dev/workit.git
cd workit
make
```

Run:

```bash
./bin/wk --help
```

## Help Navigation

- `wk --help` shows top-level command groups.
- Drill down with `wk <group> --help` (and deeper subcommands).
- For the full expanded command list: `WK_HELP=full wk --help`.
- Make shortcut: `make wk -- --help` (or `make wk -- gmail --help`).
- `make wk-help` shows CLI help (note: `make wk --help` is Make's own help; use `--`).
- Version: `wk --version` or `wk version`.

## Installation

### One command

```bash
curl -fsSL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/install.sh | bash
```

### Update

```bash
wk update
```

## Make Targets

After cloning, install tools:

```bash
make tools
```

Pinned tools (installed into `.tools/`):

| Target | Description |
|---|---|
| `make` / `make build` | Build `bin/wk` |
| `make tools` | Install pinned dev tools into `.tools/` |
| `make fmt` | Format code (goimports + gofumpt) |
| `make lint` | Lint code (golangci-lint) |
| `make test` | Run tests |
| `make ci` | Full local gate: format checks + lint + test |

### Make Shortcut

Build and run in one step:

```bash
make wk auth add you@gmail.com
```

For clean stdout when scripting:

- Use `--` when the first arg is a flag: `make wk -- --json gmail search "from:me" | jq .`

## Testing Guidelines

### Unit Tests

Standard Go `testing` package (and `httptest` where needed). Tests live next to the code in `*_test.go` files.

```bash
make test
```

### Integration Tests (Live Google APIs)

Opt-in tests that hit real Google APIs using your stored `wk` credentials/tokens.

```bash
# Optional: override which account to use
export WK_IT_ACCOUNT=you@gmail.com
export WK_CLIENT=work
go test -tags=integration ./...
```

Tip: if you want to avoid macOS Keychain prompts during these runs, set `WK_KEYRING_BACKEND=file` and `WK_KEYRING_PASSWORD=...` (uses encrypted on-disk keyring).

### Live Test Script (CLI)

Fast end-to-end smoke checks against live APIs:

```bash
scripts/live-test.sh --fast
scripts/live-test.sh --account you@gmail.com --skip groups,keep,calendar-enterprise
scripts/live-test.sh --client work --account you@company.com
```

Script toggles:

- `--auth all,groups` to re-auth before running
- `--client <name>` to select OAuth client credentials
- `--strict` to fail on optional features (groups/keep/enterprise)
- `--allow-nontest` to override the test-account guardrail

Go test wrapper (opt-in):

```bash
WK_LIVE=1 go test -tags=integration ./internal/integration -run Live
```

Optional env:

| Variable | Description |
|---|---|
| `WK_LIVE_FAST=1` | Fast mode |
| `WK_LIVE_SKIP=groups,keep` | Skip specific services |
| `WK_LIVE_AUTH=all,groups` | Re-auth before running |
| `WK_LIVE_ALLOW_NONTEST=1` | Override test-account guardrail |
| `WK_LIVE_EMAIL_TEST=you+wktest.com` | Test email address |
| `WK_LIVE_GROUP_EMAIL=group@domain` | Test group email |
| `WK_LIVE_CLASSROOM_COURSE=<courseId>` | Classroom course ID |
| `WK_LIVE_CLASSROOM_CREATE=1` | Allow creating classroom courses |
| `WK_LIVE_CLASSROOM_ALLOW_STATE=1` | Allow classroom state changes |
| `WK_LIVE_TRACK=1` | Enable email tracking tests |
| `WK_LIVE_GMAIL_BATCH_DELETE=1` | Enable batch delete tests |
| `WK_LIVE_GMAIL_FILTERS=1` | Enable Gmail filter tests |
| `WK_LIVE_GMAIL_WATCH_TOPIC=projects/.../topics/...` | Pub/Sub topic for watch tests |
| `WK_LIVE_CALENDAR_RESPOND=1` | Enable calendar respond tests |
| `WK_LIVE_CALENDAR_RECURRENCE=1` | Enable recurrence tests |
| `WK_KEEP_SERVICE_ACCOUNT=/path/to/service-account.json` | Service account for Keep tests |
| `WK_KEEP_IMPERSONATE=user@workspace-domain` | User to impersonate for Keep tests |

## CI

CI runs format checks, tests, lint, deadcode, race, and coverage gates on push/PR.

Required checks for protected branches (`main`, `dev`) should include at least:

- `ci / test`
- `ci / worker`
- `ci / darwin-cgo-build`
- `version / version-artifact`

### Branch Protection Recommendation

- Require pull requests before merge.
- Require all required checks to pass.
- Restrict direct pushes to `main`.
- Use `dev` as the integration branch and merge `dev -> main` for release promotion.

## Version Info

```bash
wk --version
wk version --json
```

See [docs/agent.md](agent.md) for the version artifact contract details and [docs/RELEASING.md](RELEASING.md) for release procedures.
