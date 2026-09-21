# OpenCode configuration reference

**Reference date:** 2026-09-21. **Status:** profile-mango has an exact-version,
inert preview renderer for OpenCode `v1.18.31`; native applicability, installation,
and runtime enforcement remain blocked.

## Immutable release evidence

The adapter is qualified against the official `anomalyco/opencode` repository,
release tag `v1.18.31`, commit
`a97622c801f4ca571530ddc51076af659a9c32cd`, published 2026-09-14:

- source repository: <https://github.com/anomalyco/opencode>
- source tag: <https://github.com/anomalyco/opencode/tree/v1.18.31>
- source archive SHA-256: `76f69fe27ec2b44e23fa1749029e7c012eb7e975a0f0c7819e9458198dfd3896`
- Linux x64 release asset SHA-256: `e9312be75ed803b7415fc2aeabda1f4fe938912a39673762dc0c38c0e11ebde4`

The source review retained these exact file hashes:

| Source file | SHA-256 |
| --- | --- |
| `packages/opencode/src/config/config.ts` | `87a9071af1ddb04d65947dba49be3fb4c94ecf63e120f17ce46ee81ff25dd45a` |
| `packages/core/src/v1/config/config.ts` | `b99bcbd98df6da79e59cda482363f759cea9d4b9792c6c8e83b6a8d686138d30` |
| `packages/core/src/v1/config/provider.ts` | `c496dea619e8e9d2d09b7a2353f2e62bff6a2276471c161c3988059a1b6ac303` |

These immutable release and source hashes are separate from the mutable official
documentation and schema snapshot below. A tag, source hash, or static source
review does not prove native parser acceptance, effective configuration, or
runtime enforcement.

## Configuration paths, syntax, and precedence

The reviewed source and official documentation describe JSON and JSONC
configuration layers. The relevant surfaces are:

- the global user configuration under `${XDG_CONFIG_HOME:-~/.config}/opencode/`,
  conventionally `opencode.json` or `opencode.jsonc`
- the project configuration at the project root, conventionally
  `opencode.json` or `opencode.jsonc`
- an explicitly selected custom configuration path supported by the target

OpenCode merges the global, project, and custom layers with target-defined
precedence. The exact winning value and per-field provenance were not observed
for this release. The adapter therefore never chooses an active destination,
merges target files, or presents a candidate as effective state.

The source establishes that the `model` value uses the string form
`"provider/model"`. Provider configuration is a separate configuration surface.
The source and schema also expose instruction, permission, and tool-related keys.
Those facts establish candidate syntax only. They do not establish that a
portable permission policy, closed tool set, instruction or skill delivery,
provider option, or runtime policy is preserved or enforced.

Target-owned credentials are separate from configuration at
`~/.local/share/opencode/auth.json`. The adapter never reads, writes, copies, or
emits that file, credentials, authentication identity, provider options, or
transport settings.

## Inert adapter boundary

`pkg/adapters/opencode` emits only this deterministic preview candidate when the
route is complete and uses the native transport:

```jsonc
// profile-mango: INERT PREVIEW ONLY
// NON-APPLICABLE: candidate syntax for OpenCode 1.18.31.
// This is not an active OpenCode config.json or config.jsonc. Authentication, provider options, effort, delivery, precedence, plugins, MCP, and enforcement are unverified.

{
  model: "provider/model",
}
```

The candidate is written under
`preview/<profile>.opencode.jsonc.preview` by the caller's explicit preview
staging flow. It contains only the canonical provider/model string. It does not
emit effort, authentication identity, credentials, provider options, transport,
permissions, tools, instructions, skills, plugins, MCP configuration, or active
configuration paths. The render report always sets `applicable: false` and keeps
these capabilities blocking or partial as appropriate.

The renderer is pure and inert. It does not inspect target homes, credentials,
installed binaries, sessions, providers, plugins, MCP servers, or the network.
Resource bytes are validated and copied only as inert preview artifacts under
caller-controlled staging; resource delivery and precedence remain unverified.

## Evidence gaps and deferred validation

The following remain blocking for this exact release:

- native JSON or JSONC parser acceptance
- merged effective configuration and per-field provenance
- global, project, and custom precedence
- authentication identity and credential selection
- effort and transport mapping
- permission and tool equivalence or enforcement
- instruction and skill discovery or delivery
- plugin and MCP discovery or enforcement
- runtime route and policy enforcement

The official mutable configuration schema snapshot used during source review has
SHA-256 `e8cb6e287a3852ee3403f4803be5ad6b19db94948037eaa9672125c333427922`.
That snapshot and current documentation are explicitly mutable and are not
immutable release evidence. Re-fetch and re-hash them before relying on later
claims.

Native execution, isolated parser or effective-state checks, and real
installation are deferred to Bead `ap-6fu.11` and require fresh explicit user
approval. That work must use the exact release artifact, synthetic isolated
paths, no personal configuration or credentials, blocked network, bounded
execution, and recovery checks. No OpenCode binary or target configuration was
executed or inspected for this adapter.

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
