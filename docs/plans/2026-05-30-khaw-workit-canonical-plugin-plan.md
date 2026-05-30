# KHAW Workit Canonical Plugin Integration Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Make Workit a canonical KHAW companion component surfaced through Hermes' plugin system, while preserving Workit as the workspace-intelligence CLI and keeping upstream Hermes updateable.

**Architecture:** Workit remains a standalone Go CLI (`wk`) for office/workspace operations. KHAW owns the Hermes integration layer: installer/component manifest entries, a KHAW plugin command/tool surface that delegates to `wk`, sanitized setup/doctor checks, and profile skills/templates. Microsoft 365 support should be added to Workit as first-class provider/connectors, not hidden inside Omni or Hermes core.

**Tech Stack:** Go 1.25 Workit CLI, Hermes plugin system (`ctx.register_tool`, `ctx.register_cli_command`, `ctx.register_skill`), KHAW additive distribution repo, Microsoft Graph for M365, existing Workit OAuth/keyring/config patterns.

---

## Current source map

### Workit repository

- Repo: `https://github.com/automagik-dev/workit.git`
- Local path: `/home/genie/prod/workit`
- Branch: `marotta`, created from `origin/main`
- Head: `65b7a01` / tag `v2.260309.1`
- Current status after plan creation: one new uncommitted doc under `docs/plans/`.

### Workit core shape

- Entrypoint: `cmd/wk/main.go`
- CLI root and global safety flags: `internal/cmd/root.go`
  - Important flags already available for KHAW policy: `--read-only`, `--dry-run`, `--enable-commands`, `--command-tier`, `--json`, `--no-input`.
- Auth commands: `internal/cmd/auth.go`
- Google OAuth/service scope mapping: `internal/googleauth/scopes.go`
- Google API client construction: `internal/googleapi/client.go`
- Per-service Google clients: `internal/googleapi/*.go`
- Per-service CLI commands: `internal/cmd/{gmail,calendar,drive,docs,sheets,slides,...}.go`
- Sync engine: `internal/sync/*`
- Config/secrets/keyring: `internal/config/*`, `internal/secrets/*`
- Existing OpenClaw/Claude plugin assets: `plugins/workit/*`
- Live test scripts: `scripts/live-tests/*`

### KHAW/Hermes plugin facts to respect

Hermes plugin docs say general plugins can provide:

- `ctx.register_tool(...)`
- `ctx.register_hook(...)`
- `ctx.register_command(...)`
- `ctx.register_cli_command(...)`
- `ctx.register_skill(...)`
- `ctx.dispatch_tool(...)`
- `ctx.llm.complete(...)`

Plugin discovery paths:

- General plugin: `~/.hermes/plugins/<name>/`
- Bundled Hermes plugin: `<hermes-repo>/plugins/<name>/`
- KHAW additive plugin source: `/home/genie/workspace/agents/khaw/plugins/khaw/`

KHAW update model says Workit-like behavior should be additive, not Hermes core drift:

- Prefer config/profile/SOUL/templates first.
- Then general Hermes plugin for KHAW tools, hooks, slash commands, CLI subcommands, bundled read-only skills, and host-owned LLM calls.
- Sidecar services/templates are appropriate for external components.
- Hermes core patch only as escape hatch.

### Hermes harness permission surface found

Felipe feedback: do not invent a separate M365/email safety mechanism before using the Hermes harness that already exists.

Relevant Hermes source:

- `tools/approval.py`
  - Central dangerous-operation approval flow.
  - Gateway/API sessions block the agent thread and send an approval request to the user instead of letting the model proceed silently.
  - Exposes plugin hooks: `pre_approval_request` and `post_approval_response`.
  - Approval data includes `command`, `description`, `pattern_key`, and `pattern_keys`.
- `tools/terminal_tool.py`
  - Registers the dangerous-command approval callback for terminal executions.
- `tools/send_message_tool.py`
  - Cross-channel send tool already forces explicit target/message arguments and requires target discovery when sending to a named person/channel.
- `plugins/workit/skills/safety.md` in Workit
  - Existing Workit doctrine: `--read-only` first, `--dry-run` before writes, `--force` only after explicit user confirmation.

Implication for Workit/M365:

