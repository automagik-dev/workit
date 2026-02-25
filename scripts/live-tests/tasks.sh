#!/usr/bin/env bash

set -euo pipefail

run_tasks_tests() {
  if skip "tasks"; then
    echo "==> tasks (skipped)"
    return 0
  fi

  run_required "tasks" "tasks lists list" wk tasks lists list --json --max 1 >/dev/null

  local created_list_json created_list_id list_json list_id
  echo "==> tasks lists create"
  created_list_json=$(wk tasks lists create "workit-smoke-$TS" --json)
  created_list_id=$(extract_id "$created_list_json")
  [ -n "$created_list_id" ] || { echo "Failed to parse task list id" >&2; exit 1; }
  list_id="$created_list_id"

  run_required "tasks" "tasks list" wk tasks list "$list_id" --json --max 1 >/dev/null

  local task_json task_id
  task_json=$(wk tasks add "$list_id" --title "workit-smoke-$TS" --due "$DAY1" --json)
  task_id=$(extract_id "$task_json")
  [ -n "$task_id" ] || { echo "Failed to parse task id" >&2; exit 1; }

  run_required "tasks" "tasks get" wk tasks get "$list_id" "$task_id" --json >/dev/null
  run_required "tasks" "tasks update" wk tasks update "$list_id" "$task_id" --title "workit-smoke-updated-$TS" --json >/dev/null
  run_required "tasks" "tasks done" wk tasks done "$list_id" "$task_id" --json >/dev/null
  run_required "tasks" "tasks undo" wk tasks undo "$list_id" "$task_id" --json >/dev/null
  run_required "tasks" "tasks delete" wk tasks delete "$list_id" "$task_id" --force >/dev/null

  local repeat_json repeat_ids
  repeat_json=$(wk tasks add "$list_id" --title "workit-smoke-repeat-$TS" --due "$DAY1" --repeat daily --repeat-count 2 --json)
  repeat_ids=$(extract_task_ids "$repeat_json")
  [ -n "$repeat_ids" ] || { echo "Failed to parse repeat task ids" >&2; exit 1; }
  while IFS= read -r tid; do
    [ -n "$tid" ] && run_required "tasks" "tasks delete repeat" wk tasks delete "$list_id" "$tid" --force >/dev/null
  done <<<"$repeat_ids"

  local done_json done_id
  done_json=$(wk tasks add "$list_id" --title "workit-smoke-done-$TS" --due "$DAY1" --json)
  done_id=$(extract_id "$done_json")
  [ -n "$done_id" ] || { echo "Failed to parse done task id" >&2; exit 1; }
  run_required "tasks" "tasks done (for clear)" wk tasks done "$list_id" "$done_id" --json >/dev/null
  run_required "tasks" "tasks clear" wk --force tasks clear "$list_id" --json >/dev/null
}
