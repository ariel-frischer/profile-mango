# Target evidence ledger

**Evidence dates:** initial 2026-09-20 local / 2026-09-21 UTC; installation updates 2026-09-22 UTC; Codex runtime diagnostics 2026-09-23 UTC
**Platform:** Linux x86_64
**Scope:** Canonical contract, shared transaction, exact-version renderer, and
narrow installation evidence. Installer qualification and validation use synthetic
disposable state, not personal agent homes or authenticated providers. Separate,
explicitly authorized Codex runtime diagnostics are recorded below and do not
upgrade the installer's evidence level.
The dated target sections below distinguish initial source/package review from
subsequent isolated native commands or settings-module probes. Native probes may
write only within their disposable sandbox. No full-profile runtime or policy
compatibility is implied by installation of a qualified configuration subset.

M0 freezes the portable contract. It does not claim that either target enforces that contract yet. Target claims use three separately reported levels:

1. schema, golden, and negative fixtures
2. offline native parsing or effective-config inspection in an isolated test home
3. explicitly authorized runtime observation

A lower evidence level never implies a higher one. Parsing is not runtime enforcement, and instruction text is not a permission boundary.

The [validation and reference strategy](agent-validation.md) defines future
no-inference probes, exact-version support records, local documentation provenance
and on-demand upstream drift checks. It adds no executed evidence to this ledger.
Native `doctor` commands require effect review, not automatic trust. Documentation
versions, installed binaries and tested support must remain distinct.

## Product boundary

Codex is the first intended public adapter candidate. Public targets have inert
exact-version renderers. Installable subsets apply at an explicit `--config` path
or, when omitted, at the documented default user path listed under
[default config destinations](#default-config-destinations-2026-09-22).
Codex permits only three route settings, written by default as a native
`$CODEX_HOME/<name>.config.toml` profile selected with `codex --profile <name>`
and, only with `--default`, also as root `config.toml` settings; the exact installed
binary loaded both forms in an isolated synthetic home,
but OAuth identity and full-profile applicability remain unverified. Claude
Code permits model plus `low`/`medium`/`high`/`xhigh` `effortLevel` installation; Pi permits three route-default
settings, Hermes three model/reasoning fields, and OpenCode a named primary
agent definition (model, effort variant, and instructions) written by default as
`agents/<profile>.md` next to `opencode.json` and selected with
`opencode --agent <profile>`, or explicitly as `primary:<name>`/
`subagent:<name>`; only with `--default` does it also patch the top-level
`model` field and one qualified skill resource into `opencode.json`, since
OpenCode skills are a directory-wide `skills.paths` setting, not per-agent.
OpenClaw permits model-primary plus thinking-default settings. Other target gates
remain blocked unless explicitly qualified below. By default `install` applies
only a target's qualified subset and lists every known profile requirement it
cannot install (permissions, tools, instructions, skills) as skipped for that
target, in the human plan and as JSON `skippedRequirements`; a route effort a
target cannot write is listed the same way (`effort <value>: NOT APPLIED
(<reason>)`, JSON `requirement: effort` with `value` and `reason`). A skipped
requirement is never claimed as honored. `install --strict` blocks on them
instead. Unknown required properties and unqualified targets always block. Field consumption, full-route
authentication, precedence, delivery, and runtime enforcement are separate claims.
Native probes require approved disposable scope and exact evidence. The Jcode fork custom
Jcode fork is experimental developer evidence, not a supported public target.

the Jcode fork must not appear in the README support list, release
promise, or public compatibility matrix. Public support for any Jcode variant
would require a stable external distribution, versioned public documentation,
reproducible fixtures, and evidence independent of the developer's machine.

## Pinned local observations

| Target | Observed build | Evidence source | Current status |
| --- | --- | --- | --- |
| OpenCode | release `v1.18.31` at `a97622c801f4ca571530ddc51076af659a9c32cd`  | Exact binary debug config, skill and named-agent consumption (including `variant`); lossless JSONC/resource patch tests, compiled transactional lifecycle, isolated negative controls and backup/restore inventory | Main model plus one SKILL.md and skills.paths, or an explicit named primary/subagent definition with model, effort `variant` and ordered instructions. Directory discovery is not an exclusive allowlist. Full auth, precedence, runtime delegation and enforcement remain blocked |
| Codex CLI | `codex-cli 0.154.0` | Exact-source resolver and isolated installed-binary `config/read` consumed installer-generated provider/model/high-effort fields (and project/session `medium`/`minimal` effort overrides); exact-source `ReasoningEffort` parser; exact-source profile-v2 loader plus isolated installed-binary `--profile` controls loaded a generated `<name>.config.toml`; built profile-mango CLI synthetic plan/apply/reapply/undo | Three settings installable for an OpenAI/native/OAuth binding with `none`/`minimal`/`low`/`medium`/`high`/`xhigh` effort, as a named `<name>.config.toml` profile beside the explicit or documented default `config.toml`, and in root `config.toml` only with `--default`; trusted project and runtime overrides can shadow them. OAuth identity, model availability and per-model effort support, delivery and full-profile applicability remain unverified |
| Oh My Pi | source `omp/18.2.6` at `78b753124d11f8dd3ae73e2524125890ff7c977e` | Exact source/addon, pinned Bun/nightly Rust, read-only getter positive/control probes against compiled CLI output | Two model-role/thinking-default fields installable; standalone startup, authentication, precedence, delivery and enforcement remain blocked |
| Claude Code | npm `2.1.278`, release commit `bf7d404e26a5fb6167d21b46c93a2bf6c22ab274`  | Immutable release/package provenance plus exact-ELF explicit-file model consumption before no-auth termination, exact-ELF offline `/model` effort status, and compiled transaction checks | Model and `effortLevel` (low/medium/high/xhigh) installer qualified; full effective state, precedence, route/auth, resources and policy enforcement remain blocked |
| OpenClaw | source `v2026.9.5` at `ec9c1a13db8938e5a3eaa51fca2e981cde2395a9` | Exact source/native getter hashes, override/fallback checks and actual compiled-output consumption | Two model/thinking defaults installable into the pinned-source `openclaw --profile <name>` config `<home>/.openclaw-<name>/openclaw.json`, derived only from a main config at `<home>/.openclaw/openclaw.json`, and into the main config only with `--default` (the `default` name is the main config); full startup, auth, delivery and enforcement remain blocked |
| Hermes Agent | source `v2026.9.14` at `345cd2b057a452236de401d3534b8502a7465e8d` (`v0.21.3`)  | Immutable source/archive plus hash-gated native read-only config merge under isolated Python 3.12.13 and compiled transaction checks; source-only profile resolution evidence plus a built profile-mango CLI synthetic plan/apply/reapply/undo | Three model/reasoning config fields installable, as a named `profiles/<name>/config.yaml` profile below the resolved `config.yaml`'s directory that `hermes -p <name>` reads, and in the main `config.yaml` only with `--default` (the `default` name is the main config); full startup, authentication, delivery and runtime enforcement remain blocked |
| Pi | source tag `v0.86.1` at `13cbf77df2396303013a41646bcfa77b4271ae56`, package `@earendil-works/pi-coding-agent@0.86.1`  | Immutable release/package provenance plus exact settings-module getter and project-override evidence, compiled transaction checks | Three route-default settings installable; full startup, authentication, delivery and runtime enforcement remain blocked |
| Jcode fork | `jcode v0.83.909-dev (ca8017a3a)` | isolated synthetic `profile list/show/resolve` probe, exact binary SHA-256, retained golden/negative adapter tests, and bundled `README.md`/`docs/WRAPPERS.md` | Experimental-only inert preview renderer; native applicability remains blocked |

The observations are version-qualified snapshots, not compatibility ranges.
Dated initial M0 source reviews below are historical; later installation
qualification sections and this summary record the current narrow support gates.

## Installed-version policy (ap-uuz.14)

Agents auto-update, so the installed binary often differs from the qualified
version above. Each install adapter states a *tested range*. By default it is
the tilde range of the qualified version: at least the qualified version and
below the next minor (`2.1.278` gives `>=2.1.278 <2.2.0`; `0.154.0` gives
`>=0.154.0 <0.155.0`). CalVer `year.month.patch` versions keep the same
year and month (`2026.9.5` gives `>=2026.9.5 <2026.10.0`). An adapter may set an
explicit `compatibleRange`, but only when new evidence recorded in this ledger
supports the wider range.

- `install` runs the target's documented command with `--version` (bounded
  timeout, capped output, and credential-free environment, as `doctor` does)
  for each ready or no-op target. It records the result as `versionCheck` in the
  plan JSON and plan ID, and prints an `installed version:` line.
- Inside the range, the plan proceeds with that note only.
- Outside the range, the command is missing, or its version is unreadable, the
  plan still proceeds with status `ready`. It shows a visible warning (human
  and JSON diagnostic `install.version_out_of_range`, `install.version_not_found`
  or `install.version_unknown`). The settings are still written for the qualified
  version, and profile-mango makes no compatibility claim for that binary.
- An explicit `--target name@version` is still exact. An unregistered version
  stays blocked, and no version check runs for it.
- `doctor` shows `IN RANGE` (`yes`, `no (tested <range>)`, or `-`) and adds
  `inRange` and `compatibleRange` to `--json`. `versionMatch` keeps its earlier
  exact-substring meaning.
- A tested range is a stated claim, not new evidence. Qualification evidence
  still belongs to the exact version recorded above.

## Pi v0.86.1 evidence

### Immutable artifact, release, and platform

The immutable Git tag `v0.86.1` for `earendil-works/pi` resolved with
`git ls-remote https://github.com/earendil-works/pi.git refs/tags/v0.86.1` to
commit `13cbf77df2396303013a41646bcfa77b4271ae56`. The retrieved tag archive
`https://github.com/earendil-works/pi/archive/refs/tags/v0.86.1.tar.gz` has
SHA-256 `16d65ce53bfab1ae24d625538d434c341c4789d34b352d8c6ef699cf1d1d567d`.
The commit-pinned archive
`https://github.com/earendil-works/pi/archive/13cbf77df2396303013a41646bcfa77b4271ae56.tar.gz`
has SHA-256 `016d83312289ca9b8d3a9d2a5ad804b265277c659472833cfd602cdceecf3818`.

The exact npm package `@earendil-works/pi-coding-agent@0.86.1` has registry
metadata SHA-256 `8e8e5cf99a033e9aae1db15e8455f819e1921b7c8351b1044acdab9525c14fd7`
and tarball SHA-256
`8dff93e6fa03e0d498e72a78d2c7bb5f094f5e06ee268e6abd000ba2984a0b6a`. Its
registry integrity is
`sha512-vZBuNfJnruxZyemZ3O05V0S/Ylze08ahFTIQ1Mik++gVdOevPl89gt/Uv0U97BPAJaj9cj6Vf9rcIgKtUrd0BA==`.
The integrity was recomputed from the retrieved tarball. The package declares
Node.js `>=22.19.0`, its `bin.pi` entrypoint is `dist/bundle/cli.js`, and the
extracted entrypoint SHA-256 is
`e79626f2dd6f94aa45d30f3fa63cd84319a6eefcd150b353cfaf274366926774`.
Qualification used Linux x86_64, Node.js `v24.21.0`, and npm `11.19.0`. This is
immutable source/package review, not an installed Pi observation.

