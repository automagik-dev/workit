# workit spec

## Goal

Build a single, clean, modern Go CLI that talks to:

- Gmail API
- Google Calendar API
- Google Chat API
- Google Classroom API
- Google Drive API
- Google Docs API
- Google Sheets API
- Google Forms API
- Apps Script API
- Google Tasks API
- Cloud Identity API (Groups)
- Google People API (Contacts + directory)
- Google Keep API (Workspace-only, service account)

This replaces the existing separate CLIs (`gmcli`, `gccli`, `gdcli`) and the Python contacts server conceptually, but:

- no backwards compatibility
- no migration tooling

## Non-goals

- Preserving legacy command names/flags/output formats
- Importing existing `~/.gmcli`, `~/.gccli`, `~/.gdcli` state
- Running an MCP server (this is a CLI)

## Language/runtime

- Go `1.25` (see `go.mod`)

## CLI framework

- `github.com/alecthomas/kong`
- Root command: `wk`
- Global flag:
  - `--color=auto|always|never` (default `auto`)
  - `--json` (JSON output to stdout)
  - `--plain` (TSV output to stdout; stable/parseable; disables colors)
  - `--force` (skip confirmations for destructive commands)
  - `--no-input` (never prompt; fail instead)
  - `--version` (print version)

Notes:

- We run `SilenceUsage: true` and print errors ourselves (colored when possible).
- `NO_COLOR` is respected.

Environment:

- `WK_COLOR=auto|always|never` (default `auto`, overridden by `--color`)
- `WK_JSON=1` (default JSON output; overridden by flags)
- `WK_PLAIN=1` (default plain output; overridden by flags)

## Output (TTY-aware colors)

- `github.com/muesli/termenv` is used to detect rich TTY capabilities and render colored output.
- Colors are enabled when:
  - output is a rich terminal and `--color=auto`, and `NO_COLOR` is not set; or
  - `--color=always`
- Colors are disabled when:
  - `--color=never`; or
  - `NO_COLOR` is set

Implementation: `internal/ui/ui.go`.

## Auth + secret storage

### OAuth client credentials (non-secret-ish)

- Stored on disk in the per-user config directory:
  - `$(os.UserConfigDir())/workit/credentials.json` (default client)
  - `$(os.UserConfigDir())/workit/credentials-<client>.json` (named clients)
- Written with mode `0600`.
- Command:
  - `wk auth credentials <credentials.json>`
  - `wk --client <name> auth credentials <credentials.json>`
  - `wk auth credentials list`
- Supports Google’s downloaded JSON format:
  - `installed.client_id/client_secret` or `web.client_id/client_secret`

Implementation: `internal/config/*`.

### Refresh tokens (secrets)

- Stored in OS credential store via `github.com/99designs/keyring`.
- Key namespace is `workit` (keyring `ServiceName`).
- Key format: `token:<client>:<email>` (default client uses `token:default:<email>`)
- Legacy key format: `token:<email>` (migrated on first read)
- Stored payload is JSON (refresh token + metadata like selected services/scopes).
- Fallback: if no OS credential store is available, keyring may use its encrypted "file" backend:
  - Directory: `$(os.UserConfigDir())/workit/keyring/` (one file per key)
  - Password: prompts on TTY; for non-interactive runs set `WK_KEYRING_PASSWORD`

Current minimal management commands (implemented):

- `wk auth tokens list` (keys only)
- `wk auth tokens delete <email>`

Implementation: `internal/secrets/store.go`.

### OAuth flow

- Desktop OAuth 2.0 flow using local HTTP redirect on an ephemeral port.
- Supports a browserless/manual flow (paste redirect URL) for headless environments.
- Supports a remote/server-friendly 2-step manual flow:
  - Step 1 prints an auth URL (`wk auth add ... --remote --step 1`)
  - Step 2 exchanges the pasted redirect URL and requires `state` validation (`--remote --step 2 --auth-url ...`)
- Refresh token issuance:
  - requests `access_type=offline`
  - supports `--force-consent` to force the consent prompt when Google doesn't return a refresh token
  - uses `include_granted_scopes=true` to support incremental auth re-runs

Scope selection note:

- The consent screen shows the scopes the CLI requested.
- Users cannot selectively un-check individual requested scopes in the consent screen; they either approve all requested scopes or cancel.
- To request fewer scopes, choose fewer services via `wk auth add --services ...` or use `wk auth add --readonly` where applicable.

## Config layout

