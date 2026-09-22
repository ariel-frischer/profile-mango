#!/usr/bin/env bash
set -euo pipefail

readonly expected_source_commit='345cd2b057a452236de401d3534b8502a7465e8d'
readonly expected_source_tree='6e14b9791cdc5a47068685e9429dd5d6bdc5ef5f'
readonly expected_archive_sha256='71f2db39a64fbba282e3bd3be4b0f7b935585948a59a368d61deeec0f0827c47'
readonly expected_config_module_sha256='d76471ce54d40e68165e2cce7c2ade9c2164ed5ce4dbcf673b1b289cb89c7d84'
readonly expected_version_module_sha256='0d78a58a9f27f32adfdac959e89767cecde93a424bcfbd39d64fb95f6cf13e6c'
readonly default_timeout_seconds=45

source_root=''
source_archive=''
python_path=''
site_packages=''
state_root=''
timeout_seconds="$default_timeout_seconds"
config_relative='home/.hermes/config.yaml'

fail() {
	printf 'Hermes native probe error: %s\n' "$*" >&2
	exit 1
}

usage() {
	cat >&2 <<'EOF'
Usage: hermes_native_probe.sh --source-root DIR --source-archive FILE \
  --python FILE --site-packages DIR --state-root DIR [--config-relative PATH] [--timeout SECONDS]

The state root must contain the generated Hermes config at the relative config path.
The script leaves the result at STATE_ROOT/result/hermes-native-result.json.
EOF
	exit 2
}

while (($# > 0)); do
	case "$1" in
		--source-root) source_root="${2:?missing --source-root value}"; shift 2 ;;
		--source-archive) source_archive="${2:?missing --source-archive value}"; shift 2 ;;
		--python) python_path="${2:?missing --python value}"; shift 2 ;;
		--site-packages) site_packages="${2:?missing --site-packages value}"; shift 2 ;;
		--state-root) state_root="${2:?missing --state-root value}"; shift 2 ;;
		--config-relative) config_relative="${2:?missing --config-relative value}"; shift 2 ;;
		--timeout) timeout_seconds="${2:?missing --timeout value}"; shift 2 ;;
		-h|--help) usage ;;
		*) usage ;;
	esac
done

