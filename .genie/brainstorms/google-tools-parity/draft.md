# Brainstorm: Google Tools Feature Parity & Enhancement

## Problem Statement
Compare automagik-tools (Python/MCP) google-* tools with gog-cli (Go CLI) to identify gaps, optimizations, and new features we can port or be inspired by — making gog-cli the most complete Google Workspace CLI for AI agents.

## Scope
- **IN**: All 12 google-* tools from automagik-tools vs all gog-cli commands
- **IN**: Feature gaps, architectural patterns, new capabilities
- **OUT**: Non-Google tools (evolution_api, gemini_assistant, spark, etc.)
- **OUT**: Rewriting gog-cli in Python / changing language

---

## Service-by-Service Comparison

### 1. 📅 Google Calendar

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| List calendars | ✅ `list_calendars` | ✅ `calendar calendars` | ✅ Parity |
| List/search events | ✅ `get_events` (time range, query, detailed) | ✅ `calendar events` (time-range, all-cals, property filters) | ✅ Parity (gog richer) |
| Create event | ✅ `create_event` (reminders, attendees, attachments) | ✅ `calendar create` | ✅ Parity |
| Modify event | ✅ `modify_event` | ✅ `calendar update` | ✅ Parity |
| Delete event | ✅ `delete_event` | ✅ `calendar delete` | ✅ Parity |
| RSVP/Respond | ✅ `respond_to_event` | ✅ `calendar respond` | ✅ Parity |
| Free/Busy | ❌ | ✅ `calendar freebusy` | **gog leads** |
| Propose time | ❌ | ✅ `calendar propose-time` | **gog leads** |
| Conflicts detection | ❌ | ✅ `calendar conflicts` | **gog leads** |
| Focus Time blocks | ❌ | ✅ `calendar focus-time` | **gog leads** |
| Out of Office | ❌ | ✅ `calendar out-of-office` | **gog leads** |
| Working Location | ❌ | ✅ `calendar working-location` | **gog leads** |
| Team calendar | ❌ | ✅ `calendar team` | **gog leads** |
| Calendar colors | ❌ | ✅ `calendar colors` | **gog leads** |
| ACL/Permissions | ❌ | ✅ `calendar acl` | **gog leads** |

**Verdict: gog-cli leads significantly.** No features to port from automagik-tools.

---

### 2. 💬 Google Chat

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| List spaces | ✅ `list_spaces` (filtered by type) | ✅ `chat spaces` | ✅ Parity |
| Get messages | ✅ `get_messages` (ordering) | ✅ `chat messages` | ✅ Parity |
| Send message | ✅ `send_message` (threaded) | ✅ `chat dm` | ✅ Parity |
| Search messages | ✅ `search_messages` (cross-space) | ❓ Need to verify | **Potential gap** |
| Thread operations | ❌ | ✅ `chat threads` | **gog leads** |

**Verdict: Mostly parity.** Check if gog has cross-space message search.

---

### 3. 📄 Google Docs

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| Search docs | ✅ `search_docs` (via Drive API) | ✅ (via `drive search`) | ✅ Parity |
| Get doc content | ✅ `get_doc_content` (full structure) | ✅ `docs cat` (plain text) | ⚠️ automagik richer |
| Create doc | ✅ `create_doc` | ✅ `docs create` (with markdown import!) | **gog leads** |
| Modify/Insert text | ✅ `modify_doc_text` (positional) | ✅ `docs write/insert/delete` | ✅ Parity |
| Find & replace | ✅ `find_and_replace_doc` | ✅ `docs find-replace` | ✅ Parity |
| Insert image | ✅ `insert_doc_image` | ✅ via markdown import (image refs) | ✅ Parity (different mechanism) |
| Headers/footers | ✅ `update_doc_headers_footers` | ❌ No dedicated command | **GAP: port from automagik** |
| Batch update | ✅ `batch_update_doc` | ✅ Uses BatchUpdate internally for all edits | ✅ Parity (internal use) |
| Document structure inspect | ✅ `inspect_doc_structure` | ❌ `docs cat` returns plain text only | **GAP: add structure view** |
| Table creation w/ data | ✅ `create_table_with_data` | ✅ `docs_table_inserter.go` (native tables) | ✅ Parity |
| Debug table structure | ✅ `debug_table_structure` | ❌ No equivalent | **Low-priority gap** |
| Export to PDF | ✅ `export_doc_to_pdf` | ✅ `docs export` (pdf, docx, txt) | ✅ Parity |
| Comments | ❌ | ✅ `docs comments` | **gog leads** |
| List tabs | ❌ | ✅ `docs list-tabs` | **gog leads** |
| Copy doc | ❌ | ✅ `docs copy` | **gog leads** |

