#!/bin/sh
# Offline fixture tests for install.sh.

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH='' cd -- "${SCRIPT_DIR}/.." && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/profile-mango-installer-tests.XXXXXX")
FIXTURE_ROOT="${TEST_ROOT}/fixtures"
RELEASE_ROOT="${FIXTURE_ROOT}/releases"
CURL_LOG="${TEST_ROOT}/curl.log"
mkdir -p "$RELEASE_ROOT" "$TEST_ROOT/bin"
: > "$CURL_LOG"

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

assert_equal() {
    expected=$1
    actual=$2
    description=$3
    if [ "$expected" != "$actual" ]; then
        fail "${description}: expected '${expected}', got '${actual}'"
    fi
}

assert_file_contains() {
    file=$1
    needle=$2
    description=$3
    if ! grep -F "$needle" "$file" >/dev/null 2>&1; then
        fail "${description}: '${needle}' not found in ${file}"
    fi
}

assert_file_not_contains() {
    file=$1
    needle=$2
    description=$3
    if grep -F "$needle" "$file" >/dev/null 2>&1; then
        fail "${description}: unexpected '${needle}' found in ${file}"
    fi
}

assert_file_equals() {
    expected_file=$1
    actual_file=$2
    description=$3
    if ! diff -u "$expected_file" "$actual_file" >/dev/null 2>&1; then
        fail "${description}: files differ"
    fi
}

digest_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

cat > "$TEST_ROOT/fake-uname" <<'EOF_UNAME'
#!/bin/sh
set -eu
case "${1:-}" in
    -s) printf '%s\n' "${FIXTURE_OS}" ;;
    -m) printf '%s\n' "${FIXTURE_ARCH}" ;;
    *) exit 2 ;;
esac
EOF_UNAME
chmod +x "$TEST_ROOT/fake-uname"

cat > "$TEST_ROOT/fake-curl" <<'EOF_CURL'
#!/bin/sh
set -eu
output=''
url=''
while [ "$#" -gt 0 ]; do
    case "$1" in
        -o|--output)
            output=$2
            shift 2
            ;;
        -*)
            shift
            ;;
        *)
            url=$1
            shift
            ;;
    esac
done
printf '%s\n' "$url" >> "$PROFILE_MANGO_CURL_LOG"

if [ "$url" = "fixture://latest" ]; then
    if [ "${FIXTURE_LATEST_FAIL:-0}" = "1" ]; then
        exit 22
    fi
    printf '{"tag_name":"%s"}\n' "${FIXTURE_LATEST_VERSION}"
    exit 0
fi

case "$url" in
    fixture://releases/*)
        relative_path=${url#fixture://releases/}
        case "$relative_path" in
            */checksums.txt)
                if [ "${FIXTURE_FAIL_CHECKSUM:-0}" = "1" ]; then
                    exit 22
                fi
                ;;
        esac
        if [ "${FIXTURE_INTERRUPT:-0}" = "1" ] &&
            [ "$relative_path" != */checksums.txt ]; then
            kill -TERM "$PPID"
            exit 143
        fi
        source_path="${FIXTURE_ROOT}/releases/${relative_path}"
        [ -f "$source_path" ] || exit 22
        if [ -n "$output" ]; then
            cp "$source_path" "$output"
        else
            cat "$source_path"
        fi
        ;;
    *)
        exit 22
        ;;
esac
EOF_CURL
chmod +x "$TEST_ROOT/fake-curl"

export FIXTURE_ROOT
export PROFILE_MANGO_CURL_CMD="$TEST_ROOT/fake-curl"
export PROFILE_MANGO_UNAME_CMD="$TEST_ROOT/fake-uname"
export PROFILE_MANGO_RELEASE_BASE_URL='fixture://releases/'
export PROFILE_MANGO_LATEST_URL='fixture://latest'
export PROFILE_MANGO_CURL_LOG="$CURL_LOG"
export FIXTURE_LATEST_VERSION='v1.2.3'

