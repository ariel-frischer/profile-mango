# Codex configuration reference

**Reference date:** 2026-09-21. **Status:** exact-version inert preview support for Codex CLI `0.154.0`; native applicability is blocked. The renderer emits candidate provider/model/effort syntax and copied resources only under an explicit staging directory. It never emits active `config.toml`, `AGENTS.md`, or skill locations, and it always reports `applicable: false` because authentication, delivery, precedence, and enforcement remain unverified.

## Configuration and precedence

Official docs describe user configuration at `${CODEX_HOME}/config.toml`
(`~/.codex/config.toml` by default), project `.codex/config.toml` layers, named
profiles, and runtime overrides. Target-owned authentication is separate from a
portable profile. The preview adapter does not select a live destination or claim
that its candidate syntax becomes effective state.

See the unversioned official [basic configuration][basic],
[advanced configuration][advanced], [configuration reference][reference], and
[schema][schema].

## Route, permissions, instructions, and skills

The documented configuration exposes provider, model, reasoning effort,
sandbox, approval, and tool-related settings. These are candidate concepts, not
evidence that canonical authentication, closed tool allowlists, read-only access,
network policy, or descendant behavior are enforced. Runtime and project layers
remain override surfaces.

Codex reads scoped `AGENTS.md` instructions and discovers skills from documented
system, administrator, user, and repository locations. Instruction delivery is
not a permission boundary, and skill visibility is not proven skill exclusion.
See [AGENTS.md guidance][agents] and [skill guidance][skills].

## Preview renderer boundary

`profile-mango render <name> --target codex --target-version 0.154.0` is an
offline compiler boundary, not an installer. Without `--preview`, applicability
blockers produce diagnostics and no output. With `--preview`, the command may
atomically create a new explicit `--out` directory containing `render.json`,
a `preview/<name>.config.toml.preview` candidate, and inert resource copies. It
still exits nonzero while blockers remain.

The report pins the tested Codex build hash, lists field-level capabilities and
blocking diagnostics, and records deterministic artifact digests. No target home,
credentials, subprocess, provider, or network is accessed.

## Candidate inspection and gaps

The observed version/help commands are recorded separately in the evidence
ledger. No credential-free command has been established that prints the complete
effective configuration with per-field provenance. `--strict-config` participates
in startup, and `doctor` may inspect installation, configuration, authentication,
or runtime health; neither is accepted as a safe inspector without exact-version
effect review.

Future checks must use an isolated `CODEX_HOME` and project with no credentials
or provider access. Native parsing does not prove runtime enforcement.

[basic]: https://learn.chatgpt.com/docs/config-file/config-basic
[advanced]: https://learn.chatgpt.com/docs/config-file/config-advanced
[reference]: https://learn.chatgpt.com/docs/config-file/config-reference
[schema]: https://developers.openai.com/codex/config-schema.json
[agents]: https://learn.chatgpt.com/docs/agent-configuration/agents-md
[skills]: https://learn.chatgpt.com/docs/build-skills
