# Pi configuration reference

**Reference and qualification date:** 2026-09-21 UTC. **Release pin:** `v0.86.1`, commit `13cbf77df2396303013a41646bcfa77b4271ae56`. **Status:** exact-version inert preview plus a three-default settings installer. Native settings-module getters and project-over-global precedence were qualified on 2026-09-22; full startup, authentication, delivery, and enforcement remain blocked. Pi is qualified independently from [Oh My Pi](oh-my-pi.md).

## Immutable release and package evidence

The release tag resolved with `git ls-remote` to commit
`13cbf77df2396303013a41646bcfa77b4271ae56`. Retrieved source archive hashes:

- Tag archive `pi-v0.86.1.tar.gz`: SHA-256 `16d65ce53bfab1ae24d625538d434c341c4789d34b352d8c6ef699cf1d1d567d`.
- Commit-pinned archive: SHA-256 `016d83312289ca9b8d3a9d2a5ad804b265277c659472833cfd602cdceecf3818`.

The immutable npm package is `@earendil-works/pi-coding-agent@0.86.1`.
Its registry tarball has SHA-256
`8dff93e6fa03e0d498e72a78d2c7bb5f094f5e06ee268e6abd000ba2984a0b6a` and
registry integrity
`sha512-vZBuNfJnruxZyemZ3O05V0S/Ylze08ahFTIQ1Mik++gVdOevPl89gt/Uv0U97BPAJaj9cj6Vf9rcIgKtUrd0BA==`.
The registry integrity was recomputed from the retrieved tarball. The package
metadata declares Node.js `>=22.19.0`, and its `bin.pi` entrypoint is
`dist/bundle/cli.js`; the extracted entrypoint SHA-256 is
`e79626f2dd6f94aa45d30f3fa63cd84319a6eefcd150b353cfaf274366926774`.
Qualification used Linux x86_64, Node.js `v24.21.0`, and npm `11.19.0`.
This is immutable source/package review, not an installed target observation.

## Configuration and precedence

The release-pinned settings documentation and package source describe JSON settings
at `~/.pi/agent/settings.json` and project `.pi/settings.json`. Project settings
override global settings; nested objects merge and arrays such as `defaultTools`
replace. The exact documented candidate keys used by the preview adapter are:

- `defaultProvider` for the startup provider.
- `defaultModel` for the startup model ID.
- `defaultThinkingLevel` for the startup thinking level: `off`, `minimal`, `low`,
  `medium`, `high`, `xhigh`, or `max`.

Provider and model selection also has CLI, custom-provider, model-catalog, and
credential surfaces. Authentication may use a CLI API key, `auth.json`,
environment variables, or `models.json`; portable profiles never copy credentials
or infer the effective route.

## Tools, instructions, and skills

Pi documents read, write, edit, and shell tools. Its security guide does not
present a permission popup or built-in sandbox boundary; trusting a project is
not containment. Instructions can come from global and ancestor/project
`AGENTS.md` or `CLAUDE.md`; an `AGENTS.override.md` in the same directory replaces
the normal candidate. Skill discovery and instruction delivery are not evidence
of access enforcement. Extensions and package resources execute with the Pi
process permissions and remain target-owned.

## Static effect review and native execution boundary

The exact source review covered the CLI, startup, settings, migrations, auth
storage, model runtime, session manager, package manager, resource loader,
extensions, and the version-pinned settings/providers/models/security/skills docs.
The source resolves `~/.pi/agent` or `PI_CODING_AGENT_DIR`, including settings,
`auth.json`, `models.json`, `models-store.json`, sessions, prompts, tools, and
debug logs. Startup constructs settings and HTTP services, runs cleanup and
migrations, and can read, rename, create, or rewrite target files. Runtime setup
loads project settings and trust state, context files, extensions, skills,
prompts, themes, packages, providers, auth state, models, and session state.
Package/config/update paths can use npm/git subprocesses, network refreshes, and
writes. First-time setup can persist theme and analytics settings.

No Pi native command was run. Considered commands included `pi --version`,
`pi --help`, `pi --list-models`, `pi config --help`, `pi config -l`,
`pi update --models`, and noninteractive print/model paths. The exact startup
review did not establish a bounded, no-write, credential-free, no-provider,
no-extension, noninteractive invocation. Synthetic `HOME` alone would not isolate
all agent, project, XDG, package, session, and extension paths. Native acceptance,
effective state, precedence, delivery, enforcement, authentication, and runtime
behavior remain unverified. This is an explicit evidence gap, not a claim that
the commands are unsafe in every environment.

## Inert preview adapter boundary

The `mango render --target pi --target-version 0.86.1` adapter emits a
candidate at `preview/<profile>.settings.json.preview` using only the exact
source-grounded `defaultProvider`, `defaultModel`, and `defaultThinkingLevel`
keys. It never emits authentication, API keys, auth-file data, model-store data,
transport, provider URLs, session paths, or resource paths in the candidate. It
copies validated canonical resources only under `resources/` in explicit inert
staging output.

Reports always set `applicable: false` and separately identify candidate syntax,
native config acceptance, effective state, precedence, route authentication,
delivery, extension discovery, permissions/tools, and runtime enforcement. The
adapter does not install output, read a Pi home, start a session, load an
extension, call a provider, or use a target network connection. See the
[target evidence ledger](../target-evidence.md) for the retained hashes and
qualification decision.

## Pinned sources

[settings]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/settings.md
[providers]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/providers.md
[models]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/models.md
[security]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/security.md
[skills]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/skills.md
[readme]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/README.md
[package]: https://registry.npmjs.org/@earendil-works%2fpi-coding-agent/0.86.1
[source]: https://github.com/earendil-works/pi/tree/13cbf77df2396303013a41646bcfa77b4271ae56/packages/coding-agent

## Installation qualification update, 2026-09-22

The opt-in settings-module probe verifies all three global defaults and project
model override without full CLI startup. It pins the imported module hash, uses
allowlisted runtime/package mounts and isolated network/PID/IPC namespaces, and
checks a no-write inventory. Installation touches only an explicit settings file
and shared transaction paths. See the [dated evidence](../target-evidence.md#bounded-settings-module-installation-2026-09-22)
for hashes, checks, and the distinction between native module consumption and
full authenticated runtime behavior.
