# Target evidence ledger

**Evidence dates:** initial 2026-09-20 local / 2026-09-21 UTC; installation updates 2026-09-22 UTC
**Platform:** Linux x86_64
**Scope:** Canonical contract, shared transaction, exact-version renderer, and
narrow installation evidence. All native qualification and install validation use
synthetic disposable state, not personal agent homes or authenticated providers.
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
exact-version renderers. OpenCode and Claude Code currently permit only model-field
installation; Pi permits only its three route-default settings. Other target gates
remain blocked unless explicitly qualified below. Field consumption, full-route
authentication, precedence, delivery, and runtime enforcement are separate claims.
Native probes require approved disposable scope and exact evidence. Ariel's custom
Jcode fork is experimental developer evidence, not a supported public target.

Ariel's custom Jcode fork must not appear in the README support list, release
promise, or public compatibility matrix. Public support for any Jcode variant
would require a stable external distribution, versioned public documentation,
reproducible fixtures, and evidence independent of Ariel's machine.

## Pinned local observations

| Target | Observed build | Evidence source | Current status |
| --- | --- | --- | --- |
| OpenCode | release `v1.18.31` at `a97622c801f4ca571530ddc51076af659a9c32cd` | immutable GitHub release/source/archive/extracted-binary hashes, isolated network-blocked `debug config` probe, lossless JSONC patch tests, transactional disposable-path apply/reapply/recovery checks, backup/restore inventory, and deterministic inert renderer tests | Top-level `model` field installable at one explicit path with destination-bound consent and transactional safeguards; auth identity, provider options, full precedence/provenance, delivery, permissions, tools, plugins, MCP, and enforcement remain blocked |
| Codex CLI | `codex-cli 0.154.0` | isolated native parsing probe, exact binary SHA-256, golden preview rendering, and negative applicability tests | Inert preview renderer only; native applicability remains blocked |
| Oh My Pi | source `omp/18.2.6` at `78b753124d11f8dd3ae73e2524125890ff7c977e` | exact source review, direct `--version`, frozen lockfile, source manifest/runtime hashes, and deterministic inert preview tests | Inert preview renderer only; standalone native artifact and applicability remain blocked |
| Claude Code | npm `2.1.278`, release commit `bf7d404e26a5fb6167d21b46c93a2bf6c22ab274` | immutable wrapper/native package tarballs, registry integrity, extracted ELF hash, release tag verification, static launcher/command review, and deterministic inert JSON tests; no native command executed | Inert preview renderer only; native acceptance, effective state, precedence, route/auth, permissions/tools, instruction/skill delivery, and enforcement remain blocked |
| OpenClaw | source `v2026.9.5` at `ec9c1a13db8938e5a3eaa51fca2e981cde2395a9` | immutable source archive/build-input hashes, entrypoint/config effect review, and deterministic inert JSON5 preview tests; no native command executed | Inert preview renderer only; runtime artifact, native config acceptance, effective state, delivery, precedence, auth, and enforcement remain blocked |
| Hermes Agent | source `v2026.9.14` at `345cd2b057a452236de401d3534b8502a7465e8d` (`v0.21.3`) | immutable source archive/build metadata hashes, entrypoint/config/status/profile effect review, and deterministic inert YAML preview tests; no native command executed | Inert preview renderer only; runtime artifact, native config acceptance, effective state, delivery, precedence, auth, permissions, tools, skills, context, and enforcement remain blocked |
| Pi | source tag `v0.86.1` at `13cbf77df2396303013a41646bcfa77b4271ae56`, package `@earendil-works/pi-coding-agent@0.86.1` | immutable source archives and npm package/integrity hashes, bundled entrypoint identity, static startup/config effect review, and deterministic inert JSON tests; no native command executed | Inert preview renderer only; native config acceptance, effective state, precedence, auth, delivery, extensions, permissions/tools, and enforcement remain blocked |
| Ariel custom Jcode fork | `jcode v0.83.909-dev (ca8017a3a)` | isolated synthetic `profile list/show/resolve` probe, exact binary SHA-256, retained golden/negative adapter tests, and bundled `README.md`/`docs/WRAPPERS.md` | Experimental-only inert preview renderer; native applicability remains blocked |

