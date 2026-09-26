#!/usr/bin/env bash
# Point GitHub's "Latest" release at the tag on the newest main commit.
#
# Release runs publish in parallel, so the order they finish in says nothing
# about which release is newest. This script ignores which run called it: it
# fetches main, finds the tag on the newest tagged main commit, and marks that
# release Latest once it is published. It never moves Latest to an older
# release, and an in-flight newer release is left for its own run to promote.
#
# Runs under a global `concurrency:` group, so the read-then-edit below never
# interleaves with another run's.
#
# Env: GITHUB_REPOSITORY (owner/repo, required), GH_TOKEN (for gh),
#      RELEASE_BRANCH (default main), RELEASE_REMOTE (default origin).
set -euo pipefail

repo="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
branch="${RELEASE_BRANCH:-main}"
remote="${RELEASE_REMOTE:-origin}"

git fetch --quiet --force --tags "$remote" "+refs/heads/${branch}:refs/remotes/${remote}/${branch}"
head="$(git rev-parse "${remote}/${branch}^{commit}")"

# The tag on the newest tagged commit of main's first-parent history. Release
# tags sit on main's merge commits, so walking first parents from the tip
# finds the newest one first.
if ! target="$(git describe --tags --abbrev=0 --first-parent --match 'v[0-9]*' "$head" 2>/dev/null)"; then
  echo "No release tag on ${branch} (${head}); Latest unchanged."
  exit 0
fi

target_commit="$(git rev-parse "${target}^{commit}")"
if ! git merge-base --is-ancestor "$target_commit" "$head"; then
  echo "::error::${target} (${target_commit}) is not on ${branch} (${head}); refusing to mark it Latest."
  exit 1
fi

if ! state="$(gh release view "$target" -R "$repo" --json isDraft,isPrerelease \
  --jq 'if .isDraft then "draft" elif .isPrerelease then "prerelease" else "published" end' 2>/dev/null)"; then
  state="missing"
fi

if [ "$state" != "published" ]; then
  echo "::notice::Newest ${branch} release tag is ${target} (${target_commit}), but its release is ${state}; Latest unchanged. Its own release run promotes it once published."
  exit 0
fi

current="$(gh api "repos/${repo}/releases/latest" --jq .tag_name 2>/dev/null || true)"
if [ "$current" = "$target" ]; then
  echo "Latest is already ${target} (${target_commit}, ${branch} ${head})."
  exit 0
fi

gh release edit "$target" -R "$repo" --latest >/dev/null
echo "Latest: ${current:-none} -> ${target} (${target_commit}, ${branch} ${head})."
