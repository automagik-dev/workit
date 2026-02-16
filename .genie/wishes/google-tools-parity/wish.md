# Wish: Google Tools Feature Parity & Enhancement

**Status:** DRAFT
**Slug:** `google-tools-parity`
**Created:** 2026-02-15
**Design:** `.genie/brainstorms/google-tools-parity/design.md`

---

## Summary

Enhance gog-cli with 7 verified feature gaps and 3 infrastructure optimizations discovered by comparing all 12 google-* tools from automagik-tools (Python/MCP) against gog-cli's existing capabilities. gog-cli already leads in 7 of 9 services; this wish closes the remaining gaps to make it the most complete Google Workspace CLI for AI agents.

---

## Scope

### IN
- 7 new commands/features across Docs, Forms, Sheets, and Drive
- 3 infrastructure improvements (error handling, scope management)
- Unit tests for every new command (`*_test.go` co-located)
- JSON output (`--json`) for all new commands
- Following upstream gogcli patterns (Kong structs, `internal/cmd/`)

### OUT
- No changes to Calendar, Gmail, Slides, Tasks, Chat (gog already leads)
- No Python/MCP code (this is Go CLI only)
- No new Google API client libraries (use existing `internal/googleapi/`)
- No breaking changes to existing commands
- No integration tests (would require live Google account)
- No JSON-to-Docs template format design (Phase 3 stretch - spec separately if reached)

---

## Decisions

- **DEC-1:** All 10 items shipped in 3 phases (quick wins -> agent workflow -> stretch) to manage scope and deliver value incrementally.
- **DEC-2:** Follow existing Kong command struct pattern from `internal/cmd/` — no new frameworks or patterns.
- **DEC-3:** New Docs/Forms/Sheets commands use the existing `googleapi.New*` service constructors — no new auth flows needed.
- **DEC-4:** Error improvements go into `internal/googleapi/errors.go` (existing file) — not a new package.
- **DEC-5:** Phase 3 (JSON-to-Docs) deferred as stretch — may become a separate wish if scope is too large.

---

## Success Criteria

- [ ] All Phase 1 commands exist and produce correct JSON + plain output
- [ ] All Phase 2 commands exist and produce correct JSON + plain output
- [ ] `make ci` passes after each phase
- [ ] No regressions in existing command behavior
- [ ] Error messages for disabled APIs include GCP console URL hints

---

## Assumptions

- **ASM-1:** Existing OAuth scopes already cover the new operations (Docs, Forms, Sheets write scopes are already requested).
- **ASM-2:** The `--enable-commands` flag infrastructure exists and can be used for dynamic scope filtering.

## Risks

- **RISK-1:** Upstream compatibility — New commands may not align with upstream gogcli patterns. Mitigation: Follow existing command patterns exactly; keep PRable.
- **RISK-2:** Scope creep — 10 features could balloon. Mitigation: Hard phase boundaries; each phase is independently shippable.
- **RISK-3:** Test coverage — Each feature needs tests. Mitigation: TDD per item; `make ci` gate before merge.

---

## Execution Groups

### Group 1: Docs Structure Inspection

**Goal:** Add a `docs structure` command that returns the document element tree (headings, paragraphs, tables, images) so agents can understand layout before editing.

**Deliverables:**
- `internal/cmd/docs.go` — New `DocsStructureCmd` Kong struct
- Command: `gog docs structure <docId>` with `--json` and plain output
- Unit test in `internal/cmd/docs_commands_test.go` or new `docs_structure_test.go`

**Acceptance Criteria:**
- [ ] `gog docs structure <docId>` returns element tree with type, index, and content summary
- [ ] `gog docs structure <docId> --json` returns structured JSON array of elements
- [ ] Unit test verifies element extraction from mock document response

**Validation:** `make test && grep -l 'DocsStructureCmd' internal/cmd/docs*.go`

---

### Group 2: Forms Publish Settings

**Goal:** Add `forms publish` command to control form publish settings (template mode and authentication requirements).

**Deliverables:**
- `internal/cmd/forms.go` — New `FormsPublishCmd` Kong struct
- Command: `gog forms publish <formId> --publish-as-template --require-authentication`
- API: Uses Forms API `setPublishSettings` endpoint (NOT batchUpdate)
- Unit test

