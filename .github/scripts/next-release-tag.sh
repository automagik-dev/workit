#!/usr/bin/env bash
# Decide whether HEAD gets a release tag, and which one. Prints `tag=<tag>`
# (tag HEAD) or `tag=` (skip) on stdout for $GITHUB_OUTPUT; reasons go to
# stderr.
#
# The skip markers are evaluated over every untagged commit since the last
# release tag, not just HEAD. auto-release runs one push at a time and GitHub
# keeps only the newest pending run, so the run for a `[skip release]` push can
# replace the pending run of an earlier push that wanted a release. That
# release must still happen: HEAD is tagged when any untagged commit on main's
# first-parent history lacks a skip marker, and skipped only when all of them
# ask for a skip.
#
# Env: RELEASE_DATE (yymmdd, default today in UTC), RELEASE_MAJOR (default 2).
set -euo pipefail

major="${RELEASE_MAJOR:-2}"
date="${RELEASE_DATE:-$(date -u +%y%m%d)}"

existing="$(git tag --points-at HEAD --list 'v[0-9]*' | head -1)"
if [ -n "$existing" ]; then
  echo "HEAD is already tagged ${existing}; nothing to release." >&2
  echo "tag="
  exit 0
fi

# Untagged first-parent commits since the last release tag (all of them when
# there is none). Each first-parent commit of main is one push or one merge.
if last="$(git describe --tags --abbrev=0 --first-parent --match 'v[0-9]*' HEAD 2>/dev/null)"; then
  range="${last}..HEAD"
else
  last=""
  range="HEAD"
fi

wanted=""
while IFS= read -r commit; do
  message="$(git log -1 --format=%B "$commit")"
  case "$message" in
    *"[skip release]"* | *"[skip ci]"*) ;;
    *)
      wanted="$commit"
      break
      ;;
  esac
done < <(git rev-list --first-parent "$range")

if [ -z "$wanted" ]; then
  echo "Every commit since ${last:-the first commit} asks to skip the release; not tagging." >&2
  echo "tag="
  exit 0
fi

today_latest="$(git tag --list "v${major}.${date}.*" --sort=-version:refname | head -1)"
if [ -z "$today_latest" ]; then
  n=1
else
  n=$((${today_latest##*.} + 1))
fi

tag="v${major}.${date}.${n}"
echo "Commit $(git rev-parse --short "$wanted") since ${last:-the first commit} wants a release; tagging HEAD as ${tag}." >&2
echo "tag=${tag}"
