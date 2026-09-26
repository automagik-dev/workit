---
summary: "Release checklist for workit (GitHub release + installer smoke test)"
---

# Releasing `workit`

Always do **all** steps below (CI + changelog + tag + GitHub release artifacts + installer sanity install). No partial releases.

Shortcut scripts (preferred, keep notes non-empty):
```sh
scripts/release.sh X.Y.Z
scripts/verify-release.sh X.Y.Z
```

Assumptions:
- Repo: `automagik-dev/workit`
- Installer script: `scripts/install.sh`

## Branch model and protections

Use a `dev` -> `main` promotion flow:

- Feature work merges into `dev`.
- `main` is release-only and receives tested merges from `dev`.
- Protect both branches with required checks.

Recommended required checks:

- `ci / test`
- `ci / worker`
- `ci / darwin-cgo-build`
- `version / version-artifact`

`version / version-artifact` enforces the version artifact contract (`version`, `branch`, `commit`, `date`) and uploads `version-contract` JSON.

## 0) Prereqs
- Clean working tree on `main`.
- Go toolchain installed (Go version comes from `go.mod`).
- `make` works locally.

## 1) Verify build is green
```sh
make ci
```

Confirm GitHub Actions `ci` and `version` are green for the commit you’re tagging:
```sh
gh run list -L 5 --branch main --workflow ci.yml
gh run list -L 5 --branch main --workflow version.yml
```

## 2) Update changelog
- Update `CHANGELOG.md` for the version you’re releasing.

Example heading:
- `## 0.1.0 - 2025-12-12`

## 3) Commit, tag & push
```sh
git checkout main
git pull

# commit changelog + any release tweaks
git commit -am "release: vX.Y.Z"

git tag -a vX.Y.Z -m "Release X.Y.Z"
git push origin main --tags
```

## 4) Verify GitHub release artifacts
The tag push triggers `.github/workflows/release.yml` (GoReleaser + changelog-derived release notes). Ensure it completes successfully and the release has assets.

```sh
gh run list -L 5 --workflow release.yml
gh release view vX.Y.Z
```

Ensure GitHub release notes are not empty and match the matching changelog section.

If the workflow needs a rerun:
```sh
gh workflow run release.yml -f tag=vX.Y.Z
```

### Automatic releases and "Latest"

Every push to `main` runs `auto-release.yml`, which tags the commit with the
next calver tag (`v2.YYMMDD.N`) and dispatches `release.yml` for it. A merge
train can start several of these at once, so:

- `auto-release.yml` runs one at a time (`concurrency: auto-release`). It skips
  a commit that is already tagged. `[skip release]` and `[skip ci]` are read
  from every untagged first-parent commit since the last tag
  (`.github/scripts/next-release-tag.sh`): HEAD is tagged when any of them
  lacks a marker, and skipped only when all of them carry one.
- `release.yml` runs for the same tag wait for each other; runs for different
  tags build in parallel.
- GoReleaser creates each release as a draft, uploads every asset, then
  publishes it with `make_latest: "false"`. A build that fails mid-upload
  leaves a draft.
- The last job, `latest`, runs one at a time across all release runs
  (`.github/scripts/set-latest-release.sh`). It walks `main` from the tip and
  marks Latest on the newest release that is published (not a draft, not a
  prerelease, with assets). It never moves Latest backwards. A newer tag on
  `main` whose release is missing, a draft or incomplete gets a `::warning::`
  naming it, and Latest falls back to the newest published release.
- The `latest` job fails when its own tag is the newest on `main` but its
  release is not published after GoReleaser, and on any GitHub API error. It
  never reads an API error as "no release".

**Cancelled runs are expected.** GitHub keeps only the newest pending run in a
concurrency group. When a merge train queues several, the older pending
`auto-release` run or `latest` job is replaced and shows as *cancelled*.
Nothing is lost: the replacing `auto-release` run tags HEAD when any untagged
commit wants a release, and every `latest` job recomputes the same target from
`main`. A *failed* `latest` job is the signal to act on.

To check or repair Latest by hand:
```sh
gh api repos/automagik-dev/workit/releases/latest --jq .tag_name
gh release edit vX.Y.Z --latest   # only for the newest published release on main
```

## 5) Sanity-check installer and update flow
```sh
curl -fsSL https://raw.githubusercontent.com/automagik-dev/workit/main/scripts/install.sh | bash
wk --version
wk update

wk --help
wk version --json
```

## Notes
- `wk --version` / `wk version` should report version + branch/commit/date metadata post-install.
