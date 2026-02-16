# Brainstorm: Agent CLI Power Features (v2)

## Status
WRS: ██████████ 100/100
 Problem ✅ | Scope ✅ | Decisions ✅ | Risks ✅ | Criteria ✅

## Problem Statement
After v1 ships discoverability (help topics, field discovery, --generate-input, pagination), agents will hit power ceilings: `--select` can't do complex transforms, and long content must be piped via stdin. v2 adds the two deferred power features to close these gaps.

## Scope

### IN — 2 Features

#### 1. --jq Filter (from gh CLI)
- `gog drive ls --json --jq '.[].name'` — built-in jq filtering
- Uses itchyny/gojq (pure Go, no CGO, same lib gh CLI uses)
- Complementary to `--select` (simple fields) — --jq handles complex transforms
- Applied AFTER --results-only and --select in the output pipeline
- Requires --json mode (error if used with --plain)

#### 2. file:// Input (from AWS CLI)
- `--body file://report.txt` — reads file content for text flags
- `--body fileb://image.png` — binary variant (base64-encoded)
- Falls back to literal string if no prefix
- Applied to specific flags: --body, --content, --description, --notes, --message, --text
- Security: resolve relative to CWD, reject traversal outside CWD subtree

### OUT
- Batch workflow JSON execution (Stripe pattern) — separate future effort
- Additional output formats beyond JSON/plain — not needed

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| jq library | itchyny/gojq | Pure Go, well-maintained, same as gh CLI |
| jq pipeline position | After --results-only → --select → --jq | Most intuitive: strip envelope, pick fields, then transform |
| file:// scope | Opt-in per flag, not universal | Only text-content flags; prevents confusion on path flags |
| fileb:// behavior | Read binary, base64 encode | Matches AWS pattern; useful for attachment content |
| Error on --jq + --plain | Exit code 2 (usage) | Clear incompatibility signal |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| gojq adds ~2MB to binary | Low | Acceptable; gh CLI does same; no CGO needed |
| gojq expression errors confuse agents | Medium | Wrap errors with hint: "jq syntax error — see https://jqlang.github.io/jq/manual/" |
| file:// path traversal | Medium | Resolve relative to CWD; block .. escapes; reject symlinks pointing outside subtree |
| file:// on non-text flags causes confusion | Low | Only apply to explicitly opted-in flags; document which flags support it |
| Depends on v1 shipping first | Low | v2 builds on output pipeline from v1; if v1 isn't done, --jq still works standalone |

## Acceptance Criteria

1. `gog drive ls --json --jq '.[].name'` outputs only file names
2. `gog gmail search "in:inbox" --json --jq '[.[] | {from, subject}]'` outputs restructured JSON
3. `gog drive ls --json --jq 'invalid[' --` exits with code 2 and helpful error on stderr
4. `gog drive ls --plain --jq '.'` exits with code 2 (incompatible flags)
5. `gog gmail send --body file://test.txt` reads file content as email body
6. `gog gmail send --body "literal text"` still works (no regression)
7. `gog gmail send --body file://../../etc/passwd` is rejected (traversal blocked)
8. Unit tests for both features
9. `make ci` passes
10. Backward compatible
11. AGENTS.md updated

## Dependency on Other Wishes

- **Depends on**: agent-cli-ux-unified (v1) — output pipeline changes (--select, --results-only ordering)
- **Complements**: google-workspace-mcp-integration — command tiers + read-only + office text + batch contacts