- Base config dir: `$(os.UserConfigDir())/workit/`
- Files:
  - `config.json` (JSON5; comments and trailing commas allowed)
  - `credentials.json` (OAuth client id/secret; default client)
  - `credentials-<client>.json` (OAuth client id/secret; named clients)
- State:
  - `state/gmail-watch/<account>.json` (Gmail watch state)
  - `oauth-manual-state-<state>.json` (temporary manual OAuth state cache; expires quickly; no tokens)
- Secrets:
  - refresh tokens in keyring

We intentionally avoid storing refresh tokens in plain JSON on disk.

Environment:

- `WK_ACCOUNT=you@gmail.com` (email or alias; used when `--account` is not set; otherwise uses keyring default or a single stored token)
- `WK_CLIENT=work` (select OAuth client bucket; see `--client`)
- `WK_KEYRING_PASSWORD=...` (used when keyring falls back to encrypted file backend in non-interactive environments)
- `WK_KEYRING_BACKEND={auto|keychain|file}` (force backend; use `file` to avoid Keychain prompts and pair with `WK_KEYRING_PASSWORD` for non-interactive)
- `WK_TIMEZONE=America/New_York` (default output timezone; IANA name or `UTC`; `local` forces local timezone)
- `WK_ENABLE_COMMANDS=calendar,tasks` (optional allowlist of top-level commands)
- `config.json` can also set `keyring_backend` (JSON5; env vars take precedence)
- `config.json` can also set `default_timezone` (IANA name or `UTC`)
- `config.json` can also set `account_aliases` for `wk auth alias` (JSON5)
- `config.json` can also set `account_clients` (email -> client) and `client_domains` (domain -> client)

Flag aliases:
- `--out` also accepts `--output`.
- `--out-dir` also accepts `--output-dir` (Gmail thread attachment downloads).

## Commands (current + planned)

### Implemented

