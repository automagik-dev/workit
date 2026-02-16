# Wish: Agent-First CLI UX -- Discoverability + Control (v1)

**Status:** DRAFT
**Slug:** `agent-cli-ux-unified`
**Created:** 2026-02-16
**Design:** `.genie/brainstorms/agent-cli-ux-unified/design.md`

---

## Summary

Make `gog` enable zero-shot agent task success -- no external skills, no prior schema memorization. Ships 4 features: discoverability core (help topics + field discovery) AND control layer (pagination + input templates). Zero new dependencies.

---

## Scope

### IN
- `gog agent help <topic>` OR explicit Kong help override for `gog help <topic>` with hybrid schema-generated + curated concept docs (see DEC-6 for strategy)
- Field discovery via `--select ""` (explicit empty string) lists available JSON fields
- Pagination control via `--max-results N` and `--page-token TOKEN` global flags
- `--generate-input` flag prints JSON input template for any command
- Unit tests for every new feature
- AGENTS.md updated with all new discovery features (including field discovery stderr behavior)

### OUT
- No `--jq` filter (deferred to v2 -- `--select` covers 80% of agent needs)
- No `file://` input (deferred to v2 -- agents can pipe stdin)
- No batch workflow JSON execution (separate future effort)
- No interactive auto-prompt (agents don't need it)
- No additional output formats beyond JSON/plain
- No breaking changes to existing commands

---

## Coordination Notes

- **root.go shared modification:** This wish modifies `internal/cmd/root.go` (Groups 1, 2, 3, 4). The wishes `google-workspace-mcp-integration` and `agent-cli-power-features-v2` also modify `root.go`. Recommend execution order: `google-workspace-mcp-integration` first, then this wish, then `agent-cli-power-features-v2`. Each wish must rebase against the latest `root.go` before merging.

---

## Decisions

- **DEC-1:** Help content is hybrid -- schema-generated facts (commands, flags, exit codes) + curated concept prose. Solves drift risk.
- **DEC-2:** Field discovery output goes to stderr -- preserves stdout parseable discipline.
- **DEC-3:** Pagination flag names match Google API convention: `--max-results`, `--page-token`.
- **DEC-4:** `--generate-input` derives templates from Kong command model (same source as `gog schema`).
- **DEC-5:** Zero new Go dependencies for v1.
- **DEC-6:** Kong `help` command conflict resolution. Kong registers a built-in `help` command. Two viable strategies: **(A)** Use `gog agent help <topic>` to avoid the conflict entirely (help topics live under the `agent` subcommand namespace), or **(B)** Override Kong's built-in help by defining an explicit `HelpCmd` struct in the CLI root that takes precedence. **Chosen strategy: (A)** -- `gog agent help <topic>`. This avoids any Kong internals manipulation and makes the agent-specific nature of these docs explicit. The Kong built-in `--help` flag continues to work normally for per-command help.
- **DEC-7:** Pagination precedence. Per-command `--max` (or `--limit`) takes priority over the global `--max-results` flag. The global `--max-results` serves as a fallback default for commands that do not define their own pagination flag. When both are provided, the per-command flag wins.

---

## Success Criteria

### Primary: Zero-Shot Agent Task Success
- [ ] Agent discovers auth setup via `gog agent help auth`, runs `gog login`, verifies with `gog status`
- [ ] Agent discovers selectable fields via `--select ""` (explicit empty string), lists Drive with projected fields
- [ ] Agent discovers input flags via `--generate-input`, constructs and sends an email

### Secondary: Technical Criteria
- [ ] `gog agent help topics` lists all topics; `gog agent help agent` renders hybrid content
- [ ] `gog drive ls --json --select ""` (explicit empty string) prints fields to stderr, exits 0
- [ ] `gog drive ls --max-results 5 --json` returns <=5 results + nextPageToken
- [ ] `gog gmail send --generate-input` prints JSON template with required markers
- [ ] `make ci` passes
- [ ] No regressions in existing behavior
- [ ] `gog drive ls --max 3 --max-results 5 --json` respects `--max 3` (per-command wins)

---

## Assumptions

- **ASM-1:** Output structs have `json:"..."` tags that can be introspected for field discovery.
- **ASM-2:** Kong command model exposes flag metadata (name, type, required, help, default, enum) needed for `--generate-input`.
- **ASM-3:** Existing per-command `--max`/`--limit`/`--page` flags can coexist with new global pagination flags. Per-command flags take precedence over global flags (see DEC-7). The global flags serve as defaults for commands without local pagination flags.

## Risks

- **RISK-1:** Help topic concept prose drifts from code -- Mitigation: keep curated sections minimal; lean on schema-generated facts; CI lint that verifies topic references exist in the Kong command model.
- **RISK-2:** Field discovery output diverges from `--select` behavior -- Mitigation: generate field list from SAME code path that processes `--select` (single source of truth).
- **RISK-3:** Pagination flags conflict with existing per-command `--max`/`--page` -- Mitigation: clear precedence rule (per-command wins, see DEC-7); audit all commands; add test covering both flags together.
- **RISK-4:** Discoverability features accidentally erode stdout discipline -- Mitigation: field discovery -> stderr only; help topics -> stdout when tty; `--generate-input` -> stdout (it IS the result).
- **RISK-5:** `--results-only` + `--max-results` interaction confusion -- Mitigation: document that `--results-only` strips `nextPageToken` from output; recommend omitting `--results-only` when paginating across multiple pages.

---

## Execution Groups

### Group 1: Help Topics (Hybrid Schema-Generated + Curated)

**Goal:** Add `gog agent help <topic>` providing concept-level docs with live schema facts that never drift.

**Note on Kong help conflict:** Kong registers a built-in `help` command. To avoid conflict, help topics are registered under `gog agent help <topic>` (see DEC-6). The Kong built-in `--help` flag continues to work normally for per-command usage help.

**Deliverables:**
- NEW: `internal/cmd/help_topics.go` -- topic registry, hybrid renderer (curated prose + schema-generated facts)
- NEW: `internal/cmd/help_topics_test.go` -- verify topics render, facts match schema
- Modify: `internal/cmd/root.go` -- register `agent help` command group at root level

**Acceptance Criteria:**
- [ ] `gog agent help topics` lists all available topics with descriptions
- [ ] `gog agent help agent` renders agent integration guide with live schema facts (services, exit codes)
- [ ] `gog agent help auth` explains authentication workflow
- [ ] `gog agent help output` explains --json, --plain, --select, --results-only
- [ ] `gog agent help <unknown>` suggests closest match (fuzzy)
- [ ] Output: colored on tty, plain when piped, JSON-wrapped with `--json`
- [ ] Unit test verifies topic content renders and generated facts match schema
- [ ] CI lint verifies that all topic references (command names, flag names) exist in the Kong command model

**Validation:** `make test && go test -run TestHelpTopics ./internal/cmd/...`

---

### Group 2: Field Discovery for --select

**Goal:** `gog drive ls --json --select ""` (explicit empty string) lists available JSON fields without making an API call.

**Kong parsing note:** An explicit empty string `--select ""` is distinct from an omitted flag. Kong parses `--select ""` as the flag being present with value `""`. The implementation must detect this specific case (flag present, value is empty string) to trigger discovery mode. `--select` without any value (bare flag) may cause a Kong parse error; that is acceptable.

**Deliverables:**
- Modify: `internal/outfmt/outfmt.go` -- detect empty `--select ""`, enumerate fields from output struct `json:"..."` tags via reflection
- Modify: `internal/cmd/root.go` -- handle empty `--select` flag edge case with Kong
- NEW: `internal/outfmt/field_discovery_test.go` -- unit tests

**Acceptance Criteria:**
- [ ] `gog drive ls --json --select ""` (explicit empty string) prints available field names to stderr
- [ ] Field list matches what `--select` actually accepts (single source of truth)
- [ ] Exit 0 -- no API call made
- [ ] Hint printed: `gog drive ls --json --select "name,id,size"`
- [ ] Works for every command that supports `--json` output
- [ ] Unit test verifies field enumeration matches struct tags
- [ ] Field discovery output documented in AGENTS.md (stderr behavior, exit code, hint format)

**Validation:** `make test && go test -run TestFieldDiscovery ./internal/outfmt/...`

---

### Group 3: --generate-input

**Goal:** `gog gmail send --generate-input` prints JSON input template with all flags for the command.

**Deliverables:**
- NEW: `internal/cmd/generate_input.go` -- template generator from Kong command model
- NEW: `internal/cmd/generate_input_test.go` -- verify output matches Kong model
- Modify: `internal/cmd/root.go` -- add `--generate-input` flag, intercept before command execution

**Inclusion/exclusion rules for --generate-input:**
- **Include:** All global `RootFlags` fields (e.g., `--json`, `--select`, `--max-results`) + all command-specific flags for the target command.
- **Exclude:** Kong built-in flags (`--help`, `--version`), hidden flags (flags with `kong:"hidden"`), and internal implementation flags not meant for user input.

**Acceptance Criteria:**
- [ ] `gog gmail send --generate-input` prints JSON with all flag fields
- [ ] Required fields marked with `(required)` prefix in value
- [ ] Types, defaults, and enum values included
- [ ] Exit 0 without executing the command
- [ ] Works for any command (not just gmail send)
- [ ] Template includes global RootFlags and command-specific flags; excludes Kong built-ins and hidden flags
- [ ] Unit test verifies output matches Kong command model
- [ ] Unit test verifies Kong built-ins (`--help`) are excluded from output

**Validation:** `make test && go test -run TestGenerateInput ./internal/cmd/...`

---

### Group 4: Pagination Control

**Goal:** Expose `--max-results N` and `--page-token TOKEN` for agent-controlled pagination across all services.

**API parameter name mapping:** Different Google APIs use different parameter names for pagination. The global `--max-results` and `--page-token` flags must be mapped to the correct API parameter per service:

| Service      | Max Results Param | Page Token Param |
|-------------|-------------------|------------------|
| Calendar    | `maxResults`      | `pageToken`      |
| Classroom   | `pageSize`        | `pageToken`      |
| Drive       | `pageSize`        | `pageToken`      |
| Gmail       | `maxResults`      | `pageToken`      |
| People      | `pageSize`        | `pageToken`      |
| Admin       | `maxResults`      | `pageToken`      |
| Tasks       | `maxResults`      | `pageToken`      |
| Groups      | `maxResults`      | `pageToken`      |
| Sheets      | N/A (row-based)   | N/A              |
| Chat        | `pageSize`        | `pageToken`      |
| Keep        | `pageSize`        | `pageToken`      |

**Deliverables:**
- Modify: `internal/cmd/root.go` -- add `--max-results` (int) and `--page-token` (string) to `RootFlags`
- Modify: `internal/cmd/paging.go` -- wire global flags into `collectAllPages` and per-command fetch functions; implement per-service parameter name mapping
- Modify: Various `*_cmd.go` files -- map global flags to per-service Google API parameters
- NEW: `internal/cmd/pagination_test.go` -- unit tests for flag wiring

**Acceptance Criteria:**
- [ ] `gog drive ls --max-results 5 --json` returns <=5 results + `nextPageToken` in envelope
- [ ] `gog drive ls --max-results 5 --page-token TOKEN --json` fetches next page
- [ ] `--all` overrides `--max-results` (collects all pages)
- [ ] `--results-only` still strips envelope fields (note: this also strips `nextPageToken` -- document this interaction)
- [ ] Existing `--max`/`--limit` flags become aliases for `--max-results` (backward compatible)
- [ ] Per-command `--max` takes priority over global `--max-results` when both are provided (see DEC-7)
- [ ] Commands without pagination silently ignore these flags
- [ ] Unit test verifies flag mapping for at least 3 services
- [ ] Unit test verifies per-command flag takes precedence over global flag

**Validation:** `make test && grep -l 'MaxResults\|PageToken\|max.results' internal/cmd/*.go`

---

## Files to Create/Modify

```
# Group 1: Help Topics
internal/cmd/help_topics.go             # NEW — topic registry + hybrid renderer
internal/cmd/help_topics_test.go        # NEW — topic tests
internal/cmd/root.go                    # Modify — register agent help command

# Group 2: Field Discovery
internal/outfmt/outfmt.go              # Modify — empty --select "" detection
internal/outfmt/field_discovery_test.go # NEW — field discovery tests
internal/cmd/root.go                    # Modify — --select edge case

# Group 3: --generate-input
internal/cmd/generate_input.go          # NEW — template generator
internal/cmd/generate_input_test.go     # NEW — generator tests
internal/cmd/root.go                    # Modify — --generate-input flag

# Group 4: Pagination Control
internal/cmd/root.go                    # Modify — global pagination flags
internal/cmd/paging.go                  # Modify — wire flags + parameter name mapping
internal/cmd/*_cmd.go                   # Modify — per-service mapping
internal/cmd/pagination_test.go         # NEW — pagination tests

# Documentation
AGENTS.md                              # Modify — document new features (including field discovery stderr, --results-only + pagination interaction)
```
