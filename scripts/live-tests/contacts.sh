#!/usr/bin/env bash

set -euo pipefail

run_contacts_tests() {
  if skip "contacts"; then
    echo "==> contacts (skipped)"
    return 0
  fi

  run_required "contacts" "contacts list" wk contacts list --json --max 1 >/dev/null

  local contact_json contact_id
  contact_json=$(wk contacts create --given "workit" --family "smoke-$TS" --email "workit-smoke-$TS@example.com" --phone "+1555555$TS" --json)
  contact_id=$(extract_field "$contact_json" resourceName)
  [ -n "$contact_id" ] || { echo "Failed to parse contact resourceName" >&2; exit 1; }

  run_required "contacts" "contacts get" wk contacts get "$contact_id" --json >/dev/null
  run_required "contacts" "contacts update" wk contacts update "$contact_id" --given "workit" --family "smoke-updated-$TS" --email "workit-smoke-$TS@example.com" --json >/dev/null
  run_required "contacts" "contacts search" wk contacts search "workit-smoke-$TS@example.com" --json --max 1 >/dev/null
  run_required "contacts" "contacts delete" wk contacts delete "$contact_id" --force >/dev/null

  if is_consumer_account "$ACCOUNT"; then
    echo "==> contacts directory (skipped; Workspace only)"
    echo "==> contacts other (skipped; Workspace only)"
  else
    run_optional "contacts-directory" "contacts directory list" wk contacts directory list --json --max 1 >/dev/null
    run_optional "contacts-directory" "contacts directory search" wk contacts directory search "workit" --json --max 1 >/dev/null
    run_optional "contacts-other" "contacts other list" wk contacts other list --json --max 1 >/dev/null
    run_optional "contacts-other" "contacts other search" wk contacts other search "workit" --json --max 1 >/dev/null
  fi
}
