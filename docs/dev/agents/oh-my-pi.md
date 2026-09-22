# Oh My Pi configuration reference

**Reference date:** 2026-09-21. **Documentation pin:** release `v18.2.6`,
commit `78b753124d11f8dd3ae73e2524125890ff7c977e`. **Status:** profile-mango
ships an inert preview renderer and, since 2026-09-22, a two-field installer
qualified through exact source-native read-only getters. Broader applicability
remains blocked. Oh My Pi is qualified independently from
[Pi](pi.md).

## Configuration and precedence

Pinned documentation describes YAML at `~/.omp/agent/config.yml` and
`<cwd>/.omp/config.yml`, plus legacy JSON migration. Precedence is built-ins,
global configuration, project configuration, repeated CLI overlays, then runtime
or environment overrides. Objects deep-merge and arrays replace.

The pinned source defines `modelRoles` values such as `provider/modelId` and
`defaultThinkingLevel` values such as `minimal`, `low`, `medium`, `high`,
`xhigh`, `max`, and `auto`. The inert adapter emits only these documented route
and effort fields. It never emits authentication values, provider URLs, API keys,
credential references, or active target paths.

## Permissions, tools, instructions, and skills

Approval mode and per-tool allow, deny, and prompt rules are documented, with
deny winning. Approval is not assumed to be process or filesystem containment.
Oh My Pi has native and compatibility context-file discovery and explicit skill
source precedence. The adapter therefore emits delivery, precedence, permission,
and tool diagnostics instead of claiming equivalence or enforcement.

## Exact source observation

The task-owned checkout is `.external/oh-my-pi` at the exact tag and commit above.
The frozen `bun.lock` was installed with Bun `1.3.14`. The direct source entrypoint
was run only with `--version`, an isolated task-owned `HOME`, and a sanitized
environment:

```text
omp/18.2.6
Bun SHA-256: 9fd36f87e4b90b07632b987a2e4ec81ca15a62c81bf983190cea6d715be2ad74
packages/coding-agent/package.json SHA-256: 4d9558530fdd8c76798181545d7cde8b558731596515b2f300e61c8d403bcb6e
bun.lock SHA-256: b74fbff79c5acbacc6bc46f4180e070cb5edb9853c8c5a00ff97d9c8734d4a70
```

The adapter evidence hash is the exact `packages/coding-agent/package.json` hash,
not a claim that a standalone target binary was produced.

The official pinned build entrypoint was attempted after the frozen dependency
install:

```bash
bun run packages/coding-agent/scripts/build-binary.ts
```

It stopped in native embedding with:

```text
No native addons found for linux-x64.
Expected pi_natives.linux-x64-modern.node / pi_natives.linux-x64-baseline.node
```

This missing native addon was the initial support blocker. No mutable release download,
wrapper, global tool installation, or alternate version was used.

## Candidate inspection and effects

Source review covered `packages/coding-agent/src/cli.ts`,
`packages/coding-agent/src/cli/config-cli.ts`,
`packages/coding-agent/src/config/settings.ts`, and
`packages/coding-agent/src/config/config-file.ts`. `config path` and
`config list --json` were **not executed**. The command wrapper initializes
`Settings` before dispatch, while settings initialization opens agent storage,
loads global and project configuration, discovers overlays, and may migrate legacy
JSON or seed state. Those effects are not accepted as a side-effect-free inspector
for profile-mango M0.

A config probe was not retained in the initial review because the source and missing native
addon hit the stop conditions. In particular, no provider, authentication, session,
hook, extension, child, personal home, credential store, or target-home operation
was run. A future probe must use a direct exact artifact, task-owned config and
project roots, sanitized environment, blocked network, bounded timeout, and unique
synthetic secret sentinels before it can add runtime evidence.

## Narrow installer qualification, 2026-09-22

The exact source addon was subsequently built with `nightly-2026-08-12`
(`rustc 1.99.0-nightly (3d6c19bb9 2026-08-11)`) and the pinned Bun runtime above.
`pi_natives.linux-x64-modern.node` has SHA-256
`9632a05bc6460c65b7fcbe8653c44cb757217a91502ab7faeb0d5241be70372e`.
The build used upstream `scripts/bazel-natives.ts host --dest packages/natives/native`.

The separate installer patches only `modelRoles.default` and
`defaultThinkingLevel` at an explicit path. Actual compiled CLI output was loaded
through exact `Settings.loadReadOnly`, with model-role and thinking getters
returning the installed synthetic values. A control without either field returned
no default model role and the built-in `high` thinking level. Unknown-key and
unmanaged-role sentinels survived unchanged. The root probe used a cleared
environment, isolated network/PID/IPC/UTS, dropped capabilities, no `/etc` mount,
hidden personal homes/sockets, read-only source/runtime mounts, separate disposable
cache, and a 180-second timeout. Content hashes and modes showed no target writes.

This is settings-module consumption, not a standalone binary, model resolution,
authentication, full precedence, delivery, session startup, or enforcement test.
The installer never launches the target. Unsupported requirements remain blocking.
Lossless patch tests cover comments, unknown fields, CRLF, Unicode columns,
document-end markers and rejection of duplicate keys, multiline managed scalars,
anchors, aliases and merge keys. Shared transaction tests cover backups, stale
plans, ownership, fault rollback and guarded recovery.

## Preview adapter boundary

The CLI target is `oh-my-pi` with the exact version `18.2.6`. The renderer emits
`preview/<profile>.config.yml.preview` and digest-verified resource copies that
preserve their original relative paths.
Every report is `applicable: false` and includes explicit blockers for:

- authentication identity and credential handling
- config-inspector effects and source/project precedence
- missing standalone native artifact
- route transport mapping
- permissions and closed tool or approval enforcement when requested
- instruction and skill delivery precedence when requested

The candidate is deterministic syntax, not an active `~/.omp` configuration. See
the [target evidence ledger](../target-evidence.md) and
[adapter architecture](../adapter-architecture.md) for the exact report and
failure-boundary evidence.

[settings]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/settings.md
[providers]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/providers.md
[models]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/models.md
[approval]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/approval-mode.md
[context]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/context-files.md
[skills]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/skills.md
