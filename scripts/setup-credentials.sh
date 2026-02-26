#!/bin/bash
# workit Setup Script
# Configures OAuth credentials for workit on any server
#
# Usage:
#   ./scripts/setup-credentials.sh
#   ./scripts/setup-credentials.sh --non-interactive --env-file .env --force
#   curl -sL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/setup-credentials.sh | bash -s -- --non-interactive --force
#   OR
#   curl -sL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/setup-credentials.sh | bash

set -euo pipefail

usage() {
	cat <<'EOF'
workit credentials setup

Writes a local credentials file (default: ~/.config/workit/credentials.env) suitable for:
- `source ~/.config/workit/credentials.env`
- systemd `EnvironmentFile=...`

Options:
  --config-dir DIR        Config dir (default: $WK_CONFIG_DIR or ~/.config/workit)
  --env-file PATH         Optional .env to source (default: ./.env if present)
  --non-interactive       Do not prompt; fail if values are missing
  --force                 Overwrite existing credentials.env without prompting
  --keyring-backend NAME  Keyring backend (default: file)
  --callback-server URL   Callback server (default: $WK_CALLBACK_SERVER or https://auth.example.com)
  --client-id VALUE       OAuth client id (discouraged: use env/.env instead)
  --client-secret VALUE   OAuth client secret (discouraged: use env/.env instead)
  -h, --help              Show help

Inputs (preferred):
  WK_CLIENT_ID, WK_CLIENT_SECRET, WK_CALLBACK_SERVER

Security note:
  Do not commit secrets. Keep them in a local .env (gitignored) or injected by your runtime.
EOF
}

CONFIG_DIR="${WK_CONFIG_DIR:-$HOME/.config/workit}"
ENV_FILE=""
NON_INTERACTIVE="0"
FORCE="0"
KEYRING_BACKEND="${WK_KEYRING_BACKEND:-file}"
CLIENT_ID="${WK_CLIENT_ID:-}"
CLIENT_SECRET="${WK_CLIENT_SECRET:-}"
CALLBACK_SERVER="${WK_CALLBACK_SERVER:-https://auth.example.com}"

while [[ $# -gt 0 ]]; do
	case "$1" in
	-h|--help)
		usage
		exit 0
		;;
	--config-dir)
		CONFIG_DIR="${2:-}"; shift 2
		;;
	--env-file)
		ENV_FILE="${2:-}"; shift 2
		;;
	--non-interactive)
		NON_INTERACTIVE="1"; shift
		;;
	--force)
		FORCE="1"; shift
		;;
	--keyring-backend)
		KEYRING_BACKEND="${2:-}"; shift 2
		;;
	--callback-server)
		CALLBACK_SERVER="${2:-}"; shift 2
		;;
	--client-id)
		CLIENT_ID="${2:-}"; shift 2
		;;
	--client-secret)
		CLIENT_SECRET="${2:-}"; shift 2
		;;
	*)
		echo "ERROR: unknown argument: $1" >&2
		echo "" >&2
		usage >&2
		exit 2
		;;
	esac
done

CRED_FILE="$CONFIG_DIR/credentials.env"

echo "🔧 workit Credentials Setup"
echo ""

env_quote() {
	# Single-quote for a shell-compatible env file.
	local s="${1:-}"
	s="${s//\'/\'\\\'\'}"
	printf "'%s'" "$s"
}

load_env_file() {
	local f="${1:-}"
	[[ -z "$f" ]] && return 0
	if [[ ! -f "$f" ]]; then
		echo "ERROR: env file not found: $f" >&2
		exit 1
	fi
	set -a
	# shellcheck disable=SC1091
	. "$f"
	set +a
}

# If no env file explicitly provided, try CWD .env.
if [[ -z "$ENV_FILE" && -f ".env" ]]; then
	ENV_FILE=".env"
fi

# Load env file (if present). This updates WK_* env vars; refresh defaults after.
load_env_file "$ENV_FILE"

CLIENT_ID="${CLIENT_ID:-${WK_CLIENT_ID:-}}"
CLIENT_SECRET="${CLIENT_SECRET:-${WK_CLIENT_SECRET:-}}"
CALLBACK_SERVER="${CALLBACK_SERVER:-${WK_CALLBACK_SERVER:-https://auth.example.com}}"
KEYRING_BACKEND="${KEYRING_BACKEND:-${WK_KEYRING_BACKEND:-file}}"

# Create config directory
mkdir -p "$CONFIG_DIR"
chmod 700 "$CONFIG_DIR"

# Check if credentials already exist
if [[ -f "$CRED_FILE" && "$FORCE" != "1" ]]; then
	echo "⚠️  Credentials file already exists: $CRED_FILE"
	read -r -p "Overwrite? [y/N] " -n 1
	echo
	if [[ ! "${REPLY:-}" =~ ^[Yy]$ ]]; then
		echo "Aborted."
		exit 0
	fi
fi

if [[ "$NON_INTERACTIVE" != "1" ]]; then
	# Prompt for credentials (defaults come from env/.env, not hardcoded).
	echo "Enter OAuth credentials (press Enter to use current environment values):"
	echo ""

	read -r -p "Client ID [${CLIENT_ID:-}]: " CLIENT_ID_IN
	if [[ -n "${CLIENT_ID_IN:-}" ]]; then
		CLIENT_ID="$CLIENT_ID_IN"
	fi

	secret_hint=""
	if [[ -n "${CLIENT_SECRET:-}" ]]; then
		secret_hint="<set>"
	fi
	read -r -s -p "Client Secret [${secret_hint}]: " CLIENT_SECRET_IN
	echo
	if [[ -n "${CLIENT_SECRET_IN:-}" ]]; then
		CLIENT_SECRET="$CLIENT_SECRET_IN"
	fi

	read -r -p "Callback Server [${CALLBACK_SERVER:-https://auth.example.com}]: " CALLBACK_SERVER_IN
	if [[ -n "${CALLBACK_SERVER_IN:-}" ]]; then
		CALLBACK_SERVER="$CALLBACK_SERVER_IN"
	fi
fi

if [[ -z "${CLIENT_ID:-}" || -z "${CLIENT_SECRET:-}" ]]; then
	echo "ERROR: Missing OAuth client credentials." >&2
	echo "Set WK_CLIENT_ID and WK_CLIENT_SECRET in the environment (or in a local .env file)." >&2
	exit 1
fi

# Write credentials file
umask 077
cat > "$CRED_FILE" << EOF
# workit OAuth Credentials
# Generated: $(date -Iseconds)
# Source this file or use with systemd EnvironmentFile=

WK_CLIENT_ID=$(env_quote "$CLIENT_ID")
WK_CLIENT_SECRET=$(env_quote "$CLIENT_SECRET")
WK_CALLBACK_SERVER=$(env_quote "$CALLBACK_SERVER")
WK_KEYRING_BACKEND=$(env_quote "$KEYRING_BACKEND")
EOF

chmod 600 "$CRED_FILE"

echo ""
echo "✅ Credentials saved to: $CRED_FILE"
echo ""
echo "To use:"
echo "  source $CRED_FILE"
echo "  wk auth add you@gmail.com --headless"
echo ""
echo "For systemd services, add to your unit file:"
echo "  EnvironmentFile=$CRED_FILE"