**Verdict: gog-cli leads.** After verification, gog already has images (via markdown), tables, and batch update internally. Real gaps: headers/footers management and document structure inspection.

---

### 4. 📁 Google Drive

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| Search files | ✅ `search_drive_files` | ✅ `drive search` (full text + raw query) | ✅ Parity |
| Get file content | ✅ `get_drive_file_content` | ✅ `drive download` (with format conversion) | **gog leads** |
| List folder | ✅ `list_drive_items` | ✅ `drive ls` (with shared drives) | ✅ Parity |
| Create file | ✅ `create_drive_file` | ✅ `drive upload` (auto-convert) | ✅ Parity |
| Get permissions | ✅ `get_drive_file_permissions` | ✅ `drive permissions` | ✅ Parity |
| Check public access | ✅ `check_drive_file_public_access` | ❓ Via permissions list | **Nice-to-have** |
| Update file | ✅ `update_drive_file` | ✅ `drive upload` (replace mode) | ✅ Parity |
| Copy file | ❌ | ✅ `drive copy` | **gog leads** |
| Mkdir | ❌ | ✅ `drive mkdir` | **gog leads** |
| Delete | ❌ | ✅ `drive delete` | **gog leads** |
| Move | ❌ | ✅ `drive move` | **gog leads** |
| Rename | ❌ | ✅ `drive rename` | **gog leads** |
| Share/Unshare | ❌ | ✅ `drive share/unshare` | **gog leads** |
| Comments | ❌ | ✅ `drive comments` | **gog leads** |
| URL generation | ❌ | ✅ `drive url` | **gog leads** |
| Shared drives list | ❌ | ✅ `drive drives` | **gog leads** |
| **Bidirectional sync** | ❌ | ✅ `sync init/list/status` | **gog leads** |

**Verdict: gog-cli leads massively.** automagik has a convenience `check_public_access` wrapper.

---

### 5. 📝 Google Forms

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| Create form | ✅ `create_form` | ✅ `forms create` (via AppScript) | ✅ Parity |
| Get form | ✅ `get_form` (questions, metadata) | ✅ `forms get` | ✅ Parity |
| Publish settings | ✅ `set_publish_settings` | ❌ Not implemented | **GAP: add publish settings** |
| Get responses | ✅ `get_form_response` | ✅ `forms responses get` | ✅ Parity |
| List responses | ✅ `list_form_responses` | ✅ `forms responses list` | ✅ Parity |
| Add questions | ✅ (likely via batch update) | ❌ Not implemented | **GAP: add question mgmt** |

**Verdict: Mostly parity.** gog has response retrieval already. Gaps: publish settings and question management.

---

### 6. 📧 Google Gmail

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| Search/list messages | ✅ `search_emails` | ✅ `gmail search` (thread-aware, query syntax) | ✅ Parity |
| Read email | ✅ `get_email` | ✅ `gmail get` (full/metadata/raw) | **gog richer** |
| Send email | ✅ `send_email` | ✅ `gmail send` (attachments, HTML, CC/BCC) | ✅ Parity |
| Reply | ✅ `reply_to_email` | ✅ (via send with in-reply-to) | ✅ Parity |
| Labels | ✅ `list_labels` | ✅ `gmail labels` (CRUD) | **gog leads** |
| Manage labels on messages | ✅ `modify_email_labels` | ✅ `gmail batch` | ✅ Parity |
| List attachments | ✅ `list_email_attachments` | ✅ `gmail attachment` | ✅ Parity |
| Download attachment | ✅ `download_attachment` | ✅ `gmail attachment` | ✅ Parity |
| Drafts | ❌ | ✅ `gmail drafts` (CRUD + send) | **gog leads** |
| Filters | ❌ | ✅ `gmail filters` | **gog leads** |
| Watch/Push | ❌ | ✅ `gmail watch` (pub/sub) | **gog leads** |
| Track opens | ❌ | ✅ `gmail track` | **gog leads** |
| Vacation | ❌ | ✅ `gmail vacation` | **gog leads** |
| Auto-forward | ❌ | ✅ `gmail autoforward` | **gog leads** |
| Delegates | ❌ | ✅ `gmail delegates` | **gog leads** |
| Send-as | ❌ | ✅ `gmail sendas` | **gog leads** |
| Forwarding | ❌ | ✅ `gmail forwarding` | **gog leads** |
| History | ❌ | ✅ `gmail history` | **gog leads** |
| Thread operations | ❌ | ✅ `gmail thread` | **gog leads** |

