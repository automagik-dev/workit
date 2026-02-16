# Wish: Agent CLI Power Features (v2)

**Status:** DRAFT
**Slug:** `agent-cli-power-features-v2`
**Created:** 2026-02-16
**Design:** `.genie/brainstorms/agent-cli-power-features-v2/design.md`
**Depends-on:** `agent-cli-ux-unified` (v1 output pipeline for --jq). Note: only Group 2 (--jq) depends on v1. Group 1 (file://) has NO dependency on v1 and can be implemented independently.

---

## Summary

Add the two power features deferred from v1: built-in `--jq` filtering (via gojq) and `file://` input support for text-content flags. These close the remaining agent UX gaps after v1 ships discoverability and control.

---

## Scope

### IN
- `--jq` global flag for built-in jq filtering on JSON output (uses `itchyny/gojq`)
- `file://` and `fileb://` input prefix for text-content flags (--body, --content, etc.)
- Security: path traversal protection, symlink validation, sensitive file rejection, size limits for file:// input
- `file://` coexists with `--body-file` (different mechanism: `file://` is a value prefix on any opted-in text flag; `--body-file` is a dedicated flag). No deprecation of `--body-file`.
- Unit tests for both features
- AGENTS.md updated (including file:// security section)

### OUT
- No batch workflow JSON execution (Stripe pattern -- separate future wish)
- No additional output formats beyond JSON/plain
- No changes to `--select` behavior (--jq is complementary, not a replacement)
- No breaking changes to existing commands
- No deprecation of `--body-file`

---

## Decisions

- **DEC-1:** jq library is `itchyny/gojq` (pure Go, no CGO) -- same library GitHub CLI uses.
- **DEC-2:** `--jq` pipeline position: after `--results-only` -> `--select` -> `--jq`. Most intuitive order.
- **DEC-3:** `file://` is opt-in per flag -- only text-content flags, not all string flags.
- **DEC-4:** `fileb://` reads binary and base64-encodes -- matches AWS CLI pattern.
- **DEC-5:** `--jq` + `--plain` is an error (exit 2) -- clear incompatibility signal. Validated early in `Execute()` before any context setup or API calls.
- **DEC-6:** `file://` coexists with `--body-file`. They are different mechanisms: `file://` is a value prefix that works on any opted-in text flag (--body, --content, --description, --message, etc.), while `--body-file` is a dedicated flag on specific commands. Neither is deprecated.

---

## Success Criteria

- [ ] `gog drive ls --json --jq '.[].name'` outputs only file names
- [ ] `gog gmail send --body file://test.txt` reads file content as email body
- [ ] `gog gmail send --body "literal text"` still works unchanged
- [ ] `gog gmail send --body file://../../etc/passwd` is rejected
- [ ] `gog gmail send --body file://symlink-to-etc-passwd` is rejected (symlink pointing outside CWD)
- [ ] `make ci` passes
- [ ] No regressions in existing behavior

---

## Assumptions

- **ASM-1:** v1 output pipeline (--results-only -> --select) is stable and merged before --jq integration. Note: this only blocks Group 2. Group 1 (file://) can proceed independently.
- **ASM-2:** ~~REMOVED.~~ gojq Go version compatibility must be verified by checking `go.mod` in the `itchyny/gojq` repository before adding to `go.mod`. Do not assume compatibility; verify the minimum Go version requirement.

## Risks

- **RISK-1:** gojq adds ~2MB to binary -- Mitigation: acceptable tradeoff; GitHub CLI does the same.
- **RISK-2:** jq expression errors confuse agents -- Mitigation: wrap errors with hint including the expression and a docs link.
- **RISK-3:** file:// path traversal and symlink attacks -- Mitigation: resolve relative to CWD with full resolution chain (see Group 1 design); reject symlinks pointing outside CWD subtree; reject sensitive file patterns.
- **RISK-4:** file:// on non-content flags causes confusion -- Mitigation: only apply to explicitly opted-in flags; document which flags support it.

---

## Execution Groups

### Group 1: file:// Input Support

**Goal:** Enable `--body file://report.txt` to read file content for text-content flags.

**No dependency on v1.** This group can be implemented independently of `agent-cli-ux-unified`.

**Security design -- path resolution order:**
1. `filepath.Abs(path)` -- resolve to absolute path
2. `filepath.Clean(absPath)` -- normalize (remove `.`, `..`, double slashes)
3. Validate that cleaned absolute path is within CWD subtree (`strings.HasPrefix(cleaned, cwd)`)
4. `os.Lstat(cleaned)` -- check file info WITHOUT following symlinks
5. If `Lstat` reports a symlink: `filepath.EvalSymlinks(cleaned)` -> validate that the symlink TARGET is also within CWD subtree
6. If all checks pass, `os.ReadFile(cleaned)`

**Sensitive file patterns (case-insensitive matching):**
- `.env`, `.env.*` (e.g., `.env.local`, `.env.production`)
- `.ssh/*`
- `.aws/*`
- `.gcloud/*`
- `*credentials*` (any file with "credentials" in the name)
- `*secret*` (any file with "secret" in the name)
- `*.pem`, `*.key`, `*.p12`, `*.pfx` (certificate/key files)
- `*token*` (any file with "token" in the name)
- `id_rsa`, `id_ed25519`, `id_dsa` (SSH private keys)

**Deliverables:**
- NEW: `internal/input/file_input.go` -- `ResolveFileInput(value string) (string, error)` with security checks (path resolution, symlink validation, sensitive file rejection, size limit)
- NEW: `internal/input/file_input_test.go` -- unit tests (read, literal passthrough, traversal blocking, symlink attack, size limit, sensitive path rejection)
- Modify: `internal/cmd/gmail_send.go` -- resolve `--body` flag via `ResolveFileInput`
- Modify: `internal/cmd/docs_cmd.go` -- resolve `--content` flag
- Modify: `internal/cmd/calendar_cmd.go` -- resolve `--description` flag
- Modify: Other commands with text-content flags (chat send `--message`, tasks `--notes`, etc.)
- Modify: `AGENTS.md` -- add security section documenting file:// behavior, allowed patterns, and rejected patterns

**Acceptance Criteria:**
- [ ] `gog gmail send --body file://test.txt` reads file content as email body
- [ ] `gog gmail send --body fileb://image.png` reads binary, base64 encodes
- [ ] `gog gmail send --body "literal text"` unchanged (no prefix = literal)
- [ ] `gog gmail send --body file://../../etc/passwd` rejected with exit 2 and error: `"error: file path escapes working directory: ../../etc/passwd"`
- [ ] `gog gmail send --body file://.env` rejected (sensitive file) with exit 2 and error: `"error: access to sensitive file blocked: .env"`
- [ ] Symlink pointing outside CWD rejected with exit 2 and error: `"error: symlink target escapes working directory: <target>"`
- [ ] Symlink pointing to file within CWD is allowed (not all symlinks are rejected)
- [ ] Files >10MB rejected with clear error
- [ ] `file://` coexists with `--body-file` -- both work, neither is deprecated
- [ ] Sensitive file pattern matching is case-insensitive (`.ENV`, `.Env` also rejected)
- [ ] Unit tests cover: read success, literal passthrough, traversal, symlink within CWD (allowed), symlink outside CWD (rejected), sensitive path, case-insensitive sensitive match, size limit
- [ ] AGENTS.md updated with file:// security section

**Validation:** `make test && go test -run TestResolveFileInput ./internal/input/...`

---

### Group 2: Built-in --jq Filter

**Goal:** Add `--jq` global flag for built-in jq filtering on JSON output.

**Depends on v1:** Requires v1 output pipeline (--results-only -> --select) to be merged first.

**Early validation design:** `--jq` + `--plain` incompatibility is checked early in `Execute()` before any context setup, authentication, or API calls. This ensures fast failure with a clear error message.

**Output pipeline integration:** The `WriteJSON` function (or equivalent output path) gains a new `--jq` step: after JSON marshal and any `--select` field projection, if `--jq` is set: parse the JSON bytes, apply the gojq expression, and write the raw output directly to stdout. The result is NOT re-encoded into a JSON envelope. The identity filter `--jq '.'` must produce valid JSON identical to the input (no re-encoding artifacts).

**Deliverables:**
- NEW: `internal/outfmt/jq.go` -- `ApplyJQ(jsonBytes []byte, expression string) ([]byte, error)` wrapper around gojq
- NEW: `internal/outfmt/jq_test.go` -- unit tests (filter, transform, count, error cases, identity filter)
- Modify: `go.mod` -- add `github.com/itchyny/gojq` dependency (verify minimum Go version compatibility first)
- Modify: `internal/cmd/root.go` -- add `--jq` flag to `RootFlags`; add early validation of `--jq` + `--plain` in `Execute()`
- Modify: `internal/outfmt/outfmt.go` -- apply jq filter as final step in JSON output pipeline (after --results-only -> --select -> --jq); integrate into `WriteJSON` path

**Acceptance Criteria:**
- [ ] `gog drive ls --json --jq '.[].name'` outputs only file names
- [ ] `gog gmail search "in:inbox" --json --jq '[.[] | {from, subject}]'` restructures JSON
- [ ] `gog drive ls --json --jq 'length'` outputs count
- [ ] `gog drive ls --json --jq 'invalid['` exits 2 with stderr error: `"error: invalid jq expression: invalid[' — <parse error detail>"`
- [ ] `gog drive ls --plain --jq '.'` exits 2 with stderr error: `"error: --jq requires --json output (incompatible with --plain)"`
- [ ] `--jq` + `--plain` validation happens before any API call or auth
- [ ] `gog drive ls --json --jq '.'` (identity filter) produces valid JSON identical to without `--jq`
- [ ] Raw jq output written directly (no re-wrapping in JSON envelope)
- [ ] Unit tests cover: filter, transform, error, incompatible flag combo, identity filter producing valid JSON

**Validation:** `make test && go test -run TestApplyJQ ./internal/outfmt/...`

---

## Files to Create/Modify

```
# Group 1: file:// Input
internal/input/file_input.go            # NEW — ResolveFileInput + security (path resolution, symlink validation)
internal/input/file_input_test.go       # NEW — unit tests (including symlink attack cases)
internal/cmd/gmail_send.go              # Modify — resolve --body
internal/cmd/docs_cmd.go                # Modify — resolve --content
internal/cmd/calendar_cmd.go            # Modify — resolve --description
internal/cmd/chat_cmd.go               # Modify — resolve --message (if applicable)
AGENTS.md                              # Modify — add file:// security section

# Group 2: --jq Filter
internal/outfmt/jq.go                  # NEW — jq wrapper
internal/outfmt/jq_test.go             # NEW — jq tests (including identity filter)
go.mod                                 # Modify — add gojq dep (after verifying Go version compatibility)
internal/cmd/root.go                   # Modify — --jq flag + early --jq/--plain validation
internal/outfmt/outfmt.go              # Modify — pipeline integration into WriteJSON

# Documentation
AGENTS.md                              # Modify — document new features (file:// security in Group 1, --jq usage in Group 2)
```
