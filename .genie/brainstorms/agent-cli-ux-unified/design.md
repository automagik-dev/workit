# Design: Agent-First CLI UX — Unified v1

> Crystallized from unified brainstorm merging two parallel analyses at WRS 100/100

## Overview

Make `gog` enable **zero-shot agent task success** — no external skills, no prior schema memorization. The agent discovers everything from the CLI itself. v1 ships 4 features: discoverability core (help topics + field discovery) AND control layer (pagination + input templates), with zero new dependencies.

## Design Principles (from parallel analysis)

1. **Discoverability first, power later** — agents can't use features they can't find
2. **Drift-proof content** — schema-generated facts + curated concept prose
3. **stdout discipline** — parseable output on stdout, human hints on stderr
4. **Zero-shot acceptance test** — validate the OUTCOME, not just the code

## Features

### Feature 1: Help Topics (hybrid schema-generated + curated)

**Goal**: `gog help <topic>` provides concept-level docs that stay accurate.

**UX**:
```bash
$ gog help topics
Available topics:
  auth         Authentication & account management
  output       Output formatting (--json, --plain, --select)
  pagination   Controlling result pagination
  agent        Agent integration guide
  scopes       OAuth scopes and permissions
  services     Available Google services

$ gog help agent
# AGENT INTEGRATION

## Quick Start
  gog schema                    # Full CLI tree as JSON
  gog exit-codes --json         # Stable exit codes
  gog drive ls --json --select  # Discover selectable fields

## Available Services
  [auto-generated from schema: gmail, drive, calendar, ...]

## Exit Codes
  [auto-generated from gog agent exit-codes]

## Environment Variables
  GOG_AUTO_JSON=1    Auto-enable JSON when stdout is piped
  GOG_HELP=full      Expand all subcommands in --help

## Output Modes
  --json / -j        JSON to stdout (best for agents)
  --plain / -p       TSV to stdout (stable, no colors)
  --results-only     Drop envelope (nextPageToken, etc.)
  --select FIELDS    Project specific fields (dot paths)

## Sandboxing
  --enable-commands  Restrict to specific top-level commands
  --no-input         Never prompt; fail instead
  --dry-run          Preview changes without executing
```

**Design**:
- New `HelpCmd` struct in `internal/cmd/help_topics.go`
- Each topic is a Go function that mixes:
  - **Curated prose**: static strings explaining concepts (when to use --json vs --plain, how auth flows work)
  - **Generated facts**: pulled live from the same Kong model that powers `gog schema` (service list, flag names, exit codes)
- `gog help topics` — lists all topics with descriptions
- `gog help <unknown>` — suggests closest match (fuzzy)
- Output: rendered with colors to tty, plain text when piped, JSON-wrapped with `--json`
- Single source of truth: facts come from code, prose lives next to the code it describes

**Files to create**:
- `internal/cmd/help_topics.go` — topic registry, hybrid renderer
- `internal/cmd/help_topics_test.go` — verify topics render, facts match schema

**Files to modify**:
- `internal/cmd/root.go` — register `help` command at root level

**Effort**: 2-3 days

---

### Feature 2: Field Discovery for --select

**Goal**: `gog drive ls --json --select` (empty value) lists available JSON fields.

**UX**:
```bash
$ gog drive ls --json --select
Available fields for 'drive ls':
  id, name, mimeType, size, modifiedTime, createdTime,
  parents, webViewLink, iconLink, owners, shared,
  trashed, starred, capabilities

Hint: gog drive ls --json --select "name,id,size"
```

**Design**:
- Detect empty `--select` value in output pipeline
- Use the **same code path** that processes `--select` to enumerate available fields (single source of truth — mitigates divergence risk from parallel analysis)
- Introspect the output struct's `json:"..."` tags via reflection
- Output to **stderr** (preserves stdout discipline)
- Exit 0 — no API call made
- Works for every command that supports `--json` output

**Files to modify**:
- `internal/outfmt/outfmt.go` — detect empty `--select`, enumerate fields from struct tags
- `internal/cmd/root.go` — handle empty `--select` flag value edge case with Kong

**Effort**: 1-2 days

---

### Feature 3: Pagination Control

**Goal**: Expose `--max-results N` and `--page-token TOKEN` for agent-controlled pagination.

