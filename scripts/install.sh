#!/bin/bash
# workit Full Server Setup
# Installs workit with Namastex OAuth configuration
#
# Usage:
#   curl -sL https://raw.githubusercontent.com/automagik-genie/workit/main/scripts/install.sh | bash

set -e

REPO_URL="https://github.com/automagik-genie/workit.git"
INSTALL_DIR="${WK_INSTALL_DIR:-$HOME/workit}"
CONFIG_DIR="${WK_CONFIG_DIR:-$HOME/.config/workit}"
BIN_DIR="${HOME}/.local/bin"

echo "🚀 workit Installation Script"
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
    echo "📦 Cloning workit..."
    git clone "$REPO_URL" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Create config directory
mkdir -p "$CONFIG_DIR"
chmod 700 "$CONFIG_DIR"

# Set up credentials via setup-credentials.sh (reads from env or prompts).
CRED_FILE="$CONFIG_DIR/credentials.env"
if [ ! -f "$CRED_FILE" ]; then
    echo "🔐 Setting up credentials..."
    bash "$INSTALL_DIR/scripts/setup-credentials.sh" --config-dir "$CONFIG_DIR"
else
    echo "   ℹ️  Credentials already exist: $CRED_FILE"
fi

# Build with embedded credentials
echo ""
echo "🔨 Building workit with embedded credentials..."
make build-namastex

# Install binary
mkdir -p "$BIN_DIR"
cp bin/wk "$BIN_DIR/wk"
chmod +x "$BIN_DIR/wk"

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
echo "  wk auth add you@gmail.com --headless"
echo ""
echo "  # List Gmail labels"
echo "  wk gmail labels list"
echo ""
echo "  # List Drive files"  
echo "  wk drive list"
echo ""
echo "  # Start sync"
echo "  wk sync init ~/GoogleDrive"
echo "  wk sync start"
