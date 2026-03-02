#!/bin/sh
# ---------------------------------------------------------------------------
# workit installer
#
# Downloads and installs the wk binary and workit plugin from GitHub Releases.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/automagik-dev/workit/main/install.sh | sh
#   sh install.sh [--force] [--version VERSION] [--help]
#
# Environment overrides:
#   WK_RELEASE_URL  Base URL for downloads (default: https://github.com/automagik-dev/workit/releases/download)
#                   Useful for offline/local testing with file:// URLs.
# ---------------------------------------------------------------------------
set -e

# ---------------------------------------------------------------------------
# Color support (detect terminal)
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    NC=''
fi

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { printf "${BLUE}[INFO]${NC}  %s\n" "$*"; }
ok()    { printf "${GREEN}[OK]${NC}    %s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC}  %s\n" "$*"; }
fail()  { printf "${RED}[ERROR]${NC} %s\n" "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
FORCE=false
VERSION=""
GITHUB_REPO="automagik-dev/workit"
GITHUB_API="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
RELEASE_BASE_URL="${WK_RELEASE_URL:-https://github.com/${GITHUB_REPO}/releases/download}"
INSTALL_DIR="${HOME}/.local/bin"
PLUGIN_DIR="${HOME}/.workit/plugin"

# ---------------------------------------------------------------------------
# Flag parsing
# ---------------------------------------------------------------------------
while [ $# -gt 0 ]; do
    case "$1" in
        --help|-h)
            cat <<EOF
Usage: sh install.sh [OPTIONS]

Downloads and installs workit (wk binary + plugin) from GitHub Releases.

Options:
  --force              Skip confirmation prompts and overwrite existing install
  --version VERSION    Install a specific version (e.g. 2.260227.5)
                       Default: latest release
  --help, -h           Show this help message and exit

Environment variables:
  WK_RELEASE_URL       Override the base download URL (for offline/local testing)

Examples:
  sh install.sh
  sh install.sh --force
  sh install.sh --version 2.260227.5
  WK_RELEASE_URL=file:///tmp/releases sh install.sh
EOF
            exit 0
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --version)
            shift
            [ $# -gt 0 ] || fail "--version requires a VERSION argument"
            VERSION="$1"
            shift
            ;;
        --version=*)
            VERSION="${1#--version=}"
            shift
            ;;
        *)
            fail "Unknown option: $1 (try --help)"
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Temp dir + cleanup trap
# ---------------------------------------------------------------------------
TMPDIR_WORK="$(mktemp -d)"
cleanup() {
    rm -rf "$TMPDIR_WORK"
}
trap cleanup EXIT INT TERM

# ---------------------------------------------------------------------------
# Detect downloader (curl or wget)
# ---------------------------------------------------------------------------
if command -v curl > /dev/null 2>&1; then
    DOWNLOADER="curl"
elif command -v wget > /dev/null 2>&1; then
    DOWNLOADER="wget"
else
    fail "Neither curl nor wget found. Please install one and re-run."
fi

download() {
    _url="$1"
    _dest="$2"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -sSfL "$_url" -o "$_dest" || return 1
    else
        wget -q "$_url" -O "$_dest" || return 1
    fi
}

download_stdout() {
    _url="$1"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -sSfL "$_url" || return 1
    else
        wget -q "$_url" -O - || return 1
    fi
}

# ---------------------------------------------------------------------------
# OS / arch detection
# ---------------------------------------------------------------------------
OS_RAW="$(uname -s)"
ARCH_RAW="$(uname -m)"

case "$OS_RAW" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)      fail "Unsupported OS: ${OS_RAW}. Only linux and darwin are supported." ;;
esac

case "$ARCH_RAW" in
    x86_64)          ARCH="amd64" ;;
    aarch64|arm64)   ARCH="arm64" ;;
    *)               fail "Unsupported architecture: ${ARCH_RAW}. Only amd64 and arm64 are supported." ;;
esac

info "Detected platform: ${OS}/${ARCH}"

# ---------------------------------------------------------------------------
# Version resolution
# ---------------------------------------------------------------------------
if [ -z "$VERSION" ]; then
    info "Fetching latest release version..."
    RELEASE_JSON="$(download_stdout "$GITHUB_API" 2>/dev/null)" || \
        fail "Failed to fetch release info from ${GITHUB_API}"

    # Parse tag_name from JSON without jq (handles both "v2.x.y" and "2.x.y")
    VERSION="$(printf '%s' "$RELEASE_JSON" | grep '"tag_name"' | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/')"
    [ -n "$VERSION" ] || fail "Could not parse version from GitHub API response."
fi

# Ensure TAG has the v prefix, VERSION does not
VERSION="${VERSION#v}"
TAG="v${VERSION}"

info "Installing workit ${TAG}"

# ---------------------------------------------------------------------------
# Binary download and install
# ---------------------------------------------------------------------------
BINARY_FILENAME="workit_${VERSION}_${OS}_${ARCH}.tar.gz"
BINARY_URL="${RELEASE_BASE_URL}/${TAG}/${BINARY_FILENAME}"
BINARY_ARCHIVE="${TMPDIR_WORK}/${BINARY_FILENAME}"

info "Downloading binary: ${BINARY_URL}"
download "$BINARY_URL" "$BINARY_ARCHIVE" || \
    fail "Failed to download binary from ${BINARY_URL}"

