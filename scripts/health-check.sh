#!/usr/bin/env bash
# workit plugin SessionStart hook — verify wk binary is available
#
# Runs at session start. Must always exit 0.

find_wk() {
  local p
  p="$(command -v wk 2>/dev/null)" && echo "$p" && return 0
  [ -x "$HOME/.local/bin/wk" ] && echo "$HOME/.local/bin/wk" && return 0
  return 1
}

WK="$(find_wk)" || true

if [ -z "$WK" ]; then
  echo "[workit] wk not installed — run: curl -fsSL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/install.sh | bash" >&2
  exit 0
fi

VER="$("$WK" --version 2>/dev/null | head -1)" || VER="unknown"
echo "[workit] ${VER}" >&2
exit 0
