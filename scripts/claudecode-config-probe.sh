#!/usr/bin/env bash
set -euo pipefail

readonly timeout_seconds="${PROFILE_MANGO_PROBE_TIMEOUT_SECONDS:-20}"
readonly requested_root="${PROFILE_MANGO_PROBE_ROOT:-}"
readonly claude_requested="${PROFILE_MANGO_CLAUDE_BIN:-}"
readonly expected_version='2.1.278 (Claude Code)'
readonly expected_sha256='5c4735937844e84f8a93306e841a5b0e12252909b07870f789b190468da147ab'

die() {
	printf 'probe error: %s\n' "$*" >&2
	exit 1
}

assert_contains() {
	local output="$1"
	local expected="$2"
	grep -Fq -- "$expected" <<<"$output" || die "output missing $expected"
}

assert_not_contains() {
	local output="$1"
	local unexpected="$2"
	if grep -Fq -- "$unexpected" <<<"$output"; then
		die "output unexpectedly contains $unexpected"
	fi
}

assert_equal() {
	local actual="$1"
	local expected="$2"
	[[ "$actual" == "$expected" ]] || die "expected $expected, got $actual"
}

require_absolute_executable() {
	local name="$1"
	local path="$2"
	[[ -n "$path" ]] || die "$name must be set to a direct executable path"
	[[ "$path" == /* ]] || die "$name must be an absolute path: $path"
	[[ -x "$path" ]] || die "$name is not executable: $path"
	readlink -f "$path"
}

make_layout() {
	local root="$1"
	mkdir -p "$root/home/.claude" "$root/project/.claude" "$root/xdg" "$root/runtime"
}

sandbox() {
	local root="$1"
	shift
	"$timeout_bin" "$timeout_seconds" "$bwrap_bin" \
		--die-with-parent \
		--new-session \
		--unshare-net \
		--unshare-pid \
		--unshare-ipc \
		--ro-bind /usr /usr \
		--ro-bind /bin /bin \
		--ro-bind /lib /lib \
		--ro-bind /lib64 /lib64 \
		--ro-bind /etc /etc \
		--ro-bind "$claude_bin" /probe/claude \
		--bind "$root" /state \
		--proc /proc \
		--dev /dev \
		--tmpfs /tmp \
		--dir /probe \
		--chdir /state/project \
		/usr/bin/env -i \
		PATH=/usr/bin:/bin \
		HOME=/state/home \
		XDG_CONFIG_HOME=/state/xdg \
		CLAUDE_CONFIG_DIR=/state/home/.claude \
		CLAUDE_CODE_DISABLE_TELEMETRY=1 \
		CLAUDE_CODE_DISABLE_VITALS_EMITTER=1 \
		CLAUDE_CODE_DISABLE_AUTOUPDATER=1 \
		CLAUDE_CODE_SIMPLE=1 \
		NO_COLOR=1 \
		TERM=dumb \
		/probe/claude "$@"
}

claude_bin="$(require_absolute_executable PROFILE_MANGO_CLAUDE_BIN "$claude_requested")"
bwrap_bin="$(command -v bwrap || true)"
timeout_bin="$(command -v timeout || true)"
[[ -n "$bwrap_bin" ]] || die "bwrap is required for network isolation"
[[ -n "$timeout_bin" ]] || die "timeout is required for bounded execution"
[[ "$timeout_seconds" =~ ^[1-9][0-9]*$ ]] || die "timeout must be a positive integer"

assert_equal "$(sha256sum "$claude_bin" | awk '{print $1}')" "$expected_sha256"

if [[ -n "$requested_root" ]]; then
	[[ "$requested_root" == /* ]] || die "PROFILE_MANGO_PROBE_ROOT must be absolute"
	mkdir -p "$requested_root"
	probe_root="$(mktemp -d "$requested_root/run.XXXXXX")"
	cleanup_parent=0
else
	probe_root="$(mktemp -d "${TMPDIR:-/tmp}/profile-mango-claudecode-probe.XXXXXX")"
	cleanup_parent=1
fi

cleanup() {
	if [[ "${PROFILE_MANGO_KEEP_PROBE:-0}" != 1 ]]; then
		rm -rf -- "$probe_root"
	fi
	if [[ "$cleanup_parent" == 1 && "${PROFILE_MANGO_KEEP_PROBE:-0}" != 1 ]]; then
		rmdir --ignore-fail-on-non-empty "$(dirname "$probe_root")" 2>/dev/null || true
	fi
}
trap cleanup EXIT

positive="$probe_root/positive"
malformed="$probe_root/malformed"
make_layout "$positive"
make_layout "$malformed"
printf 'EXTERNAL-SENTINEL\n' >"$probe_root/external-sentinel"

cat >"$positive/home/.claude/settings.json" <<'EOF'
{
  "model": "SENTINEL-MODEL",
  "unknownSentinel": "SENTINEL-UNKNOWN",
  "apiKey": "SYNTHETIC-CREDENTIAL"
}
EOF
printf '{\n  "model": "SENTINEL-MODEL",\n' >"$malformed/home/.claude/settings.json"

settings_path="$positive/home/.claude/settings.json"
settings_hash="$(sha256sum "$settings_path" | awk '{print $1}')"
external_hash="$(sha256sum "$probe_root/external-sentinel" | awk '{print $1}')"

version_output="$(sandbox "$positive" --version)"
assert_equal "$version_output" "$expected_version"
printf 'claude-version=%s\n' "$version_output"
printf 'claude-binary-sha256=%s\n' "$expected_sha256"

set +e
sandbox "$positive" --bare --settings /state/home/.claude/settings.json \
	--print --no-session-persistence --permission-prompts none \
	--output-format text SENTINEL-PROMPT >"$positive/native.stdout" 2>"$positive/native.stderr"
positive_status=$?
set -e
[[ "$positive_status" -ne 0 ]] || die 'sentinel model unexpectedly completed a native request'
positive_output="$(cat "$positive/native.stdout" "$positive/native.stderr")"
assert_contains "$positive_output" 'claude-code:unrecognized_model'
assert_contains "$positive_output" '"model":"SENTINEL-MODEL"'
assert_contains "$positive_output" 'Not logged in'
assert_not_contains "$positive_output" 'SYNTHETIC-CREDENTIAL'
assert_not_contains "$positive_output" 'SENTINEL-UNKNOWN'
printf 'claude-model-sentinel=consumed-before-auth-or-provider\n'

assert_equal "$(sha256sum "$settings_path" | awk '{print $1}')" "$settings_hash"
printf 'claude-settings-file=byte-identical-after-native-startup\n'

set +e
sandbox "$malformed" --settings '{"model":"SENTINEL-MODEL"' --help \
	>"$malformed/negative.stdout" 2>"$malformed/negative.stderr"
malformed_status=$?
set -e
[[ "$malformed_status" -ne 0 ]] || die 'malformed --settings argument unexpectedly succeeded'
malformed_output="$(cat "$malformed/negative.stdout" "$malformed/negative.stderr")"
assert_contains "$malformed_output" 'Settings file not found'
printf 'claude-malformed-settings-argument=rejected\n'

if find "$positive" -type s -print -quit | grep -q .; then
	die 'native probe created an unexpected socket in synthetic state'
fi
if grep -R --exclude=settings.json -F -q -- 'SYNTHETIC-CREDENTIAL' "$positive/home/.claude"; then
	die 'synthetic credential-shaped value escaped the settings input boundary'
fi
assert_equal "$(sha256sum "$probe_root/external-sentinel" | awk '{print $1}')" "$external_hash"
assert_equal "$(sha256sum "$claude_bin" | awk '{print $1}')" "$expected_sha256"
printf 'claude-native-writes=sandbox-only\n'
printf 'claude-isolation=env-i-network-unshared-timeout-bounded\n'
printf 'claude-native-entrypoint=bare-noninteractive-no-auth-no-provider\n'
printf 'PASS Claude Code config probe\n'