- Read operations may run through `wk --read-only ... --json`.
- Email/calendar/chat write operations must be classified as high-risk even when they are not shell-dangerous.
- The KHAW Workit plugin should add a Workit-specific approval boundary before executing commands that send email, create/update/delete meetings, modify labels, share files, post Teams messages, or request broader OAuth scopes.
- Preferred implementation: a plugin/tool wrapper that emits a structured Hermes approval request using the existing approval/gateway path, with a human-readable prompt containing exact account, recipients, subject/title, operation type, and side effects.
- Approval TTL must be configurable, defaulting to **30 minutes / 1800 seconds** for gateway popups.
- All write operations fail closed if the approval harness is unavailable, cannot deliver the popup, times out, receives a malformed response, or if the approved payload hash does not match the command about to execute.
- No `--force`, no `Mail.Send`, no `Calendars.ReadWrite`, no Teams send/write scopes in the Bernardo/Hapvida pilot without explicit approval and audit trail.

Dedicated contract doc: `docs/plans/2026-05-30-workit-m365-write-approval-contract.md`.
RLMX Fleet evidence ledger: `/home/genie/workspace/agents/khaw/plugins/khaw/rlmx-council/ledgers/2026-05-30T21-23-47-build.md`.

---

## Product decision

**Canonical ownership:**

- Workit owns office/workspace API connectors and CLI semantics.
- KHAW owns the enterprise distribution/install/doctor/profile surface.
- Hermes owns runtime/plugin APIs and should not gain permanent Workit-specific core code.
- Omni remains messaging/event backbone, not the M365 workspace-intelligence core.

**M365 direction:**

- Add Microsoft 365 as Workit's second provider family, not as a Hermes-only plugin.
- Use official Microsoft Graph; no scraping or user simulation.
- Start read-only for Hapvida/Bernardo: mail/calendar first, Teams/SharePoint after policy decisions.

---

## Implementation plan

### Task 1: Add KHAW component manifest entry for Workit

**Objective:** Make Workit a named, canonical KHAW additive component.

**Files:**
- Modify in KHAW repo: `/home/genie/workspace/agents/khaw/manifest/khaw-additions.yaml`
- Test: `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_doctor.py` or a new contract test if needed.

**Step 1: Add a `khaw-workit` extension point**

Add an entry like:

```yaml
  - id: khaw-workit
    kind: sidecar-component-and-hermes-plugin-surface
    path: external: /home/genie/prod/workit or selected release artifact
    install_target: $KHAW_HOME/components/workit plus $HERMES_HOME/plugins/khaw workit commands
    default_state: optional-enterprise
    owns:
      - Workit CLI installation and update pinning
      - office/workspace provider integrations
      - wk health/status/auth checks surfaced through KHAW doctor
      - read-only skills and command recipes for agents
      - M365 provider rollout plan for Hapvida
    verification:
      - wk binary is discoverable
      - wk --version exits zero
      - wk auth status --json exits zero or returns machine-readable setup-needed
      - khaw status/doctor reports Workit installed|missing|degraded
      - no secrets, account tokens, live mail, live docs, or keyring files are shipped in the KHAW repo
    forbidden:
      - raw mailbox/document contents in repo
      - refresh/access tokens in repo
      - trainer Workit config/keyring state
      - broad write permissions by default
```

**Step 2: Verify YAML parses**

Run:

```bash
python - <<'PY'
import yaml
p='/home/genie/workspace/agents/khaw/manifest/khaw-additions.yaml'
yaml.safe_load(open(p))
print('ok')
PY
```

Expected: `ok`.

**Step 3: Commit only if KHAW dirty state is understood**

There is existing unrelated dirty state in KHAW (`scripts/install_local_harness.py`). Do not commit over it blindly. Either stash/snapshot it with Felipe's approval or commit only the manifest/doc changes after reviewing the existing diff.

---

### Task 2: Extend KHAW plugin status with Workit component checks

**Objective:** Make `/khaw status`, `khaw_status`, and `hermes khaw status` aware of Workit without reading secrets.

**Files:**
- Modify: `/home/genie/workspace/agents/khaw/plugins/khaw/__init__.py`
- Test: `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_doctor.py` or new `test_khaw_workit.py`

**Step 1: Add a no-secret Workit probe**

Add helper:

