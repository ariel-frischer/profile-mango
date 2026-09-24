#!/bin/sh
set -eu

repo=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
dest="$tmp/skills/profile-mango/SKILL.md"
source="$repo/.agents/skills/profile-mango/SKILL.md"

make -s -C "$repo" link-skill SKILL_DEST="$dest"
test -L "$dest"
test "$(readlink "$dest")" = "$(git -C "$repo" worktree list --porcelain | sed -n '1s/^worktree //p')/.agents/skills/profile-mango/SKILL.md"
test -f "$source"

make -s -C "$repo" link-skill SKILL_DEST="$dest"
test "$(find "$tmp/skills/profile-mango" -maxdepth 1 -name 'SKILL.md.backup.*' | wc -l)" -eq 0

rm "$dest"
printf 'old global skill\n' >"$dest"
make -s -C "$repo" link-skill SKILL_DEST="$dest"
test -L "$dest"
backup=$(find "$tmp/skills/profile-mango" -maxdepth 1 -name 'SKILL.md.backup.*' -type d)
test -n "$backup"
test "$(cat "$backup/SKILL.md")" = 'old global skill'

make -s -C "$repo" link-skill SKILL_DEST="$dest"
test "$(find "$tmp/skills/profile-mango" -maxdepth 1 -name 'SKILL.md.backup.*' | wc -l)" -eq 1

rm "$dest"
ln -s "$tmp/other-skill" "$dest"
make -s -C "$repo" link-skill SKILL_DEST="$dest"
test "$(find "$tmp/skills/profile-mango" -maxdepth 1 -name 'SKILL.md.backup.*' | wc -l)" -eq 2
test "$(find "$tmp/skills/profile-mango" -maxdepth 2 -name SKILL.md -type l | wc -l)" -eq 2

rm "$dest"
mkdir "$dest"
if make -s -C "$repo" link-skill SKILL_DEST="$dest" >/dev/null 2>&1; then
  echo 'link-skill replaced a directory' >&2
  exit 1
fi
test -d "$dest"
printf 'skill link tests passed\n'