info "Extracting binary..."
tar -xzf "$BINARY_ARCHIVE" -C "$TMPDIR_WORK"

# Find the wk binary in the extracted files
WK_BINARY="$(find "$TMPDIR_WORK" -type f -name 'wk' | head -n 1)"
[ -n "$WK_BINARY" ] || fail "Could not find 'wk' binary in archive ${BINARY_FILENAME}"

# Check for existing install
TARGET="${INSTALL_DIR}/wk"
if [ -f "$TARGET" ] && [ "$FORCE" = false ]; then
    EXISTING_VERSION="$("$TARGET" --version 2>/dev/null || echo "unknown")"
    warn "Existing installation found at ${TARGET}"
    info "Installed version : ${EXISTING_VERSION}"
    info "New version       : ${VERSION}"
    if [ ! -t 0 ]; then
        warn "Running non-interactively; use --force to overwrite existing install."
        exit 0
    fi
    printf "Overwrite? [y/N] "
    read -r REPLY
    case "$REPLY" in
        [Yy]*) ;;
        *) info "Installation cancelled. Existing install unchanged."; exit 0 ;;
    esac
fi

mkdir -p "$INSTALL_DIR"
cp "$WK_BINARY" "$TARGET"
chmod +x "$TARGET"
ok "Binary installed: ${TARGET}"

# macOS quarantine removal
if [ "$OS" = "darwin" ]; then
    xattr -d com.apple.quarantine "$TARGET" 2>/dev/null || true
fi

# ---------------------------------------------------------------------------
# Plugin download and install
# ---------------------------------------------------------------------------
PLUGIN_FILENAME="workit-plugin_${VERSION}.tar.gz"
PLUGIN_URL="${RELEASE_BASE_URL}/${TAG}/${PLUGIN_FILENAME}"
PLUGIN_ARCHIVE="${TMPDIR_WORK}/${PLUGIN_FILENAME}"

info "Downloading plugin: ${PLUGIN_URL}"
download "$PLUGIN_URL" "$PLUGIN_ARCHIVE" || \
    fail "Failed to download plugin from ${PLUGIN_URL}"

info "Installing plugin to ${PLUGIN_DIR}..."

# Remove old plugin contents and recreate directory
rm -rf "$PLUGIN_DIR"
mkdir -p "$PLUGIN_DIR"

# The tarball extracts to a workit/ root directory; move contents into PLUGIN_DIR
PLUGIN_EXTRACT="${TMPDIR_WORK}/plugin_extract"
mkdir -p "$PLUGIN_EXTRACT"
tar -xzf "$PLUGIN_ARCHIVE" -C "$PLUGIN_EXTRACT"

# Move the extracted contents (inside workit/ subdirectory) to PLUGIN_DIR
if [ -d "${PLUGIN_EXTRACT}/workit" ]; then
    cp -r "${PLUGIN_EXTRACT}/workit/." "$PLUGIN_DIR/"
else
    # Fallback: move everything directly
    cp -r "${PLUGIN_EXTRACT}/." "$PLUGIN_DIR/"
fi

ok "Plugin installed: ${PLUGIN_DIR}"

# ---------------------------------------------------------------------------
# Claude Code integration
# ---------------------------------------------------------------------------
mkdir -p "${HOME}/.claude/plugins"
ln -sf "$PLUGIN_DIR" "${HOME}/.claude/plugins/workit"
ok "Claude Code plugin linked: ~/.claude/plugins/workit"

# ---------------------------------------------------------------------------
# OpenClaw integration (optional)
# ---------------------------------------------------------------------------
if command -v openclaw > /dev/null 2>&1; then
    openclaw plugins install "$PLUGIN_DIR/" 2>/dev/null || true
    ok "OpenClaw plugin registered"
fi

# ---------------------------------------------------------------------------
# PATH check
# ---------------------------------------------------------------------------
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
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
        info "Then reload your shell: source ~/.bashrc  (or source ~/.zshrc)"
        echo ""
        ;;
esac

# ---------------------------------------------------------------------------
# Bootstrap prompt
# ---------------------------------------------------------------------------
printf "\n"
printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
printf "${GREEN}✅ workit v%s installed${NC}\n" "$VERSION"
printf "\n"
printf "Binary:  ~/.local/bin/wk\n"
printf "Plugin:  ~/.workit/plugin/\n"
printf "Skills:  loaded (Gmail, Calendar, Drive, Sheets, Docs, Slides, Chat, ...)\n"
printf "Relay:   https://auth.automagik.dev (no GCP setup needed)\n"
printf "\n"
printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
printf "AGENT BOOTSTRAP PROMPT\n"
printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
printf "\n"
printf "You now have access to Google Workspace via the \`wk\` CLI (workit v%s).\n" "$VERSION"
printf "No GCP setup needed — auth runs through https://auth.automagik.dev.\n"
printf "\n"
printf "Quick start:\n"
printf "  wk auth status          → check if accounts are already connected\n"
printf "  wk auth manage          → add an account (opens browser or prints URL for headless)\n"
printf "  wk auth add EMAIL --headless --no-input  → for fully unattended agent flow\n"
printf "\n"
printf "Available services: Gmail, Calendar, Drive, Sheets, Docs, Slides,\n"
printf "                    Chat, Classroom, Tasks, Contacts, Keep, Forms, AppScript\n"
printf "\n"
printf "Ask the user what they want to do, then load the relevant skill file.\n"
printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
printf "\n"
