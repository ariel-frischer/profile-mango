#!/bin/sh
# profile-mango release installer.
#
# This script is for explicitly invoked, configured installations. It does not
# edit shell profiles, and it refuses to install unless a matching SHA-256
# checksum is downloaded and verified.
#
# Environment variables:
#   PROFILE_MANGO_INSTALL_DIR       Installation directory (default: ~/.local/bin)
#   PROFILE_MANGO_VERSION           Version/tag, or latest (default: latest)
#   PROFILE_MANGO_RELEASE_BASE_URL  Release downloads base URL
#   PROFILE_MANGO_LATEST_URL        Latest-release metadata URL
#   PROFILE_MANGO_*_CMD             Optional command paths for offline fixtures

set -eu

BINARY_NAME="profile-mango"
RELEASE_NAME="profile-mango"
GITHUB_REPO="ariel-frischer/profile-mango"
if [ -n "${HOME:-}" ]; then
    DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
else
    DEFAULT_INSTALL_DIR=""
fi
DEFAULT_RELEASE_BASE_URL="https://github.com/${GITHUB_REPO}/releases/download"
DEFAULT_LATEST_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

CURL_CMD="${PROFILE_MANGO_CURL_CMD:-${PROFILE_MANGO_CURL:-curl}}"
UNAME_CMD="${PROFILE_MANGO_UNAME_CMD:-${PROFILE_MANGO_UNAME:-uname}}"
TAR_CMD="${PROFILE_MANGO_TAR_CMD:-tar}"
MKTEMP_CMD="${PROFILE_MANGO_MKTEMP_CMD:-mktemp}"
MKDIR_CMD="${PROFILE_MANGO_MKDIR_CMD:-mkdir}"
CP_CMD="${PROFILE_MANGO_CP_CMD:-cp}"
MV_CMD="${PROFILE_MANGO_MV_CMD:-mv}"
RM_CMD="${PROFILE_MANGO_RM_CMD:-rm}"
CHMOD_CMD="${PROFILE_MANGO_CHMOD_CMD:-chmod}"
DATE_CMD="${PROFILE_MANGO_DATE_CMD:-date}"
SHA256SUM_CMD="${PROFILE_MANGO_SHA256SUM_CMD:-${PROFILE_MANGO_SHA256_CMD:-sha256sum}}"
SHASUM_CMD="${PROFILE_MANGO_SHASUM_CMD:-shasum}"
RELEASE_BASE_URL="${PROFILE_MANGO_RELEASE_BASE_URL:-${PROFILE_MANGO_BASE_URL:-$DEFAULT_RELEASE_BASE_URL}}"
LATEST_URL="${PROFILE_MANGO_LATEST_URL:-${PROFILE_MANGO_API_URL:-$DEFAULT_LATEST_URL}}"

TMP_DIR=""
STAGE_PATH=""
TARGET_BINARY=""
BACKUP_PATH=""
BACKUP_CREATED=0
REPLACEMENT_DONE=0
INSTALL_ACTIVE=0
INSTALL_VERIFIED=0

info() {
    printf '%s\n' "==> $*" >&2
}

success() {
    printf '%s\n' "==> $*" >&2
}

warn() {
    printf '%s\n' "Warning: $*" >&2
}

error() {
    printf '%s\n' "Error: $*" >&2
    exit 1
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

require_command() {
    if ! command_exists "$1"; then
        error "Required command not found: $1"
    fi
}

check_dependencies() {
    require_command "$CURL_CMD"
    require_command "$UNAME_CMD"
    require_command "$TAR_CMD"
    require_command "$MKTEMP_CMD"
    require_command "$MKDIR_CMD"
    require_command "$CP_CMD"
    require_command "$MV_CMD"
    require_command "$RM_CMD"
    require_command "$CHMOD_CMD"
    require_command "$DATE_CMD"
    require_command awk
    require_command grep
}

detect_os() {
    detected_os=$("$UNAME_CMD" -s) || error "Unable to detect operating system"
    case "$detected_os" in
        Linux) printf '%s\n' "linux" ;;
        Darwin) printf '%s\n' "darwin" ;;
        *) error "Unsupported operating system: ${detected_os} (supported: Linux, Darwin)" ;;
    esac
}

