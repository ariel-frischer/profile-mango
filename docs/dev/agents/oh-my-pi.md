# Oh My Pi configuration reference

**Reference date:** 2026-09-21, requalified 2026-10-03. **Documentation pin:**
release `v18.6.0`, commit `89d2610993af69427574bde17791df63906ec4e5` (earlier
observations below remain versioned).
See [requalification](#requalification-to-v1860-2026-10-03). **Status:** profile-mango
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

This section records the original `v18.2.6` qualification. The task-owned
checkout was `.external/oh-my-pi` at the original tag and commit above.
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
slot of a role the new route drops. It also records
`written-sha256:config.modelRoles.<slot>=<hex>` (and the same for
`config.task.maxEffort`), the hash of the value it wrote: when Oh My Pi
re-serializes `config.yml` or other keys change, install compares each owned
value with that hash instead of the whole-file hash, so only an edited owned
value needs `--override`. Profile role definitions (description,
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

Profile `agentFiles.oh-my-pi` copies native `*.md` agent files verbatim into the
same directory (kind `agent-file`, same gates and ownership). Because user agents
precede bundled ones, a user `scout.md` replaces the bundled `scout`. See
[target evidence](../target-evidence.md#profile-agent-files-2026-09-26-ap-8x5).

## Named profile overlay

`mango install <name> --target oh-my-pi` writes a Mango-owned whole file,
`profiles/<name>.yml` beside `config.yml`, holding the same `modelRoles`
selectors and `task.maxEffort` the default install writes, and prints
`use it: omp --config <path>`. `config.yml` stays byte-identical unless
`--default` is also given, which patches it as above. Undo removes the overlay
or restores its previous bytes. Native `--profile` is not used: it relocates
the agent home, including `auth.json` and sessions
(`packages/coding-agent/src/cli/profile-bootstrap.ts:77-229`,
`packages/utils/src/dirs.ts:110-132`).

Pinned source, `78b753124d11f8dd3ae73e2524125890ff7c977e`:

| Claim | Pinned source (file SHA-256) |
| --- | --- |
| `--config <file>` is repeatable and handed to settings init | `packages/coding-agent/src/cli/flag-tables.ts` (`934b82db61b5567defebbf98f42a81e53615210fb89d175feb4e8343fd98a27b`): 117-119; `packages/coding-agent/src/main.ts` (`1c4173f9b4a07b41fe4e699346a567ba44704e6cd087001b439834e35f59b55d`): 1735 |
| Overlays resolve `~` and relative paths, deep-merge after global and project config and before runtime overrides; a missing, unparsable, or non-mapping file throws | `packages/coding-agent/src/config/settings.ts` (`b21706c954cf21c545a333aba7537212b5ce10b72370d4f49dddf1fcfd0ccfdc`): 603-605, 1957-1998, 3131-3135 |
| The session's default model and thinking level come from the merged `modelRoles.default` | `settings.ts` 1342-1346 `getModelRole`; `packages/coding-agent/src/sdk.ts` (`566103f262f8e5dd80df2d0ea3e46b57ffe024eac7444d00be230654d2df9c64`): 1538, `pickInitialThinkingLevel` |

Installed-binary observation (`omp/18.3.2`, not the pinned version; ELF
SHA-256 `8cbbcd4bea7a7b86116a13352f31e3778fd4d93df931036bb1771738b0702534`):
in a disposable `HOME` with `unshare -rn` (no network), an emptied environment
and a synthetic `ANTHROPIC_API_KEY`, `config.yml` held
`default: "anthropic/claude-sonnet-4-5:low"` and the overlay
`default: "anthropic/claude-opus-4-5:high"`. `omp --config <overlay> --mode json
-p hi` recorded `model_change` `anthropic/claude-opus-4-5` and
`thinking_level_change` `high`; without `--config`, `claude-sonnet-4-5` and
`low`. Both requests failed with a connection error, and neither file changed.
A missing overlay path aborted startup with `Config overlay not found`.
Evidence level: pinned source plus that newer-binary observation; slot
consumers and `task.maxEffort` in the overlay rest on the source review above.

## Requalification to v18.3.2, 2026-09-26

Tag `v18.3.2` is commit `7853b4e499936f9dcc13c9b64adb55f6b342aabf` (blobless
clone of `can1357/oh-my-pi`, read with `git show v18.3.2:<path>`). The
installed `omp` ELF, SHA-256
`8cbbcd4bea7a7b86116a13352f31e3778fd4d93df931036bb1771738b0702534`, equals
the digest of the GitHub release asset `omp-linux-x64` for `v18.3.2` (the
release is not marked immutable). `packages/coding-agent/package.json` is
`2d07016b28d5d866c4947b4aa273ae31dbd29655513c508d9c658d024a721830` (the new
`EvidenceSHA256`); `bun.lock` is
`aa191001b0daac1cc4b21a4a6be4fd613234e19958308657b56313708d9fafb1`.

Every file cited above was diffed from `v18.2.6` to `v18.3.2`:

| Claim | v18.3.2 source (file SHA-256) | Result |
| --- | --- | --- |
| `modelRoles` is a string record | `settings-schema.ts` was deleted; settings are now declared per domain. `packages/coding-agent/src/config/model-settings.ts` (`b8a89791a8c32f59fd6732e668352bdd21dba24c92e1f2f395e817c4c4d68de2`): 97 `cfgModelRoles = register({ id: "modelRoles", type: "record" })`, no `env` | unchanged |
| Built-in roles | `packages/coding-agent/src/config/model-roles.ts` (`7a306f6445810c3361de70dc9cce799589ef0abc11b618194758c443711f479e`): 55-90 `MODEL_ROLES` | new slots `memory` (chat section, accepts tiny or chat models), `image`, `web`, `speech`, `dictation`, `judge` (kind section); earlier nine unchanged |
| `memory` falls back to a configured `tiny` | `packages/coding-agent/src/config/model-resolver.ts` (`a4496c33782ad2bcb0529c0e88067e130b4046f37d62904e1589a986be88e597`): 1091-1109 `ROLE_CONFIGURED_FALLBACK` | new; Mango's `tiny` slot now also drives an unset `memory` |
| `:<level>` suffix parsing, `@<role>` resolution | `packages/tui/src/overlays/model-selector.ts` (`32cf3259…`, identical); `model-resolver.ts` 1012-1038, 1261-1282 identical to the earlier 984-1010, 1225-1246 | unchanged |
| Bare wire-tier ids imply a level | `model-resolver.ts` 852-866 `inferWireRouteThinkingLevel`, used on exact full-pattern matches (896) | new; a bare selector naming a retired wire-tier id (such as `gemini-3.8-flash-high`) now counts as an explicit level. Suffixed selectors are unaffected |
| Default role's suffix picks the initial level | `packages/coding-agent/src/sdk.ts` (`79c3ad9645a34d82b0dde2ce6033313aa018f0810dcce5ab029e6d00e1e25c0e`): 1734, 1826 `pickInitialThinkingLevel` | unchanged logic |
| `task.maxEffort` | `packages/coding-agent/src/task/settings.ts` (`34f0a771313071ead747961df008b519fedc785800659699e5ef3c22065cc85f`): 343-356, enum `THINKING_EFFORTS`, default `max`, no `env`; `packages/catalog/src/effort.ts` and `packages/tui/src/thinking.ts` identical | unchanged values |
| Ceiling read only with a per-spawn hint | `packages/coding-agent/src/task/executor.ts` (`82057a1a96dfc3cd7d1a94ce3392a846467c32e7ced361f6381a9aa66b7cadc0`): 3701-3704 | unchanged; new: 3841 also passes the ceiling to the child session as `thinkingLevelCeiling`, so retry fallback cannot raise effort past it |
| Slot consumers | `task/agents.ts` identical (54 `@task`, 68 `@smol`); `modes/interactive-mode.ts` (`19e7e4f7…`) 3972, 4000 `plan`; `prompts/agents/reviewer.md` (`35d17237…`) `@slow`; `prompts/agents/scout.md` (`04bfabcf…`) `@smol`; `commit/model-selection.ts` (`8f0b8ded…`) 46 `["commit", "smol", ...]`; `utils/title-generator.ts` (`cb637d01…`) 128 `["tiny", "commit", "smol"]` | unchanged |
| `--config` overlays | `cli/flag-tables.ts` (`af6578ff…`) 117-119; `main.ts` (`8b303fb4…`) 1798; `config/settings.ts` (`2ae0e992e2cbd0b5c3b89a5ec08b473eba97dcf1704097ae306205d306394219`) 610-612, 2243-2287 (missing file: `Config overlay not found`), 3614-3618 `#mergeOwnLayers` global, project, overlay, runtime | unchanged order; `docs/config-usage.md` now lists a setting's declared environment variable above all layers, and neither `modelRoles` nor `task.maxEffort` declares one |
| `getModelRole`, `setByPath` | `settings.ts` 1617-1621, 192 | unchanged |
| Legacy `providers.tinyModel` / `memoryModel` | `settings.ts` 3171-3185 in `#migrateRawSettings` (2357), applied to each loaded layer | new; a legacy value is prepended as `local/<model>` to that layer's `tiny`/`memory` slot. Mango does not write these keys; a user's legacy key in the same file changes the effective `tiny` chain |
| Agent frontmatter and discovery | `discovery/helpers.ts` (`946f9775…`) 280, 306-365; `task/discovery.ts` identical | unchanged |
| Native `--profile` relocation | `cli/profile-bootstrap.ts` identical; `packages/utils/src/dirs.ts` (`a47c7f44…`) 113-135 | unchanged |
| Global/home instruction files | `discovery/builtin.ts` (`70a3e5ec…`) 397 `RULES.md`, 917 `AGENTS.md`; `discovery/agents-md.ts` identical; `helpers.ts` 687-714 | unchanged |

Installed-binary observation (the exact `v18.3.2` ELF above): the worktree's
compiled `mango install default --target oh-my-pi --apply` in a disposable
`HOME` wrote `profiles/default.yml` with all seven slots (`default`, `task`,
`plan`, `slow`, `smol`, `tiny`, `commit`, each suffixed) and `task.maxEffort:
"medium"`, leaving `config.yml` (`default` `:low`, bare `tiny`, `maxEffort:
xhigh`) byte-identical. Under `unshare -rn` with `env -i`, `omp config list
--json` with `PI_CONFIG_FILES=<overlay>` returned the overlay's seven slots and
`medium`; without it, `config.yml`'s two slots and `xhigh`. `config.yml` and the
overlay were unchanged afterwards (omp created only logs, `agent.db`, and
extracted natives in the sandbox). `omp --config <overlay> config ...` did not
apply the overlay to the `config` subcommand, so the overlay was supplied through
`PI_CONFIG_FILES`, which `settings.ts` 610-612 loads before `--config` files in
the same overlay layer. Slot consumers and the effort ceiling at spawn time rest
on the source review above.

## Requalification to v18.4.6, 2026-09-30

The [evidence ledger](../target-evidence.md#oh-my-pi-requalification-to-v1846-2026-09-30)
records the relevant `v18.3.2..v18.4.6` config/skills source diff and the
isolated installed-ELF positive/control probe. Tag `v18.4.6` resolves to
`8b25ad4a05625dde65df41d057756b4815f4837c`. Installed release ELF SHA-256
`9eb0668d3ccab78c39598b7717461f8621136114618387da22b3ddd454fb163b` matches
the official Linux x64 asset; the release is not immutable. The new tested
range is `>=18.4.6 <18.5.0`, not all earlier 18.4 builds.

Source file hashes (paths relative to `packages/coding-agent/`):

| File | SHA-256 |
| --- | --- |
| `package.json` (adapter evidence hash) | `bca974df221fecc1e706c642e1329a761849e46f76554d3df587f3629c8864f4` |
| `src/config/model-settings.ts` | `d0cef41cc2053d55f78d459c11d7efd3a85311cd9332730a846cde2664340f91` |
| `src/config/settings.ts` | `7746bc9f3b881868faaf99c72d757ed85e17fef764c42204684b49f1797bd91b` |
| `src/config/model-resolver.ts` | `85e9b57c5a804bd8edeaf90c9eb2f5e8beb85a2867e4279b55bd774ccfda571d` |
| `src/sdk.ts` | `97fc3bb3cd9ffe43b190da942c8ac326a62975484d46bc87f8c8d1ebf3db982d` |
| `src/task/settings.ts` | `fb71841ec1d77e87833b0a1320b9b40028f2f56652060f4a7c633d68b8bcc73c` |
| `src/task/executor.ts` | `1d1d31ea16b74bcde15366ee94dcd93fc84ca130330805b4becdd596dadaa077` |
| `src/discovery/builtin.ts` | `71652a999ae41e0e95a0e7b9e86e23d33f9322397cfad00eb9ec3cfae5b31600` |
| `src/discovery/helpers.ts` | `2285212a57765dbb314f73f695cec2958a4a7955a3a4847b88f2652f932c4123` |
| `src/extensibility/skills.ts` | `50892b528c20ab1fc901dad90e3900f05433d64767710e288d1821c96f50ea85` |

Settings storage, overlay consumption and all seven generated role values
were observed with `omp config list --json` in a disposable, timeout-bounded
network/PID/IPC/UTS-isolated Bubblewrap sandbox with cleared environment and
hidden personal homes. With `PI_CONFIG_FILES=<Mango overlay>`, the getter
returned generated suffixed selectors and `task.maxEffort: medium`; without
it, control selectors and `xhigh`. Both input files remained byte-identical.
No live model, suffix interpretation, task consumer, skill delivery, auth or
enforcement qualification is added. Pi remains at installed/qualified `0.87.1`;
`0.99.1` was not requalified without an installed binary.

## Preview adapter boundary

The CLI target is `oh-my-pi` with the exact version `18.4.6`. The renderer emits
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

[settings]: https://github.com/can1357/oh-my-pi/blob/v18.4.6/docs/settings.md
[providers]: https://github.com/can1357/oh-my-pi/blob/v18.4.6/docs/providers.md
[models]: https://github.com/can1357/oh-my-pi/blob/v18.4.6/docs/models.md
[approval]: https://github.com/can1357/oh-my-pi/blob/v18.4.6/docs/approval-mode.md
[context]: https://github.com/can1357/oh-my-pi/blob/v18.4.6/docs/context-files.md
[skills]: https://github.com/can1357/oh-my-pi/blob/v18.4.6/docs/skills.md

## Global instruction files (ap-nym, 2026-09-25)

`globalInstructions` may own `AGENTS.md` and `RULES.md` in `~/.omp/agent` (source `v18.4.6` `8b25ad4`, `packages/coding-agent/src/discovery/builtin.ts:398,918`; the RULES loader also honors an SDK agent-dir override). Written only when the profile is the default (`mango use`, `install --default`); whole-file ownership with create-only backup, drift checks, release on `use`, and undo. See [target evidence](../target-evidence.md#global-instruction-files-2026-09-25-ap-nym).

## Home `~/AGENTS.md` (ap-5mp, 2026-09-25)

`~/AGENTS.md` is read only when no repository encloses the working directory or the repository root is `$HOME`: `v18.4.6` `8b25ad4`, `packages/coding-agent/src/discovery/agents-md.ts:20-22`, `packages/coding-agent/src/discovery/helpers.ts:704-731` (same traversal logic as `v18.3.2`). Oh My Pi therefore does not own `globalInstructions.home`. See [target evidence](../target-evidence.md#home-instruction-file-agentsmd-2026-09-25-ap-5mp).

## Skill folders (ap-794, 2026-09-27)

Pinned source `v18.4.6`: the [native loader](https://github.com/can1357/oh-my-pi/blob/v18.4.6/packages/coding-agent/src/discovery/builtin.ts) reads user skills from `~/.omp/agent/skills` (the directory holding `config.yml`; `PI_CONFIG_DIR` moves both, see [dirs](https://github.com/can1357/oh-my-pi/blob/v18.4.6/packages/utils/src/dirs.ts)) and requires a `description` for native skills; `name` defaults to the folder name. Provider priority still gives a native skill the bare name over a same-name Codex skill under `~/.agents/skills`; the [skill loader](https://github.com/can1357/oh-my-pi/blob/v18.4.6/packages/coding-agent/src/extensibility/skills.ts) now collapses identical body/frontmatter duplicates and namespaces differing same-name skills, while explicit custom directories can displace the bare name. Raw names with path separators are rejected. These discovery/collision changes are source-reviewed only.

Each `skills` entry's whole folder (every regular file beside its `SKILL.md`, at most 256, modes normalized to `0755` when any execute bit is set and `0644` otherwise) is copied to `<folder name>/` under `<config dir>/skills` (by default `~/.omp/agent/skills`), with `{{route.…}}` placeholders rendered per target in UTF-8 text files (provenance digests cover the source bytes), written only when the profile is the default (`mango use`, `install --default`) and never for a named agent destination; a named profile lists `skills` as skipped. Files are whole-file owned (ownership kind `skill`) with create-only backups of adopted files, drift checks, release on `use` (released folders are removed once empty; adopted files are restored), and undo. A symlink or non-directory on the destination path is a conflict naming it; files in an adopted folder that profile-mango does not manage are kept and listed in a plan warning. The compact plan prints `skills: a, b` per target.

## Requalification to v18.6.0, 2026-10-03

Tag `v18.6.0` resolves to `89d2610993af69427574bde17791df63906ec4e5`.
The evidence identity remains the source `packages/coding-agent/package.json`
SHA-256: `2fcbf1a46b27ad7c631bb210abaa217dc092a6e629a74f977a8bde07b59c6d24`.
The exact qualified version moves to 18.6.0 (tested range `>=18.6.0 <18.7.0`).
18.4.6's evidence remains historical, not an exact registered install target:
18.5 changed the effort cap's scope and introduced a task setting migration.

Reviewed source contracts:

- `config/model-roles.ts:56-66` retains all seven Mango slots, with hash
  `7a306f6445810c3361de70dc9cce799589ef0abc11b618194758c443711f479e`.
- `config/model-settings.ts` still registers `modelRoles` as a record;
  `config/model-resolver.ts:924-949` interprets `:effort`. Task agent `@task`,
  scout `@smol`, reviewer `@slow`, plan-mode `plan`, title `tiny/commit/smol`,
  and commit `commit/smol` consumers retain the portable mapping.
- `task/settings.ts:369-382` retains `task.maxEffort`, the `THINKING_EFFORTS`
  enum and `max` default. `task/executor.ts:4032-4035,4166` still caps a per-call
  effort hint; 18.5 additionally caps the interactive `/effort` picker.
  `task.completionProbeMs` migrates per layer to boolean `task.completionProbe`
  (`config/settings.ts:3118-3133`); Mango owns neither key. Task items now
  accept per-call model selectors; Mango does not manage that runtime input.
- `config/settings.ts:661-663,2425-2463,3826-3830` retains repeatable overlay
  loading and global → project → overlay → runtime ordering.
- `task/discovery.ts:55-58,78-96` discovers user `agents/*.md`;
  `discovery/helpers.ts:312-318,368` still requires name/description and parses
  the model list. These are source-reviewed, not runtime discovery evidence.

Native parsing, model resolution, authentication and enforcement are separate
claims. Earlier native storage evidence is not relabelled as 18.6.0 runtime
evidence.


## Binding-route presets (ap-ynf)

`config/model-presets.ts` SHA-256
`642cdcd78af5493f2f038900667af50803c540ed7ce957a4fc01e37610ef6472`
defines the name grammar (24-28), validates roles/thinking (34-50), saves only
one global entry (104-114), and switches complete role snapshots.
`settings.ts:1793-1825` resolves a preset whole from its highest owning layer,
not by merging same-name snapshots; overlay/runtime null hides a lower entry.
Mango writes `mango-<escaped route>` entries to global config for default
installs, and only the overlay for named-only installs. Seven managed slots
are always present (unbound slots use default); task.maxEffort and resources
are deliberately not part of a preset. The field ownership manifest carries
each preset key, a semantic hash, and raw prior-entry bytes for explicit
adoption/release. Ordinary YAML comments and unrelated presets stay untouched.


## Recognizing preset switches (ap-aqw)

Mango status recognizes complete config-file matches of manifest-owned,
hash-intact preset entries (all role keys and the recorded thinking level).
It does not infer a match from a partial/default-only overlap or inspect a
running omp session. Such a match permits the next Mango plan to replace role
values without a drift override; task.maxEffort and preset-entry edits remain
protected. Status exposes the matching route and any difference between that
route's subagent cap and the applied config cap. The cap never follows an omp
preset switch. Undo remains transaction/whole-file guarded rather than treating
a preset switch as an implicit approval to restore the entire file.