The observations are version-qualified snapshots, not compatibility ranges.

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

### Capability classification

| Portable property | Evidence | Applicability consequence |
| --- | --- | --- |
| Fidelity | Model-only field mapping | Partial preview fidelity |
| Native acceptance | Exact ELF consumes explicit settings-file model sentinel | Narrow model-field installation only |
| Effective state | No merged per-key report | Blocking |
| Precedence | Explicit `--settings` path only; default and override hierarchy unverified | Blocking outside explicit-file consumption |
| Model | Exact native sentinel consumption and disposable CLI apply | Top-level model field installable; required exact route remains unverified |
| Effort | No release-qualified mapping | Blocking |
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
PROFILE_MANGO_JCODE_BIN=/absolute/path/to/ariel-custom-jcode-ca8017a3a \
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

## Oh My Pi v18.2.6 source and preview-renderer evidence

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


## Experimental Ariel custom Jcode fork evidence

All profile commands, profile fixtures, and profile observations in this section
refer only to Ariel's custom Jcode fork. They do not describe upstream Jcode or
establish compatibility with any independently distributed Jcode build.

### Destination and precedence

The Ariel custom Jcode fork defines named profiles as entries under
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

The Ariel custom-fork CLI distinguishes provider routes such as `openai` and
`openai-api`, and supports explicit provider, model, and reasoning-effort
selection. That distinction is promising for exact route mapping, but it is
custom-fork behavior and no local profile values or credential state were read
during M0.

The Ariel custom-fork `jcode profile show`, `current`, and `resolve` commands
are documented to avoid credential exposure and provider initialization. M0 did
not run them against Ariel's live profiles because doing so would read personal
target configuration. The isolated probe below uses only synthetic custom-fork
configuration before making a route observation.

### Offline inspection and effects

| Command | Documented effect | M0 use |
| --- | --- | --- |
| Ariel custom fork: `jcode --version` | Prints build version | Run |
| Ariel custom fork: `jcode profile --help` | Prints profile inspection and override surface | Run |
| Ariel custom fork: `jcode --quiet profile list --json` | Lists configured names without provider initialization | Run only in synthetic config |
| Ariel custom fork: `jcode --quiet profile show NAME --json` | Shows safe configured fields and instruction presence/length | Run only in synthetic config |
| Ariel custom fork: `jcode --quiet profile current --json` | Shows current effective policy and explicit no-profile state | Run only in synthetic config |
| Ariel custom fork: `jcode --quiet profile resolve NAME --json` | Resolves effective tools, skills, and field sources without mutating a session | Run only in synthetic config |

The first adapter experiment must use an isolated synthetic config and a private/new
server socket where required. It must not reuse Ariel's live custom-fork config or
shared daemon as a fixture. These profile commands remain custom-fork-only
evidence and are not upstream Jcode evidence.

### Isolated 2026-09-21 UTC custom-fork profile probe

The retained [`scripts/agent-config-probe.sh`](../../scripts/agent-config-probe.sh)
ran only Ariel's direct custom-fork `jcode` executable with synthetic
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

The last result is an explicit compatibility and safety gap: this current Ariel
custom-fork profile parser is not demonstrated to reject unknown profile keys.
Do not treat it as upstream Jcode behavior or as evidence of fail-closed
validation for a future adapter.

### Experimental inert adapter boundary

The `pkg/adapters/arieljcode` package is pinned to target identity
`ariel-jcode`, tested build `jcode v0.83.909-dev (ca8017a3a)`, commit
`ca8017a3a`, and SHA-256
`392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992`. It emits
only a deterministic inert TOML preview under `preview/` and keeps
`applicable: false` unconditionally. The report, diagnostic, and candidate
comments identify it as the **Ariel custom Jcode fork, experimental-only**.

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

| Portable field | Codex 0.154.0 | Ariel custom Jcode fork 0.83.909-dev (ca8017a3a) | M0 applicability consequence |
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
- Ariel's custom Jcode fork is a private experimental comparison target only.
- exact authentication-route proof is unresolved for each target without isolated target inspection
- security-sensitive fields remain non-applicable unless M1 proves equivalent or stronger enforcement
- no target artifact is applied, no launch recipe is emitted, and no live-state mutation is part of M0

