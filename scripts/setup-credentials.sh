#!/bin/bash
# gog-cli Setup Script
# Configures OAuth credentials for gog-cli on any server
#
# Usage:
#   curl -sL https://raw.githubusercontent.com/automagik-genie/gog-cli/main/scripts/setup-credentials.sh | bash
#   OR
#   ./scripts/setup-credentials.sh

set -e

CONFIG_DIR="${GOG_CONFIG_DIR:-$HOME/.config/gog}"
CRED_FILE="$CONFIG_DIR/credentials.env"

echo "🔧 gog-cli Credentials Setup"
echo ""

# Create config directory
mkdir -p "$CONFIG_DIR"
chmod 700 "$CONFIG_DIR"

# Check if credentials already exist
if [ -f "$CRED_FILE" ]; then
    echo "⚠️  Credentials file already exists: $CRED_FILE"
    read -p "Overwrite? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi
fi

# Prompt for credentials or use defaults
echo "Enter OAuth credentials (or press Enter for Namastex defaults):"
echo ""

read -p "Client ID [namastex-default]: " CLIENT_ID
if [ -z "$CLIENT_ID" ]; then
    CLIENT_ID="151804783833-b818q8mtv5tmc2i640cg4h6uq3nm6uj2.apps.googleusercontent.com"
fi

read -p "Client Secret [namastex-default]: " CLIENT_SECRET
if [ -z "$CLIENT_SECRET" ]; then
    CLIENT_SECRET="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"
fi

read -p "Callback Server [https://gogoauth.namastex.io]: " CALLBACK_SERVER
if [ -z "$CALLBACK_SERVER" ]; then
    CALLBACK_SERVER="https://gogoauth.namastex.io"
fi

# Write credentials file
cat > "$CRED_FILE" << EOF
# gog-cli OAuth Credentials
# Generated: $(date -Iseconds)
# Source this file or use with systemd EnvironmentFile=

GOG_CLIENT_ID=$CLIENT_ID
GOG_CLIENT_SECRET=$CLIENT_SECRET
GOG_CALLBACK_SERVER=$CALLBACK_SERVER
GOG_KEYRING_BACKEND=file
EOF

chmod 600 "$CRED_FILE"

echo ""
echo "✅ Credentials saved to: $CRED_FILE"
echo ""
echo "To use:"
echo "  source $CRED_FILE"
echo "  gog auth add you@gmail.com --headless"
echo ""
echo "For systemd services, add to your unit file:"
echo "  EnvironmentFile=$CRED_FILE"
