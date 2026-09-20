# Hermes configuration reference

**Reference date:** 2026-09-20. **Documentation pin:** repository release
`v2026.9.14` (Hermes Agent `v0.21.3`), commit
`345cd2b057a452236de401d3534b8502a7465e8d`. **Status:** intended MVP target;
not installed, tested, or supported by profile-mango, and no adapter ships.

## Configuration and precedence

Pinned documentation describes `~/.hermes/config.yaml`, `.env`, `auth.json`,
SOUL, memories, and skills. `HERMES_HOME` and named profiles can relocate state.
CLI values precede YAML, `.env` is a fallback, and defaults follow; managed policy
can pin values. Provider, default model, base URL, API mode, custom providers,
ordered fallbacks, reasoning effort, and provider-specific keys or OAuth are
documented. A portable profile must not copy credentials or infer route identity.

## Tools, instructions, and skills

Toolsets gate availability, but local execution otherwise has the user's
filesystem authority unless another backend is selected. Smart approvals use an
auxiliary model, cover shell commands only, and do not cover file writes; they are
not a no-inference validation surface. Project context prefers Hermes-native
conventions, while `AGENTS.md` files load root-to-current-directory. Skills
primarily live under `HERMES_HOME/skills`.

## Candidate inspection and gaps

`hermes config get model --json`, `hermes status`, and `hermes profile show` are
candidate selection/state inspectors. No complete redacted effective-config and
provenance report is documented. These commands were not executed. Future probes
must review exact-version effects, use an isolated `HERMES_HOME` and project,
synthetic data, no real credentials, and blocked provider access. Do not exercise
smart approvals, mutate profiles, or start an agent session.

Pinned sources: [configuration][configuration], [profiles][profiles],
[models][models], [providers][providers], [fallbacks][fallbacks], [tools][tools],
[security][security], [context files][context], and [skills][skills].

[configuration]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/user-guide/configuration.md
[profiles]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/reference/profile-commands.md
[models]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/user-guide/configuring-models.md
[providers]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/integrations/providers.md
[fallbacks]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/features/fallback-providers.md
[tools]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/user-guide/features/tools.md
[security]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/skills/security/references/security-privacy.md
[context]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/features/context-files.md
[skills]: https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/website/docs/features/skills.md
