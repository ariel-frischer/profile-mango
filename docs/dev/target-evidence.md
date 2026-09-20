# Target evidence ledger

**Evidence date:** 2026-09-20

M0 freezes the portable contract. It does not claim that any target enforces that contract yet. Target claims require three separately reported evidence levels:

1. schema, golden, and negative fixtures
2. offline native parsing or effective-config inspection in an isolated test home
3. explicitly authorized runtime observation

A lower evidence level never implies a higher one. Parsing is not runtime enforcement, and instruction text is not a permission boundary.

## Codex CLI

| Field | Evidence |
| --- | --- |
| Intended status | First public adapter candidate for M1; no adapter ships in M0 |
| Locally observed version | `codex-cli 0.154.0` |
| M0 evidence | Canonical profile, exact route binding, deterministic fixtures, and schemas only |
| Native verification | Not run in M0 |
| Runtime enforcement | Unverified |
| Application | Not implemented |

Before Codex becomes supported, M1 must pin a version/platform, document configuration precedence, enumerate native tool expansion, classify every portable field by fidelity/delivery/enforcement, and prove deterministic golden output. Any required property with unknown effective enforcement must make the plan non-applicable.

## Jcode

| Field | Evidence |
| --- | --- |
| Product status | **Not a supported public target** |
| Why retained here | Experimental developer comparison against Ariel's custom fork |
| Locally observed version | `jcode v0.83.901-dev (43b62abcf)` |
| Distribution assumption | Ariel-specific behavior and configuration; external adoption is not established |
| Native verification | Not run in M0 |
| Runtime enforcement | Unverified |
| Application | Not implemented |

Jcode must not appear in the README support list, release promise, or compatibility matrix. A future adapter may be developed privately or experimentally, but public support requires a stable external distribution, versioned documentation, reproducible fixtures, and evidence independent of Ariel's machine.

## Deferred checks

- No agent home, credentials, sessions, global configuration, or provider endpoint was read.
- No target process, hook, extension, child agent, or provider request was launched.
- Claude Code, Pi, Oh My Pi, OpenClaw, and Hermes remain roadmap research, not M0 targets.
- The placeholder `agentprofiles.dev` schema identifier is not a claim that the domain is registered or controlled. Rename it before public release if ownership is unavailable.
