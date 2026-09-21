#!/usr/bin/env bash
set -euo pipefail

readonly expected_version='1.18.31'
readonly expected_binary_sha256='f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11'
readonly timeout_seconds="${PROFILE_MANGO_PROBE_TIMEOUT_SECONDS:-20}"
readonly requested_root="${PROFILE_MANGO_PROBE_ROOT:-}"
readonly requested_binary="${PROFILE_MANGO_OPENCODE_BIN:-}"
readonly requested_candidate="${PROFILE_MANGO_OPENCODE_CANDIDATE:-}"
readonly keep_probe="${PROFILE_MANGO_KEEP_PROBE:-0}"

fail() {
	printf 'probe error: %s\n' "$*" >&2
	exit 1
}

require_file() {
	local name="$1"
	local path="$2"
	[[ -n "$path" ]] || fail "$name must be set"
	[[ "$path" == /* ]] || fail "$name must be an absolute path: $path"
	[[ -f "$path" ]] || fail "$name is not a regular file: $path"
	readlink -f "$path"
}

inventory() {
	local tree="$1"
	find "$tree" -mindepth 1 -print0 | sort -z | while IFS= read -r -d '' path; do
		local rel="${path#"$tree"/}"
		if [[ -f "$path" && ! -L "$path" ]]; then
			printf 'f\t%s\t%s\t%s\t%s\n' "$rel" "$(stat -c '%a' "$path")" \
				"$(stat -c '%s' "$path")" "$(sha256sum "$path" | awk '{print $1}')"
		elif [[ -d "$path" ]]; then
			printf 'd\t%s\t%s\n' "$rel" "$(stat -c '%a' "$path")"
		elif [[ -L "$path" ]]; then
			printf 'l\t%s\t%s\n' "$rel" "$(readlink "$path")"
		else
			printf 'o\t%s\t%s\n' "$rel" "$(stat -c '%F' "$path")"
		fi
	done
}

make_layout() {
	mkdir -p "$state/home" "$state/xdg/config" "$state/xdg/data" "$state/xdg/cache" \
		"$state/xdg/state" "$state/config-dir" "$state/managed" "$state/project" "$state/runtime" "$results"
	printf '%s\n' 'SENTINEL-UNRELATED-CONTENT' >"$state/config-dir/unrelated.txt"
	cat >"$state/config-dir/opencode.jsonc" <<'EOF'
{
  "$schema": "https://opencode.ai/config.json",
  "autoupdate": false,
  "model": "global/sentinel-model"
}
EOF
	cp "$candidate" "$state/config-dir/profile-mango-candidate.jsonc"
}

sandbox() {
	local config_content="$1"
	local config_path="$2"
	shift 2
	"$timeout_bin" "$timeout_seconds" "$bwrap_bin" \
		--die-with-parent --new-session --unshare-net \
		--ro-bind /usr /usr --ro-bind /bin /bin --ro-bind /lib /lib --ro-bind /lib64 /lib64 --ro-bind /etc /etc \
		--ro-bind "$binary" /probe/opencode --bind "$state" /state \
		--proc /proc --dev /dev --tmpfs /tmp --dir /probe --chdir /state/project \
		/usr/bin/env -i \
		PATH=/usr/bin:/bin HOME=/state/home \
		XDG_CONFIG_HOME=/state/xdg/config XDG_DATA_HOME=/state/xdg/data \
		XDG_CACHE_HOME=/state/xdg/cache XDG_STATE_HOME=/state/xdg/state \
		OPENCODE_TEST_HOME=/state/home OPENCODE_CONFIG_DIR=/state/config-dir \
		OPENCODE_CONFIG_CONTENT="$config_content" OPENCODE_CONFIG="$config_path" \
		OPENCODE_DISABLE_PROJECT_CONFIG=1 OPENCODE_AUTH_CONTENT='{}' OPENCODE_DB=:memory: \
		OPENCODE_TEST_MANAGED_CONFIG_DIR=/state/managed OPENCODE_DISABLE_DEFAULT_PLUGINS=1 \
		OPENCODE_DISABLE_LSP_DOWNLOAD=1 OPENCODE_DISABLE_AUTOUPDATE=1 \
		OPENCODE_DISABLE_MODELS_FETCH=1 OPENCODE_DISABLE_PRUNE=1 OPENCODE_PURE=1 \
		NO_COLOR=1 TERM=dumb /probe/opencode "$@"
}

restore_state() {
	[[ "$restored" == 1 ]] && return 0
	[[ -d "$backup" ]] || return 0
	rm -rf "$state"
	cp -a "$backup" "$state"
	inventory "$state" >"$results/inventory.restored.tsv"
	cmp "$results/inventory.before.tsv" "$results/inventory.restored.tsv"
	restored=1
}

cleanup() {
	local status=$?
	trap - EXIT
	if ! restore_state; then
		printf 'probe error: failed to restore disposable state\n' >&2
		status=1
	fi
	if [[ "$keep_probe" != 1 ]]; then
		rm -rf "$probe_root"
	fi
	exit "$status"
}

binary="$(require_file PROFILE_MANGO_OPENCODE_BIN "$requested_binary")"
candidate="$(require_file PROFILE_MANGO_OPENCODE_CANDIDATE "$requested_candidate")"
[[ -x "$binary" ]] || fail "PROFILE_MANGO_OPENCODE_BIN is not executable: $binary"
[[ -n "$requested_root" && "$requested_root" == /* ]] || fail "PROFILE_MANGO_PROBE_ROOT must be an absolute path"
[[ "$timeout_seconds" =~ ^[1-9][0-9]*$ ]] || fail "timeout must be a positive integer"
bwrap_bin="$(command -v bwrap || true)"
timeout_bin="$(command -v timeout || true)"
[[ -n "$bwrap_bin" ]] || fail "bwrap is required for network isolation"
[[ -n "$timeout_bin" ]] || fail "timeout is required for bounded execution"
actual_binary_sha256="$(sha256sum "$binary" | awk '{print $1}')"
[[ "$actual_binary_sha256" == "$expected_binary_sha256" ]] || fail "unexpected OpenCode binary hash: $actual_binary_sha256"

mkdir -p "$requested_root"
probe_root="$(mktemp -d "$requested_root/opencode-1.18.31.XXXXXX")"
state="$probe_root/state"
backup="$probe_root/backup"
results="$probe_root/results"
restored=0
trap cleanup EXIT
make_layout
candidate_state_sha256="$(sha256sum "$state/config-dir/profile-mango-candidate.jsonc" | awk '{print $1}')"
inventory "$state" >"$results/inventory.before.tsv"
cp -a "$state" "$backup"

version="$(sandbox '{}' '' --version)"
[[ "$version" == "$expected_version" ]] || fail "expected OpenCode $expected_version, got $version"
help="$(sandbox '{}' '' --help 2>&1)"
grep -Fq 'opencode debug' <<<"$help" || fail "OpenCode help omitted debug command"

positive="$(cat "$candidate")"
sandbox "$positive" '' --pure debug config >"$results/config.content.json" 2>"$results/config.content.err"
grep -Fq '"model": "openai/gpt-5.6"' "$results/config.content.json" || fail "resolved config omitted Profile Mango model"
grep -Fq '"autoupdate": false' "$results/config.content.json" || fail "resolved config omitted global sentinel"

cat >"$state/config-dir/opencode.jsonc" <<'EOF'
{
  "$schema": "https://opencode.ai/config.json",
  "autoupdate": false
}
EOF
sandbox '' /state/config-dir/profile-mango-candidate.jsonc --pure debug config >"$results/config.file.json" 2>"$results/config.file.err"
grep -Fq '"model": "openai/gpt-5.6"' "$results/config.file.json" || fail "explicit candidate file was not consumed"
[[ "$(sha256sum "$state/config-dir/profile-mango-candidate.jsonc" | awk '{print $1}')" == "$candidate_state_sha256" ]] || fail "native inspection changed the candidate file"

if sandbox '{ model: ' '' --pure debug config >"$results/config.malformed.out" 2>"$results/config.malformed.err"; then
	fail "malformed config unexpectedly succeeded"
fi
sandbox '{ "model": "openai/gpt-5.6", "profileMangoUnknownSentinel": true }' '' --pure debug config >"$results/config.unknown.json" 2>"$results/config.unknown.err"
if grep -Fq 'profileMangoUnknownSentinel' "$results/config.unknown.json"; then
	fail "unknown key unexpectedly survived resolved configuration"
fi

inventory "$state" >"$results/inventory.after-probe.tsv"
if cmp -s "$results/inventory.before.tsv" "$results/inventory.after-probe.tsv"; then
	fail "native probe unexpectedly produced no scratch-state changes"
fi
restore_state
printf '%s\n' \
	'PASS OpenCode config probe' \
	"opencode-version=$version" \
	"opencode-binary-sha256=$actual_binary_sha256" \
	'opencode-jsonc-candidate=native-accepted' \
	'opencode-config-content-precedence=observed-over-global' \
	'opencode-custom-config-file=consumed' \
	'opencode-custom-config-file=byte-identical-after-inspection' \
	'opencode-malformed-config=rejected' \
	'opencode-unknown-key=accepted-and-ignored' \
	'opencode-effective-config=merged-output-without-field-provenance' \
	'opencode-native-writes=scratch-only' \
	'opencode-backup-restore=byte-for-byte-inventory-match'
