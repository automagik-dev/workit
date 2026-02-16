# Brainstorm: google_workspace_mcp → gogcli Feature Integration

## Status
WRS: ██████████ 100/100
 Problem ✅ | Scope ✅ | Decisions ✅ | Risks ✅ | Criteria ✅

## Problem Statement
gogcli already exceeds google_workspace_mcp in Google API coverage (14 vs 12 services, 337+ vs ~100 tools). The goal is to cherry-pick 4 specific features and architectural patterns that would enhance gogcli for agent deployments and power-user workflows.

## Scope

### IN — 4 Features to Implement

#### 1. Three-Tier Command System (HIGH)
- **What**: YAML-based command tier config with `--command-tier core|extended|complete`
- **Why**: Reduces visible command surface for agent integrations
- **From**: `core/tool_tiers.yaml` + `core/tool_tier_loader.py`
- **To**: Extend `internal/cmd/enabled_commands.go`
- **Implementation**: YAML config mapping commands to tiers; cumulative

#### 2. True Read-Only Mode (HIGH)
- **What**: `--read-only` flag → readonly OAuth scopes + hide mutating commands
- **Why**: Safety guarantee for agent deployments beyond `--dry-run`
- **From**: `auth/scopes.py` + `core/tool_registry.py`
- **To**: `internal/googleauth/service.go` + `internal/cmd/root.go`

#### 3. Office Format Text Extraction (MEDIUM)
- **What**: DOCX/XLSX/PPTX → plain text via Go stdlib
- **Why**: Enables `gog drive cat document.docx` for pipelines/agents
- **To**: New `internal/officetext/` package

#### 4. Batch Contacts Operations (MEDIUM)
- **What**: `gog contacts batch create/update/delete`
- **Why**: Multi-contact ops via People API batch endpoint
- **To**: New `internal/cmd/contacts_batch.go`

### OUT
- MCP transport, OAuth 2.1 multi-user, SSRF, Custom Search, attachment auto-expiry

## Decisions
| Decision | Choice | Rationale |
|----------|--------|-----------|
| Delivery | Single wish, all 4 features | Ship together |
| Tier config | YAML | Human-readable, matches MCP pattern |
| Read-only | Scope + command hiding | Dual-layer safety |
| Office text | Go stdlib only | No deps; archive/zip + encoding/xml |
| Batch contacts | People API batch | Native Google batch support |

## Risks
| Risk | Mitigation |
|------|------------|
| Tier YAML maintenance | Auto-generate initial config from command tree |
| Read-only incomplete coverage | Audit all commands; test matrix |
| Office XML edge cases | Plain text only; fail gracefully |
| Batch contacts quotas | Existing retry/backoff in transport layer |

## Acceptance Criteria
1. `--command-tier core|extended|complete` restricts available commands per tier YAML
2. `--read-only` requests only `.readonly` scopes and hides all write commands
3. `gog drive cat file.docx` outputs plain text from DOCX/XLSX/PPTX files
4. `gog contacts batch create/update/delete` operates on multiple contacts
5. All features have unit tests
6. `make ci` passes
