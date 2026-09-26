# OpenCode configuration reference

**Reference date:** 2026-09-23. **Status:** profile-mango has an exact-version,
inert preview renderer and bounded installation for OpenCode `v1.18.31`. An
install without `--agent` writes a Mango-owned primary agent definition
(model and instructions) to `agents/<profile>.md` beside the config by
default; an explicit `--agent opencode@1.18.31=primary:name` or
`subagent:name` writes the same kind of definition at an explicit
`--config` path instead. `opencode.json`/`opencode.jsonc` is left
byte-identical unless `--default` is also passed, which additionally patches
the top-level `model` field and one owned skill resource into that file,
since OpenCode's `skills.paths` is a directory-wide setting rather than a
per-agent one. Native isolated consumption is verified for the JSONC model,
single skill, and generated named agent's name/mode/model/prompt.
Authentication identity, full precedence, default activation, delegation,
permission/tool enforcement, and multi-skill delivery remain unverified.

## Immutable release evidence

The adapter is qualified against the official `anomalyco/opencode` repository,
release tag `v1.18.31`, commit
`a97622c801f4ca571530ddc51076af659a9c32cd`, published 2026-09-14:

- source repository: <https://github.com/anomalyco/opencode>
- source tag: <https://github.com/anomalyco/opencode/tree/v1.18.31>
- source archive SHA-256: `76f69fe27ec2b44e23fa1749029e7c012eb7e975a0f0c7819e9458198dfd3896`
- Linux x64 release archive SHA-256: `e9312be75ed803b7415fc2aeabda1f4fe938912a39673762dc0c38c0e11ebde4`
- extracted Linux x64 binary SHA-256: `f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11`

The source review retained these exact file hashes:

| Source file | SHA-256 |
| --- | --- |
| `packages/opencode/src/config/config.ts` | `87a9071af1ddb04d65947dba49be3fb4c94ecf63e120f17ce46ee81ff25dd45a` |
| `packages/core/src/v1/config/config.ts` | `b99bcbd98df6da79e59cda482363f759cea9d4b9792c6c8e83b6a8d686138d30` |
| `packages/core/src/v1/config/provider.ts` | `c496dea619e8e9d2d09b7a2353f2e62bff6a2276471c161c3988059a1b6ac303` |

The immutable release, source, archive, and extracted-binary hashes are separate
from the mutable official documentation and schema snapshot below.

## Configuration paths, syntax, and precedence

The reviewed source and official documentation describe JSON and JSONC layers:

- global user configuration under `${XDG_CONFIG_HOME:-~/.config}/opencode/`
- project `opencode.json` or `opencode.jsonc`
- `OPENCODE_CONFIG` for an explicit custom file
- `OPENCODE_CONFIG_DIR` for an explicit configuration directory
- `OPENCODE_CONFIG_CONTENT` for an inline final layer
- `skills.paths` for additional local skill directories scanned for `SKILL.md`

The exact source loads global state, an explicit custom file, project state,
`.opencode` and custom-directory state, inline content, and managed configuration.
The isolated probe observed inline content overriding a global model and observed
an explicit custom file being consumed. It did not establish every project,
managed, or nested-directory precedence interaction, and the resolved output does
not provide general per-field provenance.

The source and native probe establish that `model` uses `"provider/model"`.
Provider configuration remains a separate surface. Target-owned credentials remain
separate and were replaced with synthetic empty auth content during the probe. The
adapter never reads, writes, copies, or emits a real auth store, credential,
provider option, authentication identity, or transport setting; effort is written
only as a named agent's `variant` (see [Effort](#effort-as-agent-variant)).

## Isolated native validation

The exact extracted Linux x64 binary was run only through
[`scripts/opencode-config-probe.sh`](../../../scripts/opencode-config-probe.sh).
The probe used a task-owned synthetic home/project, isolated every XDG path,
provided empty auth content and an in-memory database, disabled project config,
model fetching, auto-update, pruning, default plugins, LSP downloads, and external
plugins, blocked network access with bubblewrap, and bounded every command with a
timeout. No agentic TUI, prompt, provider session, live target home, or credential
was used.

