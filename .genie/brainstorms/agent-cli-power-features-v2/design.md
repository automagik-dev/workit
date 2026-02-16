# Design: Agent CLI Power Features (v2)

> Crystallized from brainstorm at WRS 100/100. Third wish in the trilogy.

## Overview

v1 ships discoverability + control (help topics, field discovery, --generate-input, pagination). v2 adds the two power features deferred from v1: built-in `--jq` filtering and `file://` input support. Together with the google-workspace-mcp feature wish (command tiers, read-only, office text, batch contacts), these three wishes make gogcli a complete agent-deployment platform.

## Features

### Feature 1: --jq Filter

**Goal**: `gog drive ls --json --jq '.[].name'` — built-in jq filtering on JSON output.

**UX**:
```bash
# Simple field extraction
$ gog drive ls --json --jq '.[].name'
"Report.docx"
"Budget.xlsx"

# Complex transforms
$ gog gmail search "in:starred" --json --jq '[.[] | {from, subject, date: .internalDate}]'
[{"from": "boss@co.com", "subject": "Q4", "date": "1739000000"}]

# Filtering
$ gog calendar events --json --jq '[.[] | select(.status == "confirmed")]'

# Count
$ gog drive ls --json --jq 'length'
42

# Incompatible with --plain
$ gog drive ls --plain --jq '.'
Error: --jq requires --json mode (exit 2)
```

**Design**:
- Add `--jq` global flag to `RootFlags` (string, optional)
- Requires `--json` mode; error (exit 2) if combined with `--plain`
- Output pipeline order: raw data → JSON serialize → `--results-only` → `--select` → **`--jq`** → stdout
- Uses `github.com/itchyny/gojq` (pure Go, no CGO)
- jq expression errors → exit 2 with stderr message including the expression and gojq's error
- When `--jq` is active, raw jq output is written (no re-wrapping in JSON envelope)

**Files to create**:
- `internal/outfmt/jq.go` — `ApplyJQ(jsonBytes []byte, expression string) ([]byte, error)` wrapper
- `internal/outfmt/jq_test.go` — unit tests (filter, transform, error cases)

**Files to modify**:
- `go.mod` — add `github.com/itchyny/gojq`
- `internal/cmd/root.go` — add `--jq` flag to `RootFlags`
- `internal/outfmt/outfmt.go` — apply jq filter as final step in JSON output pipeline

**Effort**: 2-3 days

---

### Feature 2: file:// Input

**Goal**: `--body file://report.txt` reads file content for text-content flags.

**UX**:
```bash
# Read email body from file
$ gog gmail send --to user@example.com --subject "Report" --body file://report.txt

# Read doc content from file
$ gog docs write DOC_ID --content file://chapter.md

# Binary content (base64)
$ gog gmail send --to user@example.com --body fileb://attachment.bin

# Literal string still works (no prefix = literal)
$ gog gmail send --to user@example.com --body "Hello, this is inline text"

# Path traversal blocked
$ gog gmail send --body file://../../etc/passwd
Error: file:// path must not escape current directory (exit 2)
```

**Design**:
- Utility function `ResolveFileInput(value string) (string, error)` in `internal/input/`
- Logic:
  1. If `value` starts with `file://` → strip prefix, read file as UTF-8 string
  2. If `value` starts with `fileb://` → strip prefix, read file, base64-encode
  3. Otherwise → return value as-is (literal string)
- **Security**:
  - Resolve path relative to CWD via `filepath.Abs` + `filepath.Clean`
  - Reject if resolved path escapes CWD subtree (block `../` traversal)
  - Reject known sensitive paths (`.env`, `.ssh/`, `.aws/`, `*credentials*`)
  - Max file size: 10MB (configurable, prevents accidental huge reads)
- **Opt-in per flag**: only flags that logically accept content bodies:
  - `--body` (gmail send)
  - `--content` (docs write)
  - `--description` (calendar create, drive create)
  - `--notes` (slides, tasks)
  - `--message` (chat send)
  - `--text` (docs insert)
- Each command calls `ResolveFileInput()` on the flag value before use

**Files to create**:
- `internal/input/file_input.go` — `ResolveFileInput()` + security checks
- `internal/input/file_input_test.go` — unit tests (read, literal, traversal, size limit)

**Files to modify**:
- `internal/cmd/gmail_send.go` — resolve `--body` flag
- `internal/cmd/docs_cmd.go` — resolve `--content` flag
- `internal/cmd/calendar_cmd.go` — resolve `--description` flag
- Other commands with text-content flags (small, mechanical changes)

**Effort**: 1-2 days

---

## Implementation Order

1. **file:// Input** (1-2 days) — standalone utility, no dependency on other features
2. **--jq Filter** (2-3 days) — depends on output pipeline being stable (v1 should be merged first)

**Total**: 3-5 days | **New dependency**: `github.com/itchyny/gojq`

## Acceptance Criteria

1. `gog drive ls --json --jq '.[].name'` outputs only names
2. `gog gmail search "in:inbox" --json --jq '[.[] | {from, subject}]'` outputs restructured JSON
3. `gog drive ls --json --jq 'invalid['` exits 2 with helpful stderr error
4. `gog drive ls --plain --jq '.'` exits 2 (incompatible flags)
5. `gog gmail send --body file://test.txt` reads file as body
6. `gog gmail send --body "literal"` unchanged (backward compatible)
7. `gog gmail send --body file://../../etc/passwd` rejected (traversal)
8. `gog gmail send --body file://huge-10gb.bin` rejected (size limit)
9. Unit tests for both features
10. `make ci` passes
11. Backward compatible
12. AGENTS.md updated

## Wish Trilogy Execution Order

| Order | Wish | Features | Est. Effort | Dependencies |
|-------|------|----------|-------------|--------------|
| **1** | google-workspace-mcp-integration | Command tiers, read-only, office text, batch contacts | 8-13 days | None |
| **2** | agent-cli-ux-unified (v1) | Help topics, field discovery, --generate-input, pagination | 8-13 days | None (parallel OK) |
| **3** | agent-cli-power-features-v2 | --jq filter, file:// input | 3-5 days | v1 output pipeline for --jq |

Wishes 1 and 2 can run in parallel. Wish 3 should run after wish 2 completes (--jq depends on the output pipeline ordering from v1).
