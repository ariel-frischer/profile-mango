# Hermes native config evidence

This evidence qualifies only the exact Hermes config loader module's consumption of the generated `config.yaml` fields. It does not qualify Hermes startup, authentication, provider calls, model calls, sessions, daemons, hooks, plugins, MCP, permissions, tools, instructions, skills, or runtime enforcement.

## Pinned inputs

- Target: Hermes Agent `0.21.3`
- Source release: `v2026.9.14`
- Source commit: `345cd2b057a452236de401d3534b8502a7465e8d`
- Source tree: `6e14b9791cdc5a47068685e9429dd5d6bdc5ef5f`
- Source archive SHA-256: `71f2db39a64fbba282e3bd3be4b0f7b935585948a59a368d61deeec0f0827c47`
- Imported config module: `hermes_cli/config.py`
- Imported config module SHA-256: `d76471ce54d40e68165e2cce7c2ade9c2164ed5ce4dbcf673b1b289cb89c7d84`
- Imported version module: `hermes_cli/__init__.py`
- Imported version module SHA-256: `0d78a58a9f27f32adfdac959e89767cecde93a424bcfbd39d64fb95f6cf13e6c`
- Python runtime: `3.12.13`

The archive, source checkout, runtime, and site-packages directory are caller-supplied task-owned inputs. The probe verifies the archive, source commit/tree, and hashes before entering the sandbox, then verifies the imported module paths and hashes inside it.

## Reproduction

Generate the candidate with the Hermes adapter, write it to a disposable state root, and run the target-specific test:

```bash
PROFILE_MANGO_HERMES_NATIVE_PROBE=1 \
HERMES_NATIVE_SOURCE_ROOT=/path/to/hermes-agent \
HERMES_NATIVE_ARCHIVE=/path/to/hermes-agent-v2026.9.14.tar \
HERMES_NATIVE_PYTHON=/path/to/hermes-venv/bin/python \
HERMES_NATIVE_SITE_PACKAGES=/path/to/hermes-venv/lib/python3.12/site-packages \
go test ./pkg/adapters/hermes -run '^TestHermesNativeProbeGeneratedPatch$' -count=1 -v
```

The test calls `PatchConfig`, writes that exact output, and asks the native `hermes_cli.config.load_config_readonly()` loader to consume it. The disposable state root is the only writable mount. The result is `result/hermes-native-result.json` below the test's temporary state root.

The retained runner is [`hermes_native_probe.sh`](hermes_native_probe.sh), and its imported-module payload is [`hermes_native_probe.py`](hermes_native_probe.py).

## Isolation contract

The runner fails closed unless it can provide all of the following:

- `bwrap` network, PID, IPC, and UTS namespace isolation
- a bounded `timeout` with a two-second kill-after grace period
- `--clearenv` followed only by synthetic Hermes and Python environment variables
- explicit read-only mounts for `/usr`, `/lib`, `/lib64`, the exact Python runtime, the exact dependency directory, the exact Hermes source tree, and the probe payload
- an empty tmpfs overlay over the source tree's `plugins/` directory, so bundled provider plugins cannot be imported during this config-only check
- no `--ro-bind / /` or other broad host-root mount
- a writable bind mount containing only the disposable state root and generated config
- synthetic tmpfs mounts hiding `/home`, `/root`, `/run`, `/var/run`, `/tmp`, and `/sys`
- a synthetic `/proc` and `/dev`
- source-tree symlink checks that reject allowlist escapes
- socket, subprocess, fork, exec, SQLite, thread-start, and Python audit-event traps in the payload
- no Hermes command, agent session, authentication flow, provider/model request, daemon, hook, plugin, or MCP startup

## Observed result

On 2026-09-22, the pinned inputs produced `status: ok`. The effective native merge returned:

```json
{
  "model": {
    "provider": "openai",
    "default": "gpt-5.6"
  },
  "agent": {
    "reasoning_effort": "high"
  },
  "unknown_sentinel": {
    "value": "SYNTHETIC_SECRET_SENTINEL"
  }
}
```

The result (`probes/native-patched-repair/result/hermes-native-result.json`) and generated input (`probes/native-patched-repair/home/.hermes/config.yaml`) were retained in a private local scratch directory. The result recorded `clearenv: true`, hidden real home and run paths, `pid: 2` with `parent_pid: 1` in the isolated PID namespace, all four namespace flags, and the imported module hashes above. Hermes created only disposable home skeleton and config-backup state inside the bound state root.

## Boundary

This is isolated native config parsing and effective-merge evidence for the three promoted fields. It is not startup evidence, authentication evidence, provider or model execution evidence, session evidence, daemon or hook evidence, MCP evidence, or enforcement evidence. The Hermes installer therefore continues to reject permissions, tools, instructions, skills, resources, and every unmanaged runtime property.
