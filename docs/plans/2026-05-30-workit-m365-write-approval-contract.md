# Workit M365 Write Approval Contract

**Context:** Hapvida/Bernardo M365 integration through Workit + KHAW + Hermes.

**Decision:** The first pilot is read-only. Any future non-read-only operation is impossible unless the Hermes approval harness is available and a human approves the exact operation through the gateway popup. Telegram is the first approval surface; the contract must remain platform-generic for Discord/Slack/Teams/etc. later.

**RLMX evidence:** `/home/genie/workspace/agents/khaw/plugins/khaw/rlmx-council/ledgers/2026-05-30T21-23-47-build.md`

**Implementation plan:** `/home/genie/prod/workit/docs/plans/2026-05-30-workit-m365-write-gate-implementation-plan.md`

---

## Non-negotiable law

1. **Default mode: read-only only.**
   - M365 pilot scopes: `User.Read`, `Mail.Read`, `Calendars.Read`.
   - No `Mail.Send`, `Calendars.ReadWrite`, Teams write, SharePoint write, file-share mutation, or app-only broad Graph permission in the initial pilot.

2. **Writes are fail-closed.**
   - If the approval harness is unavailable, expired, misconfigured, unreachable, or cannot deliver the popup, the operation is denied before Workit reaches Microsoft Graph.
   - There is no fallback to `allow`, no local auto-approval, no silent bypass, and no `--force` bypass.

3. **Approval is per operation or explicit batch.**
   - A single approval authorizes only the displayed operation payload.
   - Batch approvals must display the exact bounded batch: count, recipients/resources, operation type, and summary hash.
   - No approval may be reused for a materially different operation.

4. **Default approval TTL: 30 minutes.**
   - Configurable per deployment and optionally per operation risk tier.
   - Timer starts when the approval popup is successfully delivered.
   - Expiry equals denial.

5. **Prompt must be decision-grade.**
   - The human must see enough context to make a safe decision without exposing secrets.

---

## Enforcement boundary

The enforcement boundary is the **KHAW Workit wrapper/plugin**, not the model prompt and not Workit documentation.

Flow:

```text
Hermes/KHAW tool call
  -> KHAW Workit policy wrapper
    -> classify operation
      -> read allowlist: execute `wk --read-only ... --json`
      -> write/unknown: require Hermes approval
        -> approved: execute exact Workit command/payload
        -> denied/expired/unavailable: fail closed
```

Workit remains the owner of Microsoft Graph connector logic. KHAW owns the enterprise safety boundary around Workit execution.

---

## Classification rule

Use explicit allowlisting, not broad inference.

### Read-only allowlist

Allowed without approval only when all are true:

- Workit command is in a static read allowlist.
- Command includes or internally enforces `--read-only`.
- Microsoft Graph scope set is read-only.
- Underlying HTTP method is non-mutating, normally `GET`.
- Endpoint is not known to trigger side effects.
- Output policy prevents raw body persistence beyond the active session/evidence bundle.

Examples:

- `wk --read-only m365 outlook search --since 24h --json`
- `wk --read-only m365 outlook message get <id> --json`
- `wk --read-only m365 calendar events --today --json`
- `wk --read-only m365 calendar freebusy --date YYYY-MM-DD --json`

### Approval-required / denied by default

Everything else is write-classified unless explicitly proven read-only.

Examples:

- Sending email.
- Creating/updating/deleting calendar events.
- RSVP / invite response on behalf of the user.
- Teams post/reply/reaction/update/delete.
- SharePoint/OneDrive upload/delete/move/share/permission mutation.
- Gmail/Outlook labels, flags, categories, archive/delete/mark-read if they mutate mailbox state.
- Any command using `--force`.
- Any command requesting write Graph scopes.
- Any command not present in the allowlist.

Initial pilot behavior: write-classified operations return `WRITE_DISABLED_FOR_PILOT` even before asking for approval.

---

## Approval request contract

The KHAW wrapper must create a correlation id and approval payload before any write.

Suggested payload:

```json
{
  "approval_id": "uuid",
  "system": "khaw-workit",
  "provider": "m365",
  "account": "bernardo@... or redacted principal id",
  "operation_class": "write",
  "operation": "send_email | create_event | update_event | teams_post | share_file",
  "workit_command_hash": "sha256 exact command/payload",
  "resource": "mail | calendar | teams | sharepoint",
  "recipients": ["..."],
  "title_or_subject": "...",
  "side_effects": ["external recipient will receive email"],
  "required_scopes": ["Mail.Send"],
  "ttl_seconds": 1800,
  "requested_by": "telegram user/chat/session metadata",
  "created_at": "iso8601"
}
```

