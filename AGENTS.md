# Repository Guidelines

## Project Structure

- `cmd/gog/`: CLI entrypoint.
- `internal/`: implementation (`cmd/`, Google API/OAuth, config/secrets, output/UI).
- Tests: `*_test.go` next to code; opt-in integration suite in `internal/integration/` (build-tagged).
- `bin/`: build outputs; `docs/`: specs/releasing; `scripts/`: release helpers + `scripts/gog.mjs`.

## Build, Test, and Development Commands

- `make` / `make build`: build `bin/gog`.
- `make tools`: install pinned dev tools into `.tools/`.
- `make fmt` / `make lint` / `make test` / `make ci`: format, lint, test, full local gate.
- Optional: `pnpm gog …`: build + run in one step.
- Hooks: `lefthook install` enables pre-commit/pre-push checks.

## Coding Style & Naming Conventions

- Formatting: `make fmt` (`goimports` local prefix `github.com/steipete/gogcli` + `gofumpt`).
- Output: keep stdout parseable (`--json` / `--plain`); send human hints/progress to stderr.
- Gmail labels: treat label IDs as case-sensitive opaque tokens; only case-fold label names for name lookup.

## Testing Guidelines

- Unit tests: stdlib `testing` (and `httptest` where needed).
- Integration tests (local only):
  - `GOG_IT_ACCOUNT=you@gmail.com go test -tags=integration ./internal/integration`
  - Requires OAuth client credentials + a stored refresh token in your keyring.

## Commit & Pull Request Guidelines

- Create commits with `committer "<msg>" <file...>`; avoid manual staging.
- Follow Conventional Commits + action-oriented subjects (e.g. `feat(cli): add --verbose to send`).
- Group related changes; avoid bundling unrelated refactors.
- PRs should summarize scope, note testing performed, and mention any user-facing changes or new flags.
- PR review flow: when given a PR link, review via `gh pr view` / `gh pr diff` and do not change branches.

### PR Workflow (Review vs Land)

- **Review mode (PR link only):** read `gh pr view/diff`; do not switch branches; do not change code.
- **Landing mode:** temp branch from `main`; bring in PR (squash default; rebase/merge when needed); fix; update `CHANGELOG.md` (PR #/issue + thanks); run `make ci`; final commit; merge to `main`; delete temp; end on `main`.
- If we squash, add `Co-authored-by:` for the PR author when appropriate; leave a PR comment with what landed + SHAs.
- New contributor: thank in `CHANGELOG.md` (and update README contributors list if present).

## Agent UX Features

### Field Discovery (`--select ""`)

Pass an explicit empty string to `--select` to list available JSON fields without making an API call:

```
gog drive ls --json --select ""
```

- Output goes to **stderr** (stdout stays clean).
- Exit code **0** -- no API call is made.
- A usage hint is printed: `gog drive ls --json --select "name,id,size"`.
- Works for every command that supports `--json` output.

### Input Templates (`--generate-input` / `--gen-input`)

Print a JSON template showing all flags for any command:

```
gog gmail send --generate-input
```

- Required fields are prefixed with `(required)` in the value.
- Types, defaults, and enum values are included.
- Exit code **0** -- the command is not executed.
- Excludes Kong built-ins (`--help`, `--version`) and hidden flags.
- Includes both global `RootFlags` and command-specific flags.

### Global Pagination (`--max-results`, `--page-token`)

Control pagination across all services with global flags:

```
gog drive ls --max-results 5 --json
gog drive ls --max-results 5 --page-token TOKEN --json
```

- Maps to the correct API parameter per service (`pageSize` or `maxResults`).
- **Precedence:** per-command `--max`/`--limit` overrides `--max-results` when both are provided.
- `--all` overrides `--max-results` (fetches all pages).
- `--results-only` strips `nextPageToken` from output; avoid it when paginating across multiple pages.

### Help Topics (`gog agent help <topic>`)

Concept-level documentation for agent integration:

```
gog agent help topics          # list all topics
gog agent help auth            # authentication guide
gog agent help output          # output modes, --json, --select, exit codes
gog agent help agent           # zero-shot patterns, recommended flags
gog agent help pagination      # pagination control
gog agent help errors          # error handling, exit codes, retry guidance
```

- JSON output with `--json` flag.
- Unknown topics suggest the closest match.

## Security & Configuration Tips

- Never commit OAuth client credential JSON files or tokens.
- Prefer OS keychain backends; use `GOG_KEYRING_BACKEND=file` + `GOG_KEYRING_PASSWORD` only for headless environments.

### file:// Input Security

Text-content flags (`--body`, `--content`, `--description`, `--text`) support `file://` and `fileb://` prefixes to read content from files. Security constraints:

- **CWD-scoped:** files must reside within the current working directory subtree. Paths like `../../etc/passwd` are rejected.
- **Symlink validation:** symlinks are allowed only if their resolved target is also within the CWD subtree. Symlinks pointing outside CWD are rejected.
- **Sensitive file rejection (case-insensitive):** `.env`, `.env.*`, `.ssh/*`, `.aws/*`, `.gcloud/*`, `*credentials*`, `*secret*`, `*token*`, `*.pem`, `*.key`, `*.p12`, `*.pfx`, `id_rsa`, `id_ed25519`, `id_dsa`.
- **Size limit:** 10 MB maximum.
- **Prefixes:** `file://` reads UTF-8 text; `fileb://` reads binary and returns base64-encoded content. No prefix means literal passthrough.
- **Coexistence:** `file://` coexists with dedicated `--body-file` / `--content-file` flags. They are independent mechanisms; neither is deprecated.


## ⚠️ Worktree Policy (MANDATORY)

**NEVER work on main for feature development.**

Before ANY code work:
1. Verify branch: `git branch --show-current` (must NOT be main)
2. If on main → STOP → create worktree or cd to existing one
3. THEN start editing

Full policy: `/home/genie/workspace/docs/WORKTREE-RULES.md`
