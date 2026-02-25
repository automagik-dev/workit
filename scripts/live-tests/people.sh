#!/usr/bin/env bash

set -euo pipefail

run_people_tests() {
  run_required "people" "people me" wk people me --json >/dev/null
}