**Acceptance Criteria:**
- [ ] `gog forms publish <formId> --publish-as-template` calls Forms API `setPublishSettings` with `publishAsTemplate: true`
- [ ] `gog forms publish <formId> --require-authentication` calls Forms API `setPublishSettings` with `requireAuthentication: true`
- [ ] JSON and plain output confirm the new publish settings state
- [ ] Unit test mocks the `setPublishSettings` call

**Validation:** `make test && grep -l 'FormsPublishCmd' internal/cmd/forms*.go`

---

### Group 3: Sheets Add Tab

**Goal:** Add `sheets add-tab` command to create a new worksheet in an existing spreadsheet.

**Deliverables:**
- `internal/cmd/sheets.go` — New `SheetsAddTabCmd` Kong struct
- Command: `gog sheets add-tab <spreadsheetId> <tabName>` with optional `--index`
- Unit test

**Acceptance Criteria:**
- [ ] `gog sheets add-tab <id> "Sheet2"` calls Sheets API batchUpdate with AddSheet request
- [ ] JSON output includes new sheet ID and properties
- [ ] Unit test mocks the batchUpdate call

**Validation:** `make test && grep -l 'SheetsAddTabCmd\|AddTab' internal/cmd/sheets*.go`

---

### Group 4: Centralize API Enablement Error Handling

**Goal:** Centralize existing ad-hoc API enablement error detection into a reusable function. gog-cli already has enablement hints in 4+ files (calendar_users.go, classroom_helpers.go, groups.go, people_helpers.go) using ad-hoc `strings.Contains` checks. This group extracts them into a single utility.

**Deliverables:**
- `internal/googleapi/errors.go` — Add `WrapAPIEnablementError(err error, serviceName string) error` function
- Add `API_ENABLEMENT_LINKS` map (service name to GCP console URL), similar to automagik's `api_enablement.py`
- Migrate existing ad-hoc checks in `calendar_users.go`, `classroom_helpers.go`, `groups.go`, `people_helpers.go` to call the new centralized function
- Unit test for the error detection and hint generation

**Acceptance Criteria:**
- [ ] `WrapAPIEnablementError` (or similar) function exists in `internal/googleapi/errors.go`
- [ ] `API_ENABLEMENT_LINKS` map covers all services with known enablement URLs
- [ ] 403 errors containing "has not been used" or "is not enabled" get wrapped with enable hint
- [ ] Existing ad-hoc checks in calendar_users.go, classroom_helpers.go, groups.go, people_helpers.go are replaced with calls to the centralized function
- [ ] Non-403 errors pass through unmodified
- [ ] Unit test with mock 403 response verifies hint generation

**Validation:** `make test && grep -l 'WrapAPIEnablementError\|API_ENABLEMENT_LINKS' internal/googleapi/errors*.go`

---

### Group 5: Transient Error Classification (Refactor)

**Goal:** Extract transient error classification into a named, reusable `IsTransient` function. This is a refactor/code-quality improvement, not a new feature -- `internal/googleapi/transport.go` already correctly handles retry logic for 429 and 5xx errors (lines 83-128). The `IsTransient` function makes this classification available to other callers and improves readability.

**Deliverables:**
- `internal/googleapi/errors.go` — Add `IsTransient(err)` function
- Classify: 429, 500, 502, 503 as transient; 400, 401, 403, 404 as fatal
- Update retry transport to use `IsTransient` instead of inline status checks
- Unit test

**Acceptance Criteria:**
- [ ] `IsTransient` function exists and correctly classifies HTTP status codes
- [ ] Retry transport uses `IsTransient` for retry decisions
- [ ] Unit test covers all status code categories
- [ ] No behavior change for existing retry logic (refactor only)

**Validation:** `make test && grep -l 'IsTransient' internal/googleapi/*.go`

---

### Group 6: Docs Headers/Footers

**Goal:** Add commands to get and set document headers and footers.