**Verdict: gog-cli dominates.** automagik has basic CRUD; gog has full admin/settings/tracking.

---

### 7. 📊 Google Sheets

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| Get values | ✅ `get_spreadsheet_values` | ✅ `sheets get` | ✅ Parity |
| Update values | ✅ `update_spreadsheet_values` | ✅ `sheets update` | ✅ Parity |
| Append values | ✅ `append_spreadsheet_values` | ✅ `sheets append` | ✅ Parity |
| Create spreadsheet | ✅ `create_spreadsheet` | ✅ `sheets create` | ✅ Parity |
| Get metadata | ✅ `get_spreadsheet_metadata` | ✅ `sheets metadata` | ✅ Parity |
| Clear values | ❌ | ✅ `sheets clear` | **gog leads** |
| Cell formatting | ❌ | ✅ `sheets format` | **gog leads** |
| Cell notes | ❌ | ✅ `sheets notes` | **gog leads** |
| Copy sheet | ❌ | ✅ `sheets copy` | **gog leads** |
| Export | ❌ | ✅ `sheets export` (pdf, xlsx, csv) | **gog leads** |
| Batch operations | ✅ `batch_update_spreadsheet` | ❓ Need to verify | **automagik feature** |
| Add sheet | ✅ `add_sheet` | ❓ Need to verify | **automagik feature** |

**Verdict: gog-cli leads.** automagik has batch_update and add_sheet that may be missing from gog.

---

### 8. 🎯 Google Slides

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| Create presentation | ✅ `create_presentation` | ✅ `slides create` (with template!) | ✅ Parity |
| Get presentation | ✅ `get_presentation` | ✅ `slides info` | ✅ Parity |
| Export | ❌ | ✅ `slides export` (pdf, pptx) | **gog leads** |
| Copy | ❌ | ✅ `slides copy` | **gog leads** |
| Add slide | ❌ | ✅ `slides add-slide` (with image + notes) | **gog leads** |
| List slides | ❌ | ✅ `slides list-slides` | **gog leads** |
| Delete slide | ❌ | ✅ `slides delete-slide` | **gog leads** |
| Read slide content | ❌ | ✅ `slides read-slide` | **gog leads** |
| Update notes | ❌ | ✅ `slides update-notes` | **gog leads** |
| Replace slide image | ❌ | ✅ `slides replace-slide` | **gog leads** |
| Markdown to slides | ❌ | ✅ `slides create-from-markdown` | **gog leads** |

**Verdict: gog-cli dominates.** automagik only has basic create/get.

---

### 9. ✅ Google Tasks

| Capability | automagik-tools | gog-cli | Gap? |
|-----------|----------------|---------|------|
| List task lists | ✅ `list_task_lists` | ✅ `tasks lists` | ✅ Parity |
| Create task list | ✅ `create_task_list` | ❓ Need to verify | **Potential gap** |
| List tasks | ✅ `list_tasks` | ✅ `tasks items` | ✅ Parity |
| Create task | ✅ `create_task` | ✅ `tasks items create` | ✅ Parity |
| Update task | ✅ `update_task` | ✅ `tasks items update` | ✅ Parity |
| Delete task | ✅ `delete_task` | ✅ `tasks items delete` | ✅ Parity |
| Complete task | ✅ `complete_task` | ✅ `tasks items complete` | ✅ Parity |
| Recurring tasks | ❌ | ✅ `tasks repeat` | **gog leads** |
| Due date mgmt | ❌ | ✅ `tasks due` | **gog leads** |

