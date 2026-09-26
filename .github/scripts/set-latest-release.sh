#!/usr/bin/env bash
# Point GitHub's "Latest" release at the newest published release on main.
#
# Release runs publish in parallel, so the order they finish in says nothing
# about which release is newest. This script ignores that order. It walks
# main's first-parent history from the tip and marks Latest on the first
# release tag whose release is published. It never moves Latest backwards:
# the walk stops at the commit of the current Latest.
#
# "Published" means not a draft, not a prerelease, and carrying assets.
# GoReleaser creates each release as a draft, uploads every asset, and only
# then publishes it (internal/client/github.go CreateRelease/PublishRelease),
# so a non-draft GoReleaser release already has all of its assets. A build
# that fails mid-upload leaves a draft, which is never promoted.
#
# A newer tag on main whose release is missing, a draft or incomplete is
# skipped with a ::warning:: naming it, and Latest falls back to the newest
# published release. The run fails when its own tag (RELEASE_TAG) is among
# those skipped: after GoReleaser succeeded, its release must be published.
# Any GitHub API error fails the run; it is never read as "no release".
#
# Runs under a global `concurrency:` group, so the read-then-edit below never
# interleaves with another run's.
#
# Env: GITHUB_REPOSITORY (owner/repo, required), GH_TOKEN (for gh),
#      RELEASE_TAG (the calling run's tag, optional),
#      RELEASE_BRANCH (default main), RELEASE_REMOTE (default origin).
set -euo pipefail

repo="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
branch="${RELEASE_BRANCH:-main}"
remote="${RELEASE_REMOTE:-origin}"
own="${RELEASE_TAG:-}"

git fetch --quiet --force --tags "$remote" "+refs/heads/${branch}:refs/remotes/${remote}/${branch}"
head="$(git rev-parse "${remote}/${branch}^{commit}")"

# Every release and its state, in one listing. An API error fails the run.
if ! releases="$(gh api --paginate "repos/${repo}/releases?per_page=100" \
  --jq '.[] | [.tag_name, (if .draft then "draft" elif .prerelease then "prerelease" elif (.assets | length) == 0 then "without-assets" else "published" end)] | @tsv')"; then
  echo "::error::Could not list the releases of ${repo}; Latest unchanged."
  exit 1
fi

release_state() {
  awk -F '\t' -v tag="$1" '$1 == tag { state = $2 } END { print (state == "" ? "missing" : state) }' <<<"$releases"
}

# The current Latest. 404 means there is none; any other error fails the run.
errfile="$(mktemp)"
trap 'rm -f "$errfile"' EXIT
if current="$(gh api "repos/${repo}/releases/latest" --jq .tag_name 2>"$errfile")"; then
  :
elif grep -q 'HTTP 404' "$errfile"; then
  current=""
else
  cat "$errfile" >&2
  echo "::error::Could not read the current Latest release of ${repo}; Latest unchanged."
  exit 1
fi

current_commit=""
if [ -n "$current" ]; then
  current_commit="$(git rev-parse -q --verify "refs/tags/${current}^{commit}" || true)"
fi

# Release tags by the commit they point at (annotated tags peeled).
declare -A tags_at=()
while read -r commit tag; do
  tags_at[$commit]+="${tag} "
done < <(git for-each-ref --format='%(if)%(*objectname)%(then)%(*objectname)%(else)%(objectname)%(end) %(refname:short)' 'refs/tags/v[0-9]*')

target=""
target_commit=""
own_state=""
while read -r commit; do
  # Reached the current Latest (or history older than it): keep it.
  if [ -n "$current_commit" ] && git merge-base --is-ancestor "$commit" "$current_commit"; then
    break
  fi

  [ -n "${tags_at[$commit]:-}" ] || continue

  for tag in $(tr ' ' '\n' <<<"${tags_at[$commit]}" | sort -rV); do
    state="$(release_state "$tag")"
    if [ "$tag" = "$own" ]; then
      own_state="$state"
    fi

    if [ "$state" = "published" ]; then
      target="$tag"
      target_commit="$commit"
      break 2
    fi

    echo "::warning::${tag} (${commit}) is newer on ${branch} than any published release, but its release is ${state}; Latest falls back to an older published release."
  done
done < <(git rev-list --first-parent "$head")

if [ -z "$target" ]; then
  echo "Latest stays ${current:-unset}: no published release on ${branch} is newer."
elif [ "$target" = "$current" ]; then
  echo "Latest is already ${target} (${target_commit})."
else
  gh release edit "$target" -R "$repo" --latest >/dev/null
  echo "Latest: ${current:-none} -> ${target} (${target_commit}, ${branch} ${head})."
fi

if [ -n "$own_state" ] && [ "$own_state" != "published" ]; then
  echo "::error::This run's tag ${own} is on ${branch} and newer than Latest, but its release is ${own_state} after GoReleaser finished."
  exit 1
fi
