#!/bin/bash
# gog-cli Full Server Setup
# Installs gog-cli with Namastex OAuth configuration
#
# Usage:
#   curl -sL https://raw.githubusercontent.com/automagik-genie/gog-cli/main/scripts/install.sh | bash

set -e

REPO_URL="https://github.com/automagik-genie/gog-cli.git"
INSTALL_DIR="${GOG_INSTALL_DIR:-$HOME/gog-cli}"
CONFIG_DIR="${GOG_CONFIG_DIR:-$HOME/.config/gog}"
BIN_DIR="${HOME}/.local/bin"

echo "🚀 gog-cli Installation Script"
echo "================================"
echo ""

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    echo "   See: https://go.dev/doc/install"
    exit 1
fi

# Clone or update repo
if [ -d "$INSTALL_DIR" ]; then
    echo "📦 Updating existing installation..."
    cd "$INSTALL_DIR"
    git pull --ff-only
else
    echo "📦 Cloning gog-cli..."
    git clone "$REPO_URL" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Create config directory
mkdir -p "$CONFIG_DIR"
chmod 700 "$CONFIG_DIR"

# Create credentials file with Namastex defaults
CRED_FILE="$CONFIG_DIR/credentials.env"
if [ ! -f "$CRED_FILE" ]; then
    echo "🔐 Creating credentials file..."
    cat > "$CRED_FILE" << 'EOF'
# gog-cli OAuth Credentials (Namastex)
GOG_CLIENT_ID=151804783833-b818q8mtv5tmc2i640cg4h6uq3nm6uj2.apps.googleusercontent.com
GOG_CLIENT_SECRET=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
GOG_CALLBACK_SERVER=https://gogoauth.namastex.io
GOG_KEYRING_BACKEND=file
EOF
    chmod 600 "$CRED_FILE"
    echo "   ✅ Credentials saved to: $CRED_FILE"
else
    echo "   ℹ️  Credentials already exist: $CRED_FILE"
fi

# Build with embedded credentials
echo ""
echo "🔨 Building gog-cli with embedded credentials..."
make build-namastex

# Install binary
mkdir -p "$BIN_DIR"
cp bin/gog "$BIN_DIR/gog"
chmod +x "$BIN_DIR/gog"

# Add to PATH if needed
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo ""
    echo "📝 Add to your shell profile:"
    echo "   export PATH=\"\$PATH:$BIN_DIR\""
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "Quick start:"
echo "  # Add account (opens browser for OAuth)"
echo "  gog auth add you@gmail.com --headless"
echo ""
echo "  # List Gmail labels"
echo "  gog gmail labels list"
echo ""
echo "  # List Drive files"  
echo "  gog drive list"
echo ""
echo "  # Start sync"
echo "  gog sync init ~/GoogleDrive"
echo "  gog sync start"
