# Design: google_workspace_mcp → gogcli Feature Integration

> Crystallized from brainstorm at WRS 100/100

## Overview

Port 4 features from [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp) into gogcli to enhance agent-deployment safety and power-user workflows. gogcli already has superior API coverage (14 services, 337+ commands); this work borrows architectural patterns, not API coverage.

## Features

### Feature 1: Three-Tier Command System

**Goal**: `--command-tier core|extended|complete` flag to control visible command surface.

**Design**:
- New embedded YAML file `internal/cmd/command_tiers.yaml` mapping every subcommand to a tier
- Tiers are cumulative: `core` ⊂ `extended` ⊂ `complete`
- `core`: everyday read + basic write ops (search, list, send, create)
- `extended`: management ops (settings, filters, batch, permissions, sharing)
- `complete`: all commands (default, backward-compatible)
- Integrates with existing `enforceEnabledCommands()` in `internal/cmd/enabled_commands.go`
- `--command-tier` and `--enable-commands` are composable (both filters apply)

**Files to modify**:
- `internal/cmd/enabled_commands.go` — extend with tier logic
- `internal/cmd/root.go` — add `--command-tier` flag
- NEW: `internal/cmd/command_tiers.yaml` — tier definitions (embedded via `//go:embed`)

**Reference**: `google_workspace_mcp/core/tool_tiers.yaml`

---

### Feature 2: True Read-Only Mode

**Goal**: `--read-only` flag that enforces read-only access at OAuth scope level AND hides write commands.

**Design**:
- Dual-layer enforcement:
  1. **Scope layer**: Map each service to its `.readonly` scope variant in `internal/googleauth/service.go`
  2. **Command layer**: Tag each command as `read` or `write`; hide `write` commands when `--read-only` active
- When `--read-only` is set:
  - OAuth flow requests only read scopes (e.g., `gmail.readonly` instead of `gmail.modify`)
  - Commands tagged as `write` (send, create, delete, update, modify, upload, share) are removed from the command tree
  - If a write command is invoked directly, error with: `"command unavailable in read-only mode"`
- Compatible with `--command-tier` (both filters stack)

**Files to modify**:
- `internal/googleauth/service.go` — add readonly scope map + `--readonly` scope selection
- `internal/cmd/root.go` — add `--read-only` global flag, wire into command filtering
- Various `*_cmd.go` files — tag commands with read/write annotation (Kong tag or method)

**Reference**: `google_workspace_mcp/auth/scopes.py` (`TOOL_READONLY_SCOPES_MAP`)

---

### Feature 3: Office Format Text Extraction

**Goal**: `gog drive cat document.docx` outputs plain text extracted from DOCX/XLSX/PPTX.

**Design**:
- New package `internal/officetext/` with three extractors:
  - `ExtractDOCX(r io.ReaderAt, size int64) (string, error)` — parse `word/document.xml`, extract `<w:t>` text nodes
  - `ExtractXLSX(r io.ReaderAt, size int64) (string, error)` — parse `xl/sharedStrings.xml` + `xl/worksheets/sheet*.xml`
  - `ExtractPPTX(r io.ReaderAt, size int64) (string, error)` — parse `ppt/slides/slide*.xml`, extract `<a:t>` text nodes
  - `Extract(r io.ReaderAt, size int64, mimeType string) (string, error)` — dispatcher by MIME type
- Uses Go stdlib only: `archive/zip`, `encoding/xml`, `strings`
- Integrated into `drive cat` / `drive download --text` command path
- Auto-detected by MIME type or file extension
- Falls back to raw download if extraction fails

**Files to create**:
- `internal/officetext/extract.go` — dispatcher
- `internal/officetext/docx.go` — DOCX extractor
- `internal/officetext/xlsx.go` — XLSX extractor
- `internal/officetext/pptx.go` — PPTX extractor
- `internal/officetext/extract_test.go` — unit tests with small fixture files

**Files to modify**:
- `internal/cmd/drive.go` or equivalent — integrate text extraction into `cat` command

**Reference**: `google_workspace_mcp/core/utils.py` `extract_office_xml_text()`

---

### Feature 4: Batch Contacts Operations

**Goal**: `gog contacts batch create|update|delete` for multi-contact operations.

**Design**:
- Three new subcommands under `gog contacts batch`:
  - `create` — accepts JSON array of contact objects (from file or stdin)
  - `update` — accepts JSON array of `{resourceName, fields}` objects
  - `delete` — accepts list of resource names (from args, file, or stdin)
- Uses People API `people.batchCreateContacts`, `people.batchUpdateContacts`, `people.batchDeleteContacts`
- Limits: Google allows max 200 contacts per batch request; chunk if needed
- Output: JSON array of results with success/failure per contact
- Supports `--dry-run` (preview without executing)

**Files to create**:
- `internal/cmd/contacts_batch.go` — batch subcommands
- `internal/googleapi/contacts_batch.go` — API client methods (if not in existing contacts.go)

**Files to modify**:
- `internal/cmd/contacts_cmd.go` — register batch subcommand group

**Reference**: `google_workspace_mcp/contacts/contacts_tools.py` batch operations

---

## Acceptance Criteria

1. `gog --command-tier core gmail` shows only core-tier Gmail subcommands; `--command-tier complete` shows all (default)
2. `gog --read-only gmail` hides `send`, `delete`, `batch delete` and requests `gmail.readonly` scope
3. `gog drive cat report.docx` outputs plain text from a DOCX file stored in Drive
4. `gog contacts batch create --file contacts.json` creates multiple contacts in one API call
5. All new features have unit tests; `make ci` passes
6. Existing commands and behavior are unaffected (backward compatible)

## Dependencies

- No new Go dependencies required (all features use stdlib)
- Batch contacts uses existing People API client in `internal/googleapi/contacts.go`
- YAML parsing uses existing `gopkg.in/yaml.v3` or similar (check go.mod)

## Implementation Order

1. **Office Text Extraction** — standalone package, no existing code changes needed
2. **Three-Tier Command System** — extends existing mechanism
3. **True Read-Only Mode** — builds on tier system for command tagging
4. **Batch Contacts** — independent feature, can be done in parallel with 2-3
