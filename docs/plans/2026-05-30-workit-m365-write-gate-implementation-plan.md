# Workit M365 Write Gate Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Enforce the law that no non-read-only Workit/M365 operation can execute without a Hermes gateway approval popup, with fail-closed behavior and default 30-minute TTL.

**Architecture:** Workit remains the Microsoft Graph CLI. KHAW owns a policy wrapper around Workit execution. The wrapper classifies every operation; read allowlisted commands execute with `--read-only`, while write/unknown operations require a structured Hermes approval request before Workit receives the exact approved payload.

**Tech Stack:** Python KHAW plugin wrapper, Hermes `tools/approval.py` gateway approval queue/hook seam, Workit Go CLI, Microsoft Graph scopes, pytest contract tests.

---

## Source references

- Contract: `/home/genie/prod/workit/docs/plans/2026-05-30-workit-m365-write-approval-contract.md`
- Main integration plan: `/home/genie/prod/workit/docs/plans/2026-05-30-khaw-workit-canonical-plugin-plan.md`
- RLMX Fleet evidence: `/home/genie/workspace/agents/khaw/plugins/khaw/rlmx-council/ledgers/2026-05-30T21-23-47-build.md`
- Hermes approval source: `/home/genie/.hermes/hermes-agent/tools/approval.py`
- Workit safety doctrine: `/home/genie/prod/workit/plugins/workit/skills/safety.md`

---

## Task 1: Add operation classification contract tests

**Objective:** Prove the safety classifier defaults to read-only allowlist and treats unknowns as gated/denied.

**Files:**
- Create: `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py`
- Later create implementation: `/home/genie/workspace/agents/khaw/plugins/khaw/workit_policy.py`

**Step 1: Write failing tests**

```python
from plugins.khaw.workit_policy import classify_workit_command


def test_read_only_m365_outlook_search_is_read():
    result = classify_workit_command([
        "wk", "--read-only", "m365", "outlook", "search", "--since", "24h", "--json"
    ])

    assert result.operation_class == "read"
    assert result.requires_approval is False


def test_same_command_without_read_only_is_not_read():
    result = classify_workit_command([
        "wk", "m365", "outlook", "search", "--since", "24h", "--json"
    ])

    assert result.operation_class in {"write", "unknown"}
    assert result.requires_approval is True


def test_send_email_requires_approval():
    result = classify_workit_command([
        "wk", "m365", "outlook", "send", "--to", "a@example.com", "--subject", "x"
    ])

    assert result.operation_class == "write"
    assert result.requires_approval is True


def test_unknown_m365_command_requires_approval():
    result = classify_workit_command(["wk", "m365", "mystery", "do-thing"])

    assert result.operation_class == "unknown"
    assert result.requires_approval is True


def test_force_always_requires_approval():
    result = classify_workit_command([
        "wk", "--read-only", "m365", "calendar", "events", "--today", "--force", "--json"
    ])

    assert result.requires_approval is True
    assert "force" in result.reasons
```

**Step 2: Run RED**

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py -q -o 'addopts='
```

Expected: FAIL because `plugins.khaw.workit_policy` does not exist.

---

## Task 2: Implement minimal classifier

**Objective:** Make the classification tests pass without adding execution behavior.

**Files:**
- Create: `/home/genie/workspace/agents/khaw/plugins/khaw/workit_policy.py`

**Implementation:**

```python
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Sequence


@dataclass(frozen=True)
class WorkitPolicyDecision:
    operation_class: str
    requires_approval: bool
    reasons: tuple[str, ...] = field(default_factory=tuple)


READ_ALLOWLIST: set[tuple[str, ...]] = {
    ("m365", "outlook", "search"),
    ("m365", "outlook", "message", "get"),
    ("m365", "calendar", "events"),
    ("m365", "calendar", "freebusy"),
}

WRITE_PREFIXES: set[tuple[str, ...]] = {
    ("m365", "outlook", "send"),
    ("m365", "calendar", "create"),
    ("m365", "calendar", "update"),
    ("m365", "calendar", "delete"),
    ("m365", "teams", "send"),
    ("m365", "sharepoint", "share"),
    ("m365", "onedrive", "share"),
}


def _strip_wk(argv: Sequence[str]) -> list[str]:
    args = list(argv)
    if args and args[0] == "wk":
        args = args[1:]
    return args