**UX**:
```bash
$ gog drive ls --max-results 5 --json
{"files": [...5 items...], "nextPageToken": "Cg1..."}

$ gog drive ls --max-results 5 --page-token "Cg1..." --json
{"files": [...next 5 items...], "nextPageToken": "Xk9..."}
```

**Design**:
- Add global flags to `RootFlags`:
  - `--max-results` (int) → maps to Google API `MaxResults`/`PageSize`
  - `--page-token` (string) → maps to Google API `PageToken`
- Existing `--all` overrides `--max-results` (collect all pages)
- Existing `--results-only` continues to strip envelope
- **Flag unification audit**: existing per-command `--max`, `--limit`, `--page` become aliases for the global flags where semantically equivalent
- Commands that don't support pagination silently ignore these flags

**Files to modify**:
- `internal/cmd/root.go` — add global flags
- `internal/cmd/paging.go` — wire into `collectAllPages` and per-command fetch
- Various `*_cmd.go` — map flags to Google API parameters per service

**Effort**: 3-5 days

---

### Feature 4: --generate-input

**Goal**: `gog gmail send --generate-input` prints JSON input template.

**UX**:
```bash
$ gog gmail send --generate-input
{
  "to": "(required) Recipient email addresses",
  "cc": "CC recipients",
  "bcc": "BCC recipients",
  "subject": "Email subject line",
  "body": "Email body text",
  "html": "HTML body alternative",
  "attachments": ["File paths to attach"],
  "track": "Enable tracking (boolean, default: false)",
  "reply_to": "In-Reply-To message ID"
}
```

**Design**:
- New global flag `--generate-input` (bool)
- Introspect Kong command model for the resolved command
- Extract: flag name, type, help text, required status, default, enum values
- Format as JSON: keys = flag names, values = `"(required) help text"` or `"help text"`
- Exit 0 without executing the command
- Reuses same Kong model walking as `gog schema` — consistent, single source
- **INPUT-side discoverability** complementing field discovery (OUTPUT-side)

**Files to create**:
- `internal/cmd/generate_input.go` — template generator
- `internal/cmd/generate_input_test.go` — verify output matches Kong model

**Files to modify**:
- `internal/cmd/root.go` — add `--generate-input` flag, intercept before execution

**Effort**: 2-3 days

---

## Implementation Order

1. **Help Topics** (2-3 days) — foundation; documents ALL other features
2. **Field Discovery** (1-2 days) — OUTPUT-side discoverability
3. **--generate-input** (2-3 days) — INPUT-side discoverability
4. **Pagination Control** (3-5 days) — most invasive; touches many commands

**Total**: 8-13 days | **New dependencies**: zero

## Acceptance Criteria

### Primary: Zero-Shot Agent Task Success
An agent completes 3 tasks in a clean environment using only CLI discovery:

1. **Auth**: discovers setup via `gog help auth`, runs `gog login`, verifies with `gog status`
2. **Read + project**: discovers fields via `--select` (empty), lists Drive with projected fields
3. **Write**: discovers input via `--generate-input`, constructs and sends an email

**Pass**: all 3 succeed zero-shot; stdout remains parseable JSON/TSV.

### Secondary: Technical Criteria
4. `gog help topics` lists topics; `gog help agent` renders hybrid content with live schema facts
5. `gog drive ls --json --select` (empty) prints fields to stderr, exits 0
6. `gog drive ls --max-results 5 --json` returns ≤5 results + nextPageToken
7. `gog gmail send --generate-input` prints JSON template with required markers
8. Unit tests for all 4 features
9. `make ci` passes
10. Backward compatible — no existing behavior changes
11. AGENTS.md updated

## Deferred to v2
- `--jq` filter (gojq dependency) — `--select` covers 80%
- `file://` input — agents can pipe stdin
- Batch workflow execution — too complex for now

## Cross-References
- **Supersedes**: `.genie/brainstorms/agent-friendly-cli-help/design.md` (was 6 features)
- **Informed by**: `agents/gog-cli-brainstorm/.genie/brainstorms/agent-first-cli-ux-edges/design.md` (parallel analysis — discoverability-first principle, drift risk, zero-shot test)
- **Complements**: `.genie/brainstorms/google-workspace-mcp-integration/design.md` (command tiers + read-only mode)
