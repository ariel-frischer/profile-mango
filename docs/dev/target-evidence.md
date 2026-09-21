# Target evidence ledger

**Evidence date:** 2026-09-20 local / 2026-09-21 UTC
**Platform:** Linux x86_64
**Scope:** M0 contract evidence plus isolated native-inspector evidence. No adapter,
rendering, application, target-home read, credential read, target session launch,
provider/model launch or request, provider initialization, hook/extension/child
launch, or live-state mutation. The only target processes executed were bounded
`--version`, `--help`, Codex `features list`, and Ariel custom-fork profile
inspection commands in synthetic homes; they did not start a session or initialize
a provider.

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

Codex is the first intended public adapter candidate. Ariel's custom Jcode fork
is **not** a supported public target. It remains here only as experimental
developer evidence because that fork currently provides the closest comparison
surface for named policy profiles.

Ariel's custom Jcode fork must not appear in the README support list, release
promise, or public compatibility matrix. Public support for any Jcode variant
would require a stable external distribution, versioned public documentation,
reproducible fixtures, and evidence independent of Ariel's machine.

## Pinned local observations

| Target | Observed build | Evidence source | M0 status |
| --- | --- | --- | --- |
| Codex CLI | `codex-cli 0.154.0` | local `codex --version` and `codex --help`; official `/openai/codex` configuration loader and types | First public M1 adapter candidate; no adapter ships in M0 |
| Ariel custom Jcode fork | `jcode v0.83.909-dev (ca8017a3a)` | isolated direct-binary probe, local `jcode --version` and `jcode profile --help`; bundled `README.md` and `docs/WRAPPERS.md` | Experimental developer comparison only |

The observations are version-qualified snapshots, not compatibility ranges.

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
`3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022`. The
positive/negative feature sentinels established that the binary consumed the
synthetic config layer:

- no feature file: `apps` resolved to `true`;
- synthetic `[features] apps = false`: `apps` resolved to `false`;
- malformed synthetic TOML: `codex features list` rejected it with a TOML parse
  error.

This is native feature-config parsing evidence only. It does not establish that
Codex consumes or enforces profile route, model, reasoning-effort, sandbox,
approval, tool, skill, instruction, or authentication fields. The effective
configuration and field-provenance inspector remains unavailable, and
`--strict-config` remains excluded because it participates in startup.

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

- Codex is the first public M1 adapter candidate.
- Ariel's custom Jcode fork is a private experimental comparison target only.
- exact authentication-route proof is unresolved for both targets without isolated target inspection
- security-sensitive fields remain non-applicable unless M1 proves equivalent or stronger enforcement
- no target artifact, launch recipe, or live-state mutation is part of M0

This satisfies the M0 evidence requirement by classifying unknowns and unsupported mappings rather than filling them from memory or weakening the portable policy.

## Evidence references

- Local binaries: direct `codex --version`, `codex --help`, `codex features list`,
  and Ariel custom-fork `jcode --version`, `jcode profile --help`, and custom-
  fork `jcode profile list/show/resolve` on 2026-09-20 local / 2026-09-21 UTC.
- Reproducible harness: [`scripts/agent-config-probe.sh`](../../scripts/agent-config-probe.sh)
  and its opt-in integration test `TestAgentConfigProbe`.
- OpenAI Codex source documentation: configuration loader order and `ConfigToml` types in [`openai/codex`](https://github.com/openai/codex/tree/main/codex-rs/config/src).
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
- Claude Code, Pi, Oh My Pi, OpenClaw, and Hermes remain roadmap research, not M0 targets.
- The placeholder `profilemango.dev` schema identifier is not a claim that the domain is registered or controlled. Rename it before public release if ownership is unavailable.