make_binary() {
    binary_version=$1
    binary_mode=$2
    binary_dir=$3
    mkdir -p "$binary_dir"
    if [ "$binary_mode" = "bad" ]; then
        cat > "${binary_dir}/mango" <<EOF_BAD
#!/bin/sh
if [ "\${1:-}" = "--version" ]; then
    exit 1
fi
exit 0
EOF_BAD
    else
        cat > "${binary_dir}/mango" <<EOF_GOOD
#!/bin/sh
if [ "\${1:-}" = "--version" ]; then
    printf '%s\\n' 'mango ${binary_version}'
    exit 0
fi
printf '%s\\n' 'fixture help'
EOF_GOOD
    fi
    chmod +x "${binary_dir}/mango"
}

create_release() {
    release_number=$1
    release_mode=$2
    release_dir="${RELEASE_ROOT}/v${release_number}"
    rm -rf "$release_dir"
    mkdir -p "$release_dir"
    : > "${release_dir}/checksums.txt"

    for release_os in linux darwin; do
        for release_arch in amd64 arm64; do
            archive_name="profile-mango_${release_number}_${release_os}_${release_arch}.tar.gz"
            binary_dir="${TEST_ROOT}/binaries/${release_number}-${release_os}-${release_arch}"
            make_binary "$release_number" "$release_mode" "$binary_dir"
            tar -czf "${release_dir}/${archive_name}" -C "$binary_dir" mango
            release_digest=$(digest_file "${release_dir}/${archive_name}")
            case "$release_mode" in
                mismatch)
                    release_digest='0000000000000000000000000000000000000000000000000000000000000000'
                    ;;
                missing-entry)
                    continue
                    ;;
            esac
            printf '%s  %s\n' "$release_digest" "$archive_name" >> "${release_dir}/checksums.txt"
        done
    done

    if [ "$release_mode" = "missing-entry" ]; then
        printf '%s  %s\n' \
            '0000000000000000000000000000000000000000000000000000000000000000' \
            'profile-mango_other_linux_amd64.tar.gz' > "${release_dir}/checksums.txt"
    fi
}

new_case() {
    CASE_NAME=$1
    CASE_DIR="${TEST_ROOT}/cases/${CASE_NAME}"
    rm -rf "$CASE_DIR"
    mkdir -p "$CASE_DIR/home" "$CASE_DIR/tmp"
    export HOME="$CASE_DIR/home"
    export TMPDIR="$CASE_DIR/tmp"
    : > "$CURL_LOG"
    unset FIXTURE_FAIL_CHECKSUM FIXTURE_LATEST_FAIL FIXTURE_INTERRUPT
    unset PROFILE_MANGO_SHA256SUM_CMD PROFILE_MANGO_SHASUM_CMD
}

run_installer() {
    expected_status=$1
    if sh "$ROOT_DIR/install.sh" > "${CASE_DIR}/stdout" 2> "${CASE_DIR}/stderr"; then
        installer_status=0
    else
        installer_status=$?
    fi
    assert_equal "$expected_status" "$installer_status" "${CASE_NAME} installer status"
}

assert_no_temp_entries() {
    temp_entries=$(find "$TMPDIR" -mindepth 1 -maxdepth 1 -print)
    if [ -n "$temp_entries" ]; then
        fail "${CASE_NAME}: temporary directory was not cleaned: ${temp_entries}"
    fi
}

create_release 1.2.3 good
create_release 1.2.4 bad
create_release 1.2.5 mismatch
create_release 1.2.6 missing-entry

assert_file_contains "$ROOT_DIR/.goreleaser.yaml" \
    'name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"' \
    'GoReleaser archive naming contract'
assert_file_contains "$ROOT_DIR/.goreleaser.yaml" \
    'name_template: checksums.txt' \
    'GoReleaser checksum naming contract'