**Verdict: gog-cli leads.** Has recurring + due date management.

---

### 10. 🏢 Google Workspace (meta-tool)
automagik-tools has a `google_workspace` meta-package that dynamically registers tools based on config. This is an MCP-specific pattern (not applicable to CLI).

### 11. 🔧 Google Workspace Core
automagik-tools has shared utilities:
- **Multi-user OAuth** with credential file storage
- **Service decorator** pattern for auth injection
- **Scope management** per service
- **Error handling** with retry and API enablement messages
- **Rate limiting** (implicit in decorators)

gog-cli equivalents:
- `internal/googleapi/` — transport, retry, circuit breaker
- `internal/googleauth/` — OAuth flows (browser + headless)
- `internal/secrets/` — keyring-backed credential storage
- `internal/config/` — credential management

### 12. 📋 json_to_google_docs
automagik-tools has a specialized tool for generating Google Docs from JSON templates. This is an **interesting unique capability** — programmatic document generation from structured data.

gog-cli has `docs create --markdown` (markdown import) which is adjacent but different.

---

## Services ONLY in gog-cli (NOT in automagik-tools)

| Service | gog-cli command | Description |
|---------|----------------|-------------|
| **Google Classroom** | `classroom` | Full Classroom management (courses, students, coursework, submissions, etc.) |
| **Google Contacts / People** | `contacts`, `people` | Contact CRUD, directory search, profiles |
| **Google Groups** | `groups` | Google Groups membership management |
| **Google Keep** | `keep` | Note operations (via service account) |
| **Google Apps Script** | `appscript` | Execute Apps Script projects |
| **Cloud Identity** | (in googleapi) | Cloud identity operations |
| **Drive Sync** | `sync` | Bidirectional Drive folder sync engine |

---

## Key Architectural Differences

| Aspect | automagik-tools | gog-cli |
|--------|----------------|---------|
| Language | Python (async) | Go (concurrent) |
| Interface | MCP server (tool calls) | CLI (stdout/stderr) |
| Auth storage | File-based credentials dir | OS keyring (keychain) |
| Multi-user | Per-request email param | Per-command `--account` flag |
| Output | String responses | JSON (`--json`) / plain text / tab-separated |
| Error handling | Decorator-based with retry | Circuit breaker + retry transport |
| Concurrency | `asyncio.to_thread` | Native goroutines |

---

## Verified Feature Enhancement Opportunities

### REAL GAPS — Features to port from automagik-tools → gog-cli:

| # | Feature | Service | Effort | Agent Value |
|---|---------|---------|--------|-------------|
| 1 | **Docs: Headers/Footers management** | Docs | Medium | High — agents generating reports need H/F |
| 2 | **Docs: Structure inspection** (`docs structure`) | Docs | Small | High — agents need to understand doc layout before editing |
| 3 | **Forms: Publish settings** | Forms | Small | Medium — configure form visibility/settings |
| 4 | **Forms: Question management** (add/modify questions) | Forms | Medium | High — agents creating surveys need this |
| 5 | **Sheets: Add sheet tab** | Sheets | Small | Medium — add new worksheets to existing spreadsheet |
| 6 | **Sheets: Batch update** (raw batchUpdate exposure) | Sheets | Small | Medium — power users need raw API access |
| 7 | **Drive: Public access check** (convenience) | Drive | Tiny | Low — convenience wrapper over permissions |
| 8 | **JSON-to-Docs: Template doc generation** | Docs | Large | High — generate docs from structured data |

### NOT ACTUALLY GAPS (already in gog-cli after verification):
- ~~Docs: Image insertion~~ → ✅ Already supported via markdown import
- ~~Docs: Table creation~~ → ✅ Already has `docs_table_inserter.go`
- ~~Docs: Batch update~~ → ✅ Already uses BatchUpdate internally
- ~~Forms: Response retrieval~~ → ✅ Already has `forms responses list/get`