require_absolute() {
	local name="$1"
	local path="$2"
	[[ -n "$path" && "$path" == /* ]] || fail "$name must be an absolute path"
}

require_dir() {
	local name="$1"
	local path="$2"
	require_absolute "$name" "$path"
	[[ -d "$path" ]] || fail "$name is not a directory: $path"
}

require_file() {
	local name="$1"
	local path="$2"
	require_absolute "$name" "$path"
	[[ -f "$path" ]] || fail "$name is not a file: $path"
}

require_absolute source_root "$source_root"
require_absolute source_archive "$source_archive"
require_absolute python_path "$python_path"
require_absolute site_packages "$site_packages"
require_absolute state_root "$state_root"
[[ "$timeout_seconds" =~ ^[1-9][0-9]*$ ]] || fail "timeout must be a positive integer"
[[ "$config_relative" != /* && "$config_relative" != .. && "$config_relative" != ../* && "$config_relative" != */../* ]] || fail "config path escapes state root"

source_root="$(readlink -f -- "$source_root")"
source_archive="$(readlink -f -- "$source_archive")"
python_path="$(readlink -f -- "$python_path")"
site_packages="$(readlink -f -- "$site_packages")"
state_root="$(readlink -f -- "$state_root")"
require_dir source_root "$source_root"
require_file source_archive "$source_archive"
require_file python "$python_path"
require_dir site_packages "$site_packages"
require_dir state_root "$state_root"

config_path="$state_root/$config_relative"
require_file generated_config "$config_path"
[[ ! -L "$config_path" ]] || fail "generated config must not be a symlink"
mkdir -p -- "$state_root/result" "$state_root/managed" "$state_root/xdg" "$state_root/cache" "$state_root/data"

assert_tree_links_contained() {
	local label="$1"
	local root="$2"
	local link target
	while IFS= read -r -d '' link; do
		target="$(readlink -f -- "$link")" || fail "$label has an unreadable symlink: $link"
		case "$target" in
			"$root"|"$root"/*) ;;
			*) fail "$label symlink escapes its allowlisted tree: $link -> $target" ;;
		esac
	done < <(find "$root" -type l -print0)
}

assert_tree_links_contained source_root "$source_root"
assert_tree_links_contained site_packages "$site_packages"

archive_sha256="$(sha256sum "$source_archive" | awk '{print $1}')"
[[ "$archive_sha256" == "$expected_archive_sha256" ]] || fail "source archive hash mismatch: $archive_sha256"
source_commit="$(git -C "$source_root" rev-parse HEAD 2>/dev/null)" || fail "source root is not a git checkout"
source_tree="$(git -C "$source_root" rev-parse HEAD^{tree} 2>/dev/null)" || fail "source tree cannot be resolved"
[[ "$source_commit" == "$expected_source_commit" ]] || fail "source commit mismatch: $source_commit"
[[ "$source_tree" == "$expected_source_tree" ]] || fail "source tree mismatch: $source_tree"
config_module_sha256="$(sha256sum "$source_root/hermes_cli/config.py" | awk '{print $1}')"
version_module_sha256="$(sha256sum "$source_root/hermes_cli/__init__.py" | awk '{print $1}')"
[[ "$config_module_sha256" == "$expected_config_module_sha256" ]] || fail "config module hash mismatch: $config_module_sha256"
[[ "$version_module_sha256" == "$expected_version_module_sha256" ]] || fail "version module hash mismatch: $version_module_sha256"

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
payload="$script_dir/hermes_native_probe.py"
require_file probe_payload "$payload"

python_prefix="$(dirname -- "$(dirname -- "$python_path")")"
python_prefix="$(readlink -f -- "$python_prefix")"
python_relative="${python_path#"$python_prefix/"}"
[[ "$python_relative" != "$python_path" && "$python_relative" != *..* ]] || fail "python executable is outside its runtime prefix"
python_version="$($python_path --version 2>&1)" || fail "python interpreter did not run"

bwrap_bin="$(command -v bwrap || true)"
timeout_bin="$(command -v timeout || true)"
require_file bwrap "$bwrap_bin"
require_file timeout "$timeout_bin"
[[ -f /usr/bin/env ]] || fail "/usr/bin/env is required only for host-side preflight"
[[ -d /usr && -d /lib && -d /lib64 && -f /etc/ld.so.cache ]] || fail "allowlisted runtime mounts are unavailable"

set +e
"$timeout_bin" --signal=TERM --kill-after=2s "${timeout_seconds}s" \
	"$bwrap_bin" \
		--die-with-parent \
		--new-session \
		--unshare-net \
		--unshare-pid \
		--unshare-ipc \
		--unshare-uts \
		--clearenv \
		--ro-bind /usr /usr \
		--ro-bind /lib /lib \
		--ro-bind /lib64 /lib64 \
		--dir /etc \
		--ro-bind /etc/ld.so.cache /etc/ld.so.cache \
		--dir /opt \
		--dir /probe \
		--ro-bind "$python_prefix" /opt/hermes-python \
		--ro-bind "$site_packages" /opt/hermes-site-packages \
		--ro-bind "$source_root" /opt/hermes-source \
		--tmpfs /opt/hermes-source/plugins \
		--ro-bind "$payload" /probe/hermes_native_probe.py \
		--bind "$state_root" /mnt \
		--proc /proc \
		--dev /dev \
		--tmpfs /home \
		--tmpfs /root \
		--tmpfs /run \
		--tmpfs /tmp \
		--tmpfs /sys \
		--dir /var \
		--tmpfs /var/run \
		--chdir /mnt \
		--setenv HOME /mnt/home \
		--setenv HERMES_HOME /mnt/home/.hermes \
		--setenv HERMES_MANAGED_DIR /mnt/managed \
		--setenv HERMES_NATIVE_STATE /mnt \
		--setenv HERMES_NATIVE_SOURCE /opt/hermes-source \
		--setenv HERMES_NATIVE_CONFIG /mnt/$config_relative \
		--setenv HERMES_NATIVE_ARCHIVE_SHA256 "$archive_sha256" \
		--setenv HERMES_NATIVE_SOURCE_COMMIT "$source_commit" \
		--setenv HERMES_NATIVE_SOURCE_TREE "$source_tree" \
		--setenv HERMES_NATIVE_CONFIG_MODULE_SHA256 "$config_module_sha256" \
		--setenv HERMES_NATIVE_VERSION_MODULE_SHA256 "$version_module_sha256" \
		--setenv HERMES_NATIVE_ISOLATION_FLAGS "unshare-net unshare-pid unshare-ipc unshare-uts clearenv" \
		--setenv PATH /opt/hermes-python/bin \
		--setenv PYTHONHOME /opt/hermes-python \
		--setenv PYTHONPATH /opt/hermes-source:/opt/hermes-site-packages \
		--setenv PYTHONNOUSERSITE 1 \
		--setenv PYTHONDONTWRITEBYTECODE 1 \
		--setenv XDG_CONFIG_HOME /mnt/xdg \
		--setenv XDG_CACHE_HOME /mnt/cache \
		--setenv XDG_DATA_HOME /mnt/data \
		--setenv TMPDIR /tmp \
		--setenv LANG C.UTF-8 \
		--setenv LC_ALL C.UTF-8 \
		--setenv TERM dumb \
		--setenv NO_COLOR 1 \
		"/opt/hermes-python/$python_relative" -S /probe/hermes_native_probe.py
status=$?
set -e
if [[ "$status" -ne 0 ]]; then
	fail "sandboxed native probe failed with exit $status"
fi

result="$state_root/result/hermes-native-result.json"
require_file result "$result"
printf 'Hermes native probe passed: %s\n' "$result"
