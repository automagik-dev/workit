---
summary: "Release checklist for workit (GitHub release + Homebrew tap)"
---

# Releasing `workit`

This playbook mirrors the Homebrew + GitHub flow used in `../camsnap`.

Always do **all** steps below (CI + changelog + tag + GitHub release artifacts + tap update + Homebrew sanity install). No partial releases.

Shortcut scripts (preferred, keep notes non-empty):
```sh
scripts/release.sh X.Y.Z
scripts/verify-release.sh X.Y.Z
```

Assumptions:
- Repo: `namastexlabs/workit`
- Tap repo: `../homebrew-tap` (tap: `steipete/tap`)
- Homebrew formula name: `workit` (installs the `wk` binary)

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
- Access to the tap repo (e.g. `steipete/homebrew-tap`).

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

## 5) Update (or add) the Homebrew formula
In the tap repo (assumed sibling at `../homebrew-tap`), create/update `Formula/workit.rb`.

Recommended formula shape (build-from-source, no binary assets needed):
- `version "X.Y.Z"`
- `url "https://github.com/namastexlabs/workit/archive/refs/tags/vX.Y.Z.tar.gz"`
- `sha256 "<sha256>"`
- `depends_on "go" => :build`
- Build:
  - `system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/wk"`

Compute the SHA256 for the tag tarball:
```sh
curl -L -o /tmp/workit.tar.gz https://github.com/namastexlabs/workit/archive/refs/tags/vX.Y.Z.tar.gz
shasum -a 256 /tmp/workit.tar.gz
```

Commit + push in the tap repo:
```sh
cd ../homebrew-tap
git add Formula/workit.rb
git commit -m "workit vX.Y.Z"
git push origin main
```

## 6) Sanity-check install from tap
```sh
brew update
brew uninstall workit || true
brew untap steipete/tap || true
brew tap steipete/tap
brew install steipete/tap/workit
brew test steipete/tap/workit

wk --help
wk --version
wk version --json
```

## Notes
- `wk --version` / `wk version` should report version + branch/commit/date metadata post-install.
