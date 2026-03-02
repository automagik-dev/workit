#!/usr/bin/env bash
# workit plugin SessionStart hook — verify wk binary is available
#
# Runs at session start. Must always exit 0.

BIN_DIR="${WK_BIN_DIR:-$HOME/.local/bin}"

find_wk() {
  local p
  p="$(command -v wk 2>/dev/null)" && echo "$p" && return 0
  [ -x "$BIN_DIR/wk" ] && echo "$BIN_DIR/wk" && return 0
  return 1
}

WK="$(find_wk)" || true

if [ -z "$WK" ]; then
  echo "[workit] wk not installed — run: curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh" >&2
  exit 0
fi

VER="$("$WK" --version 2>/dev/null | head -n 1)"
[ -n "$VER" ] || VER="unknown"
echo "[workit] ${VER}" >&2
exit 0
