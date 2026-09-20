#!/usr/bin/env bash
set -euo pipefail

BRANCH="${1:?Usage: worktree-setup.sh <branch-name> [base-branch]}"
BASE="${2:-HEAD}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKTREE_DIR="$REPO_ROOT/.worktrees/$BRANCH"

sync_agent_context() {
  local path
  for path in skills .agents .opencode .claude; do
    [ -e "$REPO_ROOT/$path" ] || continue
    [ ! -e "$WORKTREE_DIR/$path" ] || continue
    if command -v rsync >/dev/null 2>&1; then
      rsync -a --exclude '.git' --exclude '.beads' --exclude '.env' \
        --exclude '.env.*' --exclude 'bin' --exclude 'dist' --exclude 'build' \
        "$REPO_ROOT/$path" "$WORKTREE_DIR/"
    else
      cp -R "$REPO_ROOT/$path" "$WORKTREE_DIR/"
    fi
  done
}

link_local_state() {
  local path
  for path in .beads .codegraph; do
    [ -e "$REPO_ROOT/$path" ] || continue
    [ -e "$WORKTREE_DIR/$path" ] || [ -L "$WORKTREE_DIR/$path" ] || \
      ln -s "$REPO_ROOT/$path" "$WORKTREE_DIR/$path"
  done

  local exclude_file
  exclude_file="$(cd "$WORKTREE_DIR" && git rev-parse --git-path info/exclude)"
  mkdir -p "$(dirname "$exclude_file")"
  grep -qxF '.beads' "$exclude_file" 2>/dev/null || printf '\n.beads\n' >>"$exclude_file"
  grep -qxF '.codegraph' "$exclude_file" 2>/dev/null || printf '.codegraph\n' >>"$exclude_file"
}

setup_project() {
  (cd "$WORKTREE_DIR" && go mod download)
}

cd "$REPO_ROOT"
mkdir -p "$(dirname "$WORKTREE_DIR")"

if [ ! -d "$WORKTREE_DIR" ]; then
  git worktree add "$WORKTREE_DIR" -b "$BRANCH" "$BASE" 2>/dev/null || \
    git worktree add "$WORKTREE_DIR" "$BRANCH"
fi

link_local_state
sync_agent_context
setup_project
printf '%s\n' "$WORKTREE_DIR"
