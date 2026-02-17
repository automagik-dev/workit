#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# gog-cli Local Developer Install
#
# Run this AFTER cloning the repo. It builds the binary and optionally
# installs it to your PATH.
#
# Usage:
#   ./install.sh                  # build + install to ~/.local/bin
#   ./install.sh --system         # build + install to /usr/local/bin
#   ./install.sh --no-install     # build only (binary at bin/gog)
#   ./install.sh --force          # overwrite existing install without asking
#   ./install.sh --help           # show usage
# ---------------------------------------------------------------------------

# -- Colors ----------------------------------------------------------------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# -- Helpers ---------------------------------------------------------------

info()  { printf "${BLUE}[INFO]${NC}  %s\n" "$*"; }
ok()    { printf "${GREEN}[OK]${NC}    %s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC}  %s\n" "$*"; }
fail()  { printf "${RED}[FAIL]${NC}  %s\n" "$*"; exit 1; }
step()  { printf "\n${BOLD}--- %s ---${NC}\n" "$*"; }

# -- Defaults --------------------------------------------------------------

INSTALL_DIR="${HOME}/.local/bin"
DO_INSTALL=true
FORCE=false
REQUIRED_GO_MAJOR=1
REQUIRED_GO_MINOR=21

# -- Parse flags -----------------------------------------------------------

for arg in "$@"; do
  case "$arg" in
    --system)
      INSTALL_DIR="/usr/local/bin"
      ;;
    --no-install)
      DO_INSTALL=false
      ;;
    --force)
      FORCE=true
      ;;
    --help|-h)
      echo "Usage: ./install.sh [--system] [--no-install] [--force] [--help]"
      echo ""
      echo "Local developer install script. Run after cloning the repo."
      echo ""
      echo "Options:"
      echo "  --system      Install to /usr/local/bin instead of ~/.local/bin (may need sudo)"
      echo "  --no-install  Build only, do not copy binary to PATH"
      echo "  --force       Overwrite existing installation without asking"
      echo "  --help, -h    Show this help message"
      echo ""
      echo "Examples:"
      echo "  ./install.sh                  # build + install to ~/.local/bin/gog"
      echo "  ./install.sh --system         # build + install to /usr/local/bin/gog"
      echo "  ./install.sh --no-install     # build only (binary at ./bin/gog)"
      exit 0
      ;;
    *)
      fail "Unknown option: $arg (try --help)"
      ;;
  esac
done

# -- Step 1: Verify we are in the repo root --------------------------------

step "Checking repository"

if [ ! -f "Makefile" ]; then
  fail "Makefile not found. Please run this script from the gog-cli repo root."
fi

if [ ! -d "cmd/gog" ]; then
  fail "cmd/gog/ directory not found. Please run this script from the gog-cli repo root."
fi

if [ ! -f "go.mod" ]; then
  fail "go.mod not found. Please run this script from the gog-cli repo root."
fi

ok "Repository root detected (Makefile, cmd/gog/, go.mod present)"

# -- Step 2: Check Go is installed and version is sufficient ---------------

step "Checking Go installation"

if ! command -v go &>/dev/null; then
  fail "Go is not installed. Please install Go ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR}+ first.\n       See: https://go.dev/doc/install"
fi

GO_VERSION_RAW="$(go version)"
# Extract major.minor from "go version go1.25.0 linux/amd64" or similar
GO_VERSION="$(echo "$GO_VERSION_RAW" | sed -n 's/.*go\([0-9]*\.[0-9]*\).*/\1/p')"

if [ -z "$GO_VERSION" ]; then
  warn "Could not parse Go version from: $GO_VERSION_RAW"
  warn "Continuing anyway -- build may fail if Go is too old."
else
  GO_MAJOR="${GO_VERSION%%.*}"
  GO_MINOR="${GO_VERSION##*.}"

  if [ "$GO_MAJOR" -lt "$REQUIRED_GO_MAJOR" ] || \
     { [ "$GO_MAJOR" -eq "$REQUIRED_GO_MAJOR" ] && [ "$GO_MINOR" -lt "$REQUIRED_GO_MINOR" ]; }; then
    fail "Go ${GO_VERSION} found, but >= ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR} is required.\n       Installed: ${GO_VERSION_RAW}\n       See: https://go.dev/doc/install"
  fi

  ok "Go ${GO_VERSION} detected (>= ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR} required)"
fi

info "$(go version)"

# -- Step 3: Install dev tools via Makefile --------------------------------

step "Installing dev tools (gofumpt, goimports, golangci-lint)"

if ! make tools; then
  fail "make tools failed. Check the output above for errors."
