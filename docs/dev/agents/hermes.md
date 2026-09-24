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

**Native named profiles:** the pinned source at commit `345cd2b057a452236de401d3534b8502a7465e8d`
(`v2026.9.14`) maps `hermes -p <name>` / `hermes --profile <name>` to
`<hermes-home>/profiles/<name>/config.yaml`, where `<hermes-home>` is the
directory holding the main `config.yaml` (default `~/.hermes`). profile-mango
installs named profiles there; see
[Named-profile installation](#named-profile-installation-2026-09-23) for the
file and line evidence. These are config/state profiles, distinct from
provider-local authentication profiles.

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

`profile-mango render <name> --target hermes --target-version 0.21.3` emits a
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

## Named-profile installation, 2026-09-23

`profile-mango install <profile> --target hermes` writes the three qualified
fields to the config that `hermes -p <profile>` reads, and leaves the default
config alone unless you pass `--default`. The plan prints
`use it: hermes -p <profile>`. Evidence is **source-only**: the pinned files
below were fetched read-only (via the GitHub Contents API and
`raw.githubusercontent.com`) at commit `345cd2b057a452236de401d3534b8502a7465e8d`
(`v2026.9.14`) on 2026-09-23. The locally installed pipx package is `0.19.0`,
not the pinned version, so it was not read or run as this evidence, and no
native `hermes -p`/`hermes profile` probe was run (the config-consumption
probe in the [retained native evidence](#bounded-installation-qualification-2026-09-22)
above does not exercise profile resolution).

| Pinned file | Lines | Observation |
| --- | --- | --- |
| [`hermes_cli/profiles.py`](https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/hermes_cli/profiles.py) | 24 | `_PROFILE_ID_RE = re.compile(r"^[a-z0-9][a-z0-9_-]{0,63}$")` — the on-disk profile id pattern. |
| same | 120 | `_RESERVED_NAMES = frozenset({"hermes", "default", "test", "tmp", "root", "sudo"})`. |
| same | 132-135, 138-142 | `_get_profiles_root()` is `_get_default_hermes_home() / "profiles"`; `_get_default_hermes_home()` is `~/.hermes` (or a Docker/custom root), independent of any already-active profile home. |
| same | 172-186 | `normalize_profile_name` lowercases input and case-folds `"default"` before dispatch. |
| same | 189-206 | `validate_profile_name` special-cases `"default"` to pass instead of rejecting it, then enforces the id pattern and rejects `_RESERVED_NAMES`. |
| same | 232-244 | `get_profile_dir(name)`: `"default"` returns the root home unchanged; any other valid id returns `<profiles-root>/<name>`. |
| same | 247-256 | `profile_exists(name)`: for a named profile, true only when `profile_dir.is_dir()` and the directory is not tombstoned — no other marker file is required. |
| same | 1789-1813 | `resolve_profile_env(profile_name)`, called from `main.py` before HERMES_HOME is set: `"default"` returns the resolved root; otherwise it requires `<root>/profiles/<canon>` to already exist as a directory (`is_dir()`, not tombstoned) and raises `FileNotFoundError` otherwise — the directory is **not** auto-created by profile selection. |
| [`hermes_cli/main.py`](https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/hermes_cli/main.py) | 401, 419-446 | `_scan_profile_flag` pre-parses `-p`/`--profile <name>` (and `--profile=<name>`) from `sys.argv` before any hermes module import. |
| same | 508-556 | `_apply_profile_override()` calls `resolve_profile_env(profile_name)` and sets `os.environ["HERMES_HOME"]` to the result before any other hermes import; a missing profile directory prints `Error: Profile '<name>' does not exist. Create it with: hermes profile create <name>` and exits 1 (except a `SUDO_USER` fallback that does not apply here). |
| [`hermes_cli/config_home.py`](https://github.com/NousResearch/hermes-agent/blob/v2026.9.14/hermes_cli/config_home.py) | 43-54 | `initialize_home` can create a missing `HERMES_HOME` and its required subdirectories — but this runs only *after* `HERMES_HOME` is set, i.e. after `resolve_profile_env` has already required the profile directory to exist. It does not rescue a missing named-profile directory. |

**Path rule.** profile-mango derives the profile config from the resolved main
config path: `<dir>/config.yaml` gives `<dir>/profiles/<name>/config.yaml`.
Unlike OpenClaw, this needs no fixed directory-name check, because Hermes
always resolves named profiles to a `profiles/` subdirectory directly below
whatever directory holds `config.yaml` — `<dir>` need not be literally named
`.hermes`. A relocated `--config` is accepted the same way a relocated default
config is accepted for any other target.

**Profile validity.** Only the `profiles/<name>` directory needs to exist for
`hermes -p <name>` to resolve (`profile_exists`/`resolve_profile_env` above);
no `hermes profile create` bootstrap (memories/, sessions/, skills/, a
`profile.yaml` description file, or a shell alias) is required. Writing
`config.yaml` there creates that directory as a side effect, which is
sufficient.

**`default` profile.** Hermes maps `-p default` (any case) to the base
`HERMES_HOME`, not `profiles/default`. So installing a profile named `default`
patches the main config in place, and no second file is written.

**Reserved and invalid names.** `hermes -p <name>` itself rejects the
`_RESERVED_NAMES` set and any id outside `_PROFILE_ID_RE`, so profile-mango
blocks the same names before writing (`install.named_profile_path_unsafe`)
rather than installing a profile Hermes would refuse to start.

**Separate state.** A new profile directory has no memories, sessions,
skills, or credentials from the default profile. profile-mango doesn't copy
them, and the plan says so (`hermes.install.profile_state_separate`).

**Validation.** Unit and command-level tests cover these cases in synthetic
homes: create, noop reinstall, `--default`, the `default` name, reserved-name
blocking, and undo. A sandbox run of the built binary in a scratch `HOME`,
with `PROFILE_MANGO_HOME` inside it, installed a `coding` profile. It created
`~/.hermes/profiles/coding/config.yaml` and left `~/.hermes/config.yaml`
byte-identical (SHA-256 `7ada0c81a5c481d8836800b97e0bedc79243d22e072c966e1f0f5d9be62f3231`
before and after). Undo then removed the profile config (keeping the
now-empty `profiles/coding` directory) and left the default config untouched.