The Telegram popup should show, in Portuguese by default for Hapvida:

```text
⚠️ Aprovação necessária — Workit/M365

Operação: enviar e-mail
Conta: Bernardo / Hapvida
Destinatários: ana@..., joao@...
Assunto: Revisão do briefing
Efeito: e-mail será enviado de verdade
Escopos: Mail.Send
Validade: 30 min
ID: <short approval id>

Aprovar ou negar?
```

Future platforms should receive the same structured payload rendered in their native approval UI.

---

## Harness behavior

### Required Hermes primitive

Current Hermes source confirms the existing approval system in `tools/approval.py`:

- gateway blocking approval path;
- `/approve` and `/deny` user response path;
- `pre_approval_request` and `post_approval_response` plugin hooks;
- timeout/gateway wait behavior.

For shell commands this is already wired through dangerous-command detection. For Workit business writes, the KHAW plugin should either:

1. call/refactor a reusable non-shell approval API backed by the same queue; or
2. use a minimal core seam/plugin hook that creates an approval entry without pretending the operation is a shell command.

Avoid encoding business writes as fake dangerous shell strings if a structured API can be exposed. The approval request should be semantic and auditable.

### Fail-closed cases

Deny before execution if:

- no gateway approval callback is registered;
- approval popup cannot be delivered;
- approval queue write fails;
- approval times out;
- approval response is malformed;
- approver identity is not authorized;
- approval hash does not match the command/payload being executed;
- Workit command tries to add `--force` after approval;
- requested Graph scopes exceed approved scopes;
- process restarts between approval and execution without durable pending-state validation.

---

## Audit contract

Every write attempt produces an audit entry, even when denied.

Fields:

- `approval_id`
- `correlation_id`
- `platform`
- `chat_id` / thread id, where safe
- `requesting_user_id`, where safe
- `approved_by_user_id`, where safe
- `operation_class`
- `operation`
- `resource`
- `target_count`
- `payload_hash`
- `required_scopes`
- `status`: `write_disabled`, `pending`, `approved`, `denied`, `expired`, `harness_unavailable`, `execution_failed`, `executed`
- `created_at`
- `responded_at`
- `executed_at`
- `ttl_seconds`
- redacted error message, if any

Do not log raw email bodies, attachment contents, access tokens, refresh tokens, or Graph response bodies containing private data.

---

## Test plan

### Unit tests

- Classification matrix: every known Workit M365 command maps to `read`, `write`, or `unknown`.
- Unknown defaults to write/deny.
- `--read-only` required for read commands.
- `--force` always requires approval and is denied in pilot mode.
- Write scopes are rejected in pilot mode.

### Integration tests with fake harness

- Read command executes without approval.
- Write command creates approval request and blocks.
- Approved write executes exact approved payload only.
- Denied write does not execute.
- Expired write does not execute.
- Harness unavailable returns fail-closed error.
- Mismatched payload hash after approval returns fail-closed error.

### Gateway tests

- Telegram popup renders operation details and TTL.
- `/approve` resolves only the pending approval id.
- `/deny` resolves only the pending approval id.
- Approval times out at configured TTL, default 1800 seconds.
- Unauthorized user cannot approve someone else's operation.

### Graph safety tests

- No `Mail.Send`/`Calendars.ReadWrite` requested in pilot auth.
- Workit cannot silently upgrade scopes.
- App-only broad scopes are not accepted by default.

---

## Implementation sequence

1. Add KHAW Workit policy module with static read/write classifier.
2. Add tests proving unknown/write operations fail closed.
3. Add or expose a structured Hermes approval API using the existing gateway approval queue.
4. Add Telegram approval rendering with TTL default 1800s.
5. Add audit ledger with redaction.
6. Wire Workit M365 read-only commands.
7. Only later, behind `WRITE_ENABLED=true`, allow approved write commands.

---

## Pitfalls called out by RLMX fleet

- Misclassifying a write as read is the highest-risk failure.
- Some `GET` endpoints can still have side effects; use allowlists, not method-only inference.
- Approval popups without recipients/resources/side effects are unsafe theater.
- Long TTLs can create stale approvals; bind approval to exact payload hash.
- Batch approvals can become dangerous; require bounded preview and count.
- Current Hermes approval API is shell-oriented; structured business-operation approval may need a clean seam.
- Do not let Workit `--force` or direct binary calls bypass the KHAW wrapper in the agent tool surface.