- `wk auth credentials <credentials.json|->`
- `wk auth credentials list`
- `wk --client <name> auth credentials <credentials.json|->`
- `wk auth add <email> [--services user|all|gmail,calendar,classroom,drive,docs,contacts,tasks,sheets,people,groups] [--readonly] [--drive-scope full|readonly|file] [--manual] [--remote] [--step 1|2] [--auth-url URL] [--timeout DURATION] [--force-consent]`
- `wk auth services [--markdown]`
- `wk auth keep <email> --key <service-account.json>` (Google Keep; Workspace only)
- `wk auth list`
- `wk auth alias list`
- `wk auth alias set <alias> <email>`
- `wk auth alias unset <alias>`
- `wk auth status`
- `wk auth remove <email>`
- `wk auth tokens list`
- `wk auth tokens delete <email>`
- `wk config get <key>`
- `wk config keys`
- `wk config list`
- `wk config path`
- `wk config set <key> <value>`
- `wk config unset <key>`
- `wk version`
- `wk drive ls [--parent ID] [--max N] [--page TOKEN] [--query Q] [--[no-]all-drives]`
- `wk drive search <text> [--raw-query] [--max N] [--page TOKEN] [--[no-]all-drives]`
- `wk drive get <fileId>`
- `wk drive download <fileId> [--out PATH] [--format F]` (`--format` only applies to Google Workspace files)
- `wk drive upload <localPath> [--name N] [--parent ID] [--convert] [--convert-to doc|sheet|slides]`
- `wk drive mkdir <name> [--parent ID]`
- `wk drive delete <fileId> [--permanent]`
- `wk drive move <fileId> --parent ID`
- `wk drive rename <fileId> <newName>`
- `wk drive share <fileId> --to anyone|user|domain [--email addr] [--domain example.com] [--role reader|writer] [--discoverable]`
- `wk drive permissions <fileId> [--max N] [--page TOKEN]`
- `wk drive unshare <fileId> <permissionId>`
- `wk drive url <fileIds...>`
- `wk drive drives [--max N] [--page TOKEN] [--query Q]`
- `wk calendar calendars`
- `wk calendar acl <calendarId>`
- `wk calendar events <calendarId> [--from RFC3339] [--to RFC3339] [--max N] [--page TOKEN] [--query Q] [--weekday]`
- `wk calendar event|get <calendarId> <eventId>`
- `WK_CALENDAR_WEEKDAY=1` defaults `--weekday` for `wk calendar events`
- `wk calendar create <calendarId> --summary S --from DT --to DT [--description D] [--location L] [--attendees a@b.com,c@d.com] [--all-day] [--event-type TYPE]`
- `wk calendar update <calendarId> <eventId> [--summary S] [--from DT] [--to DT] [--description D] [--location L] [--attendees ...] [--add-attendee ...] [--all-day] [--event-type TYPE]`
- `wk calendar delete <calendarId> <eventId>`
- `wk calendar freebusy <calendarIds> --from RFC3339 --to RFC3339`
- `wk calendar respond <calendarId> <eventId> --status accepted|declined|tentative [--send-updates all|none|externalOnly]`
- `wk time now [--timezone TZ]`
- `wk classroom courses [--state ...] [--max N] [--page TOKEN]`
- `wk classroom courses get <courseId>`
- `wk classroom courses create --name NAME [--owner me] [--state ACTIVE|...]`
- `wk classroom courses update <courseId> [--name ...] [--state ...]`
- `wk classroom courses delete <courseId>`
- `wk classroom courses archive <courseId>`
- `wk classroom courses unarchive <courseId>`
- `wk classroom courses join <courseId> [--role student|teacher] [--user me]`
- `wk classroom courses leave <courseId> [--role student|teacher] [--user me]`
- `wk classroom courses url <courseId...>`
- `wk classroom students <courseId> [--max N] [--page TOKEN]`
- `wk classroom students get <courseId> <userId>`
- `wk classroom students add <courseId> <userId> [--enrollment-code CODE]`
- `wk classroom students remove <courseId> <userId>`
- `wk classroom teachers <courseId> [--max N] [--page TOKEN]`
- `wk classroom teachers get <courseId> <userId>`
- `wk classroom teachers add <courseId> <userId>`
- `wk classroom teachers remove <courseId> <userId>`
- `wk classroom roster <courseId> [--students] [--teachers]`
- `wk classroom coursework <courseId> [--state ...] [--topic TOPIC_ID] [--scan-pages N] [--max N] [--page TOKEN]`
- `wk classroom coursework get <courseId> <courseworkId>`
- `wk classroom coursework create <courseId> --title TITLE [--type ASSIGNMENT|...]`
- `wk classroom coursework update <courseId> <courseworkId> [--title ...]`
- `wk classroom coursework delete <courseId> <courseworkId>`
- `wk classroom coursework assignees <courseId> <courseworkId> [--mode ...] [--add-student ...]`
- `wk classroom materials <courseId> [--state ...] [--topic TOPIC_ID] [--scan-pages N] [--max N] [--page TOKEN]`
- `wk classroom materials get <courseId> <materialId>`
- `wk classroom materials create <courseId> --title TITLE`
- `wk classroom materials update <courseId> <materialId> [--title ...]`
- `wk classroom materials delete <courseId> <materialId>`
- `wk classroom submissions <courseId> <courseworkId> [--state ...] [--max N] [--page TOKEN]`
- `wk classroom submissions get <courseId> <courseworkId> <submissionId>`
- `wk classroom submissions turn-in <courseId> <courseworkId> <submissionId>`
- `wk classroom submissions reclaim <courseId> <courseworkId> <submissionId>`
- `wk classroom submissions return <courseId> <courseworkId> <submissionId>`
- `wk classroom submissions grade <courseId> <courseworkId> <submissionId> [--draft N] [--assigned N]`
- `wk classroom announcements <courseId> [--state ...] [--max N] [--page TOKEN]`
- `wk classroom announcements get <courseId> <announcementId>`
- `wk classroom announcements create <courseId> --text TEXT`
- `wk classroom announcements update <courseId> <announcementId> [--text ...]`
- `wk classroom announcements delete <courseId> <announcementId>`
- `wk classroom announcements assignees <courseId> <announcementId> [--mode ...]`
- `wk classroom topics <courseId> [--max N] [--page TOKEN]`
- `wk classroom topics get <courseId> <topicId>`
- `wk classroom topics create <courseId> --name NAME`
- `wk classroom topics update <courseId> <topicId> --name NAME`
- `wk classroom topics delete <courseId> <topicId>`
- `wk classroom invitations [--course ID] [--user ID]`
- `wk classroom invitations get <invitationId>`
- `wk classroom invitations create <courseId> <userId> --role STUDENT|TEACHER|OWNER`
- `wk classroom invitations accept <invitationId>`
- `wk classroom invitations delete <invitationId>`
- `wk classroom guardians <studentId> [--max N] [--page TOKEN]`
- `wk classroom guardians get <studentId> <guardianId>`
- `wk classroom guardians delete <studentId> <guardianId>`
- `wk classroom guardian-invitations <studentId> [--state ...] [--max N] [--page TOKEN]`
- `wk classroom guardian-invitations get <studentId> <invitationId>`
- `wk classroom guardian-invitations create <studentId> --email EMAIL`
- `wk classroom profile [userId]`
- `wk gmail search <query> [--max N] [--page TOKEN]`
- `wk gmail messages search <query> [--max N] [--page TOKEN] [--include-body]`
- `wk gmail thread get <threadId> [--download]`
- `wk gmail thread modify <threadId> [--add ...] [--remove ...]`
- `wk gmail get <messageId> [--format full|metadata|raw] [--headers ...]`
- `wk gmail attachment <messageId> <attachmentId> [--out PATH] [--name NAME]`
- `wk gmail url <threadIds...>`
- `wk gmail labels list`
- `wk gmail labels get <labelIdOrName>`
- `wk gmail labels create <name>`
- `wk gmail labels modify <threadIds...> [--add ...] [--remove ...]`
- `wk gmail send --to a@b.com --subject S [--body B] [--body-html H] [--cc ...] [--bcc ...] [--reply-to-message-id <messageId>] [--reply-to addr] [--attach <file>...]`
- `wk gmail drafts list [--max N] [--page TOKEN]`
- `wk gmail drafts get <draftId> [--download]`
- `wk gmail drafts create --subject S [--to a@b.com] [--body B] [--body-html H] [--cc ...] [--bcc ...] [--reply-to-message-id <messageId>] [--reply-to addr] [--attach <file>...]`
- `wk gmail drafts update <draftId> --subject S [--to a@b.com] [--body B] [--body-html H] [--cc ...] [--bcc ...] [--reply-to-message-id <messageId>] [--reply-to addr] [--attach <file>...]`
- `wk gmail drafts send <draftId>`
- `wk gmail drafts delete <draftId>`
- `wk gmail watch start|status|renew|stop|serve`
- `wk gmail history --since <historyId>`
- `wk chat spaces list [--max N] [--page TOKEN]`
- `wk chat spaces find <displayName> [--max N]`
- `wk chat spaces create <displayName> [--member email,...]`
- `wk chat messages list <space> [--max N] [--page TOKEN] [--order ORDER] [--thread THREAD] [--unread]`
- `wk chat messages send <space> --text TEXT [--thread THREAD]`
- `wk chat threads list <space> [--max N] [--page TOKEN]`
- `wk chat dm space <email>`
- `wk chat dm send <email> --text TEXT [--thread THREAD]`
- `wk tasks lists [--max N] [--page TOKEN]`
- `wk tasks lists create <title>`
- `wk tasks list <tasklistId> [--max N] [--page TOKEN]`
- `wk tasks get <tasklistId> <taskId>`
- `wk tasks add <tasklistId> --title T [--notes N] [--due RFC3339|YYYY-MM-DD] [--repeat daily|weekly|monthly|yearly] [--repeat-count N] [--repeat-until DT] [--parent ID] [--previous ID]`
- `wk tasks update <tasklistId> <taskId> [--title T] [--notes N] [--due RFC3339|YYYY-MM-DD] [--status needsAction|completed]`
- `wk tasks done <tasklistId> <taskId>`
- `wk tasks undo <tasklistId> <taskId>`
- `wk tasks delete <tasklistId> <taskId>`
- `wk tasks clear <tasklistId>`
- `wk contacts search <query> [--max N]`
- `wk contacts list [--max N] [--page TOKEN]`
- `wk contacts get <people/...|email>`
- `wk contacts create --given NAME [--family NAME] [--email addr] [--phone num]`
- `wk contacts update <people/...> [--given NAME] [--family NAME] [--email addr] [--phone num] [--birthday YYYY-MM-DD] [--notes TEXT] [--from-file PATH|-] [--ignore-etag]`
- `wk contacts delete <people/...>`
- `wk contacts directory list [--max N] [--page TOKEN]`
- `wk contacts directory search <query> [--max N] [--page TOKEN]`
- `wk contacts other list [--max N] [--page TOKEN]`
- `wk contacts other search <query> [--max N]`
- `wk people me`
- `wk people get <people/...|userId>`
- `wk people search <query> [--max N] [--page TOKEN]`
- `wk people relations [<people/...|userId>] [--type TYPE]`

