#!/usr/bin/env bash
set -euo pipefail

readonly expected_version='1.18.31'
readonly expected_binary_sha256='f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11'
readonly timeout_seconds="${PROFILE_MANGO_PROBE_TIMEOUT_SECONDS:-20}"
readonly requested_root="${PROFILE_MANGO_PROBE_ROOT:-}"
readonly requested_binary="${PROFILE_MANGO_OPENCODE_BIN:-}"
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
  mkdir -p "$state/home" "$state/config-dir" "$state/managed" "$state/project" "$state/runtime" \
    "$state/xdg/config" "$state/xdg/data" "$state/xdg/cache" "$state/xdg/state" "$results"
  cat >"$state/config-dir/SKILL.md" <<'EOF'
---
name: profile-mango-native-synthetic
description: Use for exact OpenCode native skill qualification only.
---

Synthetic OpenCode skill body.
EOF
}

sandbox() {
  local config_content="$1"
  shift
  "$timeout_bin" "$timeout_seconds" "$bwrap_bin" \
    --die-with-parent --new-session --unshare-net --unshare-pid --unshare-ipc --cap-drop ALL \
    --ro-bind /usr /usr --ro-bind /bin /bin --ro-bind /lib /lib --ro-bind /lib64 /lib64 --ro-bind /etc /etc \
    --ro-bind "$binary" /probe/opencode --bind "$state" /state \
    --proc /proc --dev /dev --tmpfs /tmp --dir /probe --chdir /state/project \
    /usr/bin/env -i \
    PATH=/usr/bin:/bin HOME=/state/home \
    XDG_CONFIG_HOME=/state/xdg/config XDG_DATA_HOME=/state/xdg/data \
    XDG_CACHE_HOME=/state/xdg/cache XDG_STATE_HOME=/state/xdg/state \
    OPENCODE_TEST_HOME=/state/home OPENCODE_CONFIG_DIR=/state/config-dir \
    OPENCODE_CONFIG_CONTENT="$config_content" OPENCODE_DISABLE_PROJECT_CONFIG=1 \
    OPENCODE_AUTH_CONTENT='{}' OPENCODE_DB=:memory: \
    OPENCODE_TEST_MANAGED_CONFIG_DIR=/state/managed OPENCODE_DISABLE_DEFAULT_PLUGINS=1 \
    OPENCODE_DISABLE_EXTERNAL_SKILLS=1 OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1 \
    OPENCODE_DISABLE_LSP_DOWNLOAD=1 OPENCODE_DISABLE_AUTOUPDATE=1 \
    OPENCODE_DISABLE_MODELS_FETCH=1 OPENCODE_DISABLE_PRUNE=1 OPENCODE_DISABLE_SHARE=1 \
    OPENCODE_PURE=1 NO_COLOR=1 TERM=dumb /probe/opencode "$@"
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
[[ -n "$requested_root" && "$requested_root" == /* ]] || fail "PROFILE_MANGO_PROBE_ROOT must be an absolute path"
[[ "$timeout_seconds" =~ ^[1-9][0-9]*$ ]] || fail "timeout must be a positive integer"
bwrap_bin="$(command -v bwrap || true)"
timeout_bin="$(command -v timeout || true)"
[[ -n "$bwrap_bin" ]] || fail "bwrap is required for network isolation"
[[ -n "$timeout_bin" ]] || fail "timeout is required for bounded execution"
actual_binary_sha256="$(sha256sum "$binary" | awk '{print $1}')"
[[ "$actual_binary_sha256" == "$expected_binary_sha256" ]] || fail "unexpected OpenCode binary hash: $actual_binary_sha256"

mkdir -p "$requested_root"
probe_root="$(mktemp -d "$requested_root/opencode-skills-1.18.31.XXXXXX")"
state="$probe_root/state"
backup="$probe_root/backup"
results="$probe_root/results"
restored=0
trap cleanup EXIT
make_layout
inventory "$state" >"$results/inventory.before.tsv"
cp -a "$state" "$backup"

version="$(sandbox '{}' --version)"
[[ "$version" == "$expected_version" ]] || fail "expected OpenCode $expected_version, got $version"
config_content='{"autoupdate":false,"skills":{"paths":["/state/config-dir"]}}'
sandbox "$config_content" debug skill >"$results/skill.json" 2>"$results/skill.err"
grep -Fq 'profile-mango-native-synthetic' "$results/skill.json" || fail "native skill listing omitted the synthetic skill"
grep -Fq '/state/config-dir/SKILL.md' "$results/skill.json" || fail "native skill listing omitted the configured skill path"
grep -Fq 'Synthetic OpenCode skill body.' "$results/skill.json" || fail "native skill listing omitted the skill body"

inventory "$state" >"$results/inventory.after-probe.tsv"
if cmp -s "$results/inventory.before.tsv" "$results/inventory.after-probe.tsv"; then
  fail "native probe unexpectedly produced no scratch-state changes"
fi
restore_state
printf '%s\n' \
  'PASS OpenCode skills probe' \
  "opencode-version=$version" \
  "opencode-binary-sha256=$actual_binary_sha256" \
  'opencode-skills-config=natively-accepted' \
  'opencode-skill-discovery=listed-and-loaded' \
  'opencode-skill-body=observed' \
  'opencode-native-writes=scratch-only' \
  'opencode-no-agent-session=no-model-call-or-TUI' \
  'opencode-backup-restore=byte-for-byte-inventory-match'
