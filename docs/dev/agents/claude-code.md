# Claude Code configuration reference

**Reference date:** 2026-09-23 local / 2026-09-21 UTC. **Documentation:** official
mutable, unversioned pages. **Release context:** `v2.1.278`, commit
`bf7d404e26a5fb6167d21b46c93a2bf6c22ab274`. The release pin does not version
the documentation. **Status:** profile-mango ships an exact-version inert preview
renderer plus a model-only installer that writes an emulated named profile file by
default. Exact explicit-file model consumption was qualified on 2026-09-22; full effective-state support remains blocked.

## Immutable artifact and release context

The immutable GitHub release tag `v2.1.278` resolves to commit
`bf7d404e26a5fb6167d21b46c93a2bf6c22ab274` and is marked immutable. The npm
wrapper package `@anthropic-ai/claude-code@2.1.278` was acquired from its exact
registry tarball and has SHA-256
`08c6dfcf3dafcfd30e09b2926c596e274f0fa20844a5801ada7f1c8e6227157e` and
registry integrity `sha512-mfNRqC0GaEXqmP97NiwJBeYBmRuqe2VzgLUreUUaEhyJxJWx2Z6ClW1tBOncGNNXdhj6EY4LPvUIWP+oq311CA==`.
The pre-postinstall wrapper `bin/claude.exe` stub is SHA-256
`6d7abae055d3b598281300a6c835086dec81bf3048f8a2294c5d3e50c8830d7b`.
The Linux x64 native package `@anthropic-ai/claude-code-linux-x64@2.1.278` has
SHA-256 `d1fb51ab0a0234d1bd7f418ee9d6b6b124c2412b2ddaf3dfc3256bad8063f1c7`
and registry integrity
`sha512-q3r+5aLGAet1MGMkCH2xPsuIW9A40ws4zftURxhYwDenheCKvXc7Gr1jBwvEhYG8uQwgY3YsfNwnvIsh1Bjmeg==`.
The extracted native ELF is SHA-256
`5c4735937844e84f8a93306e841a5b0e12252909b07870f789b190468da147ab`.

The package declares Node.js `>=22.0.0`; the native package is Linux x64 and
glibc-only. Review ran on Linux x86_64 with glibc `2.44`, Node `v24.21.0`, and
npm `11.19.0`. The npm postinstall copies or hardlinks the native binary into
the wrapper path and changes its mode, so no lifecycle script was run during
qualification.

These hashes and the release commit are artifact/release evidence, not proof
that mutable documentation describes this release.

## Configuration and precedence

Claude Code documents strict JSON settings in user `~/.claude/settings.json`,
shared project `.claude/settings.json`, project-local
`.claude/settings.local.json`, and managed settings. Documented precedence is
managed, command line, local project, shared project, then user, while some
collections merge. Verify that behavior against the exact future test build.

Model selection can come from CLI, settings, environment, and interactive
selection. Authentication may use Anthropic or documented cloud-provider routes.
Effort has CLI, environment, settings, and model-specific surfaces. These facts
do not prove an effective route or authentication mode.

The inert adapter emits only the route model as a documentation-context
`model` candidate in `preview/<profile>.settings.json.preview`. It never emits
provider, transport, authentication, credentials, or effort values.

## Emulated named profiles

Claude Code has no native concept of named profiles: `--settings <file>` loads
strict JSON from any path the caller supplies. `profile-mango install <name>
--target claude-code` uses this to emulate one: the top-level `model` is written
to a Mango-owned `profiles/<name>.json` beside `settings.json`, and the plan
prints `use it: claude --settings <path>` so you can start Claude Code with it.
`settings.json` itself is read only to check it exists and is not byte-empty,
and stays byte-for-byte unchanged unless `--default` is also given, which then
patches `settings.json` the same way the installer did before this file
existed. Profile names use the same safe ASCII letters/digits/`_`/`-` subset
as the Codex named-profile convention; this is a profile-mango naming choice,
not observed Claude Code behavior. The exact-ELF evidence below establishes
model consumption from an explicit `--settings` path in general, not a
`profiles/` directory specifically. See the
[dated evidence](../target-evidence.md#bounded-model-installation-2026-09-22).

## Permissions, instructions, and skills

Permissions include allow, ask, and deny rules plus multiple permission modes;
the documentation says deny is evaluated before ask and allow. A prompt or rule
match is not assumed to be containment, and enforcement remains untested.

User and project `CLAUDE.md` files compose through a hierarchy, with some
subdirectory context loaded lazily. Skills use `SKILL.md` in personal, project,
managed, and plugin locations. Later adapter work must account for every relevant
override and discovery layer.

## Candidate inspection and gaps

The immutable native artifact was reviewed before considering `--version`,
`--help`, `claude doctor`, `/status`, config, or schema paths. The wrapper and
embedded strings expose update, telemetry, credentials/cloud routes and SDKs,
project/user discovery, plugins, hooks, MCP, state/migration, subprocess, and
session surfaces. `claude doctor` advertises installation, extension, memory,
hook, update, and permission checks and may fix issues. `/status` is session-oriented,
and no release-qualified schema or effective-config command was established.
The initial preview review ran no native command. The later opt-in probe uses the
exact ELF, explicit settings, synthetic state, cleared environment, isolated
network/PID/IPC, and timeouts. The model sentinel is consumed before no-auth
termination. Sandbox-only startup writes are allowed; no authenticated session or
provider request is qualified. See the [dated evidence](../target-evidence.md#bounded-model-installation-2026-09-22).

## Capability classification

| Portable property | Evidence level | Applicability consequence |
| --- | --- | --- |
| Config fidelity | Partial; only a documentation-context `model` candidate is emitted | Candidate preview only |
| Native acceptance | Exact ELF consumes explicit-file model sentinel | Model-only installation |
| Effective state | Unverified; no merged per-key report | Blocking |
| Precedence | Documentation-context only; not observed for the release | Blocking |
| Model | Narrow explicit-file consumption | Model field installable, full route remains unverified |
| Effort | No release-qualified mapping | Blocking |
| Provider and transport | No release-qualified route mapping | Blocking |
| Authentication | No credential-free identity proof | Blocking |
| Permissions | Documentation describes modes, but equivalence and enforcement are untested | Blocking |
| Tools | Tool expansion and closed-allowlist enforcement are untested | Blocking |
| `CLAUDE.md` instructions | Hierarchy and delivery are unverified | Blocking |
| Skills | Discovery, precedence, and execution are unverified | Blocking |
| Plugins, hooks, and MCP | Discovery, precedence, and enforcement are unverified | Blocking |
| Runtime enforcement | No authorized session or provider observation | Blocking |

Primary sources: [settings][settings], [model configuration][model],
[CLI reference][cli], [authentication][auth], [permissions][permissions],
[memory][memory], [skills][skills], and the separate [release context][release].

[settings]: https://code.claude.com/docs/en/settings
[model]: https://code.claude.com/docs/en/model-config
[cli]: https://code.claude.com/docs/en/cli-reference
[auth]: https://code.claude.com/docs/en/authentication
[permissions]: https://code.claude.com/docs/en/permissions
[memory]: https://code.claude.com/docs/en/memory
[skills]: https://code.claude.com/docs/en/skills
[schema]: https://json.schemastore.org/claude-code-settings.json
[release]: https://github.com/anthropics/claude-code/releases/tag/v2.1.278
