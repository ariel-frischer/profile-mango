# Roadmap

This roadmap records product direction, not shipped compatibility, delivery dates,
or implementation authorization. See the [project scope](docs/dev/project-scope.md)
for the current support boundary and the
[target evidence ledger](docs/dev/target-evidence.md) for version-qualified claims.

## Direction

Profile Mango is a portable **profile manager**, not a universal manager for every
agent configuration key. Its canonical core describes reusable behavior and
resources that users reasonably expect to carry between coding agents. Adapters use
native configuration only to deliver that profile, while preserving and reporting
meaningful target differences.

New shared concepts should enter the profile contract only when concrete cross-agent
profile use cases and evidence justify their semantics. Native settings unrelated to
portable profiles remain target-owned and outside the product scope.

## Current foundation

- Strict, offline profiles for route intent, permissions, tools, instructions, and
  skills.
- Deterministic inheritance, resource hashing, diagnostics, plans, and ownership
  manifests.
- Installers for the supported subsets in the README, with `use`/`status`,
  role models on Oh My Pi, and whole-file global instructions.
- No role subagent files, MCP projection, credential handling, or runtime
  enforcement claim yet.

## Near-term priorities

### 1. Establish narrow, applicable target support

Qualify a small, useful subset for each intended target rather than waiting for
feature parity. Codex remains the first intended public adapter. Each target must
independently establish native acceptance, effective configuration, precedence,
route identity, delivery, and enforcement at the level claimed.

### 2. Make delivery and ownership inspectable

Move from inert rendering toward explicit placement and, only when justified,
managed application. Preserve unrelated target configuration, show every intended
effect, retain ownership metadata, and provide a safe rollback boundary. Import and
drift repair remain separate decisions.

### 3. Add target profile assignments

Introduce a separate selection or deployment contract that can map an agent target
to a portable profile and, where the target supports one, a target-native profile
name. This is where a per-agent default belongs. A `PolicyProfile` should not declare
itself globally active, and assignments must not contain credentials.

Target profile models are not equivalent:

| Target | Relevant configuration model |
| --- | --- |
| Codex | Named profile files selected explicitly, plus user and project configuration layers |
| Claude Code | Layered settings and named subagents, not whole-configuration named profiles |
| Pi and Oh My Pi | Global and project configuration layers |
| OpenClaw | Cross-agent defaults with per-agent entries and overrides |
| Hermes | Named profiles implemented as isolated agent homes |
| Jcode fork | Fork-specific named profiles; experimental comparison evidence only |

### 4. Keep adapters profile-scoped

Render or apply only the target settings and resources required to express the
selected profile. Adapters may understand additional native settings when necessary
for precedence, conflict detection, or unrelated-state preservation, but Profile
Mango should not expose them as a general configuration surface. Do not add an
unrestricted passthrough object or pursue broad target configuration coverage.

## Evidence-gated expansion

The concepts below are candidates only when they form part of reusable coding-agent
profiles across multiple intended targets. Their presence in one feature-rich agent
does not by itself expand Profile Mango's scope.

### Agent and subagent roles

Codex, Claude Code, Hermes, and multi-agent OpenClaw configurations provide related
but non-equivalent concepts. Explore a portable role definition only for shared
fields such as name, description, instructions, route intent, tool policy, and
permissions. Delegation, orchestration, lifecycle, and child inheritance should
remain target-specific until their behavior is demonstrably portable.

### MCP declarations

Explore secret-free MCP intent separately from machine-local execution and
authentication bindings. A portable declaration may describe the required server
or capability, while commands, URLs, credentials, trust, and process ownership stay
local or target-owned. No MCP projection should be claimed without precedence,
permission, and lifecycle evidence.

### Hooks and lifecycle automation

Codex, Claude Code, and OpenClaw expose hook or automation systems, but their event
models, trust flows, blocking behavior, and execution authority differ. Treat hook
files as target-specific executable resources first. Consider portable lifecycle
intent only after stable shared events and safe ownership semantics are demonstrated.
Hooks are not a near-term canonical `PolicyProfile` field.

## Not planned

- An exhaustive universal schema for every agent preference.
- General-purpose management of target-native configuration unrelated to profiles.
- Coverage of channels, gateways, messaging bots, UI preferences, telemetry,
  updates, or other product operations merely because a target exposes them.
- Opaque target configuration passthrough in the canonical profile.
- Credentials, tokens, or copied authentication stores.
- Silent provider, model, transport, or authentication fallback.
- Portable identity, memory, session history, UI preferences, telemetry, or update
  settings without a demonstrated cross-agent policy use case.
- Claims that instruction delivery, hook execution, or configuration syntax alone
  prove permission enforcement.

Broader agent-configuration management may be reconsidered only if sustained demand
from a substantial user base demonstrates that it is needed. It is not required to
complete the profile-management product and should not influence current contracts
or adapter scope.

Roadmap items become shipped capabilities only after their contracts, adapters,
evidence, tests, documentation, and safety boundaries land together.