**Deliverables:**
- `internal/cmd/docs.go` — New `DocsHeaderCmd` and `DocsFooterCmd` Kong structs
- Commands: `gog docs header <docId>` (get) and `gog docs header <docId> --set "text"` (set)
- Same pattern for `gog docs footer`
- Uses Docs API batchUpdate with CreateHeader/UpdateHeader requests
- Unit tests

**Acceptance Criteria:**
- [ ] `gog docs header <docId>` returns current header content (or "no header")
- [ ] `gog docs header <docId> --set "Company Report"` creates/updates the default header
- [ ] `gog docs footer <docId> --set "Page {PAGE}"` creates/updates the default footer
- [ ] JSON output for both get and set operations
- [ ] Unit tests mock Docs API calls

**Validation:** `make test && grep -l 'DocsHeaderCmd\|DocsFooterCmd' internal/cmd/docs*.go`

---

### Group 7: Sheets Batch Update

**Goal:** Expose raw Sheets batchUpdate API for power users.

**Deliverables:**
- `internal/cmd/sheets.go` — New `SheetsBatchUpdateCmd` Kong struct
- Command: `gog sheets batch-update <spreadsheetId>` reads JSON from stdin or `--file`
- Passes raw request body to Sheets API batchUpdate
- Unit test

**Acceptance Criteria:**
- [ ] `echo '{"requests":[...]}' | gog sheets batch-update <id>` sends raw batchUpdate
- [ ] `gog sheets batch-update <id> --file requests.json` reads from file
- [ ] Response includes replies from API
- [ ] Unit test mocks batchUpdate call

**Validation:** `make test && grep -l 'SheetsBatchUpdateCmd\|BatchUpdate' internal/cmd/sheets*.go`

---

### Group 8: Dynamic OAuth Scope Management

**Goal:** When `--enable-commands` is used, only request OAuth scopes for the enabled services, reducing the consent screen.

**Deliverables:**
- `internal/googleauth/scopes.go` — Add scope filtering based on enabled commands
- `internal/cmd/enabled_commands.go` — Wire scope filtering into auth flow
- Unit tests

**Acceptance Criteria:**
- [ ] `gog auth add --enable-commands gmail,drive` only requests Gmail + Drive scopes
- [ ] Default behavior (no flag) requests all scopes (backward compatible)
- [ ] Scope list is correctly derived from command-to-service mapping
- [ ] Unit test verifies scope filtering logic

**Validation:** `make test && grep -l 'ScopesForCommands\|filterScopes' internal/googleauth/*.go`

---

### Group 9: Drive Public Access Check

**Goal:** Add convenience command to check if a file is publicly accessible.

**Deliverables:**
- `internal/cmd/drive.go` — New `DriveCheckPublicCmd` Kong struct
- Command: `gog drive check-public <fileId>` returns boolean + permission details
- Unit test

**Acceptance Criteria:**
- [ ] `gog drive check-public <fileId>` returns `public: true/false` with permission type
- [ ] JSON output includes `{"public": true, "permission": {...}}` or `{"public": false}`
- [ ] Unit test with mock permissions list

**Validation:** `make test && grep -l 'DriveCheckPublicCmd\|CheckPublic' internal/cmd/drive*.go`

---

### Group 10: JSON-to-Docs Template Generation (Stretch)

**Goal:** Generate Google Docs from structured JSON templates with placeholder substitution.

**Deliverables:**
- `internal/cmd/docs_generate.go` — New `DocsGenerateCmd` Kong struct
- Command: `gog docs generate --template <templateDocId> --data <data.json> [--folder <folderId>]`
- Template format: `{{placeholder}}` keys in template doc, replaced with values from JSON
- Unit tests

**Acceptance Criteria:**
- [ ] `gog docs generate --template <id> --data data.json` creates new doc from template
- [ ] Placeholders like `{{name}}`, `{{date}}` are replaced with JSON values
- [ ] `--folder` places the new doc in specified folder
- [ ] JSON output includes new doc ID and URL
- [ ] Unit test with mock API calls

**Validation:** `make test && grep -l 'DocsGenerateCmd' internal/cmd/docs*.go`

---

## Review Results

**Pipeline:** Plan Review (wish draft)
**Verdict:** FIX-FIRST
**Date:** 2026-02-15