assert_file_not_contains "$ROOT_DIR/install.sh" \
    'curl -fsSL https://gitlab.com' \
    'private-safe installer header'

run_platform_case() {
    platform_name=$1
    fixture_os=$2
    fixture_arch=$3
    canonical_os=$4
    canonical_arch=$5
    new_case "$platform_name"
    export FIXTURE_OS="$fixture_os"
    export FIXTURE_ARCH="$fixture_arch"
    export PROFILE_MANGO_VERSION='1.2.3'
    export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/custom-bin"
    run_installer 0
    [ -x "${PROFILE_MANGO_INSTALL_DIR}/mango" ] || fail "${CASE_NAME}: binary was not installed"
    [ -e "${PROFILE_MANGO_INSTALL_DIR}/profile-mango" ] || fail "${CASE_NAME}: profile-mango alias was not installed"
    [ -x "${PROFILE_MANGO_INSTALL_DIR}/profile-mango" ] || fail "${CASE_NAME}: profile-mango alias is not executable"
    expected_archive="profile-mango_1.2.3_${canonical_os}_${canonical_arch}.tar.gz"
    assert_file_contains "$CURL_LOG" \
        "fixture://releases/v1.2.3/${expected_archive}" \
        "${CASE_NAME} archive URL"
    assert_file_contains "$CURL_LOG" \
        'fixture://releases/v1.2.3/checksums.txt' \
        "${CASE_NAME} checksum URL"
    assert_file_contains "${CASE_DIR}/stderr" \
        'Installed binary reports: mango 1.2.3' \
        "${CASE_NAME} version diagnostic"
    assert_file_contains "${PROFILE_MANGO_INSTALL_DIR}/profile-mango" \
        'mango 1.2.3' \
        "${CASE_NAME} alias runs the same binary"
}

run_platform_case linux-amd64 Linux x86_64 linux amd64
run_platform_case linux-arm64 Linux aarch64 linux arm64
run_platform_case darwin-amd64 Darwin x86_64 darwin amd64
run_platform_case darwin-arm64 Darwin arm64 darwin arm64

new_case latest-version
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
unset PROFILE_MANGO_VERSION
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/latest-bin"
run_installer 0
assert_file_contains "$CURL_LOG" 'fixture://latest' 'latest metadata URL'
assert_file_contains "$CURL_LOG" \
    'fixture://releases/v1.2.3/profile-mango_1.2.3_linux_amd64.tar.gz' \
    'latest resolved archive URL'

new_case default-install-dir
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.3
unset PROFILE_MANGO_INSTALL_DIR
printf '%s\n' 'profile sentinel' > "$HOME/.profile"
cp "$HOME/.profile" "$CASE_DIR/profile.before"
run_installer 0
[ -x "$HOME/.local/bin/mango" ] || fail 'default install directory was not used'
[ -e "$HOME/.local/bin/profile-mango" ] || fail 'default install directory did not install the profile-mango alias'
assert_file_equals "$CASE_DIR/profile.before" "$HOME/.profile" 'shell profile was not modified'
assert_file_contains "${CASE_DIR}/stderr" \
    'The installer does not edit shell profile files.' \
    'no-profile-edit diagnostic'

new_case existing-backup
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.3
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
mkdir -p "$PROFILE_MANGO_INSTALL_DIR"
printf '%s\n' 'previous installed binary' > "$PROFILE_MANGO_INSTALL_DIR/mango"
run_installer 0
assert_file_contains "$PROFILE_MANGO_INSTALL_DIR/mango" \
    'mango 1.2.3' \
    'successful replacement installed new binary'
backup_files=$(find "$PROFILE_MANGO_INSTALL_DIR" -name 'mango.backup.*' -type f -print)
[ -n "$backup_files" ] || fail 'successful replacement did not retain rollback backup'
assert_file_contains "$backup_files" \
    'previous installed binary' \
    'rollback backup preserved previous binary'