### Optimization ideas inspired by automagik-tools:

| # | Improvement | Effort | Value |
|---|------------|--------|-------|
| 9 | **API enablement hints** — When 403 error, suggest enabling API in GCP console | Small | High for onboarding |
| 10 | **Dynamic scope management** — Only request scopes for services being used | Medium | Medium — smaller consent screen |
| 11 | **Transient error categories** — Better retry-able vs fatal error classification | Small | Medium — better UX |

---

## Decision: All 11 Items — Phased Milestones

**Approach:** Ship all features in 3 phases, ordered by effort and dependency.

### Phase 1 — Quick Wins (Small effort, ship in days)
- **#2** `docs structure` — Document structure inspection command
- **#3** `forms publish` — Form publish settings command
- **#5** `sheets add-tab` — Add worksheet to existing spreadsheet
- **#9** API enablement hints in error messages
- **#11** Better transient vs fatal error classification

### Phase 2 — Agent Workflow (Medium effort)
- **#1** `docs headers-footers` — Manage document headers/footers
- **#4** `forms questions` — Add/modify/delete form questions
- **#6** `sheets batch-update` — Raw batchUpdate exposure
- **#10** Dynamic scope management (per-service OAuth)

### Phase 3 — Stretch (Large effort)
- **#7** `drive check-public` — Convenience public access check
- **#8** JSON-to-Docs template document generation

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Upstream compatibility** — New commands may not align with upstream gogcli patterns | Medium | High | Follow existing command patterns (Kong structs, same flags style); keep PRable |
| **Scope creep** — 11 features could balloon | Medium | Medium | Hard phase boundaries; ship each phase independently |
| **Forms API limitations** — Forms API is newer, less documented | Low | Medium | Spike/prototype first; fall back to AppScript if needed |
| **OAuth scope expansion** — New APIs need new scopes | Low | Low | Document required scopes per feature; test with existing auth flow |
| **Test coverage** — Each feature needs tests | Medium | Medium | TDD per item; `make ci` gate before merge |

---

## Acceptance Criteria

### Phase 1 (all must pass):
- [ ] `gog docs structure <docId>` returns document element tree (headings, paragraphs, tables, images) in JSON and plain formats
- [ ] `gog forms publish <formId> --accepting-responses=true/false` works
- [ ] `gog sheets add-tab <spreadsheetId> <tabName>` creates a new worksheet
- [ ] 403 errors from disabled APIs include a hint: "Enable the API at https://console.cloud.google.com/apis/..."
- [ ] Error handler classifies 429, 500, 503 as transient (retryable) vs 400, 404, 403 as fatal
- [ ] All new commands have `--json` output and unit tests
- [ ] `make ci` passes

### Phase 2 (all must pass):
- [ ] `gog docs header/footer <docId> --set/--get` manages doc headers/footers
- [ ] `gog forms questions <formId> add/list/delete` manages form questions
- [ ] `gog sheets batch-update <spreadsheetId>` accepts JSON payload of raw requests
- [ ] OAuth flow only requests scopes for services being used (when `--enable-commands` is set)
- [ ] All new commands have `--json` output and unit tests
- [ ] `make ci` passes

### Phase 3 (all must pass):
- [ ] `gog drive check-public <fileId>` returns boolean public access status
- [ ] `gog docs generate --from-json <template.json>` creates a Google Doc from structured template
- [ ] Template format supports: headings, paragraphs, tables, images, lists
- [ ] `make ci` passes

---

## WRS Assessment

| Dimension | Score | Notes |
|-----------|-------|-------|
| Problem | ✅ 20 | Enhance gog-cli with verified feature gaps from automagik-tools |
| Scope | ✅ 20 | 8 real features + 3 optimizations, phased into 3 milestones |
| Decisions | ✅ 20 | All 11 items, phased approach, ordered by effort |
| Risks | ✅ 20 | 5 risks identified with mitigations |
| Criteria | ✅ 20 | Testable acceptance criteria per phase |

```
WRS: ██████████ 100/100
 Problem ✅ | Scope ✅ | Decisions ✅ | Risks ✅ | Criteria ✅
```