Date/time input conventions (shared parser):

- Date-only: `YYYY-MM-DD`
- Datetime: `RFC3339` / `RFC3339Nano` / `YYYY-MM-DDTHH:MM[:SS]` / `YYYY-MM-DD HH:MM[:SS]`
- Numeric timezone offset accepted: `YYYY-MM-DDTHH:MM:SS-0800`
- Calendar range flags also accept relatives: `now`, `today`, `tomorrow`, `yesterday`, weekday names (`monday`, `next friday`)
- Tracking `--since` also accepts durations like `24h`

### Planned high-level command tree

- `wk auth …`
  - `wk auth credentials <credentials.json>`
  - `wk auth credentials list`
  - `wk --client <name> auth credentials <credentials.json>`
- `wk gmail …`
- `wk chat …`
- `wk calendar …`
- `wk drive …`
- `wk contacts …`
- `wk tasks …`
- `wk people …`

Planned service identifiers (canonical):

- `gmail`
- `calendar`
- `chat`
- `drive`
- `contacts`
- `tasks`
- `people`

## Google API dependencies (planned)

- `golang.org/x/oauth2`
- `golang.org/x/oauth2/google`
- `google.golang.org/api/option`
- `google.golang.org/api/gmail/v1`
- `google.golang.org/api/calendar/v3`
- `google.golang.org/api/chat/v1`
- `google.golang.org/api/drive/v3`
- `google.golang.org/api/people/v1`
- `google.golang.org/api/tasks/v1`

