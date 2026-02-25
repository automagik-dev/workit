#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
if [[ -z "$version" ]]; then
  echo "usage: scripts/verify-release.sh N.YYMMDD.BUILD  (calver, e.g. 2.260224.1)" >&2
  exit 2
fi
if ! [[ "$version" =~ ^[0-9]+\.[0-9]{6}\.[0-9]+$ ]]; then
  echo "version must be calver: N.YYMMDD.BUILD (e.g. 2.260224.1)" >&2
  exit 2
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

changelog="CHANGELOG.md"
if ! rg -q "^## ${version} - " "$changelog"; then
  echo "missing changelog section for $version" >&2
  exit 2
fi
if rg -q "^## ${version} - Unreleased" "$changelog"; then
  echo "changelog section still Unreleased for $version" >&2
  exit 2
fi

notes_file="$(mktemp -t workit-release-notes)"
awk -v ver="$version" '
  $0 ~ "^## "ver" " {print "## "ver; in_section=1; next}
  in_section && /^## / {exit}
  in_section {print}
' "$changelog" | sed '/^$/d' > "$notes_file"

if [[ ! -s "$notes_file" ]]; then
  echo "release notes empty for $version" >&2
  exit 2
fi

release_body="$(gh release view "v$version" --json body -q .body)"
if [[ -z "$release_body" ]]; then
  echo "GitHub release notes empty for v$version" >&2
  exit 2
fi

assets_count="$(gh release view "v$version" --json assets -q '.assets | length')"
if [[ "$assets_count" -eq 0 ]]; then
  echo "no GitHub release assets for v$version" >&2
  exit 2
fi

release_run_id="$(gh run list -L 20 --workflow release.yml --json databaseId,conclusion,headBranch -q ".[] | select(.headBranch==\"v$version\") | select(.conclusion==\"success\") | .databaseId" | head -n1)"
if [[ -z "$release_run_id" ]]; then
  echo "release workflow not green for v$version" >&2
  exit 2
fi

ci_ok="$(gh run list -L 1 --workflow ci --branch main --json conclusion -q '.[0].conclusion')"
if [[ "$ci_ok" != "success" ]]; then
  echo "CI not green for main" >&2
  exit 2
fi

make ci

tmp_assets_dir="$(mktemp -d -t workit-release-assets)"
gh release download "v$version" -p checksums.txt -D "$tmp_assets_dir" >/dev/null
checksums_file="$tmp_assets_dir/checksums.txt"

sha_for_asset() {
  local name="$1"
  awk -v n="$name" '$2==n {print $1}' "$checksums_file"
}

required_assets=(
  "workit_${version}_darwin_amd64.tar.gz"
  "workit_${version}_darwin_arm64.tar.gz"
  "workit_${version}_linux_amd64.tar.gz"
  "workit_${version}_linux_arm64.tar.gz"
  "workit_${version}_windows_amd64.zip"
  "workit_${version}_windows_arm64.zip"
)

for asset in "${required_assets[@]}"; do
  if [[ -z "$(sha_for_asset "$asset")" ]]; then
    echo "missing release asset checksum entry: $asset" >&2
    exit 2
  fi
done

tmp_bin_dir="$(mktemp -d -t workit-install-bin)"
WK_BIN_DIR="$tmp_bin_dir" WK_SKIP_SKILLS=1 WK_VERSION="$version" bash scripts/install.sh
"$tmp_bin_dir/wk" --version

rm -rf "$tmp_assets_dir"
rm -rf "$tmp_bin_dir"
rm -f "$notes_file"

echo "Release v$version verified (CI, release notes/assets, installer smoke test)."
