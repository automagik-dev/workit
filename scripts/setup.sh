#!/usr/bin/env bash
# workit plugin Setup hook — auto-install wk binary from GitHub Releases
#
# Runs during plugin Setup. Must always exit 0.
# All output goes to stderr with [wk-setup] prefix.

MARKER="${CLAUDE_PLUGIN_ROOT:-.}/.install-version"
BIN_DIR="${WK_BIN_DIR:-$HOME/.local/bin}"
REPO="${WK_REPO:-automagik-dev/workit}"

log() { echo "[wk-setup] $*" >&2; }

find_wk() {
  local p
  p="$(command -v wk 2>/dev/null)" && echo "$p" && return 0
  [ -x "$BIN_DIR/wk" ] && echo "$BIN_DIR/wk" && return 0
  return 1
}

parse_version() {
  "$1" --version 2>/dev/null | sed -n 's/.*[vV]\([0-9][0-9.]*\).*/\1/p' | head -1
}

install_wk() {
  # Require curl and tar
  if ! command -v curl >/dev/null 2>&1; then
    log "curl not found — cannot install"
    return 1
  fi
  if ! command -v tar >/dev/null 2>&1; then
    log "tar not found — cannot install"
    return 1
  fi

  # Detect OS
  local os arch goos goarch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$os" in
    linux)  goos="linux" ;;
    darwin) goos="darwin" ;;
    *)      log "unsupported OS: $os"; return 1 ;;
  esac

  case "$arch" in
    x86_64|amd64)   goarch="amd64" ;;
    arm64|aarch64)   goarch="arm64" ;;
    *)               log "unsupported arch: $arch"; return 1 ;;
  esac

  # Resolve auth token
  local auth_token=""
  auth_token="${WK_GITHUB_TOKEN:-${GH_TOKEN:-${GITHUB_TOKEN:-}}}"
  if [ -z "$auth_token" ] && command -v gh >/dev/null 2>&1; then
    auth_token="$(gh auth token 2>/dev/null || true)"
  fi

  local auth_header=""
  if [ -n "$auth_token" ]; then
    auth_header="Authorization: Bearer $auth_token"
  fi

  # Build curl auth args
  local -a api_args=(-fsSL -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28")
  if [ -n "$auth_header" ]; then
    api_args+=(-H "$auth_header")
  fi

  # Resolve latest tag
  local tag ver
  tag="$(curl "${api_args[@]}" "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

  if [ -z "$tag" ]; then
    log "could not resolve latest release tag"
    return 1
  fi

  ver="${tag#v}"
  local asset="workit_${ver}_${goos}_${goarch}.tar.gz"
  local asset_url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
  local checksum_url="https://github.com/${REPO}/releases/download/${tag}/checksums.txt"

  local tmpdir
  tmpdir="$(mktemp -d)" || { log "mktemp failed"; return 1; }

  log "downloading ${REPO} ${tag} (${goos}/${goarch})..."

  if ! curl "${api_args[@]}" -o "${tmpdir}/${asset}" "$asset_url" 2>/dev/null; then
    log "download failed: ${asset_url}"
    rm -rf "$tmpdir"
    return 1
  fi

  # Verify checksum (best-effort)
  if curl "${api_args[@]}" -o "${tmpdir}/checksums.txt" "$checksum_url" 2>/dev/null; then
    local want got
    if command -v sha256sum >/dev/null 2>&1; then
      if ! ( cd "$tmpdir" && sha256sum -c checksums.txt --ignore-missing >/dev/null 2>&1 ); then
        log "checksum verification failed"
        rm -rf "$tmpdir"
        return 1
      fi
    elif command -v shasum >/dev/null 2>&1; then
      want="$(grep " ${asset}\$" "${tmpdir}/checksums.txt" | awk '{print $1}')"
      got="$(shasum -a 256 "${tmpdir}/${asset}" | awk '{print $1}')"
      if [ -n "$want" ] && [ "$want" != "$got" ]; then
        log "checksum mismatch"
        rm -rf "$tmpdir"
        return 1
      fi
    fi
  fi

  # Extract and install
  if ! tar -xzf "${tmpdir}/${asset}" -C "$tmpdir" 2>/dev/null; then
    log "tar extraction failed"
    rm -rf "$tmpdir"
    return 1
  fi

  mkdir -p "$BIN_DIR"
  install -m 0755 "${tmpdir}/wk" "${BIN_DIR}/wk" 2>/dev/null || {
    log "failed to install wk to ${BIN_DIR}"
    rm -rf "$tmpdir"
    return 1
  }

  # Install gog if present in the release
  if [ -f "${tmpdir}/gog" ]; then
    install -m 0755 "${tmpdir}/gog" "${BIN_DIR}/gog" 2>/dev/null || true
  fi

  rm -rf "$tmpdir"
  log "installed wk to ${BIN_DIR}/wk"

  # Hint if BIN_DIR not in PATH
  case ":$PATH:" in
    *":${BIN_DIR}:"*) ;;
    *) log "hint: add to PATH — export PATH=\"\$PATH:${BIN_DIR}\"" ;;
  esac

  return 0
}

# --- Main ---

WK="$(find_wk)" || true

if [ -z "$WK" ]; then
  log "wk not found — installing latest release..."
  if ! install_wk; then
    log "install failed (non-fatal)"
    exit 0
  fi
  WK="$(find_wk)" || {
    log "wk still not found after install"
    exit 0
  }
fi

CUR_VER="$(parse_version "$WK")"
MARKER_VER=""
[ -f "$MARKER" ] && MARKER_VER="$(awk '{print $1}' "$MARKER")"

if [ "$CUR_VER" = "$MARKER_VER" ]; then
  log "wk v${CUR_VER} up to date"
  exit 0
fi

# Version changed or first run — try self-update
if [ -n "$MARKER_VER" ]; then
  log "updating wk from v${MARKER_VER} to latest..."
  "$WK" update -y 2>&1 | sed 's/^/  /' >&2 || log "update failed (non-fatal)"
  CUR_VER="$(parse_version "$WK")"
fi

mkdir -p "$(dirname "$MARKER")" 2>/dev/null || true
echo "${CUR_VER} $(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$MARKER" 2>/dev/null || log "could not write marker (non-fatal)"
log "wk v${CUR_VER} ready"
exit 0
