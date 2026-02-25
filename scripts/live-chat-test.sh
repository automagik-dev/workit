#!/usr/bin/env bash
set -euo pipefail

ACCOUNT=""
ALLOW_NONTEST=false
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

usage() {
  cat <<'USAGE'
Usage: scripts/live-chat-test.sh [options]

Options:
  --account <email>   Account to use (defaults to WK_IT_ACCOUNT or first auth)
  --allow-nontest     Allow running against non-test accounts
  -h, --help          Show this help

Env:
  WK_LIVE_CHAT_SPACE=spaces/<id>        Existing space to use for list/send
  WK_LIVE_CHAT_THREAD=<id|resource>    Thread id or resource for sends
  WK_LIVE_CHAT_DM=user@domain          DM target (workspace user)
  WK_LIVE_CHAT_DM_THREAD=<id|resource> Thread id for DM send
  WK_LIVE_CHAT_CREATE=1                Create a new space (no cleanup)
  WK_LIVE_CHAT_MEMBER=user@domain      Member to add when creating a space
  WK_LIVE_ALLOW_NONTEST=1              Allow non-test accounts
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --account)
      ACCOUNT="$2"
      shift
      ;;
    --allow-nontest)
      ALLOW_NONTEST=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

BIN="${WK_BIN:-$ROOT_DIR/bin/wk}"
if [ ! -x "$BIN" ]; then
  make -C "$ROOT_DIR" build >/dev/null
fi

PY="${PYTHON:-python3}"
if ! command -v "$PY" >/dev/null 2>&1; then
  PY="python"
fi

if [ -z "$ACCOUNT" ]; then
  ACCOUNT="${WK_IT_ACCOUNT:-}"
fi
if [ -z "$ACCOUNT" ]; then
  acct_json=$($BIN auth list --json)
  ACCOUNT=$($PY -c 'import json,sys; obj=json.load(sys.stdin); print(obj.get("accounts", [{}])[0].get("email", ""))' <<<"$acct_json")
fi
if [ -z "$ACCOUNT" ]; then
  echo "No account available for live tests." >&2
  exit 1
fi

is_test_account() {
  local a
  a=$(echo "$1" | tr 'A-Z' 'a-z')
  case "$a" in
    *test*|*bot*|*sandbox*|*qa*|*staging*|*dev*|*@example.com)
      return 0
      ;;
  esac
  case "$a" in
    *+*)
      return 0
      ;;
  esac
  return 1
}

is_consumer_account() {
  local a domain
  a=$(echo "$1" | tr 'A-Z' 'a-z')
  domain="${a##*@}"
  case "$domain" in
    gmail.com|googlemail.com)
      return 0
      ;;
  esac
  return 1
}

if [ "${ALLOW_NONTEST:-false}" = false ] && [ -z "${WK_LIVE_ALLOW_NONTEST:-}" ]; then
  if ! is_test_account "$ACCOUNT"; then
    echo "Refusing to run live tests against non-test account: $ACCOUNT" >&2
    echo "Pass --allow-nontest or set WK_LIVE_ALLOW_NONTEST=1 to override." >&2
    exit 2
  fi
fi

if is_consumer_account "$ACCOUNT"; then
  echo "==> chat (skipped; Workspace only)"
  exit 0
fi

wk() {
  "$BIN" --account "$ACCOUNT" "$@"
}

TS=$(date +%Y%m%d%H%M%S)

echo "Using account: $ACCOUNT"
echo "==> chat spaces list"
wk chat spaces list --json --max 1 >/dev/null

if [ -n "${WK_LIVE_CHAT_SPACE:-}" ]; then
  echo "==> chat messages list"
  wk chat messages list "$WK_LIVE_CHAT_SPACE" --json --max 1 >/dev/null
  echo "==> chat threads list"
  wk chat threads list "$WK_LIVE_CHAT_SPACE" --json --max 1 >/dev/null
  echo "==> chat messages send"
  if [ -n "${WK_LIVE_CHAT_THREAD:-}" ]; then
    wk chat messages send "$WK_LIVE_CHAT_SPACE" --text "workit smoke $TS" --thread "$WK_LIVE_CHAT_THREAD" --json >/dev/null
  else
    wk chat messages send "$WK_LIVE_CHAT_SPACE" --text "workit smoke $TS" --json >/dev/null
  fi
else
  echo "==> chat messages/threads (skipped; set WK_LIVE_CHAT_SPACE)"
fi

if [ -n "${WK_LIVE_CHAT_CREATE:-}" ]; then
  if [ -z "${WK_LIVE_CHAT_MEMBER:-}" ]; then
    echo "==> chat spaces create (skipped; set WK_LIVE_CHAT_MEMBER)"
  else
    echo "==> chat spaces create"
    wk chat spaces create "workit-smoke-$TS" --member "$WK_LIVE_CHAT_MEMBER" --json >/dev/null
  fi
fi

if [ -n "${WK_LIVE_CHAT_DM:-}" ]; then
  echo "==> chat dm space"
  wk chat dm space "$WK_LIVE_CHAT_DM" --json >/dev/null
  echo "==> chat dm send"
  if [ -n "${WK_LIVE_CHAT_DM_THREAD:-}" ]; then
    wk chat dm send "$WK_LIVE_CHAT_DM" --text "workit dm $TS" --thread "$WK_LIVE_CHAT_DM_THREAD" --json >/dev/null
  else
    wk chat dm send "$WK_LIVE_CHAT_DM" --text "workit dm $TS" --json >/dev/null
  fi
else
  echo "==> chat dm (skipped; set WK_LIVE_CHAT_DM)"
fi

echo "Chat live tests complete."