Observed results:

- `--version` returned `1.18.31`; the direct binary matched the pinned SHA-256.
- `--help` exposed the non-TUI `debug config` command.
- `debug config` natively accepted the deterministic Profile Mango JSONC candidate
  and emitted `"model": "openai/gpt-5.6"` in merged output.
- inline content overrode a global sentinel model.
- an explicit `OPENCODE_CONFIG` candidate file was consumed.
- the explicit candidate file remained byte-identical after inspection.
- malformed JSONC was rejected.
- a schema-unknown key was accepted and omitted from resolved output.
- the isolated `debug skill` probe listed and loaded a synthetic `SKILL.md` from a
  configured `skills.paths` directory without starting an agent session or making
  a model call.
- the command created target-owned scratch directories, logs, locks, `.gitignore`,
  and metadata. The independent backup was restored after testing, and recursive
  type, mode, size, and SHA-256 inventory matched byte-for-byte.

The command therefore provides exact-version parser acceptance and partial merged
state/precedence evidence. The separate skills probe establishes source-file
discovery and loaded skill content for the qualified local skill field only. These
probes do not establish a safe zero-write inspector, authentication identity,
policy enforcement, or instruction delivery.

## Inert adapter boundary

The adapter emits this deterministic candidate when the route is complete and uses
native transport:

```jsonc
// profile-mango: INERT PREVIEW ONLY
// NON-APPLICABLE: candidate syntax for OpenCode 1.18.31.
// This is not an active OpenCode config.json or config.jsonc. Authentication, provider options, effort, delivery, precedence, plugins, MCP, and enforcement are unverified.

{
  "model": "provider/model",
}
```

The candidate is written under
`preview/<profile>.opencode.jsonc.preview` by the caller's explicit preview staging
flow. Native parsing and lossless explicit-path application of that exact field are
supported for `1.18.31`; full profile fidelity is not. The renderer remains
`applicable: false` and does not emit effort, authentication, credentials, provider
options, transport, permissions, tools, instructions, skills, plugins, or MCP
configuration. Install support is qualified separately below and does not make the
inert renderer applicable.

## Narrow install-capable subset

With `--default`, `mango install` also patches the top-level `model`
field in the resolved OpenCode config file and, independently, installs
exactly one validated portable skill resource, alongside the named agent
definition described below. The skill resource is written as a target-owned
`SKILL.md` beside the config, and the config receives an absolute
`skills.paths` entry for that directory. OpenCode's `skills.paths` is a
directory-wide setting, not a per-agent one, which is why it only installs
alongside `--default` rather than by default. This main-config mode requires
exact target `opencode@1.18.31`, native transport, and a profile with no
permission or tool requirements. Multi-skill profiles and all other
unqualified fields remain blocked.
Existing unowned or externally edited config files require the adapter-approved
`--override` flag, including when adding a skill. The skill resource cannot override
an unowned or externally edited `SKILL.md`, even with that flag. Discovery scans the
config directory and may include other skills: this is not an exclusive allowlist.
The planner binds consent to a digest of the normalized config and manifest paths,
while omitting raw absolute paths from public plan JSON.

The patcher validates a top-level JSONC object, changes only the model string span
and the required `skills.paths` span, or inserts deterministic properties, and fails
closed on malformed input, duplicate or non-string model/skill fields, unsupported
encoding, or ambiguous syntax.
Comments, unknown keys, unrelated bytes, modes, provider options, and
credential-shaped target-owned state are preserved. The skill patch preserves
existing top-level config and unrelated `skills` members while adding only the
qualified path. Application reuses create-only
backups, stale-source and stale-target checks, atomic replacement, ownership
manifests, journals, rollback, and guarded recovery. It never reads an auth store or
runs OpenCode. On profile omission, only an unchanged owned `SKILL.md` is
removed. Mango-introduced `skills.paths` entries are removed; ambiguous legacy
entries without the provenance marker remain with a warning. An existing unowned
skill with identical generated bytes is never adopted for later deletion.

## Named agent definition installation

