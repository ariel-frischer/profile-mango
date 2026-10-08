# Pi configuration reference

**Reference and qualification date:** 2026-09-21 UTC, requalified 2026-09-26. **Release pin:** `v0.87.1`, commit `f07218c4d4bbc12bef056a7058c3dd49dfe41abe` (originally `v0.86.1`, commit `13cbf77df2396303013a41646bcfa77b4271ae56`; see [requalification](#requalification-to-v0871-2026-09-26)). **Status:** exact-version inert preview plus a three-default settings installer. Native settings-module getters and project-over-global precedence were qualified on 2026-09-22 and re-run against the `0.87.1` package on 2026-09-26; full startup, authentication, delivery, and enforcement remain blocked. Pi is qualified independently from [Oh My Pi](oh-my-pi.md).

## Immutable release and package evidence

The release tag `v0.87.1` resolved with `git ls-remote` to commit
`f07218c4d4bbc12bef056a7058c3dd49dfe41abe`. Retrieved source archive hashes:

- Tag archive `pi-v0.87.1.tar.gz`: SHA-256 `c3902f45689af9ed9c8ee225554d31a649a993b06f04ab2022bb91ed75e808dc`.
- Commit-pinned archive: SHA-256 `f6ba24ed7e1e6dbda1844ca55c61e20e3ef21e5cc66f9eb2e0611318f6c46c15`.

The immutable npm package is `@earendil-works/pi-coding-agent@0.87.1`.
Its registry tarball has SHA-256
`1423ee3c61e7c96464e1cbf3c8dc24d3056cb3410995c3671a98c3ecc527540f` and
registry integrity
`sha512-m8ArJUtVcQMSe1lLE/Ei7vX/JV7O39sWmWBsXV2NOU70F0qCp8GubA24pT3LnwTmM6LL2xV80/h6sQg85n69ew==`.
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

The `mango render --target pi --target-version 0.87.1` adapter emits a
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

[settings]: https://github.com/earendil-works/pi/blob/v0.87.1/packages/coding-agent/docs/settings.md
[providers]: https://github.com/earendil-works/pi/blob/v0.87.1/packages/coding-agent/docs/providers.md
[models]: https://github.com/earendil-works/pi/blob/v0.87.1/packages/coding-agent/docs/models.md
[security]: https://github.com/earendil-works/pi/blob/v0.87.1/packages/coding-agent/docs/security.md
[skills]: https://github.com/earendil-works/pi/blob/v0.87.1/packages/coding-agent/docs/skills.md
[readme]: https://github.com/earendil-works/pi/blob/v0.87.1/packages/coding-agent/README.md
[package]: https://registry.npmjs.org/@earendil-works%2fpi-coding-agent/0.87.1
[source]: https://github.com/earendil-works/pi/tree/f07218c4d4bbc12bef056a7058c3dd49dfe41abe/packages/coding-agent

## Installation qualification update, 2026-09-22

The opt-in settings-module probe verifies all three global defaults and project
model override without full CLI startup. It pins the imported module hash, uses
allowlisted runtime/package mounts and isolated network/PID/IPC namespaces, and
checks a no-write inventory. Installation touches only one settings file (explicit or the
[default config destinations](../target-evidence.md#default-config-destinations-2026-09-22)) and shared transaction paths. See the [dated evidence](../target-evidence.md#bounded-settings-module-installation-2026-09-22)
for hashes, checks, and the distinction between native module consumption and
full authenticated runtime behavior.

## Home `~/AGENTS.md` (ap-5mp, 2026-09-25)

`globalInstructions.home` owns `~/AGENTS.md` through Pi: source `v0.87.1` `f07218c` (unchanged lines from `v0.86.1` `13cbf77`), `packages/coding-agent/src/core/resource-loader.ts:71-72,119-156` walks every ancestor of the working directory to `/`, taking each directory's first of `AGENTS.override.md`, `AGENTS.md`, `AGENTS.MD`, `CLAUDE.md`, `CLAUDE.MD`, so `~/AGENTS.md` applies in every directory under `$HOME` unless `~/AGENTS.override.md` exists. Pi is the single owner: its manifest records the file with create-only backup, drift checks, release on `use`, and undo; plans warn that other agents share it. See [target evidence](../target-evidence.md#home-instruction-file-agentsmd-2026-09-25-ap-5mp).

## Requalification to v0.87.1, 2026-09-26

Source diff `v0.86.1..v0.87.1` (blobless clone of `earendil-works/pi`) over the
files Mango relies on:

| File (v0.87.1 SHA-256) | Result |
| --- | --- |
| `src/core/settings-manager.ts` (`5d1bdac20da52fdbac76d49fb9ce22d142c375e8efd5bf446bf6c047c5edac42`) | identical; the packaged `dist/core/settings-manager.js` is still `5368b155…` |
| `src/config.ts` (`7874d7f63a62778f4281a324f99ba4faee8dc9a5c7ae80e2c531552d199257f9`) | identical (agent dir, `PI_CODING_AGENT_DIR`) |
| `src/core/resource-loader.ts` (`60e8b49790ab7ab7997dcaa647bef5eaaabdb8273c317c72b1f9567d78e99620`) | only prompt-template diagnostics changed; context-file lines 71-72 and 119-156 identical |
| `docs/settings.md` (`f7d55373909e38b111e8e0b002401ebf3bde7e5ef25a7597eb8a29679f431cf3`) | restructured (locations moved to new `docs/configuration.md`); still lists `defaultProvider`, `defaultModel`, `defaultThinkingLevel` and states that project settings override agent-directory settings |
| `dist/bundle/cli.js` in the package | identical hash `e79626f2…` |

Other changes (a newer xAI default model, `imageResize`/`inputLimits` in
model config) do not touch fields Mango writes or reads. No behavioral
difference was found for Mango's settings or `~/AGENTS.md` ownership.

`TestNativeSettingsModuleQualification` was re-run with
`PROFILE_MANGO_PI_NATIVE_PACKAGE` set to the extracted `0.87.1` tarball and
`PROFILE_MANGO_PI_NATIVE_TARBALL` set to the tarball itself (hash checked by the
test); both the positive/precedence and malformed-settings cases passed.
