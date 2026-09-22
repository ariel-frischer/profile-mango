# OpenCode configuration reference

**Reference date:** 2026-09-22. **Status:** profile-mango has an exact-version,
inert preview renderer and a narrowly install-capable top-level `model` plus one
target-owned skill resource for OpenCode `v1.18.31`. The deterministic JSONC
`model` and `skills.paths` configuration, plus one synthetic `SKILL.md`, have
isolated native acceptance and consumption evidence. Authentication identity,
full precedence, instructions, multi-skill delivery, provider options, and runtime
enforcement remain blocked.

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
provider option, authentication identity, effort, or transport setting.

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

`profile-mango install` can patch the top-level `model` field in one explicit
OpenCode config file and, independently, install exactly one validated portable
skill resource. The skill resource is written as a target-owned `SKILL.md` beside
the explicit config, and the config receives an absolute `skills.paths` entry for
that directory. It requires exact target `opencode@1.18.31`, native transport, and
a profile with no permission, tool, or instruction requirements. Multi-skill
profiles, instruction resources, and all other unqualified fields remain blocked.
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
runs OpenCode.

## Remaining blockers

The following remain blocking or partial for this exact release:

- complete global/project/custom/managed precedence and per-field provenance
- authentication identity and credential selection
- effort and transport mapping
- instruction-file discovery, precedence, and delivery
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
