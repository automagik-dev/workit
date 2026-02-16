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
- One new Go dependency: `gopkg.in/yaml.v3` (for tier YAML parsing)

### OUT
- No MCP transport layer (explicitly excluded)
- No OAuth 2.1 multi-user sessions (CLI is single-user)
- No SSRF protections (gogcli doesn't fetch arbitrary URLs)
- No Google Custom Search (outside workspace scope; separate wish if needed)
- No attachment auto-expiry (CLI paradigm -- user manages files)
- No breaking changes to existing commands

---

## Coordination Notes

- **root.go shared modification:** This wish modifies `internal/cmd/root.go` (Groups 2 and 3). The wishes `agent-cli-ux-unified` and `agent-cli-power-features-v2` also modify `root.go`. Recommend execution order: this wish first (adds `--command-tier` and `--read-only`), then `agent-cli-ux-unified` (adds `--max-results`, `--page-token`, `--generate-input`), then `agent-cli-power-features-v2` (adds `--jq`). Each wish must rebase against the latest `root.go` before merging.

---

## Decisions

- **DEC-1:** Tier config is embedded YAML (`//go:embed`) -- human-readable, matches MCP pattern, auto-included in binary. Requires adding `gopkg.in/yaml.v3` as a new dependency.
- **DEC-2:** Read-only mode is dual-layer (scope + command hiding) for true safety guarantee -- `--dry-run` only previews.
- **DEC-3:** Office text extraction uses Go stdlib only (`archive/zip` + `encoding/xml`) -- no external deps.
- **DEC-4:** `--command-tier` and `--enable-commands` are composable -- both filters apply independently.
- **DEC-5:** Batch contacts chunks at 200 per request (Google API limit) with existing retry/backoff transport. Each chunk is retried independently. Output is per-contact JSON (success/failure per item).

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

- **ASM-1:** ~~INVALIDATED.~~ `gopkg.in/yaml.v3` is NOT currently in `go.mod`. DEC-1 updated to explicitly allow adding it as a new dependency. Scope updated accordingly.
- **ASM-2:** Existing People API client is in `internal/googleapi/people.go` (NOT `contacts.go`). Batch endpoint availability (e.g., `people.batchCreateContacts`, `people.batchDeleteContacts`, `people.batchUpdateContacts`) must be verified with `go doc google.golang.org/api/people/v1` before implementation. If batch methods are unavailable in the Go client library, fall back to sequential calls with concurrent goroutines.
- **ASM-3:** Most Google services have `.readonly` scope variants available. **Exceptions:** Google Keep and Google People (Contacts) APIs do not expose `.readonly` scope variants. These services are documented as no-ops in read-only mode (all commands hidden since no read-only scope exists).

## Risks

- **RISK-1:** Tier YAML maintenance burden -- Mitigation: auto-generate initial config from command tree; CI lint to flag unlisted commands; auto-generation script provided as Group 2 deliverable.
- **RISK-2:** Read-only mode incomplete coverage (missing write command tags) -- Mitigation: audit all commands; test matrix; default-deny (untagged commands hidden in read-only).
- **RISK-3:** Office XML edge cases (macros, embedded objects, corrupted files) -- Mitigation: plain text only; document limitations; fail gracefully with error message.
- **RISK-4:** Batch contacts API quotas -- Mitigation: existing retry/backoff in transport layer handles rate limits.

---

## Execution Groups

### Group 1: Office Format Text Extraction

**Goal:** Create `gog drive cat` subcommand and enable `gog drive cat document.docx` to output plain text from DOCX/XLSX/PPTX files.

**Deliverables:**
- NEW: `internal/officetext/extract.go` -- dispatcher by MIME type
- NEW: `internal/officetext/docx.go` -- DOCX extractor (parse `word/document.xml`, extract `<w:t>` nodes)
- NEW: `internal/officetext/xlsx.go` -- XLSX extractor (parse `xl/sharedStrings.xml` + worksheets)
- NEW: `internal/officetext/pptx.go` -- PPTX extractor (parse `ppt/slides/slide*.xml`, extract `<a:t>` nodes)
- NEW: `internal/officetext/extract_test.go` -- unit tests with small fixture files
- NEW: `internal/officetext/testdata/` -- minimal fixture files (one `.docx`, one `.xlsx`, one `.pptx`, each <10KB, committed to repo)
- NEW: `internal/cmd/drive_cat.go` -- new `DriveCatCmd` subcommand registered under `drive`
- Modify: `internal/cmd/drive.go` -- register `cat` subcommand in the `DriveCmd` struct

**Note:** There is no existing `drive cat` command. This group CREATES `DriveCatCmd` as a new subcommand of `drive`. It downloads the file from Drive (using export for Google Docs native formats, or direct download for uploaded files), then applies text extraction if the file is DOCX/XLSX/PPTX.

**Acceptance Criteria:**
- [ ] `gog drive cat report.docx` outputs plain text extracted from DOCX
- [ ] `gog drive cat data.xlsx` outputs cell contents from XLSX
- [ ] `gog drive cat slides.pptx` outputs slide text from PPTX
- [ ] Unknown/corrupted files fall back to raw download with warning on stderr
- [ ] Unit tests pass with fixture files for each format
- [ ] Fixture files located in `internal/officetext/testdata/` (one per format, minimal size)
- [ ] Zero new dependencies (uses `archive/zip` + `encoding/xml` from stdlib)

**Validation:** `make test && go test ./internal/officetext/...`

---

### Group 2: Three-Tier Command System

**Goal:** Add `--command-tier core|extended|complete` flag to control visible command surface for agent integrations.

**Deliverables:**
- NEW: `internal/cmd/command_tiers.yaml` -- tier definitions mapping every subcommand to core/extended/complete
- NEW: `internal/cmd/gen_tiers.go` -- auto-generation script that walks the Kong command tree and outputs a starter YAML with all commands listed (run via `go generate`)
- Modify: `internal/cmd/enabled_commands.go` -- extend with tier filtering logic
- Modify: `internal/cmd/root.go` -- add `--command-tier` flag to `RootFlags`
- NEW: `internal/cmd/command_tiers_test.go` -- test tier filtering + composability with `--enable-commands`
- Modify: `go.mod` -- add `gopkg.in/yaml.v3` dependency

**Acceptance Criteria:**
- [ ] `gog --command-tier core gmail --help` shows only core Gmail subcommands (search, send, get)
- [ ] `gog --command-tier extended gmail --help` shows core + extended (labels, batch, settings)
- [ ] `gog --command-tier complete gmail --help` shows all (default behavior)
- [ ] `--command-tier` and `--enable-commands` compose correctly (both filters apply)
- [ ] YAML embedded via `//go:embed` -- no external files needed at runtime
- [ ] Unit tests verify tier filtering for at least 3 services
- [ ] All services have tier definitions in the YAML; commands not listed in the YAML cause a CI lint failure
- [ ] Auto-generation script (`go generate ./internal/cmd/...`) produces a valid starter YAML

**Validation:** `make test && grep -l 'command.tier\|CommandTier' internal/cmd/*.go`

---

### Group 3: True Read-Only Mode

**Goal:** Add `--read-only` flag that enforces read-only access at OAuth scope level AND hides write commands.

**Deliverables:**
- Modify: `internal/googleauth/service.go` -- add readonly scope map (`.readonly` variants per service)
- Modify: `internal/cmd/root.go` -- add `--read-only` global flag, wire into command filtering
- Modify: Various `*_cmd.go` files -- tag commands as `read` or `write` (Kong group tag or struct tag)
- NEW: `internal/cmd/readonly_test.go` -- test scope switching + command hiding

**Acceptance Criteria:**
- [ ] `gog --read-only gmail --help` hides `send`, `delete`, `batch delete`
- [ ] `gog --read-only gmail send` errors with "command unavailable in read-only mode"
- [ ] OAuth flow with `--read-only` requests only `.readonly` scopes (e.g., `gmail.readonly`)
- [ ] `--read-only` composes with `--command-tier` (both filters stack)
- [ ] Default behavior unchanged (no `--read-only` = full access)
- [ ] Unit tests verify scope switching and command filtering
- [ ] Services without `.readonly` scope variants (Keep, People/Contacts) are documented as fully hidden in read-only mode. These are no-ops: `gog --read-only keep --help` shows no commands with a message explaining why.

**Validation:** `make test && grep -l 'ReadOnly\|read.only' internal/cmd/*.go internal/googleauth/*.go`

---

### Group 4: Batch Contacts Operations

**Goal:** Add `gog contacts batch create|update|delete` for multi-contact operations via People API batch endpoints.

**Deliverables:**
- NEW: `internal/cmd/contacts_batch.go` -- batch create/update/delete subcommands
- NEW or Modify: `internal/googleapi/people.go` -- batch API client methods (verify batch method availability with `go doc` first; fall back to concurrent sequential calls if unavailable)
- Modify: `internal/cmd/contacts_cmd.go` -- register `batch` subcommand group
- NEW: `internal/cmd/contacts_batch_test.go` -- unit tests

**Chunking contract:** Inputs larger than 200 contacts are split into chunks of 200. Each chunk is sent as an independent API call. Per-chunk retry uses the existing transport retry/backoff. Output is streamed as per-contact JSON objects (one JSON object per contact showing success/failure status), enabling agents to parse partial results even if later chunks fail.

**Acceptance Criteria:**
- [ ] `gog contacts batch create --file contacts.json` creates multiple contacts (JSON array input)
- [ ] `echo '[...]' | gog contacts batch create` accepts JSON from stdin
- [ ] `gog contacts batch delete name1 name2` deletes multiple contacts
- [ ] Batch size capped at 200 per API call; auto-chunks larger inputs
- [ ] Each chunk is retried independently on transient failure
- [ ] `--dry-run` previews without executing
- [ ] JSON output shows per-contact success/failure (one JSON object per contact)
- [ ] Unit tests mock People API batch endpoints

**Validation:** `make test && grep -l 'ContactsBatch\|BatchCreate\|BatchDelete' internal/cmd/contacts*.go`

---

## Dependencies

```
gopkg.in/yaml.v3   — NEW (Group 2: YAML tier config parsing)
```

---

## Files to Create/Modify

```
# Group 1: Office Text Extraction
internal/officetext/extract.go          # NEW — dispatcher
internal/officetext/docx.go             # NEW — DOCX extractor
internal/officetext/xlsx.go             # NEW — XLSX extractor
internal/officetext/pptx.go             # NEW — PPTX extractor
internal/officetext/extract_test.go     # NEW — unit tests
internal/officetext/testdata/           # NEW — fixture files (one per format)
internal/cmd/drive_cat.go              # NEW — DriveCatCmd subcommand
internal/cmd/drive.go                   # Modify — register cat subcommand

# Group 2: Three-Tier Command System
internal/cmd/command_tiers.yaml         # NEW — tier definitions
internal/cmd/gen_tiers.go              # NEW — auto-generation script
internal/cmd/enabled_commands.go        # Modify — tier filtering
internal/cmd/root.go                    # Modify — --command-tier flag
internal/cmd/command_tiers_test.go      # NEW — tier tests
go.mod                                 # Modify — add gopkg.in/yaml.v3

# Group 3: True Read-Only Mode
internal/googleauth/service.go          # Modify — readonly scope map
internal/cmd/root.go                    # Modify — --read-only flag
internal/cmd/*_cmd.go                   # Modify — read/write tags
internal/cmd/readonly_test.go           # NEW — readonly tests

# Group 4: Batch Contacts
internal/cmd/contacts_batch.go          # NEW — batch commands
internal/cmd/contacts_batch_test.go     # NEW — batch tests
internal/googleapi/people.go            # Modify — batch API methods
internal/cmd/contacts_cmd.go            # Modify — register batch group
```