An install without `--agent` writes a primary agent definition to
`agents/<profile>.md` beside the resolved config by default (no `--config`
needed), reports the shared named-profile install mode (`install.mode:
named-profile`, `use it: opencode --agent <profile>`), and leaves the main
config untouched unless `--default` is also passed. Explicitly,
`--agent opencode@1.18.31=primary:name` or `subagent:name` requires an explicit
`--config opencode=<path>` ending `agents/name.md` and writes the same kind of
definition there instead. Either way, a distinct Markdown definition and
adjacent ownership manifest are written, without touching the main JSONC config.
Only simple lowercase names are accepted; OpenCode built-ins are reserved.
The deterministic frontmatter sets `mode`, the exact native route model, and the
route effort as the agent's `variant` (see [Effort](#effort-as-agent-variant));
ordered portable instruction resources form the custom body. The native loader
trims the prompt and this body replaces the named agent's stock prompt, rather
than extending it. An empty instruction list cannot establish delivery. A
profile's skill requirement is skipped for this named-agent file (OpenCode
skills are a directory-wide setting, not per-agent) unless `--default` is
also passed, which installs it into the main config as described above.

Plan consent binds destination, mode and name. Reapply and profile switching
preserve an unchanged owned definition; an unowned or edited definition is never
overridden, even if its bytes match. A primary definition is selectable, not made
active by Mango. A subagent definition is eligible for delegation, but no
delegation or session was executed. Required permissions, tools and skills still
block installation. Higher-precedence target layers, provider authentication and
runtime enforcement are not qualified.

Actual compiled-Mango plan/apply produced primary and subagent files in disposable
state. The exact binary SHA-256 above resolved both via fixed `debug agent name`
under cleared environment, synthetic home/XDG/config/auth, empty `/etc`,
network/PID/IPC isolation, dropped capabilities and timeout. The result matched
the generated name, mode, model and instruction sentinel. Changed mode, model,
prompt and missing-name controls changed or removed the resolved values; source
definitions remained byte-identical and the disposable probe state was restored
to its original inventory. Primary generated SHA-256:
`1a91ae593c0eb78167948df77f83271cd391c0ade7fc9666d043251f1a6f33c5`;
subagent: `9a6fe91855defd1f71a74642bea6b29d97031a39e9f5e47c2b5dca542e0384eb`.
This is native definition consumption, not a model call, authenticated route,
session, delegation or permission enforcement.

## Effort as agent variant

OpenCode `1.18.31` has no global effort setting. Its agent schema
(`packages/core/src/v1/config/agent.ts`, SHA-256
`4f2d7bc8283ff0c8cd922c614a74f74abb1abe9d5fc2a6dba3dd03d2dca5c624`) has a
`variant` "Default model variant for this agent (applies only when using the
agent's configured model)". `session/prompt.ts` (SHA-256
`f0c5bc64c0f0e966693d4a57f7ede1e9d6e188b396152f04b55303dc75b9b768`) applies it
only when the agent's model defines a variant of that name, and
`provider/transform.ts` (SHA-256
`c07d49e48dd2478ad2813a10805781a72551db2fd847b7df994cc854bf654c16`) names
built-in reasoning variants only `none`, `minimal`, `low`, `medium`, `high`,
`xhigh`, and `max`, per provider and model. The installer writes the route effort
as `variant: "<effort>"` for those names; any other effort (such as `ultra`) is
listed as `effort <value>: NOT APPLIED` (JSON `skippedRequirements`) and blocks
with `--strict`. If the model has no variant of that name, OpenCode silently uses
the model default; the plan warns about this. The main-config `model` written
with `--default` carries no effort, so plain `opencode` without `--agent` does not
use it.

The opt-in `TestOpenCodeGeneratedAgentNativeProbe` (2026-09-25, same exact
binary and isolation) resolved the generated primary and subagent definitions
with `variant: "high"`; changing it to `low` or removing it changed or removed the
resolved `variant`. This is loader consumption, not proof that a provider request
used that reasoning level.

## Remaining blockers

The following remain blocking or partial for this exact release:

