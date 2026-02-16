# Brainstorm: Agent-First CLI UX — Unified (v1)

## Status
WRS: ██████████ 100/100
 Problem ✅ | Scope ✅ | Decisions ✅ | Risks ✅ | Criteria ✅

## Problem Statement
`gog` should enable zero-shot task success for agents — no external skills, no prior schema memorization. The agent discovers everything it needs from the CLI itself.

## Scope

### IN — v1: Discoverability + Control (4 features)

#### 1. Help Topics — hybrid schema-generated + curated
- `gog help <topic>` — concept docs accessible from CLI
- **Hybrid content model**: factual parts (commands, flags, examples) generated live from `gog schema` at render time; concept prose (how auth works, when to use --json vs --plain) is curated static content
- Topics: auth, output, pagination, agent, scopes, services
- `gog help topics` lists all available topics
- stdout discipline: rendered to stdout when human (tty), JSON when --json

#### 2. Field Discovery for --select
- `gog drive ls --json --select` (empty) → list available JSON fields
- Introspect output struct json tags via reflection
- Print to stderr (preserves stdout parseable discipline)
- Exit 0, no API call made
- Works for every command that supports --json

#### 3. Pagination Control
- `--max-results N` — limit results (maps to Google API maxResults/PageSize)
- `--page-token TOKEN` — continue from specific page
- Works alongside existing `--all` (fetch everything) and `--results-only`
- Audit existing --max/--limit/--page flags across commands for unification

#### 4. --generate-input
- `gog gmail send --generate-input` → JSON template with all flags as fields
- Marks required fields, shows types, defaults, enums
- Derived from Kong command model (same source as `gog schema`)
- INPUT-side complement to field discovery (OUTPUT-side)
- Exit 0 without executing command

### OUT — Defer to v2
- `--jq` filter (gojq dependency) — --select covers 80% of agent needs
- `file://` input — agents can pipe stdin, low urgency
- Batch workflow JSON execution (Stripe pattern) — too complex

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Help content model | **Hybrid**: schema-generated facts + curated concepts | Solves drift risk — facts auto-update, only prose needs maintenance |
| v1 scope | 4 features (not 2, not 6) | Discoverability + control together; agents need pagination once they can discover |
| Field discovery output | stderr | Preserves stdout parseable discipline |
| Pagination flag names | `--max-results`, `--page-token` | Google API naming convention; --max as backward-compatible alias |
| --generate-input format | JSON with type annotations | Machine-readable, consistent with gog schema |
| New dependency for v1 | None | All 4 features use Go stdlib. gojq deferred to v2 (--jq) |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Help topic concept prose drifts | Medium | Keep curated sections minimal; lean on schema-generated facts; CI check that topic references valid commands |
| Field discovery output diverges from --select behavior | High | Generate field list from SAME code path that processes --select; single source of truth |
| Pagination flags conflict with existing per-command --max/--page | Medium | Audit all commands; make --max alias for --max-results; keep --page if semantically different |
| --generate-input for complex nested inputs | Low | Flatten to flag-level; match schema approach; document limitations |
| Discoverability features accidentally erode stdout discipline | High | Field discovery → stderr only; help topics → stdout when tty, JSON-wrapped when --json; --generate-input → stdout (it IS the result) |

## Acceptance Criteria

### Primary: Zero-Shot Agent Task Success
An agent completes 3 representative tasks in a clean environment without external skills, using only gog CLI discovery features:

1. **Auth discovery**: agent discovers auth setup path via `gog help auth` and verifies account status
2. **Read + project**: agent lists Drive content via field discovery (`--select` empty), then fetches with projected fields
3. **Write**: agent sends an email using `--generate-input` to discover required flags, constructs and executes the command

**Pass condition**: All 3 tasks succeed zero-shot while stdout remains parseable JSON/TSV.

### Secondary: Technical Criteria
4. `gog help topics` lists all topics; `gog help agent` renders hybrid content
5. `gog drive ls --json --select` (empty) prints fields to stderr, exits 0
6. `gog drive ls --max-results 5 --json` returns ≤5 results + nextPageToken
7. `gog gmail send --generate-input` prints JSON template with required markers
8. All features have unit tests
9. `make ci` passes
10. Existing behavior unchanged (backward compatible)
11. AGENTS.md updated with new discovery features

## Cross-References
- Supersedes: `.genie/brainstorms/agent-friendly-cli-help/design.md` (v1 was 6 features, now 4)
- Complements: `.genie/brainstorms/google-workspace-mcp-integration/design.md` (command tiers + read-only mode)
- Informed by: `agents/gog-cli-brainstorm/.genie/brainstorms/agent-first-cli-ux-edges/design.md` (parallel analysis)
