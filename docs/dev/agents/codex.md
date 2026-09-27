# Codex configuration reference

**Reference date:** 2026-09-26. **Status:** exact-version inert preview plus a separate three-setting installer for Codex CLI `0.157.1` (requalified from `0.154.0`; see [the requalification evidence](../target-evidence.md#codex-01571-requalification-2026-09-26)) that writes a named profile file by default. The renderer emits candidate provider/model/effort syntax and copied resources only under an explicit staging directory. It never emits active `config.toml`, `AGENTS.md`, or skill locations, and still reports `applicable: false` for the full profile because authentication, delivery, precedence, and enforcement remain unverified.

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

`mango render <name> --target codex --target-version 0.157.1` is an
offline compiler boundary, not an installer. Without `--preview`, applicability
blockers produce diagnostics and no output. With `--preview`, the command may
atomically create a new explicit `--out` directory containing `render.json`,
a `preview/<name>.config.toml.preview` candidate, and inert resource copies. Once
staged it exits 0, prints remaining blockers as warnings, and the report stays
`applicable: false`.

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
In a bounded isolated empty-auth home, installed `codex-cli 0.154.0` and
`0.157.1` app-server `config/read` reported all three installer-generated root fields and their user
origins. A trusted synthetic project and session overrides shadowed root model
and effort; an untrusted project did not. App-server startup initializes other
subsystems, so this is not a safe personal-home inspector or OAuth proof.

## Named profiles in 0.157.1

Codex `0.157.1` profiles are files (unchanged since `0.154.0`): `codex --profile <name>` layers
`$CODEX_HOME/<name>.config.toml` over `config.toml`. Names use ASCII letters,
digits, `_` and `-`. The older `[profiles.<name>]` tables and root
`profile = "<name>"` selector are legacy: the pinned binary refuses
`--profile <name>` while `config.toml` holds either one for that name, and
refuses to start at all with a root `profile` key. A `--profile` name with no
file is treated as an empty layer, so a typo silently runs the base config.
`--profile` applies only to runtime commands, `codex mcp`, `codex sandbox` and
`codex debug prompt-input`; app-server `config/read` rejects it. Source lines
and isolated installed-binary controls are in the
[evidence ledger](../target-evidence.md#codex-01540-named-profile-installation-2026-09-23),
repeated for `0.157.1` in the
[requalification](../target-evidence.md#codex-01571-requalification-2026-09-26).

## Bounded settings-only installation

`mango install <name> --target codex` writes a named profile: the three
settings go to `<name>.config.toml` beside `$CODEX_HOME/config.toml` (default
`~/.codex/`), and the plan prints `use it: codex --profile <name>`. `config.toml`
is read, checked, and left byte-for-byte unchanged. `--default` also writes the
same three settings as root keys in `config.toml`, so plain `codex` uses them.
`--config codex=<path>` names a different `config.toml`; the profile file goes
next to it. Bare `codex` resolves to the single qualified `codex@0.157.1`. The
installer plans only `model_provider`, `model`, and `model_reasoning_effort`
(`none`, `minimal`, `low`, `medium`, `high`, or `xhigh`, which the pinned source
parses as named `ReasoningEffort` variants; `max`, `ultra`, and others block) for an
OpenAI/native/OAuth binding. Whether a model accepts a level is model-dependent
and unverified. Use all three explicit project input flags together if not using the
profile home. An existing unowned config needs `--override`; apply requires
`--apply --yes --expect-plan <planID>` or interactive terminal consent. Plans are
read-only, backups are enabled by default, and repeated application is a no-op.
See [the version-qualified evidence](../target-evidence.md#codex-01540-settings-only-installation-2026-09-22).

For an installation made with the default adjacent manifest and backups, preview a
guarded reversal with `mango undo --target codex --config
/explicit/disposable/config.toml --json` (`restore` is an alias; `--config` defaults
to the documented path and `--original-plan <64-hex-install-plan-ID>` pins a specific
install instead of the latest committed journal).
The read-only output contains a *new* restore plan ID and both file effects.
Apply with the same flags plus `--apply --yes --expect-plan <restore-plan-ID>`
for automation, or omit `--json` and use `--apply` on a terminal to answer the
default-no y/N confirmation prompt.
This requires a committed install journal beside the config, an unchanged installed
config (or `--override` to discard later edits) and manifest, matching ownership and
adjacent original backup bytes/mode.
An existing config installed with `--no-backup`, edited target state without `--override`, alternate
manifest destination, or stale preview is rejected. A manifest created by the
original install is removed; an originally owned manifest is restored from its
backup. No arbitrary journal or backup path is accepted. Restore is a file
reversal, not proof of native effective state or OAuth behavior.
The journal and manifest must match the installer's exact generated JSON form;
an originally present ownership manifest must also have that form. A previously
accepted but manually reformatted manifest is not automatically restored, even
if its parsed ownership is otherwise valid. Keep its backup for manual review.
Same-content file replacement after preview also invalidates restore consent.

The adapter rejects required permissions, tools, instructions, skills,
non-qualified routes, malformed TOML, a root `profile` key, a same-name legacy
`[profiles.<name>]` table and provider shadow state. It patches only the named
profile file, plus the selected root document with `--default` (explicit or
[default config destinations](../target-evidence.md#default-config-destinations-2026-09-22)), preserving
unrelated keys/comments, other `[profiles.*]` tables and target-owned
authentication. Undo reverses the latest install: for a named install it removes
or restores that profile file and the manifest, and leaves earlier profiles. It never runs Codex,
reads auth stores, or selects OAuth. `codex login status` can tell a user whether
the stored login is API-key or ChatGPT, but the same ChatGPT message covers
Codex-managed OAuth and externally supplied tokens. Do not share status output
containing key fragments. Project/runtime overrides were observed in a
controlled synthetic home but are not inspected or controlled by installation.
Since `0.157.1`, a managed `requirements.toml`, cloud bundle or MDM
`model_provider` also overrides the installed provider. This is source-reviewed
only, and installation does not inspect it.
Installer-verified authentication identity,
model availability, resource delivery and policy enforcement remain unverified.
Settings-only success must not be reported as full-profile applicability.
Settings-only qualification used disposable configuration, not a personal
Codex config. Separately authorized read-only authentication metadata inspection
and two bounded runtime requests later timed out without proving model delivery;
no personal configuration was installed or changed. See the
[evidence ledger](../target-evidence.md#codex-01540-settings-only-installation-2026-09-22).

[basic]: https://learn.chatgpt.com/docs/config-file/config-basic
[advanced]: https://learn.chatgpt.com/docs/config-file/config-advanced
[reference]: https://learn.chatgpt.com/docs/config-file/config-reference
[schema]: https://developers.openai.com/codex/config-schema.json
[agents]: https://learn.chatgpt.com/docs/agent-configuration/agents-md
[skills]: https://learn.chatgpt.com/docs/build-skills

## Global instruction files (ap-nym, 2026-09-25)

`globalInstructions` may own `AGENTS.md` in `$CODEX_HOME` (source `rust-v0.157.1` `3665039`, `codex-rs/codex-home/src/instructions/mod.rs:12-13,43-44`); an existing `AGENTS.override.md` wins over it. Since `0.157.1`, a failed refresh keeps the last good copy. Written only when the profile is the default (`mango use`, `install --default`); whole-file ownership with create-only backup, drift checks, release on `use`, and undo. See [target evidence](../target-evidence.md#global-instruction-files-2026-09-25-ap-nym).

## Home `~/AGENTS.md` (ap-5mp, 2026-09-25)

`~/AGENTS.md` is read only when `$HOME` is the working directory or the nearest project root: `rust-v0.157.1` `3665039`, `codex-rs/core/src/agents_md.rs:1-18,192-245` walks up only to the nearest `project_root_markers` directory (`.git` by default). Codex therefore does not own `globalInstructions.home`. See [target evidence](../target-evidence.md#home-instruction-file-agentsmd-2026-09-25-ap-5mp).

## Role subagent files (ap-6lp, 2026-09-25)

Each declared profile role becomes `$CODEX_HOME/agents/<role>.toml`, written only when the profile is the default (`install --default`, `mango use`). Source review of `rust-v0.157.1` (`36650394c5b38c2990ccf2a3457165ca3e9d9726`); the three `agent-roles` files are byte-identical to `rust-v0.154.0`:

| Claim | Pinned source (file SHA-256) |
| --- | --- |
| Role files are TOML with `name`, `description`, `nickname_candidates`, plus flattened `ConfigToml` keys; unknown keys are rejected | `codex-rs/agent-roles/src/agent_role_config.rs` (`70ba8cf41c7339a06fee896d41c57a6e8344b770b1a1ed9a04c66d56e27d9511`), lines 20-28 |
| `developer_instructions` is required and non-blank; `name` and `description` are validated | same file, lines 67-88, 120-157 |
| User role files are discovered under `<config dir>/agents`, recursively as `*.toml`; `description` is required | `codex-rs/agent-roles/src/loader.rs` (`611e202c2bb4a2bad6c201bd93555d56b6c6c7021e96a294015556fd47b16b38`), lines 75-81, 237-249; `codex-rs/agent-roles/src/discovery.rs` (`7a82592608e9310c11da591bceef6134355c3ac925597b26e6f2d03e6c5f155a`), lines 7-40 |
| `model` (line 168), `model_provider` (line 173), and `model_reasoning_effort` (line 392) are `ConfigToml` keys | `codex-rs/config/src/config_toml.rs` |

The file carries `name`, `description`, `developer_instructions` (the role's `instructions` resource, else its description), and, when the route binds the role, `model_provider`, `model`, and `model_reasoning_effort`. An effort outside `none`..`xhigh` is omitted and reported as `effort <v> (role <r>): NOT APPLIED`. Ownership is whole-file (`role-definition` in `mango status`): an unmanaged same-name file is adopted with a create-only backup, drift blocks without `--override`, `mango use` releases roles the next profile lacks (restoring adopted bytes), and undo restores bytes. Named-only installs skip `role-definitions` with the reason "subagent files are global; install with --default or mango use". Evidence level: source review plus a built-binary sandbox smoke (install, status, use, undo); Codex was not run against the generated files.

Profile `agentFiles.codex` copies native `*.toml` files verbatim into the same directory (kind `agent-file`, same gates and ownership as role files); mango does not parse them. See [target evidence](../target-evidence.md#profile-agent-files-2026-09-26-ap-8x5).

## Skill folders (ap-794, 2026-09-27)

Pinned source: the `rust-v0.157.1` [skills loader](https://github.com/openai/codex/blob/rust-v0.157.1/codex-rs/skills/src/lib.rs) scans the user root `$HOME/.agents/skills`, not `$CODEX_HOME/skills` (`$CODEX_HOME/skills/.system` is the bundled system cache); the [parser](https://github.com/openai/codex/blob/rust-v0.157.1/codex-rs/skills/src/parser.rs) requires YAML frontmatter with a nonempty `description`, and `name` falls back to the folder name. Symlinked folders are followed; supporting files are allowed; same-name skills are not merged. Each `skills` entry's whole folder (every regular file beside its `SKILL.md`, at most 256, modes normalized to `0755` when any execute bit is set and `0644` otherwise) is copied to `<folder name>/` under `$HOME/.agents/skills` (resolved from the user home, never `CODEX_HOME`), with `{{route.…}}` placeholders rendered per target in UTF-8 text files (provenance digests cover the source bytes), written only when the profile is the default (`mango use`, `install --default`) and never for a named agent destination; a named profile lists `skills` as skipped. Files are whole-file owned (ownership kind `skill`) with create-only backups of adopted files, drift checks, release on `use` (released folders are removed once empty; adopted files are restored), and undo. A symlink or non-directory on the destination path is a conflict naming it; files in an adopted folder that profile-mango does not manage are kept and listed in a plan warning. The compact plan prints `skills: a, b` per target. The same folder is also read by Oh My Pi and OpenCode by default, so a skill installed for Codex is visible to them.
