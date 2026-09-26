# Codex configuration reference

**Reference date:** 2026-09-23. **Status:** exact-version inert preview plus a separate three-setting installer for Codex CLI `0.154.0` that writes a named profile file by default. The renderer emits candidate provider/model/effort syntax and copied resources only under an explicit staging directory. It never emits active `config.toml`, `AGENTS.md`, or skill locations, and still reports `applicable: false` for the full profile because authentication, delivery, precedence, and enforcement remain unverified.

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

`mango render <name> --target codex --target-version 0.154.0` is an
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
In a bounded isolated empty-auth home, installed `codex-cli 0.154.0` app-server
`config/read` reported all three installer-generated root fields and their user
origins. A trusted synthetic project and session overrides shadowed root model
and effort; an untrusted project did not. App-server startup initializes other
subsystems, so this is not a safe personal-home inspector or OAuth proof.

## Named profiles in 0.154.0

Codex `0.154.0` profiles are files: `codex --profile <name>` layers
`$CODEX_HOME/<name>.config.toml` over `config.toml`. Names use ASCII letters,
digits, `_` and `-`. The older `[profiles.<name>]` tables and root
`profile = "<name>"` selector are legacy: the pinned binary refuses
`--profile <name>` while `config.toml` holds either one for that name, and
refuses to start at all with a root `profile` key. A `--profile` name with no
file is treated as an empty layer, so a typo silently runs the base config.
`--profile` applies only to runtime commands, `codex mcp`, `codex sandbox` and
`codex debug prompt-input`; app-server `config/read` rejects it. Source lines
and isolated installed-binary controls are in the
[evidence ledger](../target-evidence.md#codex-01540-named-profile-installation-2026-09-23).

## Bounded settings-only installation

`mango install <name> --target codex` writes a named profile: the three
settings go to `<name>.config.toml` beside `$CODEX_HOME/config.toml` (default
`~/.codex/`), and the plan prints `use it: codex --profile <name>`. `config.toml`
is read, checked, and left byte-for-byte unchanged. `--default` also writes the
same three settings as root keys in `config.toml`, so plain `codex` uses them.
`--config codex=<path>` names a different `config.toml`; the profile file goes
next to it. Bare `codex` resolves to the single qualified `codex@0.154.0`. The
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

`globalInstructions` may own `AGENTS.md` in `$CODEX_HOME` (source `rust-v0.154.0` `6b9826e`, `codex-rs/codex-home/src/instructions/mod.rs:9-10,26-27`); an existing `AGENTS.override.md` wins over it. Written only when the profile is the default (`mango use`, `install --default`); whole-file ownership with create-only backup, drift checks, release on `use`, and undo. See [target evidence](../target-evidence.md#global-instruction-files-2026-09-25-ap-nym).

## Home `~/AGENTS.md` (ap-5mp, 2026-09-25)

`~/AGENTS.md` is read only when `$HOME` is the working directory or the nearest project root: `rust-v0.154.0` `6b9826e`, `codex-rs/core/src/agents_md.rs:1-16,187-240` walks up only to the nearest `project_root_markers` directory (`.git` by default). Codex therefore does not own `globalInstructions.home`. See [target evidence](../target-evidence.md#home-instruction-file-agentsmd-2026-09-25-ap-5mp).
