#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# workit Local Developer Install
#
# Run this AFTER cloning the repo. It builds the binary and optionally
# installs it to your PATH (~/.local/bin).
#
# Usage:
#   ./install.sh                  # build + install to ~/.local/bin
#   ./install.sh --no-install     # build only (binary at bin/wk)
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
    --no-install)
      DO_INSTALL=false
      ;;
    --force)
      FORCE=true
      ;;
    --help|-h)
      echo "Usage: ./install.sh [--no-install] [--force] [--help]"
      echo ""
      echo "Local developer install script. Run after cloning the repo."
      echo ""
      echo "Options:"
      echo "  --no-install  Build only, do not copy binary to PATH"
      echo "  --force       Overwrite existing installation without asking"
      echo "  --help, -h    Show this help message"
      echo ""
      echo "Examples:"
      echo "  ./install.sh                  # build + install to ~/.local/bin/wk"
      echo "  ./install.sh --no-install     # build only (binary at ./bin/wk)"
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
  fail "Makefile not found. Please run this script from the workit repo root."
fi

if [ ! -d "cmd/wk" ]; then
  fail "cmd/wk/ directory not found. Please run this script from the workit repo root."
fi

if [ ! -f "go.mod" ]; then
  fail "go.mod not found. Please run this script from the workit repo root."
fi

ok "Repository root detected (Makefile, cmd/wk/, go.mod present)"

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

step "Building wk binary"

if ! make build; then
  fail "make build failed. Check the output above for errors."
fi

if [ ! -f "bin/wk" ]; then
  fail "Build appeared to succeed but bin/wk was not created."
fi

ok "Binary built at bin/wk"

# -- Step 5: Verify the binary works ---------------------------------------

step "Verifying binary"

WK_VERSION_OUTPUT="$(./bin/wk --version 2>&1 || true)"

if [ -z "$WK_VERSION_OUTPUT" ]; then
  warn "bin/wk --version produced no output (binary may still be functional)"
else
  ok "bin/wk --version: ${WK_VERSION_OUTPUT}"
fi

# -- Step 6: Optionally install to ~/.local/bin ----------------------------

if [ "$DO_INSTALL" = true ]; then
  step "Installing to ${INSTALL_DIR}"

  TARGET="${INSTALL_DIR}/wk"

  # Check if target already exists
  if [ -f "$TARGET" ] && [ "$FORCE" = false ]; then
    warn "Existing installation found at ${TARGET}"
    EXISTING_VERSION="$("$TARGET" --version 2>&1 || echo "unknown")"
    info "Existing version: ${EXISTING_VERSION}"
    printf "${YELLOW}Overwrite? [y/N]${NC} "
    read -r REPLY
    if [[ ! "$REPLY" =~ ^[Yy]$ ]]; then
      info "Skipped installation to PATH. Binary is available at ./bin/wk"
      DO_INSTALL=false
    fi
  fi

  if [ "$DO_INSTALL" = true ]; then
    mkdir -p "$INSTALL_DIR"
    cp bin/wk "$TARGET"
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
  info "Skipping PATH installation (--no-install). Binary is at ./bin/wk"
fi

# -- Step 7: Print next steps ----------------------------------------------

step "Next steps"

echo ""
printf "${BOLD}1. Set up OAuth credentials${NC}\n"
echo "   You need Google OAuth client credentials to authenticate."
echo ""
echo "   Option A -- Credentials file:"
echo "     mkdir -p ~/.config/workit && chmod 700 ~/.config/workit"
echo "     cat > ~/.config/workit/credentials.env << 'CRED'"
echo "     WK_CLIENT_ID=your-client-id"
echo "     WK_CLIENT_SECRET=your-client-secret"
echo "     WK_CALLBACK_SERVER=https://your-callback-server.example.com"
echo "     CRED"
echo ""
echo "   Option B -- Environment variables:"
echo "     export WK_CLIENT_ID=\"your-client-id\""
echo "     export WK_CLIENT_SECRET=\"your-client-secret\""
echo ""
echo "   Option C -- JSON credentials file (standard Google format):"
echo "     wk auth credentials ~/path/to/client_secret.json"
echo ""
printf "${BOLD}2. Authenticate a Google account${NC}\n"
echo "   wk auth add you@gmail.com --headless --services=user"
echo ""
printf "${BOLD}3. Verify everything works${NC}\n"
echo "   wk auth list --check"
echo "   wk gmail labels list --account you@gmail.com"
echo ""
printf "${BOLD}4. For headless/server environments${NC}\n"
echo "   export WK_KEYRING_BACKEND=file"
echo "   export WK_KEYRING_PASSWORD=\"your-secure-password\""
echo ""
info "Full documentation: see INSTALL.md in this repository."
echo ""
ok "workit setup complete."