def _positional(args: Sequence[str]) -> list[str]:
    out: list[str] = []
    skip_next = False
    for arg in args:
        if skip_next:
            skip_next = False
            continue
        if arg in {"--since", "--to", "--subject", "--date", "--max"}:
            skip_next = True
            continue
        if arg.startswith("-"):
            continue
        out.append(arg)
    return out


def _matches_prefix(tokens: Sequence[str], prefixes: set[tuple[str, ...]]) -> bool:
    return any(tuple(tokens[: len(prefix)]) == prefix for prefix in prefixes)


def classify_workit_command(argv: Sequence[str]) -> WorkitPolicyDecision:
    args = _strip_wk(argv)
    reasons: list[str] = []

    if "--force" in args or "-y" in args:
        reasons.append("force")

    read_only = "--read-only" in args
    tokens = _positional(args)

    if _matches_prefix(tokens, WRITE_PREFIXES):
        return WorkitPolicyDecision("write", True, tuple(reasons or ["write_prefix"]))

    if tuple(tokens) in READ_ALLOWLIST and read_only and not reasons:
        return WorkitPolicyDecision("read", False, ())

    if tuple(tokens) in READ_ALLOWLIST and not read_only:
        reasons.append("missing_read_only")
        return WorkitPolicyDecision("unknown", True, tuple(reasons))

    return WorkitPolicyDecision("unknown", True, tuple(reasons or ["not_allowlisted"]))
```

**Step 3: Run GREEN**

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py -q -o 'addopts='
```

Expected: PASS.

---

## Task 3: Add approval payload hashing tests

**Objective:** Ensure approvals bind to the exact command/payload.

**Files:**
- Modify test: `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py`
- Modify implementation: `/home/genie/workspace/agents/khaw/plugins/khaw/workit_policy.py`

**Step 1: Write RED tests**

```python
from plugins.khaw.workit_policy import build_payload_hash


def test_payload_hash_is_stable_for_same_argv():
    argv = ["wk", "m365", "outlook", "send", "--to", "a@example.com"]
    assert build_payload_hash(argv) == build_payload_hash(list(argv))


def test_payload_hash_changes_when_recipient_changes():
    base = ["wk", "m365", "outlook", "send", "--to", "a@example.com"]
    changed = ["wk", "m365", "outlook", "send", "--to", "b@example.com"]
    assert build_payload_hash(base) != build_payload_hash(changed)
```

**Step 2: Implement minimal hash**

```python
import hashlib
import json


def build_payload_hash(argv: Sequence[str]) -> str:
    raw = json.dumps(list(argv), ensure_ascii=False, separators=(",", ":"))
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()
```

**Step 3: Run tests**

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py -q -o 'addopts='
```

Expected: PASS.

---

## Task 4: Add structured approval request interface tests

**Objective:** Define a non-shell business-operation approval seam that can later call Hermes' gateway approval queue.

**Files:**
- Modify test: `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py`
- Modify implementation: `/home/genie/workspace/agents/khaw/plugins/khaw/workit_policy.py`

**Step 1: Write RED test**

```python
from plugins.khaw.workit_policy import build_approval_request


def test_build_approval_request_defaults_to_30_minutes():
    req = build_approval_request(
        argv=["wk", "m365", "outlook", "send", "--to", "a@example.com", "--subject", "Oi"],
        account="bernardo@hapvida.example",
        requested_by="telegram:123",
    )

    assert req["system"] == "khaw-workit"
    assert req["provider"] == "m365"
    assert req["ttl_seconds"] == 1800
    assert req["payload_hash"]
    assert req["operation_class"] == "write"
    assert req["requested_by"] == "telegram:123"
```

**Step 2: Implement minimal request builder**

```python
from datetime import datetime, timezone
from uuid import uuid4


def build_approval_request(
    *,
    argv: Sequence[str],
    account: str,
    requested_by: str,
    ttl_seconds: int = 1800,
) -> dict:
    decision = classify_workit_command(argv)
    return {
        "approval_id": str(uuid4()),
        "system": "khaw-workit",
        "provider": "m365",
        "account": account,
        "operation_class": decision.operation_class,
        "requires_approval": decision.requires_approval,
        "payload_hash": build_payload_hash(argv),
        "argv": list(argv),
        "requested_by": requested_by,
        "ttl_seconds": ttl_seconds,
        "created_at": datetime.now(timezone.utc).isoformat(),
    }
```

**Step 3: Run tests**

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py -q -o 'addopts='
```

Expected: PASS.

---

