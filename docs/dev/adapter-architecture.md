# Adapter architecture

`profile-mango` has one canonical profile domain and small, target-owned renderers.
The render path is intentionally an ordinary fixed boundary, not a plugin registry or
application engine.

## Ownership

```mermaid
flowchart LR
    CLI[render CLI] --> LOAD[profile/binding/resource loading]
    CLI --> DISPATCH[explicit target switch]
    DISPATCH --> CLAUDE[Claude Code adapter]
    DISPATCH --> CODEX[Codex adapter]
    DISPATCH --> OMP[Oh My Pi adapter]
    DISPATCH --> OPENCLAW[OpenClaw adapter]
    DISPATCH --> HERMES[Hermes adapter]
    CODEX --> CONTRACT[target-neutral render contract]
    OMP --> CONTRACT
    OPENCLAW --> CONTRACT
    HERMES --> CONTRACT
    CONTRACT --> STAGE[atomic inert staging]
```

- `pkg/profilemango` owns canonical parsing, inheritance, route bindings, and
  resource digests.
- `pkg/render` owns the smallest two-consumer boundary: `TargetBuild`, pure
  input/resource types, evidence, capabilities, artifacts, report serialization,
  stable sorting, safe artifact paths, and resource-byte validation.
- `pkg/adapters/claudecode`, `pkg/adapters/codex`, `pkg/adapters/ohmypi`, `pkg/adapters/openclaw`, and
  `pkg/adapters/hermes` own exact evidence pins, target syntax, capability mapping,
  target-specific diagnostics, and applicability.
- `cmd/profile-mango/render.go` owns common input loading, an explicit switch over
  the known target names, report output, and staging through `internal/staging`.
- `internal/staging` creates only a new explicit output directory and never writes
  a target home.

## Dispatch and failure behavior

The CLI selects a known adapter before reading profile, binding, or resource
inputs. Unknown target names produce a target-neutral `RenderReport` with a
`render.target.unsupported` error and no target-specific artifacts. Known adapters
still fail closed for missing or mismatched exact versions, evidence hashes,
authentication identity, delivery, precedence, permissions, tools, or enforcement.

A preview artifact is an inert candidate only. `applicable` remains false whenever a
required property is unverified, and the command returns nonzero. Candidate config
files use `preview/` paths and report metadata excludes artifact bytes. Resource
copies are content-addressed and validated against the canonical digest before they
are staged.

## Current target boundary

Claude Code `2.1.278`, Codex `0.154.0`, Oh My Pi `18.2.6`, OpenClaw `2026.9.5`,
and Hermes Agent `0.21.3` each have deterministic inert preview renderers. Claude
Code emits a documentation-context JSON `model` candidate. Codex uses TOML
candidate syntax.
Oh My Pi uses the pinned source's YAML settings fields for a `modelRoles.default`
and `defaultThinkingLevel` candidate. OpenClaw uses source-grounded JSON5 fields
for `agents.defaults.model.primary`, an explicit empty fallback list, and
`agents.defaults.thinkingDefault`. Hermes uses source-grounded YAML fields for
`model.provider`, `model.default`, and `agent.reasoning_effort`. None of these
adapters claims native applicability, installs output, reads a target home, accesses
credentials, starts a session, calls a provider, or uses a network connection.

Claude Code's immutable npm package and release commit are pinned, but its opaque
native startup and diagnostic paths were not proven side-effect-free. No native
Claude Code command was run. The adapter keeps native acceptance, effective state,
precedence, route/auth, permissions/tools, CLAUDE.md, skills, and enforcement
blocked rather than treating mutable documentation as release evidence.

Oh My Pi's source-level version observation is exact, but the standalone build is
currently blocked by its missing pinned native addon and the reviewed `config`
commands initialize settings, discovery, and migration paths. That is recorded as
an explicit support blocker rather than bypassed by running the unsafe inspector.
See [target evidence](target-evidence.md) and the [Oh My Pi reference](agents/oh-my-pi.md)
for the provenance and effect details.

OpenClaw's exact source/archive identity is pinned, but no dependency install,
runtime build, or native command was run. The reviewed `config validate --json`
path performs plugin/state/include discovery and was not accepted as a safe M0
probe. Those are explicit blockers, not reasons to add a target-home inspector or
plugin framework. See [target evidence](target-evidence.md) and the
[OpenClaw reference](agents/openclaw.md).

Hermes's exact source/archive identity is pinned, but the current system Python
is outside its `>=3.11,<3.14` requirement and no runtime was built. The reviewed
`config get`, `status`, and `profile show` paths load dotenv, credential, profile,
plugin, gateway, and session state before or during inspection, so no native
Hermes command was accepted as a safe M0 probe. See [target evidence](target-evidence.md)
and the [Hermes reference](agents/hermes.md).
