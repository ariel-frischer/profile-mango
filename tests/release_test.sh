#!/bin/sh
# Offline release fixtures using real chlog and disposable Git repositories.
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH='' cd -- "${SCRIPT_DIR}/.." && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/profile-mango-release-tests.XXXXXX")
REAL_GIT=$(command -v git)
export REAL_GIT

cleanup() {
    cleanup_status=$?
    trap - 0
    rm -rf "$TEST_ROOT"
    exit "$cleanup_status"
}
trap cleanup 0
trap 'exit 1' HUP INT TERM

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}

mkdir -p "$TEST_ROOT/bin"
cat > "$TEST_ROOT/bin/git" <<'EOF_GIT'
#!/bin/sh
set -eu
case "$1" in
    ls-remote) exit 2 ;;
    push) printf '%s\n' "$*" >> "$PUBLICATION_LOG" ;;
    *) exec "$REAL_GIT" "$@" ;;
esac
EOF_GIT
# Build/lint/snapshot validation is outside this fixture's scope.
cat > "$TEST_ROOT/bin/make" <<'EOF_MAKE'
#!/bin/sh
exit 0
EOF_MAKE
cat > "$TEST_ROOT/bin/goreleaser" <<'EOF_GORELEASER'
#!/bin/sh
exit 0
EOF_GORELEASER
chmod +x "$TEST_ROOT/bin/git" "$TEST_ROOT/bin/make" "$TEST_ROOT/bin/goreleaser"

run_case() {
    name=$1
    fixture="$TEST_ROOT/$name"
    mkdir -p "$fixture"
    (
        cd "$fixture"
        export PATH="$TEST_ROOT/bin:$PATH"
        export PUBLICATION_LOG="$TEST_ROOT/$name-publication.log"
        output_log="$TEST_ROOT/$name-output.log"
        git init -q -b main
        git config user.name 'Release fixture'
        git config user.email 'release-fixture@example.invalid'
        git config commit.gpgsign false
        git config tag.gpgsign false
        printf 'project: release-fixture\nversions:\n' > CHANGELOG.yaml
        case "$name" in
            stamped)
                cat >> CHANGELOG.yaml <<'EOF_STAMPED'
  unreleased: {}
  1.2.3:
    date: "2026-10-01"
    fixed:
      - Already stamped release note
EOF_STAMPED
                ;;
            unstamped)
                cat >> CHANGELOG.yaml <<'EOF_UNSTAMPED'
  unreleased:
    removed:
      - Normal unstamped release note
EOF_UNSTAMPED
                ;;
            empty)
                printf '  unreleased: {}\n' >> CHANGELOG.yaml
                ;;
        esac
        chlog sync >/dev/null
        git add CHANGELOG.yaml CHANGELOG.md
        git commit -qm 'Fixture changelog'
        before=$(git rev-parse HEAD)

        if bash "$ROOT_DIR/scripts/release.sh" v1.2.3 > "$output_log" 2>&1; then
            [ "$name" != empty ] || fail 'empty changelog released without notes'
        else
            [ "$name" = empty ] || {
                cat "$output_log" >&2
                fail "$name release failed"
            }
        fi

        after=$(git rev-parse HEAD)
        case "$name" in
            stamped)
                [ "$before" = "$after" ] || fail 'already-stamped notes were committed again'
                grep -F 'Already stamped release note' .release/notes.md >/dev/null || fail 'stamped notes missing'
                ;;
            unstamped)
                [ "$before" != "$after" ] || fail 'unstamped notes were not committed'
                [ "$(git log -1 --format=%s)" = 'release: v1.2.3' ] || fail 'wrong release commit'
                grep -F 'Normal unstamped release note' .release/notes.md >/dev/null || fail 'new notes missing'
                chlog validate >/dev/null
                chlog check >/dev/null
                ;;
            empty)
                [ "$before" = "$after" ] || fail 'empty changelog was committed'
                if git rev-parse -q --verify refs/tags/v1.2.3 >/dev/null; then
                    fail 'empty changelog was tagged'
                fi
                [ ! -e "$PUBLICATION_LOG" ] || fail 'empty changelog was pushed'
                if grep -F 'Stamping changelog' "$output_log" >/dev/null; then
                    fail 'empty Unreleased header was treated as entries'
                fi
                printf 'PASS: %s\n' "$name"
                exit 0
                ;;
        esac
        [ "$(git rev-parse 'v1.2.3^{commit}')" = "$after" ] || fail 'tag does not reference release commit'
        grep -Fx 'push origin main' "$PUBLICATION_LOG" >/dev/null || fail 'branch publication missing'
        grep -Fx 'push origin v1.2.3' "$PUBLICATION_LOG" >/dev/null || fail 'tag publication missing'
        printf 'PASS: %s\n' "$name"
    )
}

run_case stamped
run_case unstamped
run_case empty