### Static effect review and native execution boundary

The exact release source review covered the CLI, startup, settings, migrations,
authentication storage, model runtime, session services, package manager,
resource loader, extensions, and the version-pinned settings/providers/models/
security/skills documentation. The default agent directory is `~/.pi/agent` or
the `PI_CODING_AGENT_DIR` override, with settings, `auth.json`, `models.json`,
`models-store.json`, sessions, prompts, tools, and debug logs. Startup constructs
settings and HTTP services, runs cleanup and migrations, and can read, rename,
create, or rewrite target files. Runtime setup loads project settings and trust
state, context files, extensions, skills, prompts, themes, packages, providers,
auth state, models, and session state. Package/config/update paths can use npm or
git subprocesses, network refreshes, and writes. First-time setup can persist
theme and analytics settings.

No Pi native command was run. Considered invocations included `pi --version`,
`pi --help`, `pi --list-models`, `pi config --help`, `pi config -l`,
`pi update --models`, and noninteractive print/model paths. The source review did
not establish a bounded, no-write, credential-free, no-provider, no-extension,
noninteractive invocation. A synthetic `HOME` alone would not isolate all agent,
project, XDG, package, session, and extension paths. Native acceptance, effective
state, precedence, delivery, authentication, and enforcement remain unverified.
This is an explicit evidence gap, not a claim that every invocation is unsafe in
every environment.

### Inert adapter boundary

The Pi adapter emits only an inert JSON settings candidate at
`preview/<profile>.settings.json.preview` with the exact source-grounded keys
`defaultProvider`, `defaultModel`, and `defaultThinkingLevel`. It never emits
credentials, authentication files, model-store data, provider URLs, sessions,
transport, or resource paths in the candidate. Reports separate candidate syntax,
native acceptance, effective state, precedence, route authentication, extension
discovery, delivery, permissions/tools, and runtime enforcement. `applicable` is
always false, and validated canonical resources are copied only into explicit
inert staging output.

### Bounded settings-module installation, 2026-09-22

The separate opt-in `TestNativeSettingsModuleQualification` imports only the exact
package's `dist/core/settings-manager.js`, SHA-256
`5368b155ec26d88374cec9e66b8e588b5041a0fb0047414f70b34e13892c4f48`.
This hash was compared with the same file in the immutable tarball above. It is
not a full CLI startup test. The hardened probe uses allowlisted runtime/package
mounts, synthetic writable state, cleared environment, isolated network/PID/IPC
namespaces, dropped capabilities, and a 15-second timeout. It verifies native
getters consume all three generated global defaults, project settings override
the global model, malformed settings produce a global parse diagnostic, and no
sandbox file or content changes occur.

The installer manages only `defaultProvider`, `defaultModel`, and
`defaultThinkingLevel` in one explicitly supplied settings JSON file. It preserves
unrelated data and rejects malformed/duplicate/ambiguous managed values and
unsupported required profile effects. Compiled-CLI tests verify real disposable
plan/apply, default backup bytes, deterministic diffs, hash-bound consent, stale
rejection, no-op reapply, and backup restoration. Project overrides remain
target-owned and can supersede global defaults. No implicit global destination is
selected. Authentication identity, model catalogs, full startup, trust,
extensions, resources, tools, permissions, delivery, and enforcement remain
unverified or blocked. The full-profile renderer remains inert.

## Claude Code v2.1.278 evidence

### Immutable artifact, release, and platform

The immutable GitHub release `v2.1.278` resolves through `refs/tags/v2.1.278`
to commit `bf7d404e26a5fb6167d21b46c93a2bf6c22ab274`; the release API marks it
immutable. The exact npm wrapper tarball
`@anthropic-ai/claude-code@2.1.278` has SHA-256
`08c6dfcf3dafcfd30e09b2926c596e274f0fa20844a5801ada7f1c8e6227157e` and
integrity `sha512-mfNRqC0GaEXqmP97NiwJBeYBmRuqe2VzgLUreUUaEhyJxJWx2Z6ClW1tBOncGNNXdhj6EY4LPvUIWP+oq311CA==`.
The pre-postinstall wrapper `bin/claude.exe` stub is SHA-256
`6d7abae055d3b598281300a6c835086dec81bf3048f8a2294c5d3e50c8830d7b`.
The exact Linux x64 glibc package
`@anthropic-ai/claude-code-linux-x64@2.1.278` has SHA-256
`d1fb51ab0a0234d1bd7f418ee9d6b6b124c2412b2ddaf3dfc3256bad8063f1c7` and
integrity `sha512-q3r+5aLGAet1MGMkCH2xPsuIW9A40ws4zftURxhYwDenheCKvXc7Gr1jBwvEhYG8uQwgY3YsfNwnvIsh1Bjmeg==`.
The extracted native ELF is SHA-256
`5c4735937844e84f8a93306e841a5b0e12252909b07870f789b190468da147ab`.

The wrapper declares Node.js `>=22.0.0`; qualification observed Node `v24.21.0`,
npm `11.19.0`, Linux x86_64, and glibc `2.44`. The native ELF is dynamically
linked against the normal glibc runtime libraries. The npm postinstall copies or
hardlinks the native binary into the wrapper path and changes its mode. It was
not executed.

### Launcher and candidate-command effect review

The wrapper launcher resolves the platform package and spawns the native binary.
Static review of the wrapper, installer, ELF headers, dynamic dependencies, and
embedded release-artifact strings found update/autoupdate, telemetry, credentials
and cloud-provider routes/SDKs, project/user discovery, plugins, hooks, MCP,
state/migration, subprocess, and session surfaces. The artifact's `doctor` path advertises
installation, extensions, memory, hooks, updates, and permissions checks and may
fix issues. `/status` is session-oriented. No release-qualified schema or
effective-config command with per-key provenance was established.

Accordingly, the initial preview qualification reviewed but did not execute `--version`, `--help`, `doctor`, `/status`, config, or schema paths. The later bounded model-consumption probe below supersedes only that execution gap. No personal `~/.claude`, login, slash command, credential, or live provider operation was used.

### Bounded model installation, 2026-09-22

`scripts/claudecode-config-probe.sh` executes only the exact ELF hash above in
bubblewrap with a synthetic HOME/config/project, cleared environment, isolated
network/PID/IPC namespaces, hidden personal homes and sockets, and timeouts. Its
`--bare --settings <explicit-file> --print --no-session-persistence` sentinel
probe reports `claude-code:unrecognized_model` with `SENTINEL-MODEL`, then
`Not logged in`. This establishes consumption of the explicit file's model field,
not a provider request, authenticated route, successful session, or default-path
precedence. The malformed `--settings` argument is rejected. Native startup can
create state, backups, and telemetry files inside the sandbox, so this is an
opt-in bounded-write probe, not a no-write inspector. The settings file remains
byte-identical and the external sentinel is preserved.

The model-only installer uses this ELF hash as its evidence identity and replaces
only top-level `model` in strict JSON at one explicit path. Duplicate keys,
malformed JSON, non-string models, existing empty files, unsupported required
permissions/tools/resources, and non-native/non-Anthropic route bindings are
rejected. Unrelated bytes and target-owned values are preserved. Built-binary
tests verify planning, diffs, hash-bound noninteractive consent, default backup
bytes, stale rejection, no-op reapply, and disposable backup restoration. Shared
fault tests establish transaction rollback and tamper-checked recovery. This does
not promote the full-profile renderer or establish authentication, effort,
permissions, tools, instruction/skill delivery, or runtime enforcement.

### Emulated named profile installation, 2026-09-23

Claude Code `2.1.278` has no native named-profile mechanism; the probe above
only shows that `--settings <explicit-file>` consumes that file's `model`
regardless of the path chosen. `mango install <name> --target
claude-code` now uses this to write the model to a Mango-owned
`profiles/<name>.json` beside `settings.json` by default, and prints `use it:
claude --settings <path>`. `settings.json` is read for the existing
empty/malformed/permission checks above but is left byte-for-byte unchanged in
this mode. `--default` additionally patches `settings.json` with the same
byte-preserving single-field patch qualified above, so plain `claude` also
picks it up. Undo removes only the profile file (and manifest) from the install
it reverses; earlier named-profile installs are untouched. This is a
profile-mango file-placement and CLI-flag convention layered on the qualified
explicit-file consumption; it is not additional native evidence.

### Claude Code effort installation, 2026-09-25

The exact Linux x64 package above was re-downloaded and matched both recorded
SHA-256 values (tarball
`d1fb51ab0a0234d1bd7f418ee9d6b6b124c2412b2ddaf3dfc3256bad8063f1c7`, ELF
`5c4735937844e84f8a93306e841a5b0e12252909b07870f789b190468da147ab`). Its embedded
bundle declares settings `effortLevel` as `low`/`medium`/`high`/`xhigh` with an
invalid value discarded, and resolves the session default as `--effort`, then
`ultracode`, then merged settings `effortLevel`, with `CLAUDE_CODE_EFFORT_LEVEL`
and per-model `modelSettings.<model>.effortLevel` as further override surfaces.

`scripts/claudecode-config-probe.sh` now also runs the exact ELF's local
`/model` command with `--print` in the same isolation (cleared environment,
synthetic HOME/config/project, unshared network/PID/IPC, timeout). It prints the
effective effort offline, before any authentication or provider request:

| Synthetic settings | Mode | `/model` output |
| --- | --- | --- |
| `effortLevel` low, medium, high, xhigh | `--bare --settings <file>` | ``Current model: `Sonnet 4.5` (effort: <level>)`` |
| `effortLevel` max, bogus | `--bare --settings <file>` | no effort reported |
| user `settings.json` medium | default path, no `--settings` | `(effort: medium)` |
| user medium, project `.claude/settings.json` high | default path | `(effort: high)` |

A profile file produced by the built installer (`{"model": …, "effortLevel":
"medium"}`) passed through `--settings` also reported `(effort: medium)`. The
installed `2.1.281` binary (inside the tested range) gave the same results for
Sonnet 4.5, Opus 4.5, Opus 4.8 and Fable 5 models. This establishes effective
session effort selection, not provider-side reasoning use. The installer writes
`effortLevel` beside `model` only for the four accepted levels; other route
efforts are reported as not applied (`skippedRequirements`) and block with
`--strict`.

### Capability classification

| Portable property | Evidence | Applicability consequence |
| --- | --- | --- |
| Fidelity | Model-only field mapping | Partial preview fidelity |
| Native acceptance | Exact ELF consumes explicit settings-file model sentinel and `effortLevel` | Narrow model and effort installation only |
| Effective state | No merged per-key report; `/model` reports effective model and effort | Blocking beyond model and effort |
| Precedence | Explicit `--settings` path; project over user observed for effort only | Blocking outside explicit-file consumption |
| Model | Exact native sentinel consumption and disposable CLI apply | Top-level model field installable; required exact route remains unverified |
| Effort | Exact ELF offline `/model` status for low/medium/high/xhigh | `effortLevel` installable for those four levels; others not applied |
| Provider and transport | No release-qualified mapping | Blocking |
| Authentication | No credential-free identity proof | Blocking |
| Permissions | No equivalence or runtime enforcement evidence | Blocking |
| Tools | No closed-allowlist or runtime enforcement evidence | Blocking |
| `CLAUDE.md` instructions | Hierarchy and delivery unverified | Blocking |
| Skills | Discovery, precedence, and execution unverified | Blocking |
| Plugins, hooks, and MCP | Discovery, precedence, and enforcement unverified | Blocking |
| Runtime enforcement | No authorized session/provider observation | Blocking |

