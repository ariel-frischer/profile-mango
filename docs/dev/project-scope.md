# Project scope and goals

The [constitution](../../.autospec/constitution.yaml) defines durable principles.
This document distinguishes current delivery from evolving product scope. It is
not an implementation spec or a promise that planned targets already work.

## North star

Make profile-mango the best open system for stable profile management across AI
agents: define behavioral intent once, understand what each agent can preserve,
and change agents without silently changing policy. Earn widespread adoption
through trustworthy portability, a simple workflow, and accessible contributions,
not through a required hosted service or an exhaustive universal abstraction.

Intended users are developers who switch or combine agents, teams sharing reviewed
behavioral profiles, and contributors maintaining target integrations. Their
problems are duplicated configuration, uncertain precedence, hidden capability
gaps, and accidental changes to permissions or model/authentication routes.

## Verified current delivery: M0

As of 2026-09-20, the [README](../../README.md) describes a private M0 contract
prototype. The [canonical domain](../../pkg/profilemango/types.go) and
[schemas](../../schemas/) provide:

- Strict `PolicyProfile` and machine-local route-binding parsing, with stable,
  field-aware diagnostics and rejection of unknown/duplicate keys, nulls,
  unsupported versions, missing parents, and inheritance cycles.
- Deterministic one-parent resolution, explicit merge rules, closed tool
  allowlists, and deny-wins semantics.
- Safe instruction/skill resource hashing with escaping paths rejected.
- Versioned profile, binding, inert plan, and ownership-manifest contracts,
  plus positive, constrained, unsupported, and golden fixtures.
- An offline `validate` command and a Go library. Route bindings describe
  provider, transport, authentication mode, model, and effort, not credentials.

**No adapters are shipped.** M0 does not render, install, apply, launch, or modify
agents. It does not inspect target homes, access credentials, run target
subprocesses, or call providers or the network. Plan and manifest types are
contracts, not evidence of an application engine. Canonical policy resolution
does not prove that a target enforces that policy.

## Confirmed intended MVP targets

The accepted roadmap direction recorded in **ap-6fu** names these intended
supported targets. This is product intent, not a current compatibility matrix:

| Target | Direction | Current shipped status |
| --- | --- | --- |
| Codex | First intended public adapter | No adapter; version-qualified candidate evidence only |
| Claude Code | Intended MVP target | No adapter; compatibility not established |
| Pi / Oh My Pi | Intended MVP target family; evaluate each variant separately | No adapter; compatibility not established for either |
| OpenClaw | Intended MVP target | No adapter; compatibility not established |
| Hermes | Intended MVP target | No adapter; compatibility not established |

Ariel's **Jcode fork is experimental-only**, outside the intended supported MVP
set and public compatibility promise. Its local observations are developer
comparison evidence, not a public support commitment.

The [target evidence ledger](target-evidence.md) remains the source for observed
versions and limitations. Its M0-only scope is compatible with this broader
intended roadmap. Support begins only when version-qualified capability evidence
and golden rendering tests exist, with additional checks for any stronger claims.
No delivery dates, version ranges, or target parity are implied here.

## What unification means

The canonical core describes shared intent. Adapters own target syntax,
capability mapping, and target-specific evidence, without leaking those details
into every core operation. Keep these boundaries explicit and ordinary; modularity
does not require a plugin loader, registry, or extension protocol.

For each relevant property, distinguish:

1. **Fidelity:** Does the target have an equivalent concept, a partial one, or none?
2. **Delivery:** Can the intended setting or resource reach a known destination
   with understood precedence?
3. **Enforcement:** What behavior has actually been demonstrated, including
   relevant overrides and bypass surfaces?

For example, delivering a read-only instruction is not equivalent to enforcing
read-only access. Selecting a model does not prove the authentication route.
Required unsupported or unknown properties block applicability, with diagnostics
that explain why. An explicitly unmanaged property is not an enforced guarantee.
Useful native differences should remain visible, not be erased to fit the least
capable target.

Instructions, skills, permissions, tool access, and route identity already have
canonical representations. Additional shared concepts should enter the core only
when concrete cross-agent use cases and evidence justify their semantics. This
does not promise roles, MCP projection, or any other expansion.

## Non-goals and boundaries

**Current M0 exclusions:** adapters, rendering, application, import, drift repair,
credential handling, provider calls, target-home inspection, roles, MCP projection,
and identity management. These are milestone boundaries, not all permanent bans.

**Long-term product boundaries:** profile-mango is not a replacement agent runtime,
a model/provider service, or a credential store. It does not promise identical
model responses, exact feature parity, or universal enforcement beyond what a
target can prove. Centralized hosting, a marketplace, and governance machinery
are not prerequisites for using or contributing to the core. An exhaustive
universal agent schema and premature plugin infrastructure are not goals.

This work, **ap-e3j**, owns principles and scope only. **ap-3pa** owns Codex
implementation, **ap-3kw** owns no-model compatibility probes, and **ap-sha** owns
clean offline install validation. This document neither implements nor closes
those outcomes.

## Observable success criteria

Evaluate these as the relevant capabilities arrive, not as claims already met:

- **Portability:** A representative profile resolves consistently and preserves
  required intent across supported targets, or is explicitly rejected where that
  is impossible. Target differences are visible without rewriting shared intent.
- **Diagnostics:** Users can identify the affected field, limitation, evidence
  boundary, and available remedy without reverse-engineering generated config.
- **Safety:** Negative fixtures reject unsupported security requirements and route
  mismatches. Output excludes secrets; future application preserves unrelated
  state and exposes ownership and intended effects.
- **Stability:** Identical explicit inputs reproduce artifacts, and contract
  changes have documented compatibility and migration implications.
- **Usability:** A clean-environment example demonstrates the advertised workflow
  without hidden personal configuration. Core validation remains offline.
- **Evidence and contribution:** Another contributor can reproduce a claimed
  mapping using versioned fixtures and documented checks, without personal
  credentials or model calls for ordinary core work.

Broad adoption is the aspiration, not an invented download, market-share, or
performance threshold. Gather real user feedback before setting such targets.

## Open product decisions

The following defaults are **proposals, not adopted mandates or new work orders**.
None blocks establishing the principles above.

| Decision | Recommended default and reason |
| --- | --- |
| What depth earns initial support for each MVP target? | Ship a narrow, evidenced capability subset first. Publish limitations rather than wait for parity or imply full coverage. |
| How should Pi and Oh My Pi support relate? | Qualify each variant independently; share implementation only where observed behavior warrants it. |
| How should useful target-only settings be represented? | Keep them at adapter boundaries initially. Extend the canonical contract only for demonstrated shared intent, not opaque passthrough everywhere. |
| What comes after deterministic rendering: manual placement or managed application? | Prefer inspectable output and explicit effects first. Decide application/import/drift workflows separately with ownership and rollback evidence. |
| How should target-version support evolve? | Start with individually tested versions and a small reproducible fixture set; claim ranges only when evidence supports them. |
| What is the public schema namespace? | Verify ownership of the current `profilemango.dev` identifier before public release, and rename if unavailable as the evidence ledger notes. |