## Scopes (planned)

We store a single refresh token per Google account email.

- `wk auth add` requests a union of scopes based on `--services`.
- Each API client refreshes an access token for the subset of scopes needed for that service.
- If you later want additional services, re-run `wk auth add <email> --services ...` (may require `--force-consent` to mint a new refresh token).

- Gmail: `https://mail.google.com/` (or narrower scopes if we decide later)
- Calendar: `https://www.googleapis.com/auth/calendar`
- Chat:
  - `https://www.googleapis.com/auth/chat.spaces`
  - `https://www.googleapis.com/auth/chat.messages`
  - `https://www.googleapis.com/auth/chat.memberships`
  - `https://www.googleapis.com/auth/chat.users.readstate.readonly`
- Drive: `https://www.googleapis.com/auth/drive`
- Contacts/Directory:
  - `https://www.googleapis.com/auth/contacts`
  - `https://www.googleapis.com/auth/contacts.other.readonly`
  - `https://www.googleapis.com/auth/directory.readonly`
- People:
  - `profile` (OIDC)

## Output formats

Default: human-friendly tables (stdlib `text/tabwriter`).

- Parseable stdout:
  - `--json`: JSON objects/arrays suitable for scripting
  - `--plain`: stable TSV (tabs preserved; no alignment; no colors)
- Human-facing hints/progress are written to stderr so stdout can be safely captured.
- Colors are only used for human-facing output and are disabled automatically for `--json` and `--plain`.

We avoid heavy table deps unless we decide we need them.

## Code layout (current)

- `cmd/wk/main.go` — binary entrypoint
- `internal/cmd/*` — kong command structs
- `internal/ui/*` — color + printing
- `internal/config/*` — config paths + credential parsing/writing
- `internal/secrets/*` — keyring store

## Formatting, linting, tests

### Formatting

Pinned tools, installed into local `.tools/` via `make tools`:

- `mvdan.cc/gofumpt@v0.7.0`
- `golang.org/x/tools/cmd/goimports@v0.38.0`
- `github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2`

Commands:

- `make fmt` — applies `goimports` + `gofumpt`
- `make fmt-check` — formats and fails if Go files or `go.mod/go.sum` change

### Lint

- `golangci-lint` with config in `.golangci.yml`
- `make lint`

### Tests

- stdlib `testing` (+ `httptest` when we add OAuth/API tests)
- `make test`

### Integration tests (local only)

There is an opt-in integration test suite guarded by build tags (not run in CI).

- Requires:
  - stored `credentials.json` (or `credentials-<client>.json`) via `wk auth credentials ...`
  - refresh token in keyring via `wk auth add <email>`
- Run:
  - `WK_IT_ACCOUNT=you@gmail.com go test -tags=integration ./internal/integration`
  - optional: `WK_CLIENT=work` to select a non-default OAuth client

## CI (GitHub Actions)

Workflow: `.github/workflows/ci.yml`

- runs on push + PR
- uses `actions/setup-go` with `go-version-file: go.mod`
- runs:
  - `make tools`
  - `make fmt-check`
  - `go test ./...`
  - `golangci-lint` (pinned `v1.62.2`)

## Next implementation steps

- Expand Gmail further (labels by name everywhere, richer body rendering, compose edge cases).
- Improve People updates (multi-field + richer contact data).
- Harden UX (consistent output formats, retries/backoff on specific transient errors).