This satisfies the M0 evidence requirement by classifying unknowns and unsupported mappings rather than filling them from memory or weakening the portable policy.

## Evidence references

- Local binaries: direct `codex --version`, `codex --help`, `codex features list`,
  and Ariel custom-fork `jcode --version`, `jcode profile --help`, and custom-
  fork `jcode profile list/show/resolve` on 2026-09-20 local / 2026-09-21 UTC.
- Reproducible harness: [`scripts/agent-config-probe.sh`](../../scripts/agent-config-probe.sh)
  and its opt-in integration test `TestAgentConfigProbe`.
- OpenAI Codex source documentation: configuration loader order and `ConfigToml` types in [`openai/codex`](https://github.com/openai/codex/tree/main/codex-rs/config/src).
- Pi v0.86.1 source and documentation: [`coding-agent` source at the exact commit](https://github.com/earendil-works/pi/tree/13cbf77df2396303013a41646bcfa77b4271ae56/packages/coding-agent), [`settings.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/settings.md), [`providers.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/providers.md), [`models.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/models.md), [`security.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/security.md), [`skills.md`](https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/skills.md), and [npm package metadata](https://registry.npmjs.org/@earendil-works%2fpi-coding-agent/0.86.1). Retrieved source/package hashes and the no-execution decision are retained in the worker qualification record.
- Oh My Pi v18.2.6 source and documentation: [`coding-agent` source](https://github.com/can1357/oh-my-pi/tree/v18.2.6/packages/coding-agent), [`settings.md`](https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/settings.md), [`models.md`](https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/models.md), and [`approval-mode.md`](https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/approval-mode.md). The local source review did not execute config inspection because initialization effects were not bounded.
- OpenClaw v2026.9.5 source and documentation: [`openclaw.mjs`](https://github.com/openclaw/openclaw/blob/v2026.9.5/openclaw.mjs), [`config-cli.ts`](https://github.com/openclaw/openclaw/blob/v2026.9.5/src/cli/config-cli.ts), [`io.snapshot.ts`](https://github.com/openclaw/openclaw/blob/v2026.9.5/src/config/io.snapshot.ts), [`configuration.md`](https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/configuration.md), and [`config.md`](https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/cli/config.md). The exact source review did not execute config inspection because plugin/state/include and migration effects were not bounded.
- Bundled Ariel custom Jcode fork documentation for source snapshot `ed9b93b89`:
  `README.md` named-session-profile section and `docs/WRAPPERS.md` profile
  inspection section. This source snapshot is separate from the tested build
  `ca8017a3a`; neither is evidence for upstream Jcode.
- Design gate: `openclaw-wpii`, `Unified coding-agent profiles: support matrix and product architecture`, retained in Ariel's OpenClaw workspace at `/home/ari/.openclaw/docs/research/general/2026-09-20-unified-coding-profile-mangos.md`. The Bead ID is the durable cross-repository reference.

## Deferred checks

- Codex feature parsing is now observed in an isolated synthetic `CODEX_HOME`,
  but route/effective-config inspection remains unavailable without a safe
  inspector.
- Ariel custom-fork `profile show/current/resolve` was run only against an
  isolated synthetic config and private socket. Do not port these profile
  commands or results to upstream Jcode.
- Do not read live agent homes, credentials, sessions, global configuration values, or provider endpoints.
- Do not launch target sessions, hooks, extensions, children, or provider requests for M0 evidence.
- Claude Code's v2.1.278 package and Pi's v0.86.1 source/package release are
  qualified for inert preview boundaries, but native acceptance, effective-state
  inspection, and enforcement remain blocked pending safe exact-artifact
  inspection paths. OpenClaw and Hermes retain their source-qualified inert
  boundaries.
- OpenCode `1.18.31` native model-candidate parsing and partial merged-state
  behavior are observed. Production patch preservation, complete precedence and
  provenance, authentication identity, effort, delivery, permissions, tools,
  plugins, MCP, and runtime enforcement remain deferred.
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
instructions, authenticated route identity, policy enforcement, and full-profile
runtime behavior remain blocked.


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