detect_arch() {
    detected_arch=$("$UNAME_CMD" -m) || error "Unable to detect architecture"
    case "$detected_arch" in
        x86_64|amd64) printf '%s\n' "amd64" ;;
        aarch64|arm64) printf '%s\n' "arm64" ;;
        *) error "Unsupported architecture: ${detected_arch} (supported: amd64, arm64)" ;;
    esac
}

validate_version() {
    version_to_check=$1
    if ! printf '%s\n' "$version_to_check" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'; then
        error "Invalid version '${version_to_check}'; expected semver such as v1.2.3"
    fi
}

get_latest_version() {
    if ! latest_payload=$("$CURL_CMD" -fsSL "$LATEST_URL"); then
        error "Failed to fetch latest version metadata"
    fi

    latest_version=$(printf '%s\n' "$latest_payload" | awk '
        {
            line = $0
            sub(/^.*"tag_name"[[:space:]]*:[[:space:]]*"/, "", line)
            if (line != $0) {
                sub(/".*$/, "", line)
                print line
                exit
            }
        }
    ')
    if [ -z "$latest_version" ]; then
        error "Latest version metadata did not contain tag_name"
    fi
    LATEST_VERSION="$latest_version"
}

resolve_version() {
    requested_version=$1
    if [ -z "$requested_version" ] || [ "$requested_version" = "latest" ]; then
        get_latest_version
        requested_version=$LATEST_VERSION
    fi
    validate_version "$requested_version"

    case "$requested_version" in
        v*) VERSION_TAG="$requested_version"; VERSION_NUMBER="${requested_version#v}" ;;
        *) VERSION_TAG="v${requested_version}"; VERSION_NUMBER="$requested_version" ;;
    esac
}

trim_trailing_slashes() {
    trimmed_url=$1
    while [ "${trimmed_url%/}" != "$trimmed_url" ]; do
        trimmed_url=${trimmed_url%/}
    done
    printf '%s\n' "$trimmed_url"
}

archive_name_for() {
    printf '%s_%s_%s_%s.tar.gz\n' "$RELEASE_NAME" "$1" "$2" "$3"
}

checksum_entry() {
    checksum_file=$1
    target_name=$2
    awk -v target="$target_name" '
        {
            name = $2
            sub(/^\*/, "", name)
            if (name == target) {
                if (NF != 2 || length($1) != 64 || $1 ~ /[^0-9A-Fa-f]/) {
                    invalid = 1
                } else {
                    count++
                    value = tolower($1)
                }
            }
        }
        END {
            if (invalid || count != 1) {
                exit 1
            }
            print value
        }
    ' "$checksum_file"
}

select_verifier() {
    if command_exists "$SHA256SUM_CMD"; then
        VERIFIER_KIND="sha256sum"
        VERIFIER_CMD="$SHA256SUM_CMD"
        return 0
    fi
    if command_exists "$SHASUM_CMD"; then
        VERIFIER_KIND="shasum"
        VERIFIER_CMD="$SHASUM_CMD"
        return 0
    fi
    error "No SHA-256 verifier found; install sha256sum or shasum"
}

verify_checksum() {
    archive_path_to_verify=$1
    checksum_path=$2
    archive_name_to_verify=$3

    if ! expected_checksum=$(checksum_entry "$checksum_path" "$archive_name_to_verify"); then
        error "Checksum entry for ${archive_name_to_verify} is missing or invalid"
    fi

    digest_output="${TMP_DIR}/digest.txt"
    if [ "$VERIFIER_KIND" = "sha256sum" ]; then
        if ! "$VERIFIER_CMD" "$archive_path_to_verify" > "$digest_output"; then
            error "SHA-256 verification command failed"
        fi
    else
        if ! "$VERIFIER_CMD" -a 256 "$archive_path_to_verify" > "$digest_output"; then
            error "SHA-256 verification command failed"
        fi
    fi

    actual_checksum=$(awk 'NF { print tolower($1); exit }' "$digest_output")
    if ! printf '%s\n' "$actual_checksum" | awk 'length($0) == 64 && $0 !~ /[^0-9a-f]/ { found = 1 } END { exit !found }'; then
        error "SHA-256 verifier returned an invalid digest"
    fi
    if [ "$expected_checksum" != "$actual_checksum" ]; then
        error "Checksum verification failed for ${archive_name_to_verify}"
    fi
    success "Checksum verified for ${archive_name_to_verify}"
}

download_and_verify() {
    download_version_number=$1
    download_version_tag=$2
    download_os=$3
    download_arch=$4

    download_archive_name=$(archive_name_for "$download_version_number" "$download_os" "$download_arch")
    download_base_url=$(trim_trailing_slashes "$RELEASE_BASE_URL")
    download_archive_url="${download_base_url}/${download_version_tag}/${download_archive_name}"
    download_checksum_url="${download_base_url}/${download_version_tag}/checksums.txt"
    DOWNLOAD_ARCHIVE_PATH="${TMP_DIR}/${download_archive_name}"
    checksum_path="${TMP_DIR}/checksums.txt"

    info "Downloading ${download_archive_name}"
    if ! "$CURL_CMD" -fsSL -o "$DOWNLOAD_ARCHIVE_PATH" "$download_archive_url"; then
        error "Failed to download ${download_archive_name}"
    fi
    if [ ! -s "$DOWNLOAD_ARCHIVE_PATH" ]; then
        error "Downloaded archive is empty: ${download_archive_name}"
    fi

    info "Downloading mandatory SHA-256 checksums"
    if ! "$CURL_CMD" -fsSL -o "$checksum_path" "$download_checksum_url"; then
        error "Failed to download mandatory checksums.txt"
    fi
    if [ ! -s "$checksum_path" ]; then
        error "Downloaded checksums.txt is empty"
    fi

    select_verifier
    verify_checksum "$DOWNLOAD_ARCHIVE_PATH" "$checksum_path" "$download_archive_name"
}

extract_binary() {
    archive_to_extract=$1
    if ! "$TAR_CMD" -xzf "$archive_to_extract" -C "$TMP_DIR"; then
        error "Failed to extract archive"
    fi

    EXTRACTED_BINARY_PATH="${TMP_DIR}/${BINARY_NAME}"
    if [ ! -f "$EXTRACTED_BINARY_PATH" ]; then
        error "Binary '${BINARY_NAME}' not found at archive root"
    fi
    if ! "$CHMOD_CMD" 755 "$EXTRACTED_BINARY_PATH"; then
        error "Unable to make extracted binary executable"
    fi
}

prepare_install_dir() {
    install_dir_to_prepare=$1
    if [ -z "$install_dir_to_prepare" ]; then
        error "Installation directory is empty"
    fi
    if [ ! -d "$install_dir_to_prepare" ] && ! "$MKDIR_CMD" -p "$install_dir_to_prepare"; then
        error "Unable to create installation directory: ${install_dir_to_prepare}"
    fi
    if [ ! -d "$install_dir_to_prepare" ] || [ ! -w "$install_dir_to_prepare" ]; then
        error "Installation directory is not writable: ${install_dir_to_prepare}"
    fi
}

stage_binary() {
    source_binary=$1
    install_dir_for_stage=$2
    STAGE_PATH=$("$MKTEMP_CMD" "${install_dir_for_stage}/.${BINARY_NAME}.tmp.XXXXXX") || error "Unable to create atomic staging file"
    if ! "$CP_CMD" "$source_binary" "$STAGE_PATH"; then
        error "Unable to copy binary into staging file"
    fi
    if ! "$CHMOD_CMD" 755 "$STAGE_PATH"; then
        error "Unable to make staged binary executable"
    fi
}

create_backup() {
    if [ ! -e "$TARGET_BINARY" ]; then
        return 0
    fi
    if [ ! -f "$TARGET_BINARY" ]; then
        error "Installation target is not a regular file: ${TARGET_BINARY}"
    fi

    backup_stamp=$("$DATE_CMD" +%Y%m%d%H%M%S)
    BACKUP_PATH="${TARGET_BINARY}.backup.${backup_stamp}.$$"
    backup_suffix=0
    while [ -e "$BACKUP_PATH" ]; do
        backup_suffix=$((backup_suffix + 1))
        BACKUP_PATH="${TARGET_BINARY}.backup.${backup_stamp}.$$.${backup_suffix}"
    done
    if ! "$CP_CMD" -p "$TARGET_BINARY" "$BACKUP_PATH"; then
        error "Unable to create rollback backup: ${BACKUP_PATH}"
    fi
    BACKUP_CREATED=1
    info "Created rollback backup: ${BACKUP_PATH}"
}

replace_binary() {
    TARGET_BINARY="$1"
    replacement_stage=$2
    INSTALL_ACTIVE=1
    create_backup
    if ! "$MV_CMD" "$replacement_stage" "$TARGET_BINARY"; then
        error "Unable to atomically replace ${TARGET_BINARY}"
    fi
    REPLACEMENT_DONE=1
}

rollback_install() {
    if [ "$INSTALL_ACTIVE" -ne 1 ]; then
        return 0
    fi

    if [ "$REPLACEMENT_DONE" -eq 1 ]; then
        if [ -e "$TARGET_BINARY" ] && ! "$RM_CMD" -f "$TARGET_BINARY"; then
            warn "Rollback could not remove failed binary: ${TARGET_BINARY}"
        fi
        if [ "$BACKUP_CREATED" -eq 1 ] && [ -e "$BACKUP_PATH" ]; then
            if "$MV_CMD" "$BACKUP_PATH" "$TARGET_BINARY"; then
                warn "Previous binary restored from rollback backup"
            else
                warn "Rollback failed while restoring ${BACKUP_PATH}"
            fi
        fi
    elif [ "$BACKUP_CREATED" -eq 1 ] && [ -e "$BACKUP_PATH" ]; then
        "$RM_CMD" -f "$BACKUP_PATH" || true
    fi
    INSTALL_ACTIVE=0
}

cleanup() {
    cleanup_status=$?
    trap - 0
    if [ "$INSTALL_VERIFIED" -ne 1 ]; then
        rollback_install || true
    fi
    if [ -n "$STAGE_PATH" ] && [ -e "$STAGE_PATH" ]; then
        "$RM_CMD" -f "$STAGE_PATH" || true
    fi
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        "$RM_CMD" -rf "$TMP_DIR" || true
    fi
    exit "$cleanup_status"
}

interrupted() {
    warn "Installation interrupted; cleaning up and restoring the previous binary if needed"
    exit 130
}

path_contains_dir() {
    case ":${PATH:-}:" in
        *":$1:"*) return 0 ;;
        *) return 1 ;;
    esac
}

