#!/bin/sh
# profile-mango uninstaller for profile-mango releases
set -eu

BINARY_NAME="mango"
ALIAS_NAME="profile-mango"
INSTALL_DIR="${PROFILE_MANGO_INSTALL_DIR:-$HOME/.local/bin}"
TARGET="${INSTALL_DIR}/${BINARY_NAME}"
ALIAS_TARGET="${INSTALL_DIR}/${ALIAS_NAME}"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\e[0;31m'
    GREEN='\e[0;32m'
    YELLOW='\e[0;33m'
    NC='\e[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
fi

if [ ! -f "$TARGET" ]; then
    printf '%bError:%b %s not found at %s\n' "${RED}" "${NC}" "$BINARY_NAME" "$TARGET" >&2
    exit 1
fi

printf 'Found %s at %s\n' "$BINARY_NAME" "$TARGET"
printf 'Remove this binary? [y/N] '
read -r answer
case "$answer" in
    y|Y|yes|YES) ;;
    *) echo "Aborted."; exit 0 ;;
esac

rm -f "$TARGET"

alias_removed=0
if [ -e "$ALIAS_TARGET" ] || [ -L "$ALIAS_TARGET" ]; then
    rm -f "$ALIAS_TARGET"
    alias_removed=1
fi

# Clean up backups
backup_count=0
for f in "${TARGET}.backup."*; do
    [ -e "$f" ] || continue
    rm -f "$f"
    backup_count=$((backup_count + 1))
done

printf '%b==>%b %s has been removed from %s\n' "${GREEN}" "${NC}" "$BINARY_NAME" "$INSTALL_DIR"
if [ "$alias_removed" -eq 1 ]; then
    printf '%b==>%b %s compatibility alias has been removed from %s\n' "${GREEN}" "${NC}" "$ALIAS_NAME" "$INSTALL_DIR"
fi
if [ "$backup_count" -gt 0 ]; then
    printf '%b==>%b Cleaned up %d backup(s)\n' "${GREEN}" "${NC}" "$backup_count"
fi