The full-profile renderer therefore continues to set `applicable: false`. It emits only an inert
JSON `model` candidate and digest-verified, path-addressed resource previews through the
shared render contract. It never emits credentials, provider/authentication
values, target-home files, active settings paths, or enforcement claims.

## OpenClaw v2026.9.5 evidence

### Immutable source, build inputs, and platform

The task-owned checkout verified annotated tag object
`ce2a56d4f41a756662328f4098e303014fdbb38c`, commit
`ec9c1a13db8938e5a3eaa51fca2e981cde2395a9`, and tree
`ac00d08eb2766b4fd114bee710e78a1df6c3cc03`. The deterministic source archive
`git archive --format=tar --prefix=openclaw-v2026.9.5/ HEAD` has SHA-256
`0e15e679795134cf7d488302f2bdaf0682ad4413e19a7f5c6cc22584f03d02a4`.
`package.json`, `pnpm-lock.yaml`, `scripts/build-all.mts`, and `openclaw.mjs`
have SHA-256 values `16a72a5768e629f703a89b27417feb05d4c4ece7df816c5d1364d202c5a2e619`,
`2415d6f2835843db64b1f4eadd676fa64c2fc9f5ddd868e73ae3deb2f0ed1b78`,
`65f6726bbc3b5e18f02d268458b73648da5da5a0ddfb5dd5eca6e3b9a8984e17`, and
`538e8ee2b65a0b24bb8a5ed3421bfe66621b1e0b5f726a167758c004f566fb36`.
The review platform was Linux 7.2.5-3-omarchy x86_64 with Node `v24.21.0`
(runtime binary SHA-256
`7fde7b8afa198da66257f42ee2001d874c7355631e6d1579a5fb5ef1f246df4c`) and
PNPM `10.28.2`.

The immutable checkout contained no `node_modules` or `dist/entry.js`. No
dependency install or OpenClaw build was run, so the record contains source and
build-input evidence rather than a produced runtime artifact hash.

### Source/effect review and execution decision

Review covered `openclaw.mjs`, `src/cli/program/preaction.ts`,
`src/cli/config-cli.ts`, `src/config/io.snapshot.ts`, `src/config/io.context.ts`,
`src/config/io.plugin-metadata.ts`, and
`src/state/openclaw-state-db-readonly.ts` before considering any command.
`config validate --json` calls `readConfigFileSnapshotWithPluginMetadata({
observe: false })`; that flag suppresses config observation but does not remove
dotenv/config-environment processing, `$include` resolution, plugin/workspace/index
discovery, state-database reads, or migration-capable normalization. The source
also contains recovery-capable helpers, although this handler does not pass an
explicit suspicious-recovery opt-in. The launcher also
performs Node/SQLite runtime checks before the CLI. The documented
`OPENCLAW_CONFIG_READONLY=1` switch protects `openclaw.json`, not every state or
discovery path.

`OPENCLAW_CONFIG_READONLY=1 openclaw config validate --json` was **not executed**.
The source review did not establish a bounded, credential-free, no-write native
inspector even with synthetic homes and blocked network. No OpenClaw command,
gateway, daemon, session, provider, credential, plugin, update, or native probe
was run, and the repository probe harness was not broadened.

### Capability classification

| Portable property | OpenClaw evidence | Applicability consequence |
| --- | --- | --- |
| Config syntax fidelity | Source/docs establish strict JSON5 and the model/thinking field locations | Partial candidate fidelity only; native acceptance is unverified |
| Effective state | No safe command with merged per-field provenance was established | Blocking unknown |
| Config/runtime precedence | Documented layers and overrides were not observed in an exact isolated run | Blocking unknown |
| Route and authentication | `provider/model` and thinking candidate fields are source-grounded; auth profile identity and route selection are unverified | Authentication-required mappings remain non-applicable |
| Permissions and tools | Tool policy, plugins, sender rules, and sandbox gates are documented; enforcement and bypass surfaces were not observed | Blocking unknown |
| Instructions and skills | Workspace and project/state/bundled sources are documented; delivery and precedence were not observed | Blocking unknown when requested |
| Runtime enforcement | No gateway/session/runtime observation was authorized or performed | Blocking unknown |

The adapter emits an inert JSON5 candidate at
`preview/<profile>.config.json5.preview`, resource copies under `resources/`,
and a `v1alpha1` report. It never emits credentials, auth references, active
target paths, or an applicable result. See the [OpenClaw reference](agents/openclaw.md).

## Codex CLI evidence

### Destination and precedence

A selected Codex profile is a TOML layer at `${CODEX_HOME}/<name>.config.toml`, selected with `codex --profile <name>`. The local help describes it as a layer over the base user config. The version-qualified M0 candidate destination is a standalone profile file with supported top-level keys, for example:

```toml
model_provider = "openai"
model = "<target-qualified-model>"
model_reasoning_effort = "<target-qualified-effort>"
sandbox_mode = "read-only"
approval_policy = "on-request"
```

This shape records known key locations only. It is not an applicable generated artifact until M1 proves provider/auth mapping, allowed values, project/runtime precedence, and security equivalence for the pinned binary.

For the observed build, the official loader source orders configuration from lower to higher precedence as:

1. package defaults
2. administrator-managed preferences
3. system config
4. enterprise cloud fragments
5. `${CODEX_HOME}/config.toml`
6. selected `${CODEX_HOME}/<name>.config.toml`
7. current-directory config
8. ancestor `.codex/config.toml` layers
9. repository `.codex/config.toml`
10. runtime flags and `--config` overrides

Untrusted project/repository layers may be loaded but disabled. Consequently, a generated profile cannot by itself prove the effective session policy. Higher project and runtime layers must be inspected or treated as unknown before claiming enforcement.

### Route and authentication

The profile layer can express `model_provider`, `model`, and `model_reasoning_effort`. Runtime `--model` and `--config` overrides have higher precedence. The observed CLI also exposes `--oss` and `--local-provider`, which can select a materially different route at runtime.

Authentication remains Codex-owned state. M0 does not read it, copy it, or infer it. No credential-free profile field was verified that independently proves the requested authentication mode. Therefore a canonical binding that requires an exact OAuth-versus-API-key route is **not applicable** to Codex in M0. M1 must combine a version-qualified provider mapping with a safe authentication-state check or reject the route.

### Offline inspection and effects

| Command or source | Observed effect | M0 use |
| --- | --- | --- |
| `codex --version` | Prints build version; no session or provider request | Run |
| `codex --help` | Prints CLI options, profile destination semantics, and runtime override surface | Run |
| `codex features list` | Reads the synthetic `${CODEX_HOME}/config.toml` and emits feature state; the isolated run returned in an unshared network namespace without entering a session | Run for feature parsing only |
| `codex --strict-config ...` | Validates recognized keys only as part of starting Codex; may proceed toward session initialization | Not run; not accepted as a side-effect-free M0 inspector |
| `codex doctor` | Advertised to inspect installation, config, auth, and runtime health | Not run because effects and credential access were not bounded for M0 |
| Official loader/types source | Documents layer order and supported configuration keys without reading local target state | Used as version-adjacent design evidence; local build behavior still requires M1 verification |

No credential-free command that prints the complete effective config with field provenance was established for the observed binary. This is an explicit unknown, not inferred success.

### Isolated 2026-09-21 UTC probe

The retained [`scripts/agent-config-probe.sh`](../../scripts/agent-config-probe.sh)
ran the direct `codex` executable in a bubblewrap network namespace with
synthetic `HOME`, `XDG_CONFIG_HOME`, `CODEX_HOME`, an empty project, a cleared
environment, and a bounded ten-second timeout. It did not run `exec`, `doctor`,
`login`, `app-server`, `sandbox`, or any session-starting command.

Reproduction command, with direct binary paths supplied by the operator:

```bash
PROFILE_MANGO_CODEX_BIN=/absolute/path/to/codex-0.154.0 \
PROFILE_MANGO_JCODE_BIN=/absolute/path/to/jcode-fork-ca8017a3a \
PROFILE_MANGO_PROBE_ROOT=/task-owned-scratch/ap-3kw \
PROFILE_MANGO_PROBE_TIMEOUT_SECONDS=10 \
./scripts/agent-config-probe.sh
```

The same command is exercised by the opt-in `integration_test.go` test
`TestAgentConfigProbe`; the checked-in script is the deterministic fixture
revision for this evidence.

The exact tested build was `codex-cli 0.154.0`, SHA-256
`3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022`. Before
running any Codex target command, the probe hashes the resolved direct
executable and fails closed if this exact hash does not match. The
positive/negative feature sentinels established that the binary consumed the
synthetic config layer:

- no feature file: `apps` resolved to `true`;
- synthetic `[features] apps = false`: `apps` resolved to `false`;
- the same synthetic `apps = false` layer with `--enable apps`: `apps` resolved
  to `true`;
- the same synthetic `apps = false` layer with `--disable apps`: `apps` resolved
  to `false`;
- malformed synthetic TOML: `codex features list` rejected it with a TOML parse
  error.

This is native feature-config parsing and feature-runtime-precedence evidence
only. It does not establish that Codex consumes or enforces profile route,
model, reasoning-effort, sandbox, approval, tool, skill, instruction, or
authentication fields. The effective configuration and field-provenance
inspector remains unavailable, and `--strict-config` remains excluded because
it participates in startup.

The exact-version capability result is therefore:

| Capability | Evidence result | Applicability consequence |
| --- | --- | --- |
| Route provider/model/effort | The help surface advertises route-related options, but no safe command established consumption or effective values for these fields | Unverified and blocking |
| Authentication identity | No credential-free state or route identity was observed | Unverified and blocking |
| Precedence | Feature-file versus runtime feature overrides were observed; profile, project, and route precedence were not | Partial evidence only, blocking for canonical route claims |
| Permissions/tools | Help advertises sandbox and approval controls, but no runtime enforcement check was safe to run | Unverified and blocking |
| Instruction/skill delivery | No safe native delivery or provenance inspector was established without starting a session | Unverified and blocking |

The probe prints these field-specific results as
`codex-route-provider-model-effort=unverified`,
`codex-authentication-identity=unverified`,
`codex-precedence=feature-runtime-override-observed-route-project-unverified`,
`codex-permissions-tools=enforcement-unverified`, and
`codex-instruction-skill-delivery=unverified`. These are explicit gaps, not
inferred support claims. The adapter therefore remains non-applicable.

## OpenCode v1.18.31 native config and preview-renderer evidence

The official immutable GitHub release `v1.18.31`, published 2026-09-14, points to
commit `a97622c801f4ca571530ddc51076af659a9c32cd`. The tag archive has SHA-256
`76f69fe27ec2b44e23fa1749029e7c012eb7e975a0f0c7819e9458198dfd3896`, and the
release metadata records SHA-256
`e9312be75ed803b7415fc2aeabda1f4fe938912a39673762dc0c38c0e11ebde4` for the
Linux x64 release archive. The extracted direct binary has SHA-256
`f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11`.