```python
def _workit_status(repo: Path) -> dict[str, Any]:
    candidates = [
        Path('/home/genie/prod/workit/bin/wk'),
        Path.home() / '.local' / 'bin' / 'wk',
    ]
    binary = next((p for p in candidates if p.exists()), None)
    source_repo = Path('/home/genie/prod/workit')
    return {
        'source_repo': str(source_repo),
        'source_repo_exists': source_repo.exists(),
        'binary': str(binary) if binary else None,
        'binary_exists': bool(binary),
    }
```

Keep it read-only. Do not call `wk auth list` unless the output is sanitized because account emails may be sensitive in a generic KHAW base template.

**Step 2: Include it under status payload checks**

Add `workit_source_repo` and `workit_binary` checks. For base-template installs, `workit_binary` may be optional/degraded rather than required.

**Step 3: Test no mutation/no secret leakage**

Run:

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract -q -o 'addopts='
```

Expected: existing tests remain green. If test environment lacks Go/Workit binary, status should report missing/degraded, not fail KHAW core.

---

### Task 3: Add KHAW plugin CLI subcommands for Workit

**Objective:** Give operators a canonical command path such as `hermes khaw workit status` and eventually `hermes khaw workit doctor`.

**Files:**
- Modify: `/home/genie/workspace/agents/khaw/plugins/khaw/__init__.py`
- Test: new or existing KHAW CLI contract test.

**Step 1: Extend `_setup_cli`**

Add subparser:

```python
workit = sub.add_parser('workit', help='Inspect Workit/KHAW workspace integration')
workit_sub = workit.add_subparsers(dest='workit_command')
workit_sub.add_parser('status', help='Show Workit source/binary status')
```

**Step 2: Extend `_cli_handler`**

Route `khaw workit status` to a formatted `_workit_status(...)` output.

**Step 3: Verify CLI registration**

After installing/enabling the KHAW plugin in a Hermes home, run:

```bash
hermes khaw workit status
```

Expected: no auth calls, no tokens printed, tells whether source/binary are present.

---

### Task 4: Add Workit provider boundary in code before M365 implementation

**Objective:** Avoid scattering Microsoft-specific logic through Google-only packages.

**Files:**
- Create: `/home/genie/prod/workit/internal/provider/provider.go`
- Possibly create: `/home/genie/prod/workit/internal/msauth/*`
- Possibly create: `/home/genie/prod/workit/internal/msgraph/*`
- Modify later: `/home/genie/prod/workit/internal/cmd/root.go`

**Step 1: Introduce provider terminology without behavior change**

Create a tiny provider package:

```go
package provider

type Provider string

const (
    ProviderGoogle    Provider = "google"
    ProviderMicrosoft Provider = "microsoft"
)

func (p Provider) String() string { return string(p) }
```

**Step 2: Add tests**

Create `internal/provider/provider_test.go` with simple constants/string tests.

**Step 3: Run tests**

Requires Go installed. Current VM does **not** have `go` on PATH, so first install/provision Go 1.25 or use Workit's documented dev environment.

Expected:

```bash
go test ./internal/provider
```

---

### Task 5: Add Microsoft Graph auth package in Workit

**Objective:** Implement token acquisition/storage patterns for Microsoft without breaking Google auth.

**Files:**
- Create: `internal/msauth/config.go`
- Create: `internal/msauth/token.go`
- Create: `internal/msauth/scopes.go`
- Create tests beside each file.

**Design:**

Support two auth models explicitly:

1. Delegated OAuth per user/director.
2. App-only/client credentials for admin-approved service scenarios.

Do **not** silently default to broad app-only permissions. Hapvida pilot should be read-only and policy-bound.

**Minimum scopes for Bernardo pilot:**

- Mail read: `Mail.Read`
- Calendar read: `Calendars.Read`
- User profile: `User.Read`

Teams/SharePoint later:

- Teams chats/channels only after allowlist/governance decision.
- Files/Sites only after document-access policy decision.

---

### Task 6: Add Outlook read commands as M365 equivalent of Gmail read commands

**Objective:** Make `wk outlook search` and `wk outlook message get` available as read-only commands.

**Files:**
- Create: `internal/msgraph/mail.go`
- Create: `internal/cmd/outlook.go`
- Modify: `internal/cmd/root.go` to include `Outlook OutlookCmd`
- Modify scope/provider maps.
- Add tests.

**Initial commands:**

```bash
wk --read-only outlook search --since 7d --max 20 --json
wk --read-only outlook message get <id> --json
```

**Output contract:**

JSON-first, similar to Gmail:

```json
{
  "messages": [
    {
      "id": "...",
      "subject": "...",
      "from": "...",
      "receivedDateTime": "...",
      "importance": "high",
      "toMe": true,
      "hasAttachments": false,
      "webLink": "..."
    }
  ],
  "nextPageToken": "..."
}
```

Do not include full body by default in `search`. Require explicit `message get` for body.

---

### Task 7: Add Calendar M365 read commands

**Objective:** Make `wk m365 calendar events --today --json` or equivalent available.

**Files:**
- Create: `internal/msgraph/calendar.go`
- Create or extend: `internal/cmd/m365_calendar.go`
- Add tests.

**Initial commands:**

```bash
wk --read-only m365 calendar events --today --json
wk --read-only m365 calendar events --tomorrow --json
wk --read-only m365 calendar freebusy --date 2026-05-31 --json
```

Prefer a `m365` namespace if command-name collisions with Google `calendar` would confuse agents.

---

### Task 8: Add executive briefing command in Workit, not Hermes core

**Objective:** Give KHAW/Hermes a stable CLI to call for Bernardo's daily briefing inputs.

**Files:**
- Create: `internal/cmd/briefing.go` or `internal/cmd/m365_briefing.go`
- Add tests.

**Initial command:**

```bash
wk --read-only m365 briefing daily --since 24h --json
```

**Output:** machine-readable evidence bundle, not prose-only:

```json
{
  "mail": { "urgent": [], "actionRequired": [], "fyis": [] },
  "calendar": { "today": [], "tomorrow": [], "conflicts": [] },
  "evidence": [{ "source": "outlook", "id": "...", "webLink": "..." }]
}
```

Hermes/KHAW can turn this into an elegant Portuguese executive narrative.

---

### Task 9: Add KHAW skill/templates for Workit M365 operation

**Objective:** Give future agents the correct playbook and guardrails.

**Files:**
- Create in KHAW repo: `plugins/khaw/skills/workit-m365/SKILL.md`
- Register via KHAW plugin or manifest.

**Skill content must include:**

- Use Workit for workspace APIs.
- M365 read-only by default.
- Never request/send/write permissions unless explicitly authorized.
- Use `wk --read-only ... --json` for data collection.
- Deliver executive briefing via private authorized channel only.
- Do not log raw mail bodies in KHAW repo or memory.

---

### Task 10: Add contract tests for KHAW + Workit integration

**Objective:** Prove Workit is canonical without requiring live Microsoft credentials.

**Files:**
- Create: `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit.py`

**Test cases:**

1. KHAW manifest includes `khaw-workit`.
2. KHAW plugin status includes a `workit` section.
3. No Workit token/config/keyring paths are copied into KHAW repo.
4. Workit source repo path is outside KHAW repo (`/home/genie/prod/workit` or installed release path), preserving sidecar/component boundary.

---

## Open questions for Felipe/Bernardo

1. Should Workit be installed into fresh KHAW machines from GitHub releases, source checkout, or a pinned artifact mirror?
2. Should M365 delegated OAuth be implemented directly in Workit, or should Workit initially consume a brokered token from a Hapvida-approved gateway?
3. Should the first Bernardo pilot deliver only an evidence bundle to Hermes, or should Workit itself produce the first executive summary?
4. Should the command namespace be `wk outlook ...` / `wk teams ...` or `wk m365 outlook ...` / `wk m365 teams ...`? Recommendation: use `wk m365 ...` to avoid provider ambiguity.
5. Does Hapvida require no raw body persistence, or can Workit keep local encrypted caches for dedupe/history?

## Immediate blockers

- This VM does not currently have `go` on PATH, so Workit tests/builds cannot run until Go is installed/provisioned.
- KHAW repo has pre-existing dirty state in `scripts/install_local_harness.py`; do not mix unrelated KHAW commits until reviewed.

## First execution slice recommendation

1. Add KHAW manifest entry + status probe + contract test.
2. Add a KHAW `workit-m365` skill with privacy/read-only doctrine.
3. In Workit, add only provider package + Microsoft scope definitions + tests.
4. Then implement `wk m365 outlook search` read-only.

This gives a safe, reviewable path from product architecture to Bernardo value without forcing M365 into Omni or Hermes core.
