#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>"
  echo "  e.g. $0 v0.1.0"
  exit 1
fi

SEMVER_REGEX='^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
if [[ ! "$VERSION" =~ $SEMVER_REGEX ]]; then
  echo "Error: VERSION must be semver (e.g. v1.2.3 or 1.2.3)"
  exit 1
fi

# Strip leading v for bare semver
BARE_VERSION="${VERSION#v}"
# Ensure tag has v prefix
TAG="v${BARE_VERSION}"
# Remote that hosts the GitHub repository; its tag push triggers the release workflow.
REMOTE="${RELEASE_REMOTE:-origin}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1"
    exit 1
  fi
}

echo "==> Pre-flight checks..."
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Error: working tree is dirty"
  exit 1
fi

if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "Error: local tag already exists: ${TAG}"
  exit 1
fi

if git ls-remote --exit-code --tags "${REMOTE}" "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "Error: remote tag already exists: ${TAG}"
  exit 1
fi

require_cmd goreleaser

echo "==> Running tests..."
make test

echo "==> Running lint..."
make lint

echo "==> Building..."
make build

echo "==> Checking goreleaser config..."
goreleaser check
require_cmd chlog

echo "==> Validating changelog..."
chlog validate
chlog check

mkdir -p .release
echo "==> Checking unreleased entries..."
# Plain entry bullets are indented four spaces; version/category headers are not.
# Drain the output instead of using grep -q so pipefail cannot see an early SIGPIPE.
if chlog show unreleased --plain 2>/dev/null | grep -E '^    - ' >/dev/null; then
  echo "==> Stamping changelog: ${BARE_VERSION}..."
  chlog release "${BARE_VERSION}"

  echo "==> Syncing CHANGELOG.md..."
  chlog sync

  echo "==> Checking stamped changelog..."
  chlog validate
  chlog check

  echo "==> Committing changelog..."
  git add CHANGELOG.yaml CHANGELOG.md
  git commit -m "release: ${TAG}"
fi

echo "==> Checking release notes..."
chlog extract "${BARE_VERSION}" > .release/notes.md
if [[ ! -s .release/notes.md ]]; then
  echo "Error: release notes are empty for ${BARE_VERSION}"
  exit 1
fi

echo "==> Running goreleaser snapshot..."
goreleaser release --snapshot --clean --skip=publish

echo "==> Tagging ${TAG}..."
git tag -a "${TAG}" -m "Release ${TAG}"

echo "==> Pushing to ${REMOTE}..."
git push "${REMOTE}" main
git push "${REMOTE}" "${TAG}"

echo ""
echo "Done! ${TAG} tagged and pushed."
echo "The tag push triggers the GitHub Actions release workflow (.github/workflows/release.yml)."
echo ""
echo "Next steps:"
echo "  Watch the release:   gh run watch"
echo "  View release:        gh release view ${TAG}"