### Checklist

- [x] Problem statement is one sentence, testable
- [x] Scope IN has concrete deliverables
- [x] Scope OUT is not empty -- boundaries explicit
- [x] Every task has acceptance criteria that are testable
- [x] Tasks are bite-sized and independently shippable
- [x] Dependencies tagged (blocks/blockedBy)
- [x] Validation commands exist for each execution group

### Gaps Found

#### CRITICAL: Group 7 (Forms Question Management) -- Source Doesn't Have It Either

**Severity:** CRITICAL
**Evidence:** Grepped `/tmp/automagik-tools/automagik_tools/tools/google_forms/forms_tools.py` -- it has exactly 5 tools: `create_form`, `get_form`, `set_publish_settings`, `get_form_response`, `list_form_responses`. **Zero question management functions.** No `add_question`, `delete_question`, or `create_item` anywhere in automagik-tools.

**Impact:** Group 7 claims to close a "gap" vs automagik-tools, but automagik-tools doesn't have this feature either. This is scope creep -- a net-new feature, not parity.

**Fix:** Either (a) remove Group 7 entirely since it's not a parity gap, or (b) re-classify it as a "gog-cli enhancement" (not parity) and move it to Phase 3 stretch. Recommend (a) to keep scope tight.

#### HIGH: Group 2 (Forms Publish Settings) -- Wrong API Parameters

**Severity:** HIGH
**Evidence:** The wish says `--accepting-responses=true/false` and `Forms API batchUpdate with updateSettings`. But the actual automagik implementation uses `service.forms().setPublishSettings()` (a dedicated endpoint, NOT batchUpdate) with parameters `publishAsTemplate` and `requireAuthentication` -- NOT `accepting-responses`.

**Fix:** Update Group 2 to match the real Forms API:
- Command: `gog forms publish <formId> --publish-as-template --require-authentication`
- API call: `forms.setPublishSettings(formId, body)` not batchUpdate
- Acceptance criteria should reference correct parameters

#### HIGH: Group 4 (API Enablement Hints) -- Already Partially Implemented

**Severity:** HIGH
**Evidence:** gog-cli already has API enablement hints in multiple files:
- `internal/cmd/calendar_users.go:36`: "people API is not enabled; enable it at: https://console.developers.google.com/..."
- `internal/cmd/classroom_helpers.go:22`: "classroom API is not enabled; enable it at: ..."
- `internal/cmd/groups.go:139`: "Cloud Identity API is not enabled; enable it at: ..."
- `internal/cmd/people_helpers.go:32`: "people API is not enabled; enable it at: ..."

The current pattern is ad-hoc per-command checks. Group 4 should be rescoped as "centralize existing ad-hoc enablement checks into a reusable function in errors.go" rather than "add enablement hints" (they already exist).

**Fix:** Rescope Group 4 goal to "centralize and generalize existing API enablement error detection into `internal/googleapi/errors.go`" with a single `WrapAPIEnablementError(err, serviceName)` function that replaces the 4+ ad-hoc implementations.

#### MEDIUM: Group 5 (IsTransient) -- Already Effectively Implemented in Transport

**Severity:** MEDIUM
**Evidence:** `internal/googleapi/transport.go` already correctly classifies:
- 429 as retryable (lines 83-103)
- 5xx as retryable (lines 106-128)
- 4xx (except 429) as non-retryable (line 131)

This is exactly what `IsTransient` would do. The function would be a thin wrapper around logic that already works.

**Fix:** Downgrade to LOW priority. It's still useful as a refactor (extract the classification into a named function for other callers), but it's not a gap -- it's a code-quality improvement. Keep in Phase 1 but reduce estimated effort.

#### MEDIUM: Assumption ASM-1 is Invalid

**Severity:** MEDIUM
**Evidence:** ASM-1 states "Google Forms API supports batchUpdate for question management (v1 API)." The Forms API v1 does support `batchUpdate` for questions, BUT automagik-tools doesn't actually use it (see CRITICAL gap above). This assumption is untested and the fallback (AppScript) adds significant complexity.

**Fix:** If Group 7 is kept, spike the Forms batchUpdate API first. If removed per the CRITICAL recommendation, delete ASM-1.

