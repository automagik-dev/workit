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
