# Brainstorm: Root .md Cleanup

## Problem
Repo root has 10 .md files. Some are standard open-source (README, CHANGELOG, INSTALL) but others are agent-internal docs (AGENT.md, MEMORY.md, MILESTONES.md, TODO.md, BACKLOG.md, ENVIRONMENT.md) that belong in ~/agents/gog-cli/ not in the public repo.

## Current State — Repo Root .md Files

| File | Lines | Category | Proposed Action |
|------|-------|----------|-----------------|
| README.md | 1582 | Standard OSS | KEEP in repo |
| CHANGELOG.md | 377 | Standard OSS | KEEP in repo |
| INSTALL.md | 190 | Standard OSS | KEEP in repo |
| CLAUDE.md | 6 | Agent config | KEEP (references other files) |
| AGENT.md | 576 | Agent spec | MOVE → ~/agents/gog-cli/ |
| TODO.md | 281 | Infra setup guide | MOVE → ~/agents/gog-cli/ |
| MILESTONES.md | 142 | Progress tracking | MOVE → ~/agents/gog-cli/ |
| BACKLOG.md | 18 | Bug tracker | MOVE → ~/agents/gog-cli/ |
| MEMORY.md | 35 | Agent memory | MOVE → ~/agents/gog-cli/ (already gitignored?) |
| ENVIRONMENT.md | 44 | Agent env config | MOVE → ~/agents/gog-cli/ |

## Decision: What stays in repo root?
- README.md — public-facing project docs
- CHANGELOG.md — release history
- INSTALL.md — user setup guide
- CLAUDE.md — agent config entry point (tiny, just @refs)

## Decision: What moves to ~/agents/gog-cli/?
- AGENT.md — full implementation spec (agent internal)
- TODO.md — infrastructure setup tasks (agent internal)
- MILESTONES.md — progress tracking (agent internal)
- BACKLOG.md — bug tracking (agent internal)
- MEMORY.md — agent memory/decisions (agent internal)
- ENVIRONMENT.md — machine-specific paths/config (agent internal)

## CLAUDE.md update
Currently references ENVIRONMENT.md via @./ENVIRONMENT.md
After move: @../../../agents/gog-cli/ENVIRONMENT.md

## WRS
- Problem: ✅
- Scope: ✅
- Decisions: ✅
- Risks: need to check
- Criteria: need to define
