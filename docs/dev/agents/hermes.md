# Hermes configuration reference

**Reference date:** 2026-09-21. **Source pin:** repository release
`v2026.9.14` (Hermes Agent `v0.21.3`), commit
`345cd2b057a452236de401d3534b8502a7465e8d`. **Status:** exact-version inert
preview renderer plus a bounded three-field installer qualified on 2026-09-22.
Full startup, authentication, and runtime enforcement remain blocked.

## Exact source observation

The task-owned checkout verified annotated tag object
`7a963716b81be13ba513d4f127633b7da493aff2`, commit
`345cd2b057a452236de401d3534b8502a7465e8d`, and tree
`6e14b9791cdc5a47068685e9429dd5d6bdc5ef5f`. The deterministic source archive
has SHA-256
`71f2db39a64fbba282e3bd3be4b0f7b935585948a59a368d61deeec0f0827c47`.
`pyproject.toml` declares project version `0.21.3` and Python
`>=3.11,<3.14`; `hermes_cli/__init__.py` independently declares version
`0.21.3` and release date `2026.9.14`. The source and build metadata hashes,
runtime, and platform are recorded in the [target evidence ledger](../target-evidence.md).

The review platform was Linux x86_64 on Omarchy kernel
`7.2.5-3-omarchy`. The available system Python was `3.14.7`, outside the
source requirement, so no Hermes dependency install or runtime build was run.

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
provenance report is documented. Source review found that the launcher loads
`HERMES_HOME/.env` and the project `.env`, parses effective config, configures
logging, and can create or secure the Hermes home before dispatch. `config get`
then calls `load_config`; `status` reads dotenv, config, auth/provider state,
plugins, gateway state, and session databases, while `--deep` can use the
network and a local socket. `profile show` reads profile config, gateway/PID
state, distribution metadata, aliases, skills, `.env`, and `SOUL.md` paths.

Those effects were not bounded as credential-free and no-write even with
synthetic homes and blocked network. The commands were not executed. No
smart approvals, profile mutations, sessions, providers, credentials, target
homes, or auxiliary-model calls were used. Future probes must establish all
of those effects before execution.

## Inert preview boundary

`mango render <name> --target hermes --target-version 0.21.3` emits a
deterministic `preview/<name>.config.yaml.preview` candidate through the
versioned render report. Its source-grounded YAML fields are `model.provider`,
`model.default`, and `agent.reasoning_effort`. It never copies authentication
values, writes `~/.hermes`, reads target state, or claims applicability. The
report keeps native config acceptance, effective state, precedence,
authentication, delivery, permissions, tools, skills, context, and runtime
enforcement blocking.

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


## Bounded installation qualification, 2026-09-22

The installer losslessly patches only `model.provider`, `model.default`, and
`agent.reasoning_effort`. Exact source `hermes_cli.config.load_config_readonly()`
consumed the generated patch and merged defaults in a synthetic Python 3.12.13
environment. Both imported config/version modules are hash-gated. Root reran the
probe with network/PID/IPC isolation, allowlisted mounts, cleared environment,
dropped capabilities, hidden real homes/sockets, and a timeout. No full agent
startup, provider call, credential access, session, plugin, hook, or gateway ran.

Compiled-CLI tests exercise plan/apply, byte/comment preservation, default backup,
idempotent replan, stale rejection, and disposable byte-for-byte restoration.
Native module consumption is not proof of authenticated routing or policy
execution. Permissions, tools, instructions, skills, and other required properties
remain blocked. The initial M0 observations above remain historical evidence.
See the [retained native evidence](../../../pkg/adapters/hermes/hermes_native_evidence.md)
for exact hashes, invocation, and limitations.