Pinned source review covered the configuration loader and v1 config/provider
schemas. Their SHA-256 values are:

| Item | SHA-256 |
| --- | --- |
| `packages/opencode/src/config/config.ts` | `87a9071af1ddb04d65947dba49be3fb4c94ecf63e120f17ce46ee81ff25dd45a` |
| `packages/core/src/v1/config/config.ts` | `b99bcbd98df6da79e59cda482363f759cea9d4b9792c6c8e83b6a8d686138d30` |
| `packages/core/src/v1/config/provider.ts` | `c496dea619e8e9d2d09b7a2353f2e62bff6a2276471c161c3988059a1b6ac303` |
| mutable `https://opencode.ai/config.json` snapshot | `e8cb6e287a3852ee3403f4803be5ad6b19db94948037eaa9672125c333427922` |

The exact source establishes JSON/JSONC configuration, the `model` field in
`provider/model` form, provider/model option surfaces, and merged global, custom,
project, `.opencode`, inline, and managed configuration layers. Mutable official
documentation separately identifies `~/.config/opencode/opencode.json`,
`OPENCODE_CONFIG`, project `opencode.json`, and the target-owned credential store
`~/.local/share/opencode/auth.json`. Mutable documentation is context, not release
proof.

With explicit user approval under `ap-6fu.11`, the exact release binary was run
through `scripts/opencode-config-probe.sh` in a task-owned bubblewrap sandbox. The
probe cleared the environment, isolated every HOME/XDG/config/data/cache/state,
project, managed-config, database, and temporary path, supplied empty auth content,
disabled model fetching, auto-update, pruning, default plugins, external plugins,
project config, and LSP downloads, blocked network access, and bounded commands
with timeouts. It never started the TUI, a prompt, a provider session, or a live
agent home.

The first native probe rejected the generated candidate because its unquoted
`model` property was not valid JSONC. The renderer was corrected to emit
`"model"`. The repeated exact-binary probe then established:

- `--version` returned `1.18.31` and the direct binary matched the pinned hash;
- the deterministic Profile Mango candidate was accepted by `debug config`;
- merged output contained `"model": "openai/gpt-5.6"`;
- inline content overrode a global sentinel model;
- an explicit `OPENCODE_CONFIG` candidate file was consumed;
- the target inserted `$schema` into that file, proving the inspector is writeful;
- malformed JSONC was rejected;
- an unknown key was accepted and omitted from resolved output;
- target-owned scratch directories, logs, locks, `.gitignore`, and metadata were
  created; and
- an independent backup was restored after testing, with recursive type, mode,
  size, and SHA-256 inventory matching the pre-test state byte-for-byte.

This is exact-version native parser acceptance and partial merged-state/precedence
evidence for the model candidate. It does not establish general per-field
provenance, authentication identity, effort, transport, instruction or skill
delivery, permissions, tools, plugins, MCP, runtime enforcement, or safe Profile
Mango installation. OpenCode remains excluded from production installation and
`--all` applicability. No installed or global OpenCode state was read or modified.

## Codex preview-renderer evidence

The first adapter boundary is pinned to `codex-cli 0.154.0` and binary SHA-256
`3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022`.
Golden and negative tests demonstrate deterministic candidate TOML, resource
copies, report metadata, no-write behavior without `--preview`, atomic writes to
a new explicit staging directory with `--preview`, and rejection of unsupported
versions, invalid resources, permissions, and closed tool requirements.

This is preview-rendering support only. Every report remains `applicable: false`,
and the command returns nonzero while authentication, delivery, precedence, or
enforcement is unverified. The preview never contains active `config.toml`,
`AGENTS.md`, active skill destinations, credentials, launch recipes, or target-home
paths.

### Codex 0.154.0 settings-only installation, 2026-09-22