[ -e "$PROFILE_MANGO_INSTALL_DIR/profile-mango" ] || fail 'successful replacement did not install the profile-mango alias'

new_case checksum-mismatch
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.5
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
mkdir -p "$PROFILE_MANGO_INSTALL_DIR"
printf '%s\n' 'old binary' > "$PROFILE_MANGO_INSTALL_DIR/mango"
run_installer 1
assert_file_contains "${CASE_DIR}/stderr" 'Checksum verification failed' 'checksum mismatch failure'
assert_file_contains "$PROFILE_MANGO_INSTALL_DIR/mango" 'old binary' 'checksum failure preserved target'
assert_no_temp_entries

new_case missing-checksum-entry
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.6
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
run_installer 1
assert_file_contains "${CASE_DIR}/stderr" \
    'Checksum entry for profile-mango_1.2.6_linux_amd64.tar.gz is missing or invalid' \
    'missing checksum entry failure'
assert_no_temp_entries

new_case checksum-download-failure
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.3
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
export FIXTURE_FAIL_CHECKSUM=1
run_installer 1
assert_file_contains "${CASE_DIR}/stderr" \
    'Failed to download mandatory checksums.txt' \
    'checksum download failure'
assert_no_temp_entries

new_case missing-verifier
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.3
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
export PROFILE_MANGO_SHA256SUM_CMD="${TEST_ROOT}/missing-sha256sum"
export PROFILE_MANGO_SHASUM_CMD="${TEST_ROOT}/missing-shasum"
run_installer 1
assert_file_contains "${CASE_DIR}/stderr" \
    'No SHA-256 verifier found' \
    'missing verifier failure'
assert_no_temp_entries

new_case verification-rollback
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.4
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
mkdir -p "$PROFILE_MANGO_INSTALL_DIR"
printf '%s\n' 'previous binary' > "$PROFILE_MANGO_INSTALL_DIR/mango"
run_installer 1
assert_file_contains "$PROFILE_MANGO_INSTALL_DIR/mango" \
    'previous binary' \
    'verification failure restored previous binary'
assert_file_contains "${CASE_DIR}/stderr" \
    'Previous binary restored from rollback backup' \
    'rollback diagnostic'
assert_no_temp_entries

new_case latest-failure
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
unset PROFILE_MANGO_VERSION
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
export FIXTURE_LATEST_FAIL=1
run_installer 1
assert_file_contains "${CASE_DIR}/stderr" \
    'Failed to fetch latest version metadata' \
    'latest metadata failure'

new_case unsupported-os
export FIXTURE_OS=FreeBSD
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.3
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
run_installer 1
assert_file_contains "${CASE_DIR}/stderr" \
    'Unsupported operating system: FreeBSD' \
    'unsupported operating system failure'

new_case unsupported-arch
export FIXTURE_OS=Linux
export FIXTURE_ARCH=ppc64le
export PROFILE_MANGO_VERSION=1.2.3
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
run_installer 1
assert_file_contains "${CASE_DIR}/stderr" \
    'Unsupported architecture: ppc64le' \
    'unsupported architecture failure'

new_case interrupted
export FIXTURE_OS=Linux
export FIXTURE_ARCH=x86_64
export PROFILE_MANGO_VERSION=1.2.3
export PROFILE_MANGO_INSTALL_DIR="${CASE_DIR}/bin"
export FIXTURE_INTERRUPT=1
mkdir -p "$PROFILE_MANGO_INSTALL_DIR"
printf '%s\n' 'interrupted previous binary' > "$PROFILE_MANGO_INSTALL_DIR/mango"
run_installer 130
assert_file_contains "$PROFILE_MANGO_INSTALL_DIR/mango" \
    'interrupted previous binary' \
    'interrupted install preserved target'
assert_no_temp_entries

printf '%s\n' 'PASS: offline installer fixture suite'
