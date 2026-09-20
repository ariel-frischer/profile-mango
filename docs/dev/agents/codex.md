# Codex configuration reference

**Reference date:** 2026-09-20. **Status:** first intended public adapter
candidate; no adapter or supported capability. The official configuration pages
and schema are mutable and unversioned. Their recorded hashes in
[`sources.json`](sources.json) are retrieval snapshots, not a mapping to Codex
`0.154.0`; that version is a separate installed observation in the
[evidence ledger](../target-evidence.md).

## Configuration and precedence

Official docs describe user configuration at `${CODEX_HOME}/config.toml`
(`~/.codex/config.toml` by default), project `.codex/config.toml` layers, named
profiles, and runtime overrides. Target-owned authentication is separate from a
portable profile. A later adapter must prove precedence and route identity for
the exact tested build rather than treating a rendered file as effective state.

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