## Task 5: Add fail-closed execution wrapper tests

**Objective:** Prove that write/unknown operations do not execute when approval harness is unavailable.

**Files:**
- Modify test: `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py`
- Modify implementation: `/home/genie/workspace/agents/khaw/plugins/khaw/workit_policy.py`

**Step 1: Write RED tests**

```python
import pytest
from plugins.khaw.workit_policy import SafetyHarnessUnavailable, execute_workit_with_policy


class DenyingHarness:
    available = True
    def request_approval(self, request):
        return {"approved": False, "reason": "denied"}


class UnavailableHarness:
    available = False
    def request_approval(self, request):
        raise AssertionError("should not be called")


def test_write_fails_closed_when_harness_unavailable():
    with pytest.raises(SafetyHarnessUnavailable):
        execute_workit_with_policy(
            ["wk", "m365", "outlook", "send", "--to", "a@example.com"],
            account="bernardo@hapvida.example",
            requested_by="telegram:123",
            harness=UnavailableHarness(),
            runner=lambda argv: {"executed": True},
        )


def test_write_denied_does_not_execute():
    executed = False
    def runner(argv):
        nonlocal executed
        executed = True
        return {"executed": True}

    result = execute_workit_with_policy(
        ["wk", "m365", "outlook", "send", "--to", "a@example.com"],
        account="bernardo@hapvida.example",
        requested_by="telegram:123",
        harness=DenyingHarness(),
        runner=runner,
    )

    assert result["status"] == "denied"
    assert executed is False
```

**Step 2: Implement minimal wrapper**

```python
class SafetyHarnessUnavailable(RuntimeError):
    pass


def execute_workit_with_policy(
    argv: Sequence[str],
    *,
    account: str,
    requested_by: str,
    harness,
    runner,
):
    decision = classify_workit_command(argv)
    if not decision.requires_approval:
        return runner(list(argv))

    if not getattr(harness, "available", False):
        raise SafetyHarnessUnavailable("Workit write blocked: Hermes approval harness unavailable")

    req = build_approval_request(argv=argv, account=account, requested_by=requested_by)
    approval = harness.request_approval(req)
    if not approval or not approval.get("approved"):
        return {"status": "denied", "approval_id": req["approval_id"]}

    if approval.get("payload_hash") and approval["payload_hash"] != req["payload_hash"]:
        raise SafetyHarnessUnavailable("Workit write blocked: approved payload hash mismatch")

    return runner(list(argv))
```

**Step 3: Run tests**

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py -q -o 'addopts='
```

Expected: PASS.

---

## Task 6: Add pilot-mode write-disabled tests

**Objective:** Enforce the current Hapvida/Bernardo pilot as read-only even if a write approval harness exists.

**Files:**
- Modify test and implementation from previous tasks.

**Step 1: Write RED test**

```python
from plugins.khaw.workit_policy import execute_workit_with_policy


class ApprovingHarness:
    available = True
    def request_approval(self, request):
        return {"approved": True, "payload_hash": request["payload_hash"]}


def test_pilot_mode_blocks_writes_even_if_harness_approves():
    result = execute_workit_with_policy(
        ["wk", "m365", "outlook", "send", "--to", "a@example.com"],
        account="bernardo@hapvida.example",
        requested_by="telegram:123",
        harness=ApprovingHarness(),
        runner=lambda argv: {"executed": True},
        write_enabled=False,
    )

    assert result["status"] == "write_disabled_for_pilot"
```

**Step 2: Add `write_enabled` gate before requesting approval**

```python
# Add parameter: write_enabled: bool = False
if decision.requires_approval and not write_enabled:
    return {"status": "write_disabled_for_pilot", "operation_class": decision.operation_class}
```

**Step 3: Run tests**

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py -q -o 'addopts='
```

Expected: PASS.

---

## Task 7: Wire Hermes structured approval seam

**Objective:** Replace fake harnesses with a real adapter that uses Hermes approval machinery without faking a shell command.

**Files:**
- Create: `/home/genie/workspace/agents/khaw/plugins/khaw/hermes_approval_adapter.py`
- Add tests with monkeypatch/fake `tools.approval` entry points once the exact seam is chosen.

**Required discovery before implementation:**

Read `/home/genie/.hermes/hermes-agent/tools/approval.py` around:

- `_gateway_queues`
- `_ApprovalEntry`
- `register_gateway_notify`
- `resolve_gateway_approval`
- `_fire_approval_hook`
- timeout handling

