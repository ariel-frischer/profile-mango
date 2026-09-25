#!/usr/bin/env bash
set -euo pipefail

readonly timeout_seconds="${PROFILE_MANGO_PROBE_TIMEOUT_SECONDS:-10}"
readonly requested_root="${PROFILE_MANGO_PROBE_ROOT:-}"
readonly codex_requested="${PROFILE_MANGO_CODEX_BIN:-}"
readonly jcode_requested="${PROFILE_MANGO_JCODE_BIN:-}"
readonly keep_probe="${PROFILE_MANGO_KEEP_PROBE:-0}"
readonly codex_expected_version='codex-cli 0.154.0'
readonly codex_expected_sha256='3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022'
readonly jcode_expected_version='jcode v0.83.909-dev (ca8017a3a)'
readonly jcode_expected_sha256='392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992'

die() {
	printf 'probe error: %s\n' "$*" >&2
	exit 1
}

require_absolute_executable() {
	local name="$1"
	local path="$2"

	[[ -n "$path" ]] || die "$name must be set to a direct executable path"
	[[ "$path" == /* ]] || die "$name must be an absolute path: $path"
	[[ -x "$path" ]] || die "$name is not executable: $path"
	readlink -f "$path"
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

feature_state() {
	awk '$1 == "apps" { print $NF; found = 1 } END { if (!found) exit 1 }'
}

make_layout() {
	local root="$1"

	mkdir -p "$root/home/.jcode" "$root/xdg" "$root/codex" \
		"$root/project" "$root/runtime"
}

sandbox() {
	local root="$1"
	local target="$2"
	shift 2

	local binary
	case "$target" in
		codex) binary="$codex_bin" ;;
		jcode) binary="$jcode_bin" ;;
		*) die "unknown sandbox target: $target" ;;
	esac

	"$timeout_bin" "$timeout_seconds" "$bwrap_bin" \
		--die-with-parent \
		--new-session \
		--unshare-net \
		--ro-bind /usr /usr \
		--ro-bind /bin /bin \
		--ro-bind /lib /lib \
		--ro-bind /lib64 /lib64 \
		--ro-bind /etc /etc \
		--ro-bind "$binary" "/probe/$target" \
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
		CODEX_HOME=/state/codex \
		JCODE_HOME=/state/home/.jcode \
		JCODE_CONFIG=/state/home/.jcode/config.toml \
		JCODE_SOCKET=/state/runtime/jcode.sock \
		JCODE_NO_TELEMETRY=1 \
		NO_COLOR=1 \
		TERM=dumb \
		"/probe/$target" "$@"
}

run_jcode_profile() {
	local root="$1"
	shift

	sandbox "$root" jcode \
		--no-update --no-selfdev --socket /state/runtime/jcode.sock --quiet \
		profile "$@"
}

codex_bin="$(require_absolute_executable PROFILE_MANGO_CODEX_BIN "$codex_requested")"
jcode_bin="$(require_absolute_executable PROFILE_MANGO_JCODE_BIN "$jcode_requested")"
bwrap_bin="$(command -v bwrap || true)"
timeout_bin="$(command -v timeout || true)"
[[ -n "$bwrap_bin" ]] || die "bwrap is required for network isolation"
[[ -n "$timeout_bin" ]] || die "timeout is required for bounded execution"
[[ "$timeout_seconds" =~ ^[1-9][0-9]*$ ]] || die "timeout must be a positive integer"

codex_sha256="$(sha256sum "$codex_bin" | awk '{print $1}')"
assert_equal "$codex_sha256" "$codex_expected_sha256"
jcode_sha256="$(sha256sum "$jcode_bin" | awk '{print $1}')"
assert_equal "$jcode_sha256" "$jcode_expected_sha256"

if [[ -n "$requested_root" ]]; then
	[[ "$requested_root" == /* ]] || die "PROFILE_MANGO_PROBE_ROOT must be absolute"
	mkdir -p "$requested_root"
	probe_root="$(mktemp -d "$requested_root/run.XXXXXX")"
	cleanup_parent=0
else
	probe_root="$(mktemp -d "${TMPDIR:-/tmp}/profile-mango-agent-probe.XXXXXX")"
	cleanup_parent=1
fi

cleanup() {
	if [[ "$keep_probe" != 1 ]]; then
		rm -rf -- "$probe_root"
	fi
	if [[ "$cleanup_parent" == 1 && "$keep_probe" != 1 ]]; then
		rmdir --ignore-fail-on-non-empty "$(dirname "$probe_root")" 2>/dev/null || true
	fi
}
trap cleanup EXIT

codex_baseline="$probe_root/codex-baseline"
codex_positive="$probe_root/codex-positive"
codex_malformed="$probe_root/codex-malformed"
make_layout "$codex_baseline"
make_layout "$codex_positive"
make_layout "$codex_malformed"
printf '[features]\napps = false\n' >"$codex_positive/codex/config.toml"
printf '[features\n' >"$codex_malformed/codex/config.toml"

codex_version="$(sandbox "$codex_baseline" codex --version)"
codex_help="$(sandbox "$codex_baseline" codex --help)"
codex_features_baseline="$(sandbox "$codex_baseline" codex features list)"
codex_features_positive="$(sandbox "$codex_positive" codex features list)"
codex_features_cli_enable="$(sandbox "$codex_positive" codex --enable apps features list)"
codex_features_cli_disable="$(sandbox "$codex_positive" codex --disable apps features list)"
assert_equal "$codex_version" "$codex_expected_version"
assert_contains "$codex_help" '--strict-config'
assert_equal "$(feature_state <<<"$codex_features_baseline")" true
assert_equal "$(feature_state <<<"$codex_features_positive")" false
assert_equal "$(feature_state <<<"$codex_features_cli_enable")" true
assert_equal "$(feature_state <<<"$codex_features_cli_disable")" false
if codex_malformed_output="$(sandbox "$codex_malformed" codex features list 2>&1)"; then
	die 'malformed Codex config unexpectedly succeeded'
fi
assert_contains "$codex_malformed_output" 'TOML parse error'

jcode_positive="$probe_root/jcode-positive"
jcode_malformed="$probe_root/jcode-malformed"
jcode_unknown="$probe_root/jcode-unknown-key"
jcode_invalid_effort="$probe_root/jcode-invalid-effort"
make_layout "$jcode_positive"
make_layout "$jcode_malformed"
make_layout "$jcode_unknown"
make_layout "$jcode_invalid_effort"
cat >"$jcode_positive/home/.jcode/config.toml" <<'EOF'
[profiles.sentinel]
provider = "openai-api"
model = "sentinel-model-positive"
reasoning_effort = "low"
tool_profile = "none"
tools = ["read"]
disabled_tools = ["write"]
skills_mode = "none"
skills = ["sentinel-skill-positive"]
disabled_skills = ["sentinel-skill-disabled"]
instructions = "SENTINEL-INSTRUCTION-POSITIVE"
agents_md_path = "sentinel-agents.md"

[profiles.alternate]
provider = "openai"
model = "sentinel-model-alternate"
reasoning_effort = "minimal"
tool_profile = "minimal"
instructions = "SENTINEL-INSTRUCTION-ALTERNATE"
EOF
printf '[profiles.sentinel\n' >"$jcode_malformed/home/.jcode/config.toml"
cat >"$jcode_unknown/home/.jcode/config.toml" <<'EOF'
[profiles.sentinel]
provider = "openai-api"
model = "sentinel-model-unknown-key"
unknown_key = "SENTINEL-UNKNOWN-KEY"
EOF
cat >"$jcode_invalid_effort/home/.jcode/config.toml" <<'EOF'
[profiles.sentinel]
provider = "openai-api"
model = "sentinel-model-invalid-effort"
reasoning_effort = "SENTINEL-INVALID-EFFORT"
EOF

jcode_version="$(sandbox "$jcode_positive" jcode --version)"
jcode_help="$(sandbox "$jcode_positive" jcode profile --help)"
jcode_list="$(run_jcode_profile "$jcode_positive" list --json)"
jcode_show="$(run_jcode_profile "$jcode_positive" show sentinel --json)"
jcode_resolved="$(run_jcode_profile "$jcode_positive" resolve sentinel --json)"
jcode_current="$(sandbox "$jcode_positive" jcode \
	--no-update --no-selfdev --socket /state/runtime/jcode.sock --quiet \
	--profile sentinel profile current --json)"
jcode_tool_override="$(sandbox "$jcode_positive" jcode \
	--no-update --no-selfdev --socket /state/runtime/jcode.sock --quiet \
	--profile sentinel --tool-profile minimal profile resolve sentinel --json)"
assert_equal "$jcode_version" "$jcode_expected_version"
assert_contains "$jcode_help" 'Resolve a named profile'
assert_contains "$jcode_list" '"name": "sentinel"'
assert_contains "$jcode_list" '"name": "alternate"'
assert_contains "$jcode_show" '"provider": "openai-api"'
assert_contains "$jcode_show" '"model": "sentinel-model-positive"'
assert_contains "$jcode_show" '"reasoning_effort": "low"'
assert_contains "$jcode_show" '"tool_profile": "none"'
assert_contains "$jcode_show" '"instructions_present": true'
assert_contains "$jcode_show" '"instructions_chars": 29'
assert_contains "$jcode_show" '"agents_md_present": false'
assert_not_contains "$jcode_show" 'SENTINEL-INSTRUCTION-POSITIVE'
assert_contains "$jcode_resolved" '"provider": "openai-api"'
assert_contains "$jcode_resolved" '"model": "sentinel-model-positive"'
assert_contains "$jcode_resolved" '"reasoning_effort": "low"'
assert_contains "$jcode_resolved" '"allowed_tools": ['
assert_contains "$jcode_resolved" '"disabled_tools": ['
assert_contains "$jcode_resolved" '"sources"'
assert_contains "$jcode_resolved" '"model": "Profile"'
assert_contains "$jcode_resolved" "Profile agents_md_path '/state/home/.jcode/sentinel-agents.md' is missing"
assert_contains "$jcode_current" '"profile_name": "sentinel"'
assert_contains "$jcode_current" '"provider": "openai-api"'
assert_contains "$jcode_tool_override" '"profile": "minimal"'

if jcode_malformed_output="$(run_jcode_profile "$jcode_malformed" list --json 2>&1)"; then
	die 'malformed Jcode fork config unexpectedly succeeded'
fi
assert_contains "$jcode_malformed_output" 'TOML parse error'
jcode_unknown_output="$(run_jcode_profile "$jcode_unknown" resolve sentinel --json)"
assert_contains "$jcode_unknown_output" '"model": "sentinel-model-unknown-key"'
assert_not_contains "$jcode_unknown_output" 'SENTINEL-UNKNOWN-KEY'
if jcode_invalid_output="$(run_jcode_profile "$jcode_invalid_effort" resolve sentinel --json 2>&1)"; then
	die 'invalid Jcode fork reasoning effort unexpectedly succeeded'
fi
assert_contains "$jcode_invalid_output" 'invalid reasoning_effort'

printf 'PASS agent config probe\n'
printf 'platform=%s\n' "$(uname -srm)"
printf 'network=unshared-by-bwrap\n'
printf 'environment=env-i-with-synthetic-home-xdg-and-target-roots\n'
printf 'codex-version=%s\n' "$codex_version"
printf 'codex-sha256=%s\n' "$codex_sha256"
printf 'codex-features-apps-baseline=%s\n' "$(feature_state <<<"$codex_features_baseline")"
printf 'codex-features-apps-synthetic=%s\n' "$(feature_state <<<"$codex_features_positive")"
printf 'codex-features-apps-cli-enable=%s\n' "$(feature_state <<<"$codex_features_cli_enable")"
printf 'codex-features-apps-cli-disable=%s\n' "$(feature_state <<<"$codex_features_cli_disable")"
printf 'codex-malformed-config=rejected\n'
printf 'codex-config-consumption=features-apps-only-observed\n'
printf 'codex-route-provider-model-effort=unverified\n'
printf 'codex-authentication-identity=unverified\n'
printf 'codex-precedence=feature-runtime-override-observed-route-project-unverified\n'
printf 'codex-permissions-tools=enforcement-unverified\n'
printf 'codex-instruction-skill-delivery=unverified\n'
printf 'jcode-version=%s\n' "$jcode_version"
printf 'jcode-sha256=%s\n' "$jcode_sha256"
printf 'jcode-profile-fields=provider-model-effort-tools-skills-instruction-presence\n'
printf 'jcode-tool-profile-override=observed\n'
printf 'jcode-malformed-config=rejected\n'
printf 'jcode-invalid-effort=rejected\n'
printf 'jcode-unknown-key=accepted-and-ignored-by-current-custom-fork\n'