This is a separate, bounded installer, not a promotion of the inert preview.
Only root `model_provider = "openai"`, a non-empty safe `model`, and
`model_reasoning_effort = "high"` are patched at one explicit config path (the
effort gate was widened on 2026-09-25, see
[below](#codex-01540-reasoning-effort-values-2026-09-25)).
Permissions, tools, instructions, skills, non-native transport, non-OpenAI
providers, non-OAuth bindings, other efforts, malformed/duplicate TOML, and
ambiguous profile/provider shadow state block. The installer does not write
authentication settings or read a credential store.

Retained local qualification at `.worktrees/reports/ap-6fu.14/qualification-milestone-20260922.md`
used exact `rust-v0.154.0` source commit
`6b9826e3aa83b1a5947db50f4332cb9c65f1b340` (archive SHA-256
`848c7ffac62e21b14edc2048d2e5b7c82b31c556afa4b0d305da0f7d5aa794f4`).
Its resolver consumed actual `PatchConfig` output: effective OpenAI provider
ID/object, model `gpt-5.6`, and effort `high`. An unknown-provider control
failed, and omitted model/effort controls yielded absent values. Filtered Cargo
and direct compiled test runs each passed exactly one test. The release lockfile
required metadata-only normalization of 150 workspace version strings in a
separate copy; no external dependency or edge changed. The test bypassed
requirements.toml and ambient layers, so it does not prove installed-binary
equivalence, model availability, active profile/project/runtime precedence,
authentication identity, or runtime enforcement.

The rebuilt profile-mango CLI planned without writing, then applied with
`--override --apply --yes --expect-plan` and default backup against synthetic
TOML. Config SHA-256 changed from
`1e1a9c8dd11c8120885c38985a1c02d02bfeb4735088ef4f30de3e78637cb90c`
to `f2d8fc2f623194d6f3f2b6a5b9d6c08cd7c2c1e2212039f6479d6aaf4a9eba66`;
the backup retained the original digest. Target-owned
`forced_login_method = "api"`, an unknown key, feature table, and comment remained unchanged. A
second built plan and matching apply both reported `noop`. A read-only
permission requirement remained blocked even with apply flags. Synthetic
`auth.json` and outside-target sentinel SHA-256 values stayed unchanged at
`011e02a527fc678e665c87d60d899e3ac72be7ffcf4bd289ff400e0d44f686b3`
and `4d0646b302547417167fe19f380c26f501c6fe912e959ec788ccbdd5eb0d63d2`.
Internal transaction tests cover stale plans and synthetic guarded restoration.
The subsequently landed guarded public restore command reverses an eligible
committed two-file install transaction at an explicit path with a fresh plan hash.
Its built-binary disposable acceptance is recorded separately in
`.worktrees/reports/ap-6fu.14/`; it does not verify Codex runtime state.

On 2026-09-23, a separate offline probe fed the public installer's synthetic
root config to the direct installed `codex-cli 0.154.0` binary (SHA-256
`3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022`).
Its app-server `config/read` returned `openai`, `gpt-5.6`, and `high`, all with
user-config origins. An untrusted project override was ignored with a warning;
after explicit synthetic trust, project model/effort `project-model`/`medium`
won. Session `-c` overrides won with `runtime-model`/`minimal`. The app-server
rejects `--profile`; neither named runtime-profile selection nor live model use
was exercised. This endpoint starts auth/config/cloud, plugin and SQLite
machinery, so it was confined to an empty-auth, network/PID/IPC-isolated sandbox,
not used as a personal-home inspector. Retained requirement-to-check evidence:
`.worktrees/reports/ap-6fu.14/native-qualification-20260923.md`.

Separately authorized Codex runtime diagnostics on 2026-09-23 inspected only
read-only authentication metadata and mounted the confirmed credential file
read-only inside disposable state. The stored `chatgpt` mode and pinned-binary
ChatGPT login status were observed, but neither proves that profile-mango
enforces an exact OAuth route. Two bounded provider-capable requests timed out;
the first retained no partial events and the second retained no JSON events or
usage. Model availability, delivery, effort enforcement, and the cause of either
timeout remain unknown. Later credential-free, network-blocked cold-start
probes sometimes emitted no event, while instrumented runs reached
`thread.started`; no common blocking stage was identified. State-DB migration
counts and zero thread rows do not locate the stall because a successful
`--ephemeral` run also had zero persisted thread rows. These diagnostics do not
alter the settings-only install claim and authorize no further provider call.
Retained sanitized evidence: `.worktrees/reports/ap-6fu.14/live-qualification-20260923.md`
and `.worktrees/reports/ap-6fu.14/cold-start-stages-20260923.md`.

`codex login status` distinguishes stored API-key from ChatGPT login, but the
same ChatGPT message covers both Codex-managed OAuth and externally supplied
ChatGPT tokens in the pinned source. It is advice for a user, not a credential
read or OAuth proof by profile-mango. Human and JSON plans warn that successful
installation applies **only to three settings**, never the OAuth-required full
profile, delivery, policy enforcement, or effective runtime route.

### Codex 0.154.0 named-profile installation, 2026-09-23

Named profiles in Codex `0.154.0` are separate files, not `[profiles.<name>]`
tables. Exact release source `rust-v0.154.0` (commit
`6b9826e3aa83b1a5947db50f4332cb9c65f1b340`, archive SHA-256
`848c7ffac62e21b14edc2048d2e5b7c82b31c556afa4b0d305da0f7d5aa794f4`) shows:

- `utils/cli/src/shared_options.rs`: `--profile`/`-p` "Layer
  `$CODEX_HOME/<name>.config.toml` on top of the base user config"; names are
  ASCII letters, digits, `_` and `-` (`protocol/src/config_types.rs`
  `ProfileV2Name`).
- `config/src/loader/mod.rs`: the profile file is pushed as a second user layer
  above `config.toml` and merged into the same `ConfigToml`, so `model_provider`,
  `model` and `model_reasoning_effort` have the same meaning there as at the root.
  `--profile <name>` is a hard error while `config.toml` holds a legacy
  `profile = "<name>"` or `[profiles.<name>]`; unrelated legacy tables are allowed.
- `core/src/config/mod.rs`: any root `profile = "..."` is a hard startup error
  ("no longer supported; use `--profile`").
- `cli/src/main.rs`: `--profile` applies only to runtime commands, `codex mcp`,
  `codex sandbox`, and `codex debug prompt-input`.

The design recorded in the plan for ap-vjc.1 (`[profiles.<name>]` in
`config.toml`, and `--default` writing root `profile = "<name>"`) would therefore
have produced a profile Codex refuses to select and a config Codex refuses to
start with. The installer instead writes `$CODEX_HOME/<name>.config.toml`,
leaves `config.toml` byte-for-byte unchanged, and with `--default` writes the
same three root settings into `config.toml`. It blocks a root `profile` key, a
same-name legacy `[profiles.<name>]` table, and openai provider shadow state in
`config.toml`; the unchanged `config.toml` is also a stale-plan check at apply.

Installed-binary controls ran the direct `codex-cli 0.154.0` executable
(SHA-256 `3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022`,
verified before every run) under bubblewrap with unshared network, PID and IPC
namespaces, a cleared environment, synthetic `HOME`/`XDG_CONFIG_HOME`/
`CODEX_HOME`, no auth file, and a 20-second timeout. The only command was
`codex [--profile <name>] debug prompt-input hi`, which builds the model-visible
prompt offline (ephemeral, no state database, no provider request) and prints it.
Results:

| Synthetic state | Command | Result |
| --- | --- | --- |
| `coding.config.toml` with `model_provider = "no-such-provider"` | `--profile coding` | exit 1, ``Model provider `no-such-provider` not found`` |
| same | no `--profile` | exit 0 |
| root `model_provider = "no-such-provider"`, profile `model_provider = "openai"` | `--profile coding` | exit 0 (profile provider wins) |
| same | no `--profile` | exit 1, provider not found |
| profile `model = 42` | `--profile coding` | exit 1, ``coding.config.toml:1:9: invalid type: integer `42`, expected a string`` |
| profile `model_reasoning_effort = 42` | `--profile coding` | exit 1, same typed error at `1:26` |
| both controls above | no `--profile` | exit 0 |
| `[profiles.coding]` in `config.toml` | `--profile coding` | exit 1, legacy-table refusal naming `coding.config.toml` |
| root `profile = "coding"` | no `--profile` | exit 1, legacy `profile` no longer supported |
| unrelated `[profiles.dev]` plus `coding.config.toml` | `--profile coding` | exit 0 |
| profile-mango-generated `coding.config.toml`/`review.config.toml` (below) | `--profile coding`, `--profile review`, none | exit 0 each |
| generated files, `coding` provider edited to `no-such-provider` | `--profile coding` / `--profile review` | exit 1 provider not found / exit 0 |
| generated files | `--profile missing` | exit 0: a missing profile file is treated as empty |

This proves that the pinned binary selects the generated file with `--profile`
and consumes its `model_provider` (value-sensitive, overriding the root) and its
typed `model` and `model_reasoning_effort` keys. `debug prompt-input` does not
print the effective model or effort, so positive effective values for those two
come from the loader source plus the earlier root `config/read` observation, not
from this run. A bogus effort string was accepted, so effort values are not
validated at load. OAuth, model availability and runtime delivery were not
exercised.

The built CLI (branch `agent/vjc1-codex-profiles`) ran with a cleared
environment and synthetic `HOME`, `CODEX_HOME` and `PROFILE_MANGO_HOME`:
`install coding --target codex` and then `install review --target codex` each
planned `use it: codex --profile <name>`, created `<name>.config.toml`, and left
a `config.toml` with a comment, root model/effort, a feature table and an MCP
table at its original SHA-256. The JSON plan carried
`install: {mode: named-profile, profileName, useCommand}`. Reinstalling either
profile planned `noop`. `undo --target codex` removed only `review.config.toml`
and restored the manifest; a second undo removed `coding.config.toml` and the
manifest; `config.toml` stayed unchanged throughout. `install coding --default`
then also adopted `config.toml` with a backup and wrote the three root settings.

### Codex 0.154.0 reasoning-effort values, 2026-09-25

Exact release source `rust-v0.154.0` (tag object
`36eab01061df3cde5f95ec20a526777b430091ba`, commit
`6b9826e3aa83b1a5947db50f4332cb9c65f1b340`) file
`codex-rs/protocol/src/openai_models.rs` (SHA-256
`2e9923d405a497441a0b264efc07de6ce21cdb108442e660a8b9fb63ca415aed`) defines
`ReasoningEffort::from_str`: `none`, `minimal`, `low`, `medium`, `high`, `xhigh`,
`max`, `ultra` and `persistent` parse as named variants, an empty string is an
error, and any other string becomes `Custom`. The same enum types
`model_reasoning_effort`, so every value loads; the isolated installed-binary
`config/read` observation above already reported root `high`, trusted-project
`medium`, and session `minimal` as effective values.

The installer gate now accepts `none`, `minimal`, `low`, `medium`, `high` and
`xhigh`; `max`, `ultra` and other efforts still block with an explicit error
listing the accepted values, instead of being written or dropped. The built-CLI
sandbox test `TestInstallCodexAppliesMediumEffortInSandbox` plans and applies
`medium` with `--default` and observes `model_reasoning_effort = "medium"` in
both `config.toml` and `<name>.config.toml`. Whether a given model accepts a
level remains model-dependent and was not exercised; no provider call was made.

## Oh My Pi v18.2.6 source and preview-renderer evidence

The following records the initial preview qualification. The later installer
qualification below supersedes the missing-addon blocker only for two fields.

The task-owned source checkout is pinned to tag `v18.2.6` and commit
`78b753124d11f8dd3ae73e2524125890ff7c977e`. Frozen dependency installation used
Bun `1.3.14` and the pinned `bun.lock`. The direct source entrypoint
`packages/coding-agent/src/cli.ts --version` printed `omp/18.2.6` with an
isolated task-owned `HOME` and sanitized environment.

The exact hashes used by the observation and adapter evidence are:

| Item | SHA-256 |
| --- | --- |
| Bun `1.3.14` runtime | `9fd36f87e4b90b07632b987a2e4ec81ca15a62c81bf983190cea6d715be2ad74` |
| `packages/coding-agent/package.json` | `4d9558530fdd8c76798181545d7cde8b558731596515b2f300e61c8d403bcb6e` |
| pinned `bun.lock` | `b74fbff79c5acbacc6bc46f4180e070cb5edb9853c8c5a00ff97d9c8734d4a70` |

The adapter uses the package manifest hash as `EvidenceSHA256` and labels the
evidence `source-entrypoint-version`. It does not present that source hash as a
standalone Oh My Pi binary hash.

The official pinned build command was attempted only after source review and
frozen dependency installation:

```bash
bun run packages/coding-agent/scripts/build-binary.ts
```

The build generated intermediate client assets, then stopped at native embedding:

```text
No native addons found for linux-x64.
Expected pi_natives.linux-x64-modern.node / pi_natives.linux-x64-baseline.node
```

The exact standalone artifact was therefore not produced. No mutable wrapper,
global tool install, release download, alternate version, or native-addon
substitution was used.

### Config command source effects

Before any target execution, source review covered the pinned command and settings
paths:

- `packages/coding-agent/src/cli.ts` bypasses network bootstrap for `--version`.
- `packages/coding-agent/src/cli/config-cli.ts` initializes `Settings` before
  handling `config path` or `config list --json`.
- `packages/coding-agent/src/config/settings.ts` opens agent storage and loads
  global, project, and overlay configuration during initialization.
- `packages/coding-agent/src/config/config-file.ts` contains legacy JSON migration
  that can write a YAML settings file.

`config path` and `config list --json` were not run. The source-reviewed storage,
discovery, and migration effects are not accepted as a side-effect-free inspector
under the M0 safety boundary. No provider, credential, session, hook, extension,
child, personal home, or target-home operation was invoked.

### Capability result

| Capability | Evidence result | Applicability consequence |
| --- | --- | --- |
| Exact source version | `omp/18.2.6` printed from the pinned source entrypoint | Version-qualified source observation only |
| Standalone artifact | Official build stopped because the pinned linux-x64 native addon was missing | Blocking |
| Config inspection | `config path/list` source path initializes storage, discovery, and migration | Blocking, command not run |
| Route provider/model/effort | Pinned docs and source identify `modelRoles` and `defaultThinkingLevel` syntax | Candidate syntax only, blocking |
| Authentication identity | No credential-free route proof was established | Blocking |
| Permissions/tools | Approval and tool policy concepts are documented, but equivalence and enforcement were not tested | Blocking |
| Instruction/skill delivery | Context-file and skill precedence is documented, but no safe native delivery observation exists | Blocking |

The adapter consequently emits only deterministic inert YAML candidates and
digest-verified, path-addressed resource copies that preserve original relative paths.
Reports remain `applicable: false`, include
the source, build, and config blockers above, and return nonzero. It never writes
`~/.omp`, active config destinations, credentials, launch recipes, or target-home
paths.

### Narrow Oh My Pi installer qualification, 2026-09-22

Exact source `78b753124d11f8dd3ae73e2524125890ff7c977e`, the Bun hash above,
and `nightly-2026-08-12` produced the native addon through upstream
`scripts/bazel-natives.ts host --dest packages/natives/native` with networking
disabled. The addon SHA-256 is
`9632a05bc6460c65b7fcbe8653c44cb757217a91502ab7faeb0d5241be70372e`.
The toolchain manifest hash is
`38824251984d44a1d1eb4bbb109074b6d0e3213d6f82d9538622033a5bc60455`.
An older worker note's different Bun hash (`951ee2...`) is unreconciled metadata,
not equivalent evidence and not the runtime used by the successful qualification.

Actual compiled CLI plan/apply output was passed to exact
`Settings.loadReadOnly({agentDir, cwd})`. `getModelRole("default")` returned
`synthetic/native-proof` and `get("defaultThinkingLevel")` returned `xhigh`.
Control input without the fields returned no model role and `high`. An unmanaged
review role and synthetic unknown-key sentinel remained unchanged.
The root run removed the worker's broad read-only `/etc` mount and used cleared
environment, isolated namespaces/network, dropped capabilities, hidden personal
homes/sockets, read-only exact source/runtime, disposable state/cache and timeout.
Content/mode inventories showed no target writes during native loading.

The compiled CLI checks demonstrated deterministic read-only plans, wrong-hash
and stale-plan zero writes, lossless application, default byte-equal backup and
no-op reapply. Shared transaction tests separately establish fault rollback and
hash-guarded recovery. There is no public restore CLI command.
Only `modelRoles.default` and `defaultThinkingLevel` are installable at an explicit or
documented default path. Full-profile previews remain non-applicable. This evidence does not prove
model/provider resolution, authentication, precedence, delivery or enforcement.
See the [target reference](agents/oh-my-pi.md) for the bounded patch contract.

## Hermes Agent v0.21.3 evidence

### Immutable source, build inputs, and platform

The task-owned checkout verified annotated tag object
`7a963716b81be13ba513d4f127633b7da493aff2`, commit
`345cd2b057a452236de401d3534b8502a7465e8d`, and tree
`6e14b9791cdc5a47068685e9429dd5d6bdc5ef5f`. The deterministic source archive
`git archive --format=tar --prefix=hermes-agent-v2026.9.14/ HEAD` has SHA-256
`71f2db39a64fbba282e3bd3be4b0f7b935585948a59a368d61deeec0f0827c47`.
`pyproject.toml`, `uv.lock`, `setup.py`, `hermes_cli/main.py`, and
`hermes_cli/__init__.py` have SHA-256 values
`a674c321c63c3bfd9fa099fab5957a64092416f7771680b370a4d77e992744ba`,
`4426ffd292c32cd8edda5779db49951833e87c73cab7cee845e7afe92ecb17b9`,
`d476dd1c28d707acd72f1d0551d7178123832ae18b6e201ca1ac48cd040653ae`,
`d21135792593599715c194999a97621cb86b09180b19e51db58d1910f1bdabf3`, and
`0d78a58a9f27f32adfdac959e89767cecde93a424bcfbd39d64fb95f6cf13e6c`.

The annotated tag message and both source version declarations identify Hermes
Agent `v0.21.3`, release date `2026.9.14`. The source requires Python
`>=3.11,<3.14`. The review platform was Linux 7.2.5-3-omarchy x86_64 on
Omarchy 4.0.4. The active system runtime was Python `3.14.7`, binary SHA-256
`d78f9cf7178ecff09963551399855543c297f37ac207e626228bfe43cb26a70c`, outside
the source requirement. A compatible mise-managed Python `3.12.13` was present
but not activated. No Hermes dependency install or runtime artifact build was run.

### Source/effect review and execution decision

Review covered the `hermes` launcher, `hermes_cli/main.py`, `config.py`,
`config_defaults.py`, `env_loader.py`, `hermes_constants.py`, `profile_cmd.py`,
`profiles.py`, `status.py`, `status_auth.py`, `auth.py`,
`agent/prompt_builder.py`, and `agent/agent_init.py` before considering any
command. The source review established these effects:

- The launcher imports the main CLI, loads `HERMES_HOME/.env` and the project
  `.env`, parses effective configuration, configures file logging, and may
  create or secure Hermes-home directories before dispatch.
- `hermes config get <key> --json` dispatches to `get_config_value`, which reads
  `.env` keys directly or calls `load_config` for YAML values. Config loading
  can normalize, cache, back up malformed YAML, and perform home/path setup.
- `hermes status` reads `.env`, merged config, provider/auth status, `auth.json`
  through provider helpers, plugin/platform discovery, gateway/PID state,
  cron files, and session databases. `status --deep` can make an HTTP request
  when an OpenRouter key is present and probes a local TCP socket.
- `hermes profile show` resolves a profile under `HERMES_HOME`, reads its
  `config.yaml`, gateway state, distribution metadata, aliases, skill count,
  `.env`, and `SOUL.md` presence. Profile helpers also contain subprocess and
  service-management paths that were not part of a bounded inspector contract.
- Context and skill construction reads `SOUL.md`, `.hermes.md`/`HERMES.md`,
  root-to-current-directory `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, and
  `HERMES_HOME/skills`. Auth helpers resolve profile/global `auth.json` paths,
  and dotenv loading can evaluate configured external secret sources.

No bounded, credential-free, no-write execution was established even with
synthetic `HERMES_HOME` and project paths, sanitized environment, blocked
network, and a timeout. No Hermes command was run. Smart approvals were not
used because they invoke an auxiliary model and do not cover file writes. No
session, provider, credential, profile mutation, update, telemetry, hook,
plugin, subprocess, target-home, or personal Hermes state was accessed. The
repository probe harness was not broadened.

### Capability classification

| Portable property | Hermes Agent v0.21.3 evidence | M0 applicability consequence |
| --- | --- | --- |
| Config syntax fidelity | Source and release docs identify YAML `model.provider`, `model.default`, and `agent.reasoning_effort` fields | Partial inert candidate fidelity; native parser acceptance is unverified |
| Effective state | `config get` loads merged config without a safe per-field provenance contract; `status` also reads state and auth surfaces | Blocking unknown |
| Config/runtime precedence | Docs describe CLI, YAML, dotenv, defaults, and managed policy layers, but no exact isolated observation was safe | Blocking unknown |
| Provider/model/effort | Candidate fields are source-grounded; provider resolution, fallback selection, and effort use are not observed | Syntax partial; effective route remains blocking |
| Authentication | `auth.json`, dotenv, provider environment, and OAuth paths are target-owned; no credential-free route identity was established | Authentication-required mappings are non-applicable |
| Permissions/tools | `toolsets`, terminal backends, approvals, plugins, and command policies are documented, but closed allowlists, network/shell behavior, and bypasses were not observed | Blocking unknown |
| Instructions/context | Hermes discovers SOUL and project context files with target-owned precedence and threat scanning | Delivery and precedence remain blocking |
| Skills | Skills primarily live under `HERMES_HOME/skills` and are discovered into the prompt/tool surface | Delivery, rediscovery, and enforcement remain blocking |
| Runtime enforcement | No Hermes runtime artifact or session was built or launched | Blocking unknown |

The Hermes adapter emits only a deterministic inert YAML candidate and
digest-verified, path-addressed resource copies that preserve original relative paths.
The candidate uses no credential, auth,
memory, session, profile, or active target-home path. Reports remain
`applicable: false`, include target-owned diagnostics, and return nonzero.


## Experimental Jcode fork evidence

All profile commands, profile fixtures, and profile observations in this section
refer only to the Jcode fork. They do not describe upstream Jcode or
establish compatibility with any independently distributed Jcode build.

### Destination and precedence

The Jcode fork defines named profiles as entries under
`[profiles.<name>]` in `~/.jcode/config.toml`, selected with the custom-fork
`jcode --profile <name>` flag or its TUI profile picker. The bundled wrapper
guide states that these custom-fork profiles supply provider, model, reasoning,
tool, skill, and additional-instruction defaults without editing configuration
or affecting another session. The observed fork's generated default-config
documentation defines this secret-free destination shape:

```toml
[profiles.example]
provider = "<provider-id>"
model = "<model-id>"
reasoning_effort = "<none|minimal|low|medium|high|xhigh|max>"
tool_profile = "<full|acp|minimal|lite|none>"
tools = ["<tool-name>"]
disabled_tools = ["<tool-name>"]
skills_mode = "<all|allowlist|none>"
skills = ["<installed-skill-name>"]
disabled_skills = ["<installed-skill-name>"]
instructions = "<additive guidance>"
agents_md_path = "<path relative to JCODE_HOME>"
```

`provider_profile` may select an existing `[providers.<name>]` entry, but portable output must never create, copy, or embed credentials in that entry.

Explicit CLI flags and environment overrides are documented to retain higher
precedence than the selected custom-fork profile. The custom-fork `profile
current` and `profile resolve` commands report effective non-secret policy and
field sources. Restored sessions retain a credential-free resolved snapshot and
warn on missing or changed profiles rather than silently adopting a different
named profile. Child agents and swarm workers inherit effective restrictions
unless explicitly overridden.

Because explicit child overrides are possible, a root profile is not an organizational enforcement boundary. A portable policy requiring restrictions on all descendants must remain non-applicable unless child override behavior is separately constrained and verified.

### Route and authentication

The Jcode fork CLI distinguishes provider routes such as `openai` and
`openai-api`, and supports explicit provider, model, and reasoning-effort
selection. That distinction is promising for exact route mapping, but it is
custom-fork behavior and no local profile values or credential state were read
during M0.

The Jcode fork `jcode profile show`, `current`, and `resolve` commands
are documented to avoid credential exposure and provider initialization. M0 did
not run them against the developer's live profiles because doing so would read personal
target configuration. The isolated probe below uses only synthetic custom-fork
configuration before making a route observation.

### Offline inspection and effects

| Command | Documented effect | M0 use |
| --- | --- | --- |
| Jcode fork: `jcode --version` | Prints build version | Run |
| Jcode fork: `jcode profile --help` | Prints profile inspection and override surface | Run |
| Jcode fork: `jcode --quiet profile list --json` | Lists configured names without provider initialization | Run only in synthetic config |
| Jcode fork: `jcode --quiet profile show NAME --json` | Shows safe configured fields and instruction presence/length | Run only in synthetic config |
| Jcode fork: `jcode --quiet profile current --json` | Shows current effective policy and explicit no-profile state | Run only in synthetic config |
| Jcode fork: `jcode --quiet profile resolve NAME --json` | Resolves effective tools, skills, and field sources without mutating a session | Run only in synthetic config |

The first adapter experiment must use an isolated synthetic config and a private/new
server socket where required. It must not reuse the developer's live custom-fork config or
shared daemon as a fixture. These profile commands remain custom-fork-only
evidence and are not upstream Jcode evidence.

### Isolated 2026-09-21 UTC custom-fork profile probe

The retained [`scripts/agent-config-probe.sh`](../../scripts/agent-config-probe.sh)
ran only the developer's direct custom-fork `jcode` executable with synthetic
`HOME/.jcode/config.toml`, `XDG_CONFIG_HOME`, an empty project, a private socket,
`--no-update`, `--no-selfdev`, `JCODE_NO_TELEMETRY=1`, a cleared environment,
an unshared network namespace, and a bounded ten-second timeout. These flags and
environment values request update/telemetry suppression, but do not prove that
the target made no attempted telemetry operation. The fixture was not copied
from or compared with an upstream Jcode installation.

The exact tested build was `jcode v0.83.909-dev (ca8017a3a)`, SHA-256
`392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992`. The
custom-fork `profile list`, `show`, and `resolve` commands consumed a synthetic
`[profiles.sentinel]` entry and returned distinct positive sentinels for:

- provider `openai-api`, model `sentinel-model-positive`, and effort `low`, with
  `Profile` sources in the resolved output;
- tool profile `none`, allowed `read`, and disabled `write`;
- skill mode `none`, selected and disabled skill names, and an empty effective
  skill set;
- additive instruction presence and character count without returning the
  instruction content;
- an explicit synthetic missing `agents_md_path` warning rather than a read of
  the repository or a personal `AGENTS.md`.

The custom-fork tool-profile override `--tool-profile minimal` changed the
resolved tool profile while preserving the profile's route values. This probe did
not use the profile inspector to generalize provider/model/effort flag
precedence, so those override paths remain unverified.

Negative custom-fork profile fixtures produced these version-qualified results:

- malformed TOML was rejected with a TOML parse error;
- invalid `reasoning_effort` was rejected with the allowed-value diagnostic;
- an unknown `unknown_key` was accepted but omitted from the resolved output.

The last result is an explicit compatibility and safety gap: this current Jcode fork
custom-fork profile parser is not demonstrated to reject unknown profile keys.
Do not treat it as upstream Jcode behavior or as evidence of fail-closed
validation for a future adapter.

### Experimental inert adapter boundary

The `pkg/adapters/jcodefork` package is pinned to target identity
`jcode-fork`, tested build `jcode v0.83.909-dev (ca8017a3a)`, commit
`ca8017a3a`, and SHA-256
`392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992`. It emits
only a deterministic inert TOML preview under `preview/` and keeps
`applicable: false` unconditionally. The report, diagnostic, and candidate
comments identify it as the **Jcode fork, experimental-only**.

Candidate syntax is limited to the exact-build-observed provider, model,
reasoning effort, a `none` tool profile paired with canonical closed allow and
deny selectors, empty-skill mode, canonical skill selectors, and instruction
presence/character-count metadata. Resource bytes remain separate inert,
digest-verified artifacts addressed by their original relative paths. The adapter does not emit authentication,
provider profiles, credentials, `agents_md_path`, arbitrary target keys,
non-empty target skill modes, or skill exclusions because the canonical input
does not express those fields with exact-build evidence.

The target parser accepted and omitted an unknown profile key in the retained
probe. The adapter therefore uses an explicit projection allowlist and rejects
unsupported canonical policy shapes rather than passing through unknown target
fields. Adapter tests and the retained probe regression assertions cover this
boundary. No new native Jcode probe ran for this adapter because the exact
retained binary artifact was not re-verified in this worktree; the current
session Jcode binary and personal target state were not inspected.

The source documentation snapshot `ed9b93b894454619f73ccddd24c2ff7a3c98ddc3`
remains separate from the tested build and is not evidence of upstream Jcode
compatibility. Authentication, discovery, delivery, precedence, child
overrides, hooks, extensions, MCP, and runtime enforcement remain blocking.

## Portable-field classification

`fidelity` describes whether the target has a corresponding concept. `delivery` describes whether M0 has an approved deterministic destination. `enforcement` describes what has actually been demonstrated. `unknown` and `unsupported` block applicability when the field is required.

| Portable field | Codex 0.154.0 | Jcode fork 0.83.909-dev (ca8017a3a) | M0 applicability consequence |
| --- | --- | --- | --- |
| `spec.routeRef` provider/model/effort | **Partial fidelity.** Profile keys exist for provider, model, and effort; runtime overrides are higher precedence. Exact auth mode is unverified. | **Observed profile fields, not auth proof.** Synthetic custom-fork `profile resolve` reported provider `openai-api`, model, effort, and `Profile` sources. No credential or provider state was read. | Any binding requiring verified authentication mode is non-applicable. No silent route fallback. |
| `permissions.mode` | **Partial concept.** Sandbox and approval fields exist, but read-only target behavior and all escape surfaces were not verified. | **Profile tool policy observed only.** Custom-fork tool-profile values resolve, but a general read-only filesystem/process guarantee was not verified. | Required read-only policy is non-applicable. |
| `permissions.network` | **Partial/unknown.** Network policy concepts exist in current source, but local-build destination and enforcement were not verified. | **Unknown.** No version-qualified profile-level network enforcement was established. | Any required network allow/deny value is non-applicable. |
| `permissions.shell` | **Partial concept.** Sandbox/approval affect shell execution but are not proven equivalent to a shell deny. | **Partial concept.** Tool allow/deny can hide shell tools, but hooks, extensions, MCP, and children were not exhaustively bounded. | Required shell deny is non-applicable. |
| `tools.allow` / `tools.deny` | **Partial fidelity.** Tool configuration exists, but native tool expansion and closed-allowlist semantics were not verified. | **Observed profile concept.** Synthetic custom-fork resolution reported explicit tools, disabled tools, and tool-profile precedence; runtime enforcement and child override boundaries remain unverified. | M0 resolves canonical deny-wins semantics but emits no target artifact. M1 must prove exact expansion. |
| `instructions.append` | **Partial delivery.** Codex has instruction/config layers, but deterministic profile-local file projection and precedence were not verified. | **Observed profile metadata.** Custom-fork `profile show/resolve` reported instruction presence and character count without exposing content; delivery and full precedence remain unverified. | Canonical resolution is supported; target delivery remains M1 work. |
| `skills.include` / `skills.exclude` | **Partial fidelity.** Current config supports skill enable/disable selectors, but project discovery and rediscovery can affect effective availability. | **Observed custom-fork profile selectors.** Skill mode, selected skills, disabled skills, and effective empty mode resolved from synthetic config; discovery and enforcement remain unverified. | Required absence of a skill cannot yet be claimed as enforcement. |
| `spec.extends` | **Compiler-owned.** No native target inheritance is required. | **Compiler-owned.** No native target inheritance is required. | M0 fully resolves one parent before any adapter sees the profile. |

No candidate field is classified as runtime-enforced by M0. The route/instruction-only fixture is valid as a **canonical resolution** fixture with security explicitly unmanaged, not as proof that either target can safely load the resulting policy. The constrained read-only and intentionally unsupported fixtures must remain non-applicable until an adapter supplies version-qualified evidence.

## M0 conclusion

The minimum contract can proceed without public Jcode support and without pretending Codex is already supported. M0 establishes deterministic canonical semantics and records that:

- Codex is the first public M1 adapter candidate, and Claude Code, Pi, Oh My Pi,
  OpenClaw, and Hermes now have exact source-qualified inert preview renderers.
- the Jcode fork is a private experimental comparison target only.
- exact authentication-route proof is unresolved for each target without isolated target inspection
- security-sensitive fields remain non-applicable unless M1 proves equivalent or stronger enforcement
- no target artifact is applied, no launch recipe is emitted, and no live-state mutation is part of M0

This satisfies the M0 evidence requirement by classifying unknowns and unsupported mappings rather than filling them from memory or weakening the portable policy.

## Evidence references

- Local binaries: direct `codex --version`, `codex --help`, `codex features list`,
  and Jcode fork `jcode --version`, `jcode profile --help`, and custom-
  fork `jcode profile list/show/resolve` on 2026-09-20 local / 2026-09-21 UTC.
- Reproducible harness: [`scripts/agent-config-probe.sh`](../../scripts/agent-config-probe.sh)
  and its opt-in integration test `TestAgentConfigProbe`.
- OpenAI Codex source documentation: configuration loader order and `ConfigToml` types in [`openai/codex`](https://github.com/openai/codex/tree/main/codex-rs/config/src).
- Pi v0.86.1 source and documentation: [`coding-agent` source at the exact commit](https://github.com/earendil-works/pi/tree/13cbf77df2396303013a41646bcfa77b4271ae56/packages/coding-agent), [`settings.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/settings.md), [`providers.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/providers.md), [`models.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/models.md), [`security.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/security.md), [`skills.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/skills.md), and [npm package metadata](https://registry.npmjs.org/@earendil-works%2fpi-coding-agent/0.86.1). Retrieved source/package hashes and the no-execution decision are retained in the worker qualification record.
- Oh My Pi v18.2.6 source and documentation: [`coding-agent` source](https://github.com/can1357/oh-my-pi/tree/v18.2.6/packages/coding-agent), [`settings.md`](https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/settings.md), [`models.md`](https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/models.md), and [`approval-mode.md`](https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/approval-mode.md). The local source review did not execute config inspection because initialization effects were not bounded.
- OpenClaw v2026.9.5 source and documentation: [`openclaw.mjs`](https://github.com/openclaw/openclaw/blob/v2026.9.5/openclaw.mjs), [`config-cli.ts`](https://github.com/openclaw/openclaw/blob/v2026.9.5/src/cli/config-cli.ts), [`io.snapshot.ts`](https://github.com/openclaw/openclaw/blob/v2026.9.5/src/config/io.snapshot.ts), [`configuration.md`](https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/configuration.md), and [`config.md`](https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/cli/config.md). The exact source review did not execute config inspection because plugin/state/include and migration effects were not bounded.
- Bundled Jcode fork documentation for source snapshot `ed9b93b89`:
  `README.md` named-session-profile section and `docs/WRAPPERS.md` profile
  inspection section. This source snapshot is separate from the tested build
  `ca8017a3a`; neither is evidence for upstream Jcode.
- Design gate: `openclaw-wpii`, `Unified coding-agent profiles: support matrix and product architecture`, retained in the developer.s OpenClaw research workspace (2026-09-20 design gate document). The Bead ID is the durable cross-repository reference.

## Deferred checks

- Codex feature parsing is now observed in an isolated synthetic `CODEX_HOME`,
  but route/effective-config inspection remains unavailable without a safe
  inspector.
- Jcode fork `profile show/current/resolve` was run only against an
  isolated synthetic config and private socket. Do not port these profile
  commands or results to upstream Jcode.
- Do not read live agent homes, credentials, sessions, global configuration values, or provider endpoints.
- Do not launch target sessions, hooks, extensions, children, or provider requests for M0 evidence.
- Claude Code's v2.1.278 package and Pi's v0.86.1 source/package release are
  qualified for inert preview boundaries, but native acceptance, effective-state
  inspection, and enforcement remain blocked pending safe exact-artifact
  inspection paths. OpenClaw and Hermes retain their source-qualified inert
  boundaries.
- OpenCode `1.18.31` native model, single-skill and named-definition consumption
  are qualified below. Complete precedence and provenance, authentication
  identity, effort, permissions, tools, plugins, MCP, runtime delegation, and
  enforcement remain deferred.
- Oh My Pi remains non-applicable. Requalify the missing native addon and perform
  a new isolated config probe before making any native support claim.
- The placeholder `profilemango.dev` schema identifier is not a claim that the domain is registered or controlled. Rename it before public release if ownership is unavailable.


## OpenCode single-skill installation qualification, 2026-09-22

Exact `1.18.31` binary SHA-256
`f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11`
was requalified with `scripts/opencode-skills-probe.sh`. Its isolated `debug skill`
path accepted `skills.paths`, listed the synthetic skill, and loaded its body.
The probe uses cleared environment, allowlisted runtime mounts, hidden real homes,
network/PID/IPC isolation, dropped capabilities, and a timeout. No model call,
agent session, TUI, credential store, or personal configuration was used.
Disposable backup restoration matched the original byte inventory.

The installer may copy exactly one validated portable skill to `SKILL.md` beside
an explicit config and add that directory to `skills.paths`. Existing config
bytes and unrelated skills members are preserved. Unowned or externally edited
skill resources remain protected even with config `--override`. Compiled-CLI
checks cover actual config/resource application, backup, idempotent replan,
stale-source rejection, and unowned-resource preservation. Directory discovery
may include other skills and is not an exclusive allowlist. Multi-skill profiles,
main-config instructions, authenticated route identity, policy enforcement, and
full-profile runtime behavior remain blocked. A later omission removes only a
clean owned resource and a Mango-introduced discovery path; ambiguous legacy
paths are preserved with a warning. No-op matching bytes never adopt unowned files.

## OpenCode named primary/subagent definition qualification, 2026-09-22

Compiled Mango generated two distinct `agents/<name>.md` definitions in
disposable state using explicit `--agent opencode@1.18.31=primary:mango-review`
and `subagent:mango-research`. SHA-256 of the actual installed definitions:
`1a91ae593c0eb78167948df77f83271cd391c0ade7fc9666d043251f1a6f33c5`
and `9a6fe91855defd1f71a74642bea6b29d97031a39e9f5e47c2b5dca542e0384eb`.
The exact OpenCode binary `f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11`
resolved both name, mode, route model and instruction body via `debug agent name`.
Changing mode, model or prompt changed the observed native result. Removing or
renaming the definition prevented its named resolution. This is actual
installer-generated artifact consumption, not only handwritten feasibility.

The opt-in probe used cleared environment, synthetic home/XDG/config/auth,
empty `/etc`, allowlisted read-only runtime and binary mounts, writable synthetic
state only, network/PID/IPC isolation, dropped capabilities and a 20-second
timeout. Candidate and definition bytes remained unchanged, an unrelated
sentinel was preserved, and restored state matched its pre-probe inventory.
No `--tool`, provider/model request, TUI or session was used. Plan IDs include
destination and mode/name; only unchanged Mango-owned whole definitions can be
updated, regardless of `--override`. Primary selection, subagent delegation,
authentication, higher-precedence target state, policy enforcement, and runtime
behavior were not executed or inferred. Other targets have no qualified named
instruction/subagent installer; their required fields remain blocked.

## OpenCode default-install named primary agent, 2026-09-23 (ap-vjc.5)

An install without `--agent` now reuses the qualified primary-agent renderer
above (same `AgentDefinition` code path, same `mode: primary` frontmatter and
model/instructions rendering) at `agents/<profile>.md` beside the resolved
config, and reports it through the shared named-profile install mode already
qualified for Codex (`docs/dev/target-evidence.md` Codex sections): the plan
carries `install.mode: named-profile` and a `use it: opencode --agent
<profile>` command, and `opencode.json`/`opencode.jsonc` is left byte-identical
unless `--default` is also passed. This reuses the exact-binary evidence
recorded above for the primary definition; no new native OpenCode probe was
run for this change. `--default` additionally reuses the qualified top-level
`model` patch and one-skill install exactly as documented under "OpenCode
single-skill installation qualification, 2026-09-22", now gated behind that
flag instead of being the unconditional default, since OpenCode's
`skills.paths` is a directory-wide setting rather than a per-agent one.
Compiled-CLI checks cover: default install creates the agent file only,
`--default` also patches the model and skill fields, undo removes only the
agent file and manifest it created, and an edited or unowned agent file at the
default path is protected exactly as an explicit `--agent` destination.

## OpenCode agent effort variant, 2026-09-25

OpenCode `1.18.31` exposes effort per agent only as `variant`. Exact source at
commit `a97622c801f4ca571530ddc51076af659a9c32cd`:
`packages/core/src/v1/config/agent.ts` (SHA-256
`4f2d7bc8283ff0c8cd922c614a74f74abb1abe9d5fc2a6dba3dd03d2dca5c624`) declares the
agent `variant`; `packages/opencode/src/session/prompt.ts` (SHA-256
`f0c5bc64c0f0e966693d4a57f7ede1e9d6e188b396152f04b55303dc75b9b768`) uses it only
when the prompt uses the agent's own model and that model has a variant of that
name; `packages/opencode/src/provider/transform.ts` (SHA-256
`c07d49e48dd2478ad2813a10805781a72551db2fd847b7df994cc854bf654c16`) names
built-in reasoning variants only from `none`, `minimal`, `low`, `medium`, `high`,
`xhigh`, `max` (model- and provider-dependent), mapped to provider options such
as `reasoningEffort`. The named agent renderer now emits
`variant: "<effort>"` for those names; other efforts are listed as not applied.

The opt-in `TestOpenCodeGeneratedAgentNativeProbe`, rerun against the exact
binary (`f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11`, verified before running) in the same isolation,
resolved `variant: "high"` from both generated primary and subagent definitions;
`changed variant` (`low`) and `missing variant` controls changed or removed the
resolved value, and state restored to its baseline inventory. A missing variant
on the model silently falls back to the model default at runtime, and the
`--default` main-config model carries no effort; plans warn about both. No
provider request was made.

## Hermes bounded installation qualification, 2026-09-22

Exact Hermes `0.21.3` source release `v2026.9.14`, commit
`345cd2b057a452236de401d3534b8502a7465e8d`, consumed generated installer YAML via
native `hermes_cli.config.load_config_readonly()`. Config module SHA-256
`d76471ce54d40e68165e2cce7c2ade9c2164ed5ce4dbcf673b1b289cb89c7d84`
and version module SHA-256
`0d78a58a9f27f32adfdac959e89767cecde93a424bcfbd39d64fb95f6cf13e6c`
are verified before and inside the probe. Python 3.12.13 runs under cleared
environment, network/PID/IPC isolation, dropped capabilities, allowlisted runtime
mounts, hidden personal homes/sockets, plugin-directory overlay, traps, and timeout.
The generated `model.provider`, `model.default`, and `agent.reasoning_effort`
values were observed, with unrelated synthetic state preserved. Root reran this
probe after hardening and verified actual compiled CLI plan/apply/backups/noop/
stale rejection/disposable restoration. This is config parsing and effective
merge evidence only. Full startup, authenticated route, provider execution,
delivery, permissions/tools, sessions, hooks, plugins, MCP, and enforcement remain
unqualified. The ordinary installer never imports or launches Hermes.


## OpenClaw bounded installation qualification, 2026-09-22

Exact `2026.9.5` commit `ec9c1a13db8938e5a3eaa51fca2e981cde2395a9`, tree
`ac00d08eb2766b4fd114bee710e78a1df6c3cc03`, was mounted read-only with hash-gated
imported source modules. Native JSON5 parser, core validator, model-primary
getters/resolvers and `resolveConfiguredThinkingDefaultCore` consumed generated
model/thinking settings. Global values, per-agent overrides, explicit fallbacks,
and user-override fallback disabling were checked. Inherited fallback availability
is distinct from effective fallback projection, and an agent primary without
fallbacks does not inherit global fallbacks.

Root reran exact-source consumption under Node 24.21.0 in a cleared-environment,
network/PID/IPC-isolated sandbox with dropped capabilities, allowlisted mounts,
hidden personal homes/sockets and timeout. The independent rerun removed the
worker's broad `/etc` mount. A second rerun consumed actual compiled-CLI applied
output, then verified no-op and restored the disposable config bytes. Default
backups, preservation and stale rejection also pass compiled integration tests.
This is source-native field consumption, not full startup, authentication,
provider identity, sessions, policy or runtime enforcement. Only two defaults are
promoted. Credentials, provider options, fallback editing, resources, permissions,
tools, hooks, plugins, MCP and other required capabilities remain blocked.

The independent codeload gzip SHA-256
`3bd4d9617308808704cfdcbbdfdf87335d2f8efa4a34be49210aa34eca816841`
differs from retained archive SHA-256
`0e15e679795134cf7d488302f2bdaf0682ad4413e19a7f5c6cc22584f03d02a4`.
Neither replaces the other. Exact peeled Git identity and actual imported hashes,
recorded in the target's native evidence fixture, underpin this qualification.

### OpenClaw 2026.9.5 named-profile installation, 2026-09-23

Source-only evidence from the pinned commit
`ec9c1a13db8938e5a3eaa51fca2e981cde2395a9`, fetched read-only. `src/entry.ts`
applies root `--profile` before any command runs. `src/cli/profile.ts`
`applyCliProfileEnv` sets `OPENCLAW_CONFIG_PATH=<stateDir>/openclaw.json`.
`src/cli/profile-utils.ts` `resolveProfileStateDir` returns
`<home>/.openclaw-<name>`, or `<home>/.openclaw` for `default`. `<home>` is
`OPENCLAW_HOME` or the OS home. The installer writes the qualified fields to that
file and changes the main config only with `--default`. A main config path not
shaped `<home>/.openclaw/openclaw.json` blocks named install
(`install.named_profile_path_unsafe`). The profile named `default` patches the
main config in place. No native `--profile` probe was run: the only local binary
is `2026.9.4`. Unit, command, and compiled-binary tests plus a scratch-`HOME`
sandbox run cover create, default-config byte identity, `--default`, and undo.
File and line references are in
[openclaw.md](agents/openclaw.md#named-profile-installation-2026-09-23).

### Hermes 0.21.3 named-profile installation, 2026-09-23

Source-only evidence from the pinned commit
`345cd2b057a452236de401d3534b8502a7465e8d` (`v2026.9.14`), fetched read-only.
`hermes_cli/profiles.py` `get_profile_dir`/`resolve_profile_env` map
`hermes -p <name>` to `<hermes-home>/profiles/<name>/config.yaml`, where
`<hermes-home>` is the directory holding the main `config.yaml` (default
`~/.hermes`); `"default"` (any case) resolves to that root directory itself,
not `profiles/default`. `hermes_cli/main.py` `_apply_profile_override` pre-parses
`-p`/`--profile <name>` and requires `resolve_profile_env` to find an existing,
non-tombstoned `profiles/<name>` directory before it sets `HERMES_HOME`;
`hermes -p <name>` refuses to start otherwise (`FileNotFoundError`) rather than
creating it. The installer writes the qualified fields to
`profiles/<name>/config.yaml` (creating that directory) and changes the main
config only with `--default`. Unlike OpenClaw, no fixed directory-name check is
needed: the profile path is always `profiles/<name>` below whatever directory
holds the resolved `config.yaml`. Reserved ids (`hermes`, `test`, `tmp`,
`root`, `sudo`) and ids outside `[a-z0-9][a-z0-9_-]{0,63}` block named install
(`install.named_profile_path_unsafe`), matching what `hermes -p <name>` itself
refuses. The profile named `default` patches the main config in place. No
native `hermes -p`/`hermes profile` probe was run: the only local binary is
`0.19.0`. Unit and command tests plus a scratch-`HOME` sandbox run cover
create, default-config byte identity, `--default`, the `default` name,
reserved-name blocking, and undo. File and line references are in
[hermes.md](agents/hermes.md#named-profile-installation-2026-09-23).

## Default config destinations, 2026-09-22

When `install` receives a target without `--config`, it plans against that
target's documented user-level config file. The plan's human output shows
`config: <path> (default)` and JSON reports `config.source`; JSON omits absolute
paths. An explicit `--config` always overrides the default. Writes keep the plan
diff, create-only backups, hash/drift checks, adjacent ownership manifests, and
interactive or `--yes --expect-plan` consent. Resolution reads only `HOME`,
`XDG_CONFIG_HOME`, and the documented relocation variables below, and stats only the
OpenCode candidate files. It does not read auth stores, sessions, plugins, MCP, or
providers. Relative relocation values block with a reason.

| Target | Default path | Relocation | Documentation source |
| --- | --- | --- | --- |
| Claude Code `2.1.278` | `~/.claude/settings.json`; emulated named profiles go to `profiles/<name>.json` in the same folder | none | [claude-code.md](agents/claude-code.md#emulated-named-profiles) |
| Codex `0.154.0` | `$CODEX_HOME/config.toml`, default `~/.codex/config.toml`; named profiles go to `<name>.config.toml` in the same folder | `CODEX_HOME` | [codex.md](agents/codex.md#configuration-and-precedence) |
| OpenCode `1.18.31` | `${XDG_CONFIG_HOME:-~/.config}/opencode/opencode.json`; `opencode.jsonc` when only it exists; both present blocks | `XDG_CONFIG_HOME` | [opencode.md](agents/opencode.md#configuration-paths-syntax-and-precedence) |
| Pi `0.86.1` | `~/.pi/agent/settings.json` | `PI_CODING_AGENT_DIR` | [pi.md](agents/pi.md#configuration-and-precedence) |
| Oh My Pi `18.2.6` | `~/.omp/agent/config.yml` | none | [oh-my-pi.md](agents/oh-my-pi.md#configuration-and-precedence) |
| OpenClaw `2026.9.5` | `~/.openclaw/openclaw.json`; named profiles go to `~/.openclaw-<name>/openclaw.json`, and a relocated main config blocks named install | `OPENCLAW_CONFIG_PATH` | [openclaw.md](agents/openclaw.md#configuration-and-precedence) |
| Hermes `0.21.3` | `~/.hermes/config.yaml`; named profiles go to `<config-dir>/profiles/<name>/config.yaml` | `HERMES_HOME` | [hermes.md](agents/hermes.md#configuration-and-precedence) |

These paths are documentation-derived destinations, not new native evidence.
Field consumption remains qualified only as recorded in each target section; for
example, Claude Code consumption was observed through an explicit `--settings`
file, and project, managed, or runtime layers can still shadow a user-level value.
OpenCode named `--agent` definitions and adapters without a documented default
still require an explicit `--config`. Tests isolate `HOME`, `XDG_CONFIG_HOME`, and
the relocation variables in a temporary sandbox and refuse to run if any default
resolves outside it.
