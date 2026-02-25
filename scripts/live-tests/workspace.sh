#!/usr/bin/env bash

set -euo pipefail

run_workspace_tests() {
  if is_consumer_account "$ACCOUNT"; then
    echo "==> groups (skipped; Workspace only)"
  else
    run_optional "groups" "groups list" wk groups list --json --max 5 >/dev/null
    if [ -n "${WK_LIVE_GROUP_EMAIL:-}" ]; then
      run_optional "groups" "groups members" wk groups members "$WK_LIVE_GROUP_EMAIL" --json --max 5 >/dev/null
    fi
  fi

  if skip "keep"; then
    echo "==> keep (skipped)"
    return 0
  fi

  if is_consumer_account "$ACCOUNT"; then
    echo "==> keep (skipped; Workspace only)"
    return 0
  fi

  if [ -z "${WK_KEEP_SERVICE_ACCOUNT:-}" ] || [ -z "${WK_KEEP_IMPERSONATE:-}" ]; then
    if [ "${STRICT:-false}" = true ]; then
      echo "Missing WK_KEEP_SERVICE_ACCOUNT/WK_KEEP_IMPERSONATE for keep tests." >&2
      return 1
    fi
    echo "==> keep (optional; set WK_KEEP_SERVICE_ACCOUNT and WK_KEEP_IMPERSONATE)"
    return 0
  fi

  local notes_json note_name note_json attachment_name attachment_out
  echo "==> keep list (optional)"
  if notes_json=$(wk keep list --service-account "$WK_KEEP_SERVICE_ACCOUNT" --impersonate "$WK_KEEP_IMPERSONATE" --json --max 5); then
    echo "ok"
  else
    echo "skipped/failed"
    if [ "${STRICT:-false}" = true ]; then
      return 1
    fi
    return 0
  fi

  note_name=$(extract_keep_note_name "$notes_json")
  if [ -n "$note_name" ]; then
    echo "==> keep get (optional)"
    if note_json=$(wk keep get "$note_name" --service-account "$WK_KEEP_SERVICE_ACCOUNT" --impersonate "$WK_KEEP_IMPERSONATE" --json); then
      echo "ok"
    else
      echo "skipped/failed"
      if [ "${STRICT:-false}" = true ]; then
        return 1
      fi
      note_json=""
    fi
  else
    echo "==> keep get (skipped; no notes)"
    note_json=""
  fi

  run_optional "keep" "keep search" wk keep search "workit" --service-account "$WK_KEEP_SERVICE_ACCOUNT" --impersonate "$WK_KEEP_IMPERSONATE" --json >/dev/null

  if [ -n "$note_json" ]; then
    attachment_name=$(extract_keep_attachment_name "$note_json")
    if [ -n "$attachment_name" ]; then
      attachment_out="$LIVE_TMP/keep-attachment-$TS"
      run_optional "keep" "keep attachment" wk keep attachment "$attachment_name" --service-account "$WK_KEEP_SERVICE_ACCOUNT" --impersonate "$WK_KEEP_IMPERSONATE" --out "$attachment_out" >/dev/null
    else
      echo "==> keep attachment (skipped; no attachments)"
    fi
  else
    echo "==> keep attachment (skipped; no note)"
  fi
}