show_path_guidance() {
    guidance_dir=$1
    if path_contains_dir "$guidance_dir"; then
        success "${guidance_dir} is already in PATH"
        return 0
    fi
    warn "${guidance_dir} is not in PATH"
    printf '%s\n' "Add it for the current shell with:" >&2
    printf "  export PATH=\"%s:\$PATH\"\n" "$guidance_dir" >&2
    printf '%s\n' "The installer does not edit shell profile files." >&2
}

show_success() {
    installed_version_output=$("$TARGET_BINARY" --version 2>/dev/null || true)
    success "Installed ${BINARY_NAME} ${VERSION_TAG} at ${TARGET_BINARY}"
    if [ -n "$installed_version_output" ]; then
        info "Installed binary reports: ${installed_version_output}"
    fi
    show_path_guidance "$INSTALL_DIR"
    printf '%s\n' "Verify with: ${BINARY_NAME} --version" >&2
    printf '%s\n' "Get started with: ${BINARY_NAME} --help" >&2
}

main() {
    check_dependencies

    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    requested_version="${PROFILE_MANGO_VERSION:-latest}"
    resolve_version "$requested_version"
    info "Installing version: ${VERSION_TAG}"

    INSTALL_DIR="${PROFILE_MANGO_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
    prepare_install_dir "$INSTALL_DIR"
    info "Install directory: ${INSTALL_DIR}"

    TMP_DIR=$("$MKTEMP_CMD" -d "${TMPDIR:-/tmp}/profile-mango.XXXXXX") || error "Unable to create temporary directory"
    trap cleanup 0
    trap interrupted HUP INT TERM

    download_and_verify "$VERSION_NUMBER" "$VERSION_TAG" "$os" "$arch"
    extract_binary "$DOWNLOAD_ARCHIVE_PATH"
    stage_binary "$EXTRACTED_BINARY_PATH" "$INSTALL_DIR"
    replace_binary "${INSTALL_DIR}/${BINARY_NAME}" "$STAGE_PATH"

    info "Verifying installed binary"
    if ! "$TARGET_BINARY" --version >/dev/null 2>&1; then
        error "Installed binary failed --version verification"
    fi
    INSTALL_VERIFIED=1
    INSTALL_ACTIVE=0
    show_success
}

if [ "${PROFILE_MANGO_INSTALLER_NO_MAIN:-0}" != "1" ]; then
    main "$@"
fi
