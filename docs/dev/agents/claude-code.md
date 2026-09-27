# Claude Code configuration reference

**Reference date:** 2026-09-23 local / 2026-09-21 UTC. **Documentation:** official
mutable, unversioned pages. **Release context:** `v2.1.278`, commit
`bf7d404e26a5fb6167d21b46c93a2bf6c22ab274`. The release pin does not version
the documentation. **Status:** profile-mango ships an exact-version inert preview
renderer plus a model and effort installer that writes an emulated named profile
file by default. Exact explicit-file model consumption was qualified on
2026-09-22 and top-level `effortLevel` consumption on 2026-09-25; full
effective-state support remains blocked.

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
strict JSON from any path the caller supplies. `mango install <name>
--target claude-code` uses this to emulate one: the top-level `model` (and
`effortLevel` when the route effort is `low`, `medium`, `high`, or `xhigh`) is
written to a Mango-owned `profiles/<name>.json` beside `settings.json`, and the plan
prints `use it: claude --settings <path>` so you can start Claude Code with it.
`<path>` is the file actually planned, beside an explicit `--config` too; it is
shown as `~/...` below your home, else absolute, and shell-quoted when needed.
The JSON plan's `install.useCommand` carries the same command.
`settings.json` itself is read only to check it exists and is not byte-empty,
and stays byte-for-byte unchanged unless `--default` is also given, which then
patches `settings.json` the same way the installer did before this file
existed. Profile names use the same safe ASCII letters/digits/`_`/`-` subset
as the Codex named-profile convention; this is a profile-mango naming choice,
not observed Claude Code behavior. The exact-ELF evidence below establishes
model consumption from an explicit `--settings` path in general, not a
`profiles/` directory specifically. See the
[dated evidence](../target-evidence.md#bounded-model-installation-2026-09-22).

## Effort

The exact `2.1.278` ELF settings schema declares top-level `effortLevel` as one
of `low`, `medium`, `high`, `xhigh`, silently discarding any other value
(`max` is session-only). The isolated probe's offline `/model` status line
reported each of the four levels from an explicit `--settings` file and from
the default user `settings.json`, reported no effort for `max` or an invalid
value, and showed project settings overriding user settings. The installer
therefore writes `effortLevel` beside `model` for those four levels only. Any
other route effort is listed in the plan as `effort <value>: NOT APPLIED`
(JSON `skippedRequirements` entry `effort` with value and reason), and
`--strict` blocks. `--effort`, `CLAUDE_CODE_EFFORT_LEVEL`, per-model
`modelSettings`, and project/local settings can still override the installed
level; model support for effort is model-dependent. See the
[dated evidence](../target-evidence.md#claude-code-effort-installation-2026-09-25).

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
| Native acceptance | Exact ELF consumes explicit-file model sentinel and `effortLevel` | Model and effort installation |
| Effective state | Unverified; no merged per-key report | Blocking |
| Precedence | Documentation-context only; project over user observed for effort | Blocking |
| Model | Narrow explicit-file consumption | Model field installable, full route remains unverified |
| Effort | Exact ELF `/model` status reports `effortLevel` low/medium/high/xhigh from `--settings` and user settings | `effortLevel` installable for those four levels; others not applied |
| Provider and transport | No release-qualified route mapping | Blocking |
| Authentication | No credential-free identity proof | Blocking |
| Permissions | Documentation describes modes, but equivalence and enforcement are untested | Blocking |
| Tools | Tool expansion and closed-allowlist enforcement are untested | Blocking |
| `CLAUDE.md` instructions | Hierarchy and delivery are unverified | Blocking |
| Skills | Personal skill folders under `<config dir>/skills` are documented; precedence and execution remain unobserved | Installed as owned folders (ap-794) |
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

## Global instruction files (ap-nym, 2026-09-25)

`globalInstructions` may own `CLAUDE.md` in `~/.claude` (documentation context only: code.claude.com/docs/en/memory retrieved 2026-09-25, SHA-256 `cf73d3a5…192a8f`; not observed in the binary). Written only when the profile is the default (`mango use`, `install --default`); whole-file ownership with create-only backup, drift checks, release on `use`, and undo. See [target evidence](../target-evidence.md#global-instruction-files-2026-09-25-ap-nym).

## Home `~/AGENTS.md` (ap-5mp, 2026-09-25)

Documentation context only (code.claude.com/docs/en/memory retrieved 2026-09-25, SHA-256 `cf73d3a5…192a8f`, lines 333-360): `2.1.277+` reads `AGENTS.md` in the working directory and above only when no `CLAUDE.md`/`CLAUDE.local.md` exists there. Claude Code therefore does not own `globalInstructions.home`. See [target evidence](../target-evidence.md#home-instruction-file-agentsmd-2026-09-25-ap-5mp).

## Role subagent files (ap-6lp, 2026-09-25)

Each declared profile role becomes `~/.claude/agents/<role>.md`, written only when the profile is the default (`install --default`, `mango use`). Evidence is read-only `strings` observation of installed binaries inside the tested range `>=2.1.278 <2.2.0`: `2.1.281` (SHA-256 `56fe3da88458465fb27d7e9299dddb3fead55750fb9c2de795f233b5eea6dce1`) and `2.1.280` (SHA-256 `1e08503dbdf3c2cb0d706d32f3408277388d1c76ef108673e8fe42c1b322925b`). No `2.1.278` binary was available, and no command was run against the files.

| Claim | Observed string (2.1.281) |
| --- | --- |
| User agents load from `<config home>/agents` beside skills and commands | `e==="user"?M(p,"agents")` |
| `name` and `description` frontmatter are required | `Missing required "name" field in frontmatter`, `Missing required "description" field in frontmatter` |
| `model` is a model name or `inherit`, recorded as `agent_frontmatter` | `y&&y!=="inherit"` ... `"agent_frontmatter"` |
| `effort` must be `low`, `medium`, `high`, `xhigh`, `max`, or an integer | `ad=["low","medium","high","xhigh","max"]`; ``Agent file ${e} has invalid effort '${$e}'. Valid options: ${ad.join(", ")} or an integer`` |

The file carries `name`, `description`, the role's `instructions` (else its description) as the body, and, when the route binds the role, `model` (anthropic provider only; other providers skip the role's model as `roles` with a reason) and `effort` (`low`..`max`; anything else is `NOT APPLIED`). Ownership and the named-only `role-definitions` skip match Codex. Residual risk: acceptance is inferred from strings of later patch releases in range, not a native probe of `2.1.278`.

Profile `agentFiles.claude-code` copies native `*.md` files verbatim into the same directory (kind `agent-file`, same gates and ownership as role files); mango does not parse them. See [target evidence](../target-evidence.md#profile-agent-files-2026-09-26-ap-8x5).

## Skill folders (ap-794, 2026-09-27)

Documentation context only ([skills][skills], unversioned; retrieved 2026-09-27): personal skills live in `~/.claude/skills/<name>/SKILL.md` (beneath `CLAUDE_CONFIG_DIR` when set **[INFERENCE]**), supporting files such as `scripts/` and `references/` are allowed, symlinked skill folders are followed, `name` defaults to the folder name, and `description` falls back to the first body line. Enterprise skills outrank personal ones, which outrank project ones. Each `skills` entry's whole folder (every regular file beside its `SKILL.md`, at most 256, modes normalized to `0755` when any execute bit is set and `0644` otherwise) is copied verbatim to ``<config dir>/skills` (by default `~/.claude/skills`)/<folder name>/`, written only when the profile is the default (`mango use`, `install --default`) and never for a named agent destination; a named profile lists `skills` as skipped. Files are whole-file owned (ownership kind `skill`) with create-only backups of adopted files, drift checks, release on `use` (released folders are removed once empty; adopted files are restored), and undo. A symlink or non-directory on the destination path is a conflict naming it; files in an adopted folder that profile-mango does not manage are kept and listed in a plan warning. The compact plan prints `skills: a, b` per target. Residual risk: the skills page carries no release pin, so discovery is not tied to `2.1.278` source.