#### LOW: Missing Opportunities Discovered During Review

**Severity:** LOW (informational -- not blocking)

New features found in automagik-tools that aren't in the wish:
1. **Google Custom Search** (3 tools: `search_custom`, `get_search_engine_info`, `search_custom_siterestrict`) -- entirely missing from gog-cli. Would require new Google API service.
2. **Drive `update_drive_file`** has `starred`, `trashed`, `properties` convenience parameters -- gog may want these flags.
3. **Slides `get_page_thumbnail`** -- thumbnail URL generation at multiple sizes.
4. **Comment factory pattern** -- `create_comment_tools()` generates comment tools for Docs, Sheets, Slides via Drive API.

**Fix:** Consider adding these as a separate wish if they're valuable. Not blocking this wish.

### Summary

| # | Gap | Severity | Action |
|---|-----|----------|--------|
| 1 | Group 7 doesn't close a real parity gap | CRITICAL | Remove or reclassify |
| 2 | Group 2 uses wrong API params | HIGH | Fix parameters to match real Forms API |
| 3 | Group 4 already partially exists | HIGH | Rescope to centralize existing pattern |
| 4 | Group 5 already works in transport | MEDIUM | Downgrade priority, keep as refactor |
| 5 | ASM-1 untested, may be invalid | MEDIUM | Remove if Group 7 removed |
| 6 | Missing opportunities (Custom Search, etc.) | LOW | Separate wish |

### Next Steps

Fixes applied in loop 1. Re-reviewing.

1. ~~Fix CRITICAL: Remove Group 7~~ -- DONE. Removed and renumbered Groups 8-11 to 7-10.
2. ~~Fix HIGH: Update Group 2 parameters~~ -- DONE. Updated to use `setPublishSettings` with `--publish-as-template` and `--require-authentication`.
3. ~~Fix HIGH: Rescope Group 4~~ -- DONE. Rescoped to centralize existing ad-hoc enablement checks via `WrapAPIEnablementError`.
4. ~~Fix MEDIUM: Downgrade Group 5~~ -- DONE. Noted as refactor/code-quality improvement; transport.go already handles retry correctly.
5. ~~Fix MEDIUM: Remove ASM-1~~ -- DONE. Removed ASM-1 (Forms batchUpdate assumption) and RISK-3 (Forms API limitations). Renumbered remaining assumptions and risks.
6. ~~Update task count~~ -- DONE. Updated from 11 to 10 groups throughout.
7. Re-run `/review` after fixes.

---

## Files to Create/Modify

```
# Phase 1 (Groups 1-5)
internal/cmd/docs.go                    # Add DocsStructureCmd
internal/cmd/docs_structure_test.go     # NEW - structure tests
internal/cmd/forms.go                   # Add FormsPublishCmd
internal/cmd/forms_publish_test.go      # NEW - publish tests
internal/cmd/sheets.go                  # Add SheetsAddTabCmd
internal/cmd/sheets_addtab_test.go      # NEW - add-tab tests
internal/googleapi/errors.go            # Centralized API enablement errors + IsTransient
internal/googleapi/errors_test.go       # Updated error tests
internal/googleapi/transport.go         # Use IsTransient in retry

# Phase 2 (Groups 6-8)
internal/cmd/docs.go                    # Add DocsHeaderCmd, DocsFooterCmd
internal/cmd/docs_header_footer_test.go # NEW - header/footer tests
internal/cmd/sheets.go                  # Add SheetsBatchUpdateCmd
internal/cmd/sheets_batch_test.go       # NEW - batch update tests
internal/googleauth/scopes.go           # NEW - dynamic scope filtering
internal/googleauth/scopes_test.go      # NEW - scope tests
internal/cmd/enabled_commands.go        # Wire scope filtering

# Phase 3 (Groups 9-10)
internal/cmd/drive.go                   # Add DriveCheckPublicCmd
internal/cmd/drive_checkpublic_test.go  # NEW - check-public tests
internal/cmd/docs_generate.go           # NEW - JSON-to-Docs generation
internal/cmd/docs_generate_test.go      # NEW - generation tests
```
