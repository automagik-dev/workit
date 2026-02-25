#!/usr/bin/env bash
# workit one-command installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/install.sh | bash

set -euo pipefail

REPO="${WK_REPO:-automagik-dev/workit}"
SKILLS_REPO="${WK_SKILLS_REPO:-$REPO}"
BIN_DIR="${WK_BIN_DIR:-$HOME/.local/bin}"
SKILLS_DIR="${WK_SKILLS_DIR:-$HOME/.config/workit/skills}"
VERSION="${WK_VERSION:-latest}"
SKIP_SKILLS="${WK_SKIP_SKILLS:-0}"

need_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "error: missing required command: $1" >&2
		exit 1
	fi
}

need_cmd curl
need_cmd tar

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
linux) GOOS="linux" ;;
darwin) GOOS="darwin" ;;
*)
	echo "error: unsupported OS: $OS" >&2
	exit 1
	;;
esac

case "$ARCH" in
x86_64|amd64) GOARCH="amd64" ;;
arm64|aarch64) GOARCH="arm64" ;;
*)
	echo "error: unsupported architecture: $ARCH" >&2
	exit 1
	;;
esac

AUTH_TOKEN="${WK_GITHUB_TOKEN:-${GH_TOKEN:-${GITHUB_TOKEN:-}}}"
if [[ -z "$AUTH_TOKEN" ]] && command -v gh >/dev/null 2>&1; then
	AUTH_TOKEN="$(gh auth token 2>/dev/null || true)"
fi

api_headers=(-H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28")
if [[ -n "$AUTH_TOKEN" ]]; then
	api_headers+=(-H "Authorization: Bearer $AUTH_TOKEN")
fi

if [[ "$VERSION" == "latest" ]]; then
	TAG="$(
		curl -fsSL "${api_headers[@]}" "https://api.github.com/repos/${REPO}/releases/latest" | \
			sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
	)"
else
	TAG="$VERSION"
	[[ "$TAG" == v* ]] || TAG="v$TAG"
fi

if [[ -z "$TAG" ]]; then
	echo "error: could not resolve release tag for ${REPO}" >&2
	exit 1
fi

VER="${TAG#v}"
ASSET="workit_${VER}_${GOOS}_${GOARCH}.tar.gz"
ASSET_URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "installing ${REPO} ${TAG} (${GOOS}/${GOARCH})"

curl -fsSL "${api_headers[@]}" -o "${tmpdir}/${ASSET}" "$ASSET_URL"
if curl -fsSL "${api_headers[@]}" -o "${tmpdir}/checksums.txt" "$CHECKSUM_URL"; then
	if command -v sha256sum >/dev/null 2>&1; then
		( cd "$tmpdir" && sha256sum -c checksums.txt --ignore-missing )
	elif command -v shasum >/dev/null 2>&1; then
		want="$(grep " ${ASSET}\$" "${tmpdir}/checksums.txt" | awk '{print $1}')"
		got="$(shasum -a 256 "${tmpdir}/${ASSET}" | awk '{print $1}')"
		[[ "$want" == "$got" ]] || { echo "error: checksum mismatch for ${ASSET}" >&2; exit 1; }
	fi
fi

tar -xzf "${tmpdir}/${ASSET}" -C "$tmpdir"

mkdir -p "$BIN_DIR"
install -m 0755 "${tmpdir}/wk" "${BIN_DIR}/wk"
if [[ -f "${tmpdir}/gog" ]]; then
	install -m 0755 "${tmpdir}/gog" "${BIN_DIR}/gog"
fi

if [[ "$SKIP_SKILLS" == "1" ]]; then
	echo "skipping skills install (WK_SKIP_SKILLS=1)"
elif command -v git >/dev/null 2>&1; then
	skills_tmp="${tmpdir}/skills-repo"
	git clone --depth=1 "https://github.com/${SKILLS_REPO}.git" "$skills_tmp" >/dev/null
	if [[ -n "$TAG" ]]; then
		git -C "$skills_tmp" checkout --quiet "$TAG" >/dev/null 2>&1 || true
	fi

	if [[ -d "${skills_tmp}/skills/workit" ]]; then
		skills_src="${skills_tmp}/skills/workit"
	elif [[ -d "${skills_tmp}/workit" ]]; then
		skills_src="${skills_tmp}/workit"
	else
		echo "warning: ${SKILLS_REPO} does not contain a workit skill folder; skipping skills install" >&2
		skills_src=""
	fi

	if [[ -n "$skills_src" ]]; then
		mkdir -p "$SKILLS_DIR"
		rm -rf "${SKILLS_DIR}/workit"
		cp -R "$skills_src" "${SKILLS_DIR}/workit"
		echo "installed skills to ${SKILLS_DIR}/workit"
	fi
else
	echo "warning: git not found; skipping skills install" >&2
fi

echo
echo "installed wk to ${BIN_DIR}/wk"
if [[ ":$PATH:" != *":${BIN_DIR}:"* ]]; then
	echo "add to PATH:"
	echo "  export PATH=\"\$PATH:${BIN_DIR}\""
fi
echo
echo "next updates:"
echo "  wk update"
