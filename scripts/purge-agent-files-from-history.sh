#!/bin/bash
# Purge agent identity files from git history using git-filter-repo.
#
# These files were committed (b543678) then deleted (70a3e77, 66fc68c)
# but remain in git history with personal information.
#
# WARNING: This rewrites history and requires a force-push.
# All collaborators must re-clone after this runs.
#
# Usage:
#   1. Create a FRESH clone (do NOT run on your working copy):
#        git clone --mirror https://github.com/automagik-dev/workit.git /tmp/workit-mirror.git
#
#   2. Run this script:
#        bash scripts/purge-agent-files-from-history.sh /tmp/workit-mirror.git
#
#   3. Force-push:
#        cd /tmp/workit-mirror.git && git push --force
#
#   4. All collaborators re-clone.

set -euo pipefail

MIRROR="${1:-}"

if [ -z "$MIRROR" ]; then
    echo "Usage: $0 /path/to/mirror.git"
    echo ""
    echo "Create a mirror first:"
    echo "  git clone --mirror https://github.com/automagik-dev/workit.git /tmp/workit-mirror.git"
    exit 1
fi

if [ ! -d "$MIRROR" ]; then
    echo "ERROR: $MIRROR does not exist"
    exit 1
fi

if ! command -v git-filter-repo &>/dev/null; then
    echo "ERROR: git-filter-repo not found. Install it:"
    echo "  pip install git-filter-repo"
    echo "  # or: brew install git-filter-repo"
    exit 1
fi

echo "=== Purging agent identity files from git history ==="
echo "Mirror: $MIRROR"
echo ""
echo "Files to remove from ALL commits:"
echo "  AGENT.md AGENTS.md BACKLOG.md CLAUDE.md ENVIRONMENT.md"
echo "  HEARTBEAT.md IDENTITY.md MILESTONES.md ROLE.md SOUL.md"
echo "  TODO.md TOOLS.md USER.md"
echo ""
read -r -p "Continue? [y/N] " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

cd "$MIRROR"

git filter-repo \
    --path AGENT.md \
    --path AGENTS.md \
    --path BACKLOG.md \
    --path CLAUDE.md \
    --path ENVIRONMENT.md \
    --path HEARTBEAT.md \
    --path IDENTITY.md \
    --path MILESTONES.md \
    --path ROLE.md \
    --path SOUL.md \
    --path TODO.md \
    --path TOOLS.md \
    --path USER.md \
    --invert-paths \
    --force

echo ""
echo "=== Cleaning up ==="
git reflog expire --expire=now --all
git gc --prune=now --aggressive

echo ""
echo "=== Verification ==="
count=$(git log --all --name-only --pretty=format: | grep -cE "^(AGENT|AGENTS|BACKLOG|CLAUDE|ENVIRONMENT|HEARTBEAT|IDENTITY|MILESTONES|ROLE|SOUL|TODO|TOOLS|USER)\.md$" || true)
echo "Agent files remaining in history: $count"

if [ "$count" -eq 0 ]; then
    echo "SUCCESS: All agent files purged from history."
    echo ""
    echo "Next steps:"
    echo "  cd $MIRROR"
    echo "  git remote add origin https://github.com/automagik-dev/workit.git  # if needed"
    echo "  git push --force"
else
    echo "WARNING: $count agent file references still found in history."
    exit 1
fi