**Implementation rule:**

If Hermes exposes no clean public function for semantic approval requests, add the smallest upstreamable seam in Hermes core, e.g.:

```python
def request_gateway_approval(
    *,
    description: str,
    pattern_key: str,
    pattern_keys: list[str],
    metadata: dict | None = None,
    timeout: int | None = None,
) -> dict:
    ...
```

This function must reuse the existing gateway queue and `/approve`/`/deny` resolver.

**Do not:** encode Workit writes as fake shell strings unless this is explicitly accepted as a temporary bridge.

---

## Task 8: Add Telegram popup renderer for business approvals

**Objective:** Render decision-grade Workit/M365 approval messages in Telegram.

**Files:**
- Likely modify: `/home/genie/.hermes/hermes-agent/gateway/platforms/telegram.py`
- Or better: add platform-neutral rendering extension if existing approval callback supports metadata.

**Prompt template:**

```text
⚠️ Aprovação necessária — Workit/M365

Operação: {operation}
Conta: {account_label}
Destinatários/recursos: {targets}
Assunto/título: {title}
Efeito: {side_effects}
Escopos: {required_scopes}
Validade: {ttl_minutes} min
ID: {short_approval_id}

Aprovar ou negar?
```

**Tests:**

- metadata appears in rendered popup;
- raw tokens/body contents do not appear;
- TTL defaults to 30 minutes;
- oversized recipient lists are summarized with count and hash.

---

## Task 9: Add audit ledger

**Objective:** Every write attempt creates a redacted audit entry, even if denied.

**Files:**
- Create: `/home/genie/workspace/agents/khaw/plugins/khaw/workit_audit.py`
- Tests: extend `/home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py` or create `test_khaw_workit_audit.py`

**Audit fields:**

- `approval_id`
- `correlation_id`
- `platform`
- `chat_id` / thread id if safe
- `operation_class`
- `operation`
- `resource`
- `target_count`
- `payload_hash`
- `required_scopes`
- `status`
- `created_at`, `responded_at`, `executed_at`
- `ttl_seconds`
- redacted error

**Verification:**

- No raw email body.
- No tokens.
- No attachment contents.
- Denied and harness-unavailable attempts are logged.

---

## Task 10: Add Workit M365 scope guard

**Objective:** Prevent pilot auth from requesting write scopes.

**Files:**
- Workit M365 auth package once created, likely `/home/genie/prod/workit/internal/msauth/scopes.go`
- Tests beside it.

**Pilot allowed scopes:**

```text
User.Read
Mail.Read
Calendars.Read
```

**Denied in pilot:**

```text
Mail.Send
Calendars.ReadWrite
Chat.ReadWrite
ChannelMessage.Send
Sites.ReadWrite.All
Files.ReadWrite.All
```

**Test command after Go exists:**

```bash
go test ./internal/msauth
```

Current blocker: Go is not installed on this VM.

---

## Global verification gates

Before declaring implementation complete:

```bash
python -m pytest /home/genie/workspace/agents/khaw/tests/contract/test_khaw_workit_policy.py -q -o 'addopts='
python -m pytest /home/genie/workspace/agents/khaw/tests/contract -q -o 'addopts='
```

When Go is available:

```bash
cd /home/genie/prod/workit
go test ./...
```

Manual gateway smoke after code is installed and gateway restarted:

1. Trigger a read-only M365 command. Expected: executes without popup.
2. Trigger a write command in pilot mode. Expected: denied as `WRITE_DISABLED_FOR_PILOT`; no popup; no execution.
3. Temporarily enable writes in a fake/non-production Graph mock. Expected: Telegram approval popup appears.
4. Deny. Expected: no execution.
5. Approve. Expected: exact payload executes once.
6. Modify payload after approval. Expected: hash mismatch denial.
7. Stop approval harness. Expected: fail-closed denial.

---

## Commit boundaries

Recommended commits:

1. `test: add khaw workit policy classification contract`
2. `feat: add khaw workit read/write classifier`
3. `feat: add workit approval payload hashing`
4. `feat: add fail-closed workit execution wrapper`
5. `feat: add workit pilot write-disabled gate`
6. `feat: add hermes structured approval adapter`
7. `feat: render workit m365 approval prompts`
8. `feat: audit workit approval attempts`
9. `feat: guard m365 pilot scopes`

Do not mix unrelated KHAW dirty state (`scripts/install_local_harness.py`) into these commits.