fi

ok "Dev tools installed to .tools/"

# -- Step 4: Build the binary via Makefile ---------------------------------

step "Building gog binary"

if ! make build; then
  fail "make build failed. Check the output above for errors."
fi

if [ ! -f "bin/gog" ]; then
  fail "Build appeared to succeed but bin/gog was not created."
fi

ok "Binary built at bin/gog"

# -- Step 5: Verify the binary works ---------------------------------------

step "Verifying binary"

GOG_VERSION_OUTPUT="$(./bin/gog --version 2>&1 || true)"

if [ -z "$GOG_VERSION_OUTPUT" ]; then
  warn "bin/gog --version produced no output (binary may still be functional)"
else
  ok "bin/gog --version: ${GOG_VERSION_OUTPUT}"
fi

# -- Step 6: Optionally install to PATH ------------------------------------

if [ "$DO_INSTALL" = true ]; then
  step "Installing to ${INSTALL_DIR}"

  TARGET="${INSTALL_DIR}/gog"

  # Check if target already exists
  if [ -f "$TARGET" ] && [ "$FORCE" = false ]; then
    warn "Existing installation found at ${TARGET}"
    EXISTING_VERSION="$("$TARGET" --version 2>&1 || echo "unknown")"
    info "Existing version: ${EXISTING_VERSION}"
    printf "${YELLOW}Overwrite? [y/N]${NC} "
    read -r REPLY
    if [[ ! "$REPLY" =~ ^[Yy]$ ]]; then
      info "Skipped installation to PATH. Binary is available at ./bin/gog"
      DO_INSTALL=false
    fi
  fi

  if [ "$DO_INSTALL" = true ]; then
    # Create target directory if it does not exist
    if [ ! -d "$INSTALL_DIR" ]; then
      info "Creating directory ${INSTALL_DIR}"
      mkdir -p "$INSTALL_DIR" 2>/dev/null || {
        warn "Cannot create ${INSTALL_DIR} without elevated permissions."
        info "Retrying with sudo..."
        sudo mkdir -p "$INSTALL_DIR"
      }
    fi

    # Copy binary
    cp bin/gog "$TARGET" 2>/dev/null || {
      info "Requires elevated permissions to write to ${INSTALL_DIR}"
      sudo cp bin/gog "$TARGET"
    }
    chmod +x "$TARGET"

    ok "Installed to ${TARGET}"

    # Check if INSTALL_DIR is on PATH
    if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
      echo ""
      warn "${INSTALL_DIR} is not in your PATH."
      info "Add it to your shell profile:"
      echo ""
      echo "  # bash (~/.bashrc or ~/.bash_profile)"
      echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
      echo ""
      echo "  # zsh (~/.zshrc)"
      echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
      echo ""
      info "Then reload your shell:  source ~/.bashrc  (or source ~/.zshrc)"
    fi
  fi
else
  info "Skipping PATH installation (--no-install). Binary is at ./bin/gog"
fi

# -- Step 7: Print next steps ----------------------------------------------

step "Next steps"

echo ""
printf "${BOLD}1. Set up OAuth credentials${NC}\n"
echo "   You need Google OAuth client credentials to authenticate."
echo ""
echo "   Option A -- Credentials file:"
echo "     mkdir -p ~/.config/gog && chmod 700 ~/.config/gog"
echo "     cat > ~/.config/gog/credentials.env << 'CRED'"
echo "     GOG_CLIENT_ID=your-client-id"
echo "     GOG_CLIENT_SECRET=your-client-secret"
echo "     GOG_CALLBACK_SERVER=https://your-callback-server.example.com"
echo "     CRED"
echo ""
echo "   Option B -- Environment variables:"
echo "     export GOG_CLIENT_ID=\"your-client-id\""
echo "     export GOG_CLIENT_SECRET=\"your-client-secret\""
echo ""
echo "   Option C -- JSON credentials file (standard Google format):"
echo "     gog auth credentials ~/path/to/client_secret.json"
echo ""
printf "${BOLD}2. Authenticate a Google account${NC}\n"
echo "   gog auth add you@gmail.com --headless --services=user"
echo ""
printf "${BOLD}3. Verify everything works${NC}\n"
echo "   gog auth list --check"
echo "   gog gmail labels list --account you@gmail.com"
echo ""
printf "${BOLD}4. For headless/server environments${NC}\n"
echo "   export GOG_KEYRING_BACKEND=file"
echo "   export GOG_KEYRING_PASSWORD=\"your-secure-password\""
echo ""
info "Full documentation: see INSTALL.md in this repository."
echo ""
ok "gog-cli setup complete."
