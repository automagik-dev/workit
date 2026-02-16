# Wish: Agent CLI Power Features (v2)

**Status:** DRAFT
**Slug:** `agent-cli-power-features-v2`
**Created:** 2026-02-16
**Design:** `.genie/brainstorms/agent-cli-power-features-v2/design.md`
**Depends-on:** `agent-cli-ux-unified` (v1 output pipeline for --jq)

---

## Summary

Add the two power features deferred from v1: built-in `--jq` filtering (via gojq) and `file://` input support for text-content flags. These close the remaining agent UX gaps after v1 ships discoverability and control.

---

## Scope

### IN
- `--jq` global flag for built-in jq filtering on JSON output (uses `itchyny/gojq`)
- `file://` and `fileb://` input prefix for text-content flags (--body, --content, etc.)
- Security: path traversal protection, sensitive file rejection, size limits for file:// input
- Unit tests for both features
- AGENTS.md updated

### OUT
- No batch workflow JSON execution (Stripe pattern — separate future wish)
- No additional output formats beyond JSON/plain
- No changes to `--select` behavior (--jq is complementary, not a replacement)
- No breaking changes to existing commands

---

## Decisions

- **DEC-1:** jq library is `itchyny/gojq` (pure Go, no CGO) — same library GitHub CLI uses.
- **DEC-2:** `--jq` pipeline position: after `--results-only` → `--select` → `--jq`. Most intuitive order.
- **DEC-3:** `file://` is opt-in per flag — only text-content flags, not all string flags.
- **DEC-4:** `fileb://` reads binary and base64-encodes — matches AWS CLI pattern.
- **DEC-5:** `--jq` + `--plain` is an error (exit 2) — clear incompatibility signal.

---

## Success Criteria

- [ ] `gog drive ls --json --jq '.[].name'` outputs only file names
- [ ] `gog gmail send --body file://test.txt` reads file content as email body
- [ ] `gog gmail send --body "literal text"` still works unchanged
- [ ] `gog gmail send --body file://../../etc/passwd` is rejected
- [ ] `make ci` passes
- [ ] No regressions in existing behavior

---

## Assumptions

- **ASM-1:** v1 output pipeline (--results-only → --select) is stable and merged before --jq integration.
- **ASM-2:** `itchyny/gojq` is compatible with Go 1.25+ and has no CGO requirements.

## Risks

- **RISK-1:** gojq adds ~2MB to binary — Mitigation: acceptable tradeoff; GitHub CLI does the same.
- **RISK-2:** jq expression errors confuse agents — Mitigation: wrap errors with hint including the expression and a docs link.
- **RISK-3:** file:// path traversal — Mitigation: resolve relative to CWD, block `..` escapes, reject symlinks pointing outside subtree, reject `.env`/`.ssh`/`.aws`/`*credentials*`.
- **RISK-4:** file:// on non-content flags causes confusion — Mitigation: only apply to explicitly opted-in flags; document which flags support it.

---

## Execution Groups

### Group 1: file:// Input Support

**Goal:** Enable `--body file://report.txt` to read file content for text-content flags.

**Deliverables:**
- NEW: `internal/input/file_input.go` — `ResolveFileInput(value string) (string, error)` with security checks
- NEW: `internal/input/file_input_test.go` — unit tests (read, literal passthrough, traversal blocking, size limit, sensitive path rejection)
- Modify: `internal/cmd/gmail_send.go` — resolve `--body` flag via `ResolveFileInput`
- Modify: `internal/cmd/docs_cmd.go` — resolve `--content` flag
- Modify: `internal/cmd/calendar_cmd.go` — resolve `--description` flag
- Modify: Other commands with text-content flags (chat send `--message`, tasks `--notes`, etc.)

**Acceptance Criteria:**
- [ ] `gog gmail send --body file://test.txt` reads file content as email body
- [ ] `gog gmail send --body fileb://image.png` reads binary, base64 encodes
- [ ] `gog gmail send --body "literal text"` unchanged (no prefix = literal)
- [ ] `gog gmail send --body file://../../etc/passwd` rejected with exit 2
- [ ] `gog gmail send --body file://.env` rejected (sensitive file)
- [ ] Files >10MB rejected with clear error
- [ ] Unit tests cover: read success, literal passthrough, traversal, sensitive path, size limit

**Validation:** `make test && go test -run TestResolveFileInput ./internal/input/...`

---

### Group 2: Built-in --jq Filter

**Goal:** Add `--jq` global flag for built-in jq filtering on JSON output.

**Deliverables:**
- NEW: `internal/outfmt/jq.go` — `ApplyJQ(jsonBytes []byte, expression string) ([]byte, error)` wrapper around gojq
- NEW: `internal/outfmt/jq_test.go` — unit tests (filter, transform, count, error cases)
- Modify: `go.mod` — add `github.com/itchyny/gojq` dependency
- Modify: `internal/cmd/root.go` — add `--jq` flag to `RootFlags`
- Modify: `internal/outfmt/outfmt.go` — apply jq filter as final step in JSON output pipeline (after --results-only → --select → --jq)

**Acceptance Criteria:**
- [ ] `gog drive ls --json --jq '.[].name'` outputs only file names
- [ ] `gog gmail search "in:inbox" --json --jq '[.[] | {from, subject}]'` restructures JSON
- [ ] `gog drive ls --json --jq 'length'` outputs count
- [ ] `gog drive ls --json --jq 'invalid['` exits 2 with helpful stderr error
- [ ] `gog drive ls --plain --jq '.'` exits 2 (incompatible flags)
- [ ] Raw jq output written directly (no re-wrapping in JSON envelope)
- [ ] Unit tests cover: filter, transform, error, incompatible flag combo

**Validation:** `make test && go test -run TestApplyJQ ./internal/outfmt/...`

---

## Files to Create/Modify

```
# Group 1: file:// Input
internal/input/file_input.go            # NEW — ResolveFileInput + security
internal/input/file_input_test.go       # NEW — unit tests
internal/cmd/gmail_send.go              # Modify — resolve --body
internal/cmd/docs_cmd.go                # Modify — resolve --content
internal/cmd/calendar_cmd.go            # Modify — resolve --description
internal/cmd/chat_cmd.go               # Modify — resolve --message (if applicable)

# Group 2: --jq Filter
internal/outfmt/jq.go                  # NEW — jq wrapper
internal/outfmt/jq_test.go             # NEW — jq tests
go.mod                                 # Modify — add gojq dep
internal/cmd/root.go                   # Modify — --jq flag
internal/outfmt/outfmt.go              # Modify — pipeline integration

# Documentation
AGENTS.md                              # Modify — document new features
```
