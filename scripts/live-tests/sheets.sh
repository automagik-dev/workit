#!/usr/bin/env bash

set -euo pipefail

run_sheets_tests() {
  if skip "sheets"; then
    echo "==> sheets (skipped)"
    return 0
  fi

  local sheet_json sheet_id copy_json copy_id export_path
  sheet_json=$(wk sheets create "workit-smoke-sheet-$TS" --json)
  sheet_id=$(extract_id "$sheet_json")
  [ -n "$sheet_id" ] || { echo "Failed to parse sheet id" >&2; exit 1; }

  run_required "sheets" "sheets metadata" wk sheets metadata "$sheet_id" --json >/dev/null
  run_required "sheets" "sheets update" wk sheets update "$sheet_id" "Sheet1!A1:B2" --values-json '[["A1","B1"],["A2","B2"]]' --json >/dev/null
  run_required "sheets" "sheets get" wk sheets get "$sheet_id" "Sheet1!A1:B2" --json >/dev/null
  run_required "sheets" "sheets append" wk sheets append "$sheet_id" "Sheet1!A3:B3" --values-json '[["A3","B3"]]' --json >/dev/null
  run_required "sheets" "sheets format" wk sheets format "$sheet_id" "Sheet1!A1:B1" --format-json '{"textFormat":{"bold":true}}' --format-fields textFormat.bold --json >/dev/null
  run_required "sheets" "sheets clear" wk sheets clear "$sheet_id" "Sheet1!A1:B3" --json >/dev/null

  export_path="$LIVE_TMP/sheets-export-$TS.xlsx"
  run_required "sheets" "sheets export" wk sheets export "$sheet_id" --format xlsx --out "$export_path" >/dev/null

  copy_json=$(wk sheets copy "$sheet_id" "workit-smoke-sheet-copy-$TS" --json)
  copy_id=$(extract_id "$copy_json")
  [ -n "$copy_id" ] || { echo "Failed to parse sheet copy id" >&2; exit 1; }

  run_required "sheets" "drive delete sheet copy" wk drive delete "$copy_id" --force >/dev/null
  run_required "sheets" "drive delete sheet" wk drive delete "$sheet_id" --force >/dev/null
}
