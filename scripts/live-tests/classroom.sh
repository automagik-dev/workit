#!/usr/bin/env bash

set -euo pipefail

run_classroom_tests() {
  if skip "classroom"; then
    echo "==> classroom (skipped)"
    return 0
  fi

  run_optional "classroom" "classroom profile get" wk classroom profile get --json >/dev/null
  run_optional "classroom" "classroom courses list" wk classroom courses list --json --max 1 >/dev/null

  if [ -n "${WK_LIVE_CLASSROOM_COURSE:-}" ]; then
    local course_id cw_json cw_id
    course_id="$WK_LIVE_CLASSROOM_COURSE"
    run_optional "classroom" "classroom courses get" wk classroom courses get "$course_id" --json >/dev/null
    run_optional "classroom" "classroom courses url" wk classroom courses url "$course_id" --json >/dev/null
    run_optional "classroom" "classroom roster" wk classroom roster "$course_id" --students --teachers --max 1 --json >/dev/null
    run_optional "classroom" "classroom students list" wk classroom students "$course_id" --max 1 --json >/dev/null
    run_optional "classroom" "classroom teachers list" wk classroom teachers "$course_id" --max 1 --json >/dev/null
    run_optional "classroom" "classroom coursework list" wk classroom coursework "$course_id" --max 1 --json >/dev/null
    run_optional "classroom" "classroom materials list" wk classroom materials "$course_id" --max 1 --json >/dev/null
    run_optional "classroom" "classroom announcements list" wk classroom announcements "$course_id" --max 1 --json >/dev/null
    run_optional "classroom" "classroom topics list" wk classroom topics "$course_id" --max 1 --json >/dev/null

    cw_json=$(wk classroom coursework "$course_id" --max 1 --json 2>/dev/null || true)
    cw_id=$(extract_id "$cw_json")
    if [ -n "$cw_id" ]; then
      run_optional "classroom" "classroom submissions list" wk classroom submissions "$course_id" "$cw_id" --max 1 --json >/dev/null
    fi
  else
    if [ "${STRICT:-false}" = true ]; then
      echo "Missing WK_LIVE_CLASSROOM_COURSE for classroom coverage." >&2
      return 1
    fi
    echo "==> classroom (optional; set WK_LIVE_CLASSROOM_COURSE to expand)"
  fi

  # Disabled by default: creator account lacks course state permissions.
  if [ -n "${WK_LIVE_CLASSROOM_CREATE:-}" ] && [ -n "${WK_LIVE_CLASSROOM_ALLOW_STATE:-}" ]; then
    local course_json course_id topic_json topic_id announcement_json announcement_id material_json material_id coursework_json coursework_id

    echo "==> classroom courses create"
    if course_json=$(wk classroom courses create --name "workit-smoke-$TS" --section "workit" --state ACTIVE --json 2>/dev/null); then
      :
    elif course_json=$(wk classroom courses create --name "workit-smoke-$TS" --section "workit" --state PROVISIONED --json 2>/dev/null); then
      :
    else
      course_json=""
    fi
    course_id=$(extract_id "$course_json")
    if [ -z "$course_id" ]; then
      echo "Classroom course create failed; skipping create tests."
      if [ "${STRICT:-false}" = true ]; then
        return 1
      fi
      return 0
    fi

    run_optional "classroom" "classroom courses update" wk classroom courses update "$course_id" --name "workit-smoke-updated-$TS" --json >/dev/null
    run_optional "classroom" "classroom courses archive" wk classroom courses archive "$course_id" --json >/dev/null
    run_optional "classroom" "classroom courses unarchive" wk classroom courses unarchive "$course_id" --json >/dev/null

    echo "==> classroom topics create"
    topic_json=$(wk classroom topics create "$course_id" --name "workit topic $TS" --json 2>/dev/null || true)
    topic_id=$(extract_id "$topic_json")

    echo "==> classroom announcements create"
    announcement_json=$(wk classroom announcements create "$course_id" --text "workit announcement $TS" --json 2>/dev/null || true)
    announcement_id=$(extract_id "$announcement_json")

    echo "==> classroom materials create"
    material_json=$(wk classroom materials create "$course_id" --title "workit material $TS" --json 2>/dev/null || true)
    material_id=$(extract_id "$material_json")

    echo "==> classroom coursework create"
    coursework_json=$(wk classroom coursework create "$course_id" --title "workit coursework $TS" --type ASSIGNMENT --max-points 10 --json 2>/dev/null || true)
    coursework_id=$(extract_id "$coursework_json")

    if [ -n "$announcement_id" ]; then
      run_optional "classroom" "classroom announcements update" wk classroom announcements update "$course_id" "$announcement_id" --text "workit announcement updated $TS" --json >/dev/null
      run_optional "classroom" "classroom announcements delete" wk --force classroom announcements delete "$course_id" "$announcement_id" --json >/dev/null
    fi
    if [ -n "$material_id" ]; then
      run_optional "classroom" "classroom materials update" wk classroom materials update "$course_id" "$material_id" --title "workit material updated $TS" --json >/dev/null
      run_optional "classroom" "classroom materials delete" wk --force classroom materials delete "$course_id" "$material_id" --json >/dev/null
    fi
    if [ -n "$coursework_id" ]; then
      run_optional "classroom" "classroom coursework update" wk classroom coursework update "$course_id" "$coursework_id" --title "workit coursework updated $TS" --json >/dev/null
      run_optional "classroom" "classroom coursework delete" wk --force classroom coursework delete "$course_id" "$coursework_id" --json >/dev/null
    fi
    if [ -n "$topic_id" ]; then
      run_optional "classroom" "classroom topics update" wk classroom topics update "$course_id" "$topic_id" --name "workit topic updated $TS" --json >/dev/null
      run_optional "classroom" "classroom topics delete" wk --force classroom topics delete "$course_id" "$topic_id" --json >/dev/null
    fi

    if wk --force classroom courses delete "$course_id" --json >/dev/null; then
      :
    else
      echo "Classroom course delete failed; manual cleanup needed: $course_id" >&2
      if [ "${STRICT:-false}" = true ]; then
        return 1
      fi
    fi
  elif [ -n "${WK_LIVE_CLASSROOM_CREATE:-}" ]; then
    echo "==> classroom create (skipped; no account with course state permissions)"
  fi
}
