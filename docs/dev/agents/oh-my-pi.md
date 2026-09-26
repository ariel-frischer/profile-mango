# Oh My Pi configuration reference

**Reference date:** 2026-09-21. **Documentation pin:** release `v18.2.6`,
commit `78b753124d11f8dd3ae73e2524125890ff7c977e`. **Status:** profile-mango
ships an inert preview renderer and, since 2026-09-22, a model-role installer
whose `modelRoles.default` storage is qualified through exact source-native
read-only getters; since 2026-09-25 it writes suffixed per-role selectors on
source-review evidence only (see [per-role selectors](#per-role-selectors-2026-09-25)).
Broader applicability remains blocked. Oh My Pi is qualified independently from
[Pi](pi.md).

## Configuration and precedence

Pinned documentation describes YAML at `~/.omp/agent/config.yml` and
`<cwd>/.omp/config.yml`, plus legacy JSON migration. Precedence is built-ins,
global configuration, project configuration, repeated CLI overlays, then runtime
or environment overrides. Objects deep-merge and arrays replace.

The pinned source defines `modelRoles` values such as `provider/modelId`, with
an optional `:<level>` thinking suffix, and `defaultThinkingLevel` values such
as `minimal`, `low`, `medium`, `high`, `xhigh`, `max`, and `auto`. The inert
adapter emits only `modelRoles` selectors. It never emits authentication values,
provider URLs, API keys, credential references, or active target paths.

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
`defaultThinkingLevel` at an explicit or [default config destination](../target-evidence.md#default-config-destinations-2026-09-22) path. Actual compiled CLI output was loaded
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

## Per-role selectors, 2026-09-25

Source review of the same pinned tag and commit (a fresh read-only checkout,
`git rev-parse HEAD` = `78b753124d11f8dd3ae73e2524125890ff7c977e`) extends the
installer to one `modelRoles.<role>` selector per bound route role and moves
effort onto each selector:

| Claim | Pinned source (file SHA-256) |
| --- | --- |
| `modelRoles` is a string record; any role key is stored as written | `packages/coding-agent/src/config/settings-schema.ts` (`080e5c2b040afebdcc629a9692d0eac21aa78e75b0fd1f247abe1b59a0023153`): `modelRoles: { type: "record" }`, `Record<string, string>` |
| Built-in roles are `default`, `smol`, `slow`, `vision`, `plan`, `commit`, `tiny`, `task`, `advisor` | `packages/coding-agent/src/config/model-roles.ts` (`5cd9142582adfc6a773fa83858806914f9314506f1f504a59f484e65e0cbf54c`): `MODEL_ROLES` |
| A role value may end in `:<level>`; `minimal`..`xhigh`, `max`, and `auto` are recognized | `packages/tui/src/overlays/model-selector.ts` (`32cf3259183137dd7eca36cc2e3e9c1394df229e787b26f3bd9518eab5578d8f`): `splitThinkingSuffix`; `packages/coding-agent/src/config/model-resolver.ts` (`68b229552ea292ddf11a3c7076110dd530de425f93d113224e03028ab0574225`): `parseModelPatternWithContext` passes `MAX_THINKING_SUFFIX_OPTIONS`, `resolveModelRoleValue` returns `explicitThinkingLevel` |
| The default role's explicit suffix wins over the model default and `defaultThinkingLevel`; an explicit `--model` or a persisted session level skips it | `packages/coding-agent/src/sdk.ts` (`566103f262f8e5dd80df2d0ea3e46b57ffe024eac7444d00be230654d2df9c64`): `pickInitialThinkingLevel` |
| Documentation shows suffixed role values | `docs/settings.md` (`0b750e8d48928bec4751deebfa393c3960327646e00ba23699bfcedf64419f31`): `slow: anthropic/claude-opus-4-5:high`, "Role values may carry a thinking suffix"; `docs/models.md` (`e997ad228ceddb2686a06344b553e9e0d212834caeefec21ea5b5fe395a887ab`) |

The installer therefore writes `modelRoles.default: provider/model:effort` and
no longer writes `defaultThinkingLevel`, so hand-picked models keep their own
thinking default. A role without effort is written bare, and a bare model whose
last `:` segment the parser would read as a level (including unambiguous
prefixes such as `:hi`) is rejected until effort is set.

Evidence level: source review only. The 2026-09-22 native getter probe covered
`modelRoles.default` storage, not suffix interpretation or non-default roles,
and was not re-run. The plan warning says so.

## Portable roles and `task.maxEffort`, 2026-09-25

Bindings roles now use the portable names `worker`, `planner`, `research`, and
`tiny`; the installer expands each into the slots whose consumers match its
purpose. Source review of the same checkout
(`78b753124d11f8dd3ae73e2524125890ff7c977e`):

| Portable role | Slots | Consumer in pinned source (file SHA-256) |
| --- | --- | --- |
| `worker` | `task` | `packages/coding-agent/src/task/agents.ts` (`c978977edd83b8f05759763af9b931243390fbe95cd5ca4437a352fda90710e0`): bundled task agent `model: "@task"` (line 54) |
| `planner` | `plan`, `slow` | `packages/coding-agent/src/modes/interactive-mode.ts` (`f37f5f7389f008ba8929751b13eece5a827a49d9ac98aba2e702741551f35254`): plan mode `resolveRoleModelWithThinking("plan")` (line 3482); `packages/coding-agent/src/prompts/agents/reviewer.md` (`c6abab43dcf55776830225a8a8b277fc39b9e3bbdc49271635c28e309ec9c2c0`): `model: "@slow"` |
| `research` | `smol` | `packages/coding-agent/src/prompts/agents/scout.md` (`c65a6e928a80009acac0510c17bd70dc5cdc688121ffd30976c7e6b0e8ce58fe`): scout `model: "@smol"`; `task/agents.ts` line 68 `@smol` |
| `tiny` | `tiny`, `commit` | `packages/coding-agent/src/utils/title-generator.ts` (`f76bd8fec0d2f3e1fd0b3ecd92243ca28413729e07b5e00f4b13b8f224889c8b`): candidates `["tiny", "commit", "smol"]` (line 126); `packages/coding-agent/src/commit/model-selection.ts` (`2cdc1f844f5072e62f8924a18be9dfb613a6335d9569354b9c361f86d930605e`): `["commit", "smol", ...]` (line 46) |

`advisor` and `vision` have no portable role and are never written. Each slot
keeps its own manifest field (`config.modelRoles.<slot>`) and
`ohmypi-role-prior:<slot>=<value>` marker, so `mango use` gives back every
slot of a role the new route drops. Profile role definitions (description,
instructions) install as agent files; see the next section.

`task.maxEffort` is a cap on subagent effort, so bindings model it as the
optional route field `subagentMaxEffort`:

| Claim | Pinned source (file SHA-256) |
| --- | --- |
| `task.maxEffort` is an enum of `THINKING_EFFORTS`, default `max`, "Maximum Per-Spawn Effort"; "Lower values prevent callers from escalating subagents above this ceiling" | `packages/coding-agent/src/config/settings-schema.ts` (`080e5c2b040afebdcc629a9692d0eac21aa78e75b0fd1f247abe1b59a0023153`), line 5222 |
| `THINKING_EFFORTS` is `minimal`, `low`, `medium`, `high`, `xhigh`, `max` (no `auto`) | `packages/catalog/src/effort.ts` (`78858ce4b34cc03759dd49bcbdb6243821deaae2624afc8ee833f890b8a2dcb3`) |
| The ceiling is read only when the task caller passes a per-spawn effort hint | `packages/coding-agent/src/task/executor.ts` (`c6012cb3fa82e4cc8b2d034deffa674a9472224eabd0d81c8c803ec1c490b0f0`), line 3594 |
| The hint maps onto the model's supported range, then clamps to the highest supported level at or below the cap; a model with none throws | `packages/tui/src/thinking.ts` (`7500a8a9ccd1bd2118e207d37ad0faca5745c4400990f6ca3edbbdd6aff9aad6`): `resolveTaskEffortLevel` |
| Dotted keys are stored as nested YAML (`task:` / `maxEffort:`) | `packages/coding-agent/src/config/settings.ts` (`b21706c954cf21c545a333aba7537212b5ce10b72370d4f49dddf1fcfd0ccfdc`): `setByPath` |

The installer writes `task.maxEffort` in place or inserts a `task:` block,
owns `config.task.maxEffort`, records `ohmypi-setting-prior:task.maxEffort=<prior>`,
and on `mango use` without the field restores the prior value or removes the
key (and a `task:` block left empty). It does not affect spawns that pass no
effort, nor role slot efforts. Other targets skip it as `subagentMaxEffort`.

Evidence level: source review only; no native probe of slot consumers or of
`task.maxEffort` was run.

## Role subagent files (ap-6lp, 2026-09-25)

Each declared profile role becomes `~/.omp/agent/agents/<role>.md`, written
only when the profile is the default (`install --default`, `mango use`).
Source review of `78b753124d11f8dd3ae73e2524125890ff7c977e`:

| Claim | Pinned source (file SHA-256) |
| --- | --- |
| Agent frontmatter requires `name` and `description` (else the file is ignored); `model` is parsed as a model list | `packages/coding-agent/src/discovery/helpers.ts` (`ce3ad96706c22f49c11c8047a94d87d0cb48210d34237bb802355a6dde7a4206`), lines 279, 305-364 |
| `*.md` files load from the user `agents` config dir; precedence is project, user, extensions, plugins, bundled | `packages/coding-agent/src/task/discovery.ts` (`524de1af7e9c58ac34991a998cf6f3d60b823493cd503bbcc74b1aadd8035353`), lines 51-105 |
| An `@<role>` model resolves through `modelRoles.<role>`, keeping its `:effort` suffix | `packages/coding-agent/src/config/model-resolver.ts` (`68b229552ea292ddf11a3c7076110dd530de425f93d113224e03028ab0574225`), lines 984-1010, 1225-1246 |

The file carries `name`, `description`, the role's `instructions` (else its
description) as the body, and, when the route binds the role, `model: "@<slot>"`
naming the role's first slot (`worker` -> `@task`, `planner` -> `@plan`,
`research` -> `@smol`, `tiny` -> `@tiny`). The same install writes that slot's
`provider/model:effort` selector, so model and effort come from one place; the
agent file never repeats them. Ownership and the named-only `role-definitions`
skip match Codex and OpenCode. Evidence level: source review plus sandbox
install tests; no native probe of agent discovery was run.

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

## Global instruction files (ap-nym, 2026-09-25)

`globalInstructions` may own `AGENTS.md` and `RULES.md` in `~/.omp/agent` (source `v18.2.6` `78b7531`, `packages/coding-agent/src/discovery/builtin.ts:393,913`). Written only when the profile is the default (`mango use`, `install --default`); whole-file ownership with create-only backup, drift checks, release on `use`, and undo. See [target evidence](../target-evidence.md#global-instruction-files-2026-09-25-ap-nym).

## Home `~/AGENTS.md` (ap-5mp, 2026-09-25)

`~/AGENTS.md` is read only when no repository encloses the working directory or the repository root is `$HOME`: `v18.2.6` `78b7531`, `packages/coding-agent/src/discovery/agents-md.ts:20-22`, `packages/coding-agent/src/discovery/helpers.ts:684-711`. Oh My Pi therefore does not own `globalInstructions.home`. See [target evidence](../target-evidence.md#home-instruction-file-agentsmd-2026-09-25-ap-5mp).
