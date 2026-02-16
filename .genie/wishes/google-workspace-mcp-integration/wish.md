# Wish: Google Workspace MCP Feature Integration

**Status:** DRAFT
**Slug:** `google-workspace-mcp-integration`
**Created:** 2026-02-16
**Design:** `.genie/brainstorms/google-workspace-mcp-integration/design.md`

---

## Summary

Port 4 architectural patterns from [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp) into gogcli to enhance agent-deployment safety and power-user workflows. gogcli already has superior API coverage (14 services, 337+ commands); this wish borrows safety patterns and convenience features, not API coverage.

---

## Scope

### IN
- Three-tier command system (`--command-tier core|extended|complete`) via embedded YAML
- True read-only mode (`--read-only`) with dual-layer enforcement (OAuth scopes + command hiding)
- Office format text extraction (`gog drive cat file.docx`) for DOCX/XLSX/PPTX using Go stdlib
- Batch contacts operations (`gog contacts batch create|update|delete`) via People API batch endpoints
- Unit tests for every new feature
- JSON output support for all new commands
- No new Go dependencies (all stdlib)

### OUT
- No MCP transport layer (explicitly excluded)
- No OAuth 2.1 multi-user sessions (CLI is single-user)
- No SSRF protections (gogcli doesn't fetch arbitrary URLs)
- No Google Custom Search (outside workspace scope; separate wish if needed)
- No attachment auto-expiry (CLI paradigm — user manages files)
- No breaking changes to existing commands

---

## Decisions

- **DEC-1:** Tier config is embedded YAML (`//go:embed`) — human-readable, matches MCP pattern, auto-included in binary.
- **DEC-2:** Read-only mode is dual-layer (scope + command hiding) for true safety guarantee — `--dry-run` only previews.
- **DEC-3:** Office text extraction uses Go stdlib only (`archive/zip` + `encoding/xml`) — no external deps.
- **DEC-4:** `--command-tier` and `--enable-commands` are composable — both filters apply independently.
- **DEC-5:** Batch contacts chunks at 200 per request (Google API limit) with existing retry/backoff transport.

---

## Success Criteria

- [ ] `--command-tier core` restricts Gmail to only core subcommands; `complete` shows all (default)
- [ ] `--read-only` requests `.readonly` OAuth scopes and hides all write commands
- [ ] `gog drive cat report.docx` outputs plain text from DOCX stored in Drive
- [ ] `gog contacts batch create --file contacts.json` creates multiple contacts in one API call
- [ ] `make ci` passes
- [ ] No regressions in existing command behavior

---

## Assumptions

- **ASM-1:** `gopkg.in/yaml.v3` or equivalent YAML parser already exists in go.mod for tier config parsing.
- **ASM-2:** Existing People API client in `internal/googleapi/contacts.go` supports batch endpoints or can be extended.
- **ASM-3:** All Google services have `.readonly` scope variants available.

## Risks

- **RISK-1:** Tier YAML maintenance burden — Mitigation: auto-generate initial config from command tree; CI lint to flag unlisted commands.
- **RISK-2:** Read-only mode incomplete coverage (missing write command tags) — Mitigation: audit all commands; test matrix; default-deny (untagged commands hidden in read-only).
- **RISK-3:** Office XML edge cases (macros, embedded objects, corrupted files) — Mitigation: plain text only; document limitations; fail gracefully with error message.
- **RISK-4:** Batch contacts API quotas — Mitigation: existing retry/backoff in transport layer handles rate limits.

---

## Execution Groups

### Group 1: Office Format Text Extraction

**Goal:** Enable `gog drive cat document.docx` to output plain text from DOCX/XLSX/PPTX files.

**Deliverables:**
- NEW: `internal/officetext/extract.go` — dispatcher by MIME type
- NEW: `internal/officetext/docx.go` — DOCX extractor (parse `word/document.xml`, extract `<w:t>` nodes)
- NEW: `internal/officetext/xlsx.go` — XLSX extractor (parse `xl/sharedStrings.xml` + worksheets)
- NEW: `internal/officetext/pptx.go` — PPTX extractor (parse `ppt/slides/slide*.xml`, extract `<a:t>` nodes)
- NEW: `internal/officetext/extract_test.go` — unit tests with small fixture files
- Modify: `internal/cmd/drive.go` — integrate text extraction into `cat` command path

**Acceptance Criteria:**
- [ ] `gog drive cat report.docx` outputs plain text extracted from DOCX
- [ ] `gog drive cat data.xlsx` outputs cell contents from XLSX
- [ ] `gog drive cat slides.pptx` outputs slide text from PPTX
- [ ] Unknown/corrupted files fall back to raw download with warning on stderr
- [ ] Unit tests pass with fixture files for each format
- [ ] Zero new dependencies (uses `archive/zip` + `encoding/xml` from stdlib)

**Validation:** `make test && go test ./internal/officetext/...`

---

### Group 2: Three-Tier Command System

**Goal:** Add `--command-tier core|extended|complete` flag to control visible command surface for agent integrations.

**Deliverables:**
- NEW: `internal/cmd/command_tiers.yaml` — tier definitions mapping every subcommand to core/extended/complete
- Modify: `internal/cmd/enabled_commands.go` — extend with tier filtering logic
- Modify: `internal/cmd/root.go` — add `--command-tier` flag to `RootFlags`
- NEW: `internal/cmd/command_tiers_test.go` — test tier filtering + composability with `--enable-commands`

**Acceptance Criteria:**
- [ ] `gog --command-tier core gmail --help` shows only core Gmail subcommands (search, send, get)
- [ ] `gog --command-tier extended gmail --help` shows core + extended (labels, batch, settings)
- [ ] `gog --command-tier complete gmail --help` shows all (default behavior)
- [ ] `--command-tier` and `--enable-commands` compose correctly (both filters apply)
- [ ] YAML embedded via `//go:embed` — no external files needed at runtime
- [ ] Unit tests verify tier filtering for at least 3 services

**Validation:** `make test && grep -l 'command.tier\|CommandTier' internal/cmd/*.go`

---

### Group 3: True Read-Only Mode

**Goal:** Add `--read-only` flag that enforces read-only access at OAuth scope level AND hides write commands.

**Deliverables:**
- Modify: `internal/googleauth/service.go` — add readonly scope map (`.readonly` variants per service)
- Modify: `internal/cmd/root.go` — add `--read-only` global flag, wire into command filtering
- Modify: Various `*_cmd.go` files — tag commands as `read` or `write` (Kong group tag or struct tag)
- NEW: `internal/cmd/readonly_test.go` — test scope switching + command hiding

**Acceptance Criteria:**
- [ ] `gog --read-only gmail --help` hides `send`, `delete`, `batch delete`
- [ ] `gog --read-only gmail send` errors with "command unavailable in read-only mode"
- [ ] OAuth flow with `--read-only` requests only `.readonly` scopes (e.g., `gmail.readonly`)
- [ ] `--read-only` composes with `--command-tier` (both filters stack)
- [ ] Default behavior unchanged (no `--read-only` = full access)
- [ ] Unit tests verify scope switching and command filtering

**Validation:** `make test && grep -l 'ReadOnly\|read.only' internal/cmd/*.go internal/googleauth/*.go`

---

### Group 4: Batch Contacts Operations

**Goal:** Add `gog contacts batch create|update|delete` for multi-contact operations via People API batch endpoints.

**Deliverables:**
- NEW: `internal/cmd/contacts_batch.go` — batch create/update/delete subcommands
- NEW or Modify: `internal/googleapi/contacts.go` — batch API client methods
- Modify: `internal/cmd/contacts_cmd.go` — register `batch` subcommand group
- NEW: `internal/cmd/contacts_batch_test.go` — unit tests

**Acceptance Criteria:**
- [ ] `gog contacts batch create --file contacts.json` creates multiple contacts (JSON array input)
- [ ] `echo '[...]' | gog contacts batch create` accepts JSON from stdin
- [ ] `gog contacts batch delete name1 name2` deletes multiple contacts
- [ ] Batch size capped at 200 per API call; auto-chunks larger inputs
- [ ] `--dry-run` previews without executing
- [ ] JSON output shows per-contact success/failure
- [ ] Unit tests mock People API batch endpoints

**Validation:** `make test && grep -l 'ContactsBatch\|BatchCreate\|BatchDelete' internal/cmd/contacts*.go`

---

## Files to Create/Modify

```
# Group 1: Office Text Extraction
internal/officetext/extract.go          # NEW — dispatcher
internal/officetext/docx.go             # NEW — DOCX extractor
internal/officetext/xlsx.go             # NEW — XLSX extractor
internal/officetext/pptx.go             # NEW — PPTX extractor
internal/officetext/extract_test.go     # NEW — unit tests
internal/cmd/drive.go                   # Modify — integrate into cat

# Group 2: Three-Tier Command System
internal/cmd/command_tiers.yaml         # NEW — tier definitions
internal/cmd/enabled_commands.go        # Modify — tier filtering
internal/cmd/root.go                    # Modify — --command-tier flag
internal/cmd/command_tiers_test.go      # NEW — tier tests

# Group 3: True Read-Only Mode
internal/googleauth/service.go          # Modify — readonly scope map
internal/cmd/root.go                    # Modify — --read-only flag
internal/cmd/*_cmd.go                   # Modify — read/write tags
internal/cmd/readonly_test.go           # NEW — readonly tests

# Group 4: Batch Contacts
internal/cmd/contacts_batch.go          # NEW — batch commands
internal/cmd/contacts_batch_test.go     # NEW — batch tests
internal/googleapi/contacts.go          # Modify — batch API methods
internal/cmd/contacts_cmd.go            # Modify — register batch group
```
