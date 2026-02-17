# Design: Root .md Cleanup

## Problem
Repo root cluttered with agent-internal .md files that don't belong in a public repo.

## Scope IN
- Move 6 files from repo → ~/agents/gog-cli/: AGENT.md, TODO.md, MILESTONES.md, BACKLOG.md, MEMORY.md, ENVIRONMENT.md
- Fix stale content during move (paths, outdated info)
- Update CLAUDE.md to reference ENVIRONMENT.md at new location
- git rm tracked files from repo

## Scope OUT
- README.md, CHANGELOG.md, INSTALL.md — stay as-is
- docs/ directory contents — not touched
- .genie/ contents — not touched

## Decisions
- ENVIRONMENT.md moves to ~/agents/gog-cli/ and CLAUDE.md refs it there
- Fix stale workspace path in ENVIRONMENT.md (/home/genie/repos → /home/genie/workspace/repos)
- Fix origin remote URL (automagik-genie → namastexlabs)

## Acceptance Criteria
- [ ] Only README.md, CHANGELOG.md, INSTALL.md, CLAUDE.md remain in repo root
- [ ] All 6 files exist in ~/agents/gog-cli/
- [ ] CLAUDE.md @refs resolve correctly
- [ ] ENVIRONMENT.md has correct paths
- [ ] make ci passes