- complete global/project/custom/managed precedence and per-field provenance
- authentication identity and credential selection
- transport mapping, and effort for models without a matching variant or for the main-config model
- global/project instruction-file discovery and precedence outside named definitions
- active primary selection and runtime subagent delegation
- more than one portable skill resource or target skill-directory layout
- permission and tool equivalence or enforcement
- plugin and MCP discovery or enforcement
- runtime route and policy enforcement

The mutable configuration schema snapshot used during source review has SHA-256
`e8cb6e287a3852ee3403f4803be5ad6b19db94948037eaa9672125c333427922`.
It remains context, not immutable release evidence.

## Pinned references

- [OpenCode release tag][release]
- [OpenCode config source][config-source]
- [OpenCode core config source][core-config-source]
- [OpenCode provider source][provider-source]
- [OpenCode official documentation][docs]

[release]: https://github.com/anomalyco/opencode/tree/v1.18.31
[config-source]: https://github.com/anomalyco/opencode/blob/v1.18.31/packages/opencode/src/config/config.ts
[core-config-source]: https://github.com/anomalyco/opencode/blob/v1.18.31/packages/core/src/v1/config/config.ts
[provider-source]: https://github.com/anomalyco/opencode/blob/v1.18.31/packages/core/src/v1/config/provider.ts
[docs]: https://opencode.ai/docs/config/

## Global instruction files (ap-nym, 2026-09-25)

`globalInstructions` may own `AGENTS.md` in `${XDG_CONFIG_HOME:-~/.config}/opencode` (`v1.18.31` `a97622c`, `packages/opencode/src/session/instruction.ts:61`, `packages/core/src/global.ts:13`); OpenCode also reads `~/.claude/CLAUDE.md`. Written only when the profile is the default (`mango use`, `install --default`); whole-file ownership with create-only backup, drift checks, release on `use`, and undo. See [target evidence](../target-evidence.md#global-instruction-files-2026-09-25-ap-nym).

## Home `~/AGENTS.md` (ap-5mp, 2026-09-25)

`~/AGENTS.md` is read only outside a Git repository: `v1.18.31`, `packages/opencode/src/session/instruction.ts:122-133` runs `findUp` from the working directory to the worktree root, which is `/` only for non-Git directories (`packages/opencode/src/project/project.ts:217`). OpenCode therefore does not own `globalInstructions.home`. See [target evidence](../target-evidence.md#home-instruction-file-agentsmd-2026-09-25-ap-5mp).

## Role subagent files (ap-6lp, 2026-09-25)

Each declared profile role becomes `${XDG_CONFIG_HOME:-~/.config}/opencode/agents/<role>.md`, written only when the profile is the default (`install --default`, `mango use`). Source review of `v1.18.31` (`a97622c801f4ca571530ddc51076af659a9c32cd`):

| Claim | Pinned source (file SHA-256) |
| --- | --- |
| Agents load from `{agent,agents}/**/*.md` in each config dir, named by relative path; frontmatter is the config and the trimmed body is `prompt` | `packages/opencode/src/config/agent.ts` (`4844d4dfa48a516f5b134b0e310c3b0fc11292d0180b0b12ab8d96a43775efa7`), lines 11-31 |
| `model` (line 14), `variant` (line 15), `description` (line 25), and `mode` (`subagent`, `primary`, or `all`; line 26) are agent keys | `packages/core/src/v1/config/agent.ts` (`4f2d7bc8283ff0c8cd922c614a74f74abb1abe9d5fc2a6dba3dd03d2dca5c624`) |

The file carries `description`, `mode: subagent`, and, when the route binds the role, `model` as `provider/model` (the same form as named agents) and `variant` from its effort (the [variant rules](#effort-as-agent-variant) apply). The body is the role's `instructions` resource, else its description. Ownership (adopt with create-only backup, drift, release on `mango use`, undo, `role-definition` in `mango status`) and the named-only `role-definitions` skip match Codex. Evidence level: source review plus a built-binary sandbox smoke (install, status, use, undo); OpenCode was not run against the generated files, so runtime delegation to these subagents stays unobserved.
