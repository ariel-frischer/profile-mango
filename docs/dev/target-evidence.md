# Target evidence ledger

**Evidence date:** 2026-09-20
**Platform:** Linux x86_64
**Scope:** M0 contract evidence only. No adapter, rendering, application, target-home read, credential read, target process launch, or provider request.

M0 freezes the portable contract. It does not claim that either target enforces that contract yet. Target claims use three separately reported levels:

1. schema, golden, and negative fixtures
2. offline native parsing or effective-config inspection in an isolated test home
3. explicitly authorized runtime observation

A lower evidence level never implies a higher one. Parsing is not runtime enforcement, and instruction text is not a permission boundary.

## Product boundary

Codex is the first intended public adapter candidate. Jcode is **not** a supported public target. It remains here only as experimental developer evidence for Ariel's custom fork, because that fork currently provides the closest comparison surface for named policy profiles.

Jcode must not appear in the README support list, release promise, or public compatibility matrix. Public Jcode support would require a stable external distribution, versioned public documentation, reproducible fixtures, and evidence independent of Ariel's machine.

## Pinned local observations

| Target | Observed build | Evidence source | M0 status |
| --- | --- | --- | --- |
| Codex CLI | `codex-cli 0.154.0` | local `codex --version` and `codex --help`; official `/openai/codex` configuration loader and types | First public M1 adapter candidate; no adapter ships in M0 |
| Jcode custom fork | `jcode v0.83.907-dev (ed9b93b89)` | local `jcode --version` and `jcode profile --help`; bundled `README.md` and `docs/WRAPPERS.md` | Experimental developer comparison only |

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
| `codex --strict-config ...` | Validates recognized keys only as part of starting Codex; may proceed toward session initialization | Not run; not accepted as a side-effect-free M0 inspector |
| `codex doctor` | Advertised to inspect installation, config, auth, and runtime health | Not run because effects and credential access were not bounded for M0 |
| Official loader/types source | Documents layer order and supported configuration keys without reading local target state | Used as version-adjacent design evidence; local build behavior still requires M1 verification |

No credential-free command that prints the complete effective config with field provenance was established for the observed binary. This is an explicit unknown, not inferred success.

## Experimental Jcode evidence

### Destination and precedence

Named profiles are entries under `[profiles.<name>]` in `~/.jcode/config.toml`, selected with `jcode --profile <name>` or the TUI profile picker. The bundled wrapper guide states that profiles supply provider, model, reasoning, tool, skill, and additional-instruction defaults without editing configuration or affecting another session. The observed fork's generated default-config documentation defines this secret-free destination shape:

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

Explicit CLI flags and environment overrides retain higher precedence than the selected profile. `profile current` and `profile resolve` report effective non-secret policy and field sources. Restored sessions retain a credential-free resolved snapshot and warn on missing or changed profiles rather than silently adopting a different named profile. Child agents and swarm workers inherit effective restrictions unless explicitly overridden.

Because explicit child overrides are possible, a root profile is not an organizational enforcement boundary. A portable policy requiring restrictions on all descendants must remain non-applicable unless child override behavior is separately constrained and verified.

### Route and authentication

The observed CLI distinguishes provider routes such as `openai` and `openai-api`, and supports explicit provider, model, and reasoning-effort selection. That distinction is promising for exact route mapping, but it is custom-fork behavior and no local profile values or credential state were read during M0.

`jcode profile show`, `current`, and `resolve` are documented to avoid credential exposure and provider initialization. Even so, M0 did not run them against Ariel's profiles because doing so would read personal target configuration. M1 experimental work must use an isolated synthetic config home before claiming a route mapping.

### Offline inspection and effects

| Command | Documented effect | M0 use |
| --- | --- | --- |
| `jcode --version` | Prints build version | Run |
| `jcode profile --help` | Prints profile inspection and override surface | Run |
| `jcode --quiet profile list --json` | Lists configured names without provider initialization | Documented, not run against personal config |
| `jcode --quiet profile show NAME --json` | Shows safe configured fields and instruction presence/length | Documented, not run against personal config |
| `jcode --quiet profile current --json` | Shows current effective policy and explicit no-profile state | Documented, not run against personal config |
| `jcode --quiet profile resolve NAME --json` | Resolves effective tools, skills, and field sources without mutating a session | Documented, not run against personal config |

The first adapter experiment must use an isolated synthetic config and a private/new server socket where required. It must not reuse Ariel's live config or shared daemon as a fixture.

## Portable-field classification

`fidelity` describes whether the target has a corresponding concept. `delivery` describes whether M0 has an approved deterministic destination. `enforcement` describes what has actually been demonstrated. `unknown` and `unsupported` block applicability when the field is required.

| Portable field | Codex 0.154.0 | Experimental Jcode ed9b93b89 | M0 applicability consequence |
| --- | --- | --- | --- |
| `spec.routeRef` provider/model/effort | **Partial fidelity.** Profile keys exist for provider, model, and effort; runtime overrides are higher precedence. Exact auth mode is unverified. | **Partial fidelity.** Profile/provider/model/effort concepts exist; provider IDs appear route-specific, but isolated mapping was not run. | Any binding requiring verified authentication mode is non-applicable. No silent route fallback. |
| `permissions.mode` | **Partial concept.** Sandbox and approval fields exist, but read-only target behavior and all escape surfaces were not verified. | **Unknown enforcement.** Tool restrictions exist, but a general read-only filesystem/process guarantee was not verified. | Required read-only policy is non-applicable. |
| `permissions.network` | **Partial/unknown.** Network policy concepts exist in current source, but local-build destination and enforcement were not verified. | **Unknown.** No version-qualified profile-level network enforcement was established. | Any required network allow/deny value is non-applicable. |
| `permissions.shell` | **Partial concept.** Sandbox/approval affect shell execution but are not proven equivalent to a shell deny. | **Partial concept.** Tool allow/deny can hide shell tools, but hooks, extensions, MCP, and children were not exhaustively bounded. | Required shell deny is non-applicable. |
| `tools.allow` / `tools.deny` | **Partial fidelity.** Tool configuration exists, but native tool expansion and closed-allowlist semantics were not verified. | **Direct profile concept.** Explicit tools, disabled tools, and base-tool suppression exist; child override remains an escape unless constrained. | M0 resolves canonical deny-wins semantics but emits no target artifact. M1 must prove exact expansion. |
| `instructions.append` | **Partial delivery.** Codex has instruction/config layers, but deterministic profile-local file projection and precedence were not verified. | **Direct profile concept.** Additive instructions are documented; content delivery and full precedence were not tested in isolation. | Canonical resolution is supported; target delivery remains M1 work. |
| `skills.include` / `skills.exclude` | **Partial fidelity.** Current config supports skill enable/disable selectors, but project discovery and rediscovery can affect effective availability. | **Direct profile concept.** Skill mode, allowlist, and disabled skills are documented; other discovery layers still require inspection. | Required absence of a skill cannot yet be claimed as enforcement. |
| `spec.extends` | **Compiler-owned.** No native target inheritance is required. | **Compiler-owned.** No native target inheritance is required. | M0 fully resolves one parent before any adapter sees the profile. |

No candidate field is classified as runtime-enforced by M0. The route/instruction-only fixture is valid as a **canonical resolution** fixture with security explicitly unmanaged, not as proof that either target can safely load the resulting policy. The constrained read-only and intentionally unsupported fixtures must remain non-applicable until an adapter supplies version-qualified evidence.

## M0 conclusion

The minimum contract can proceed without public Jcode support and without pretending Codex is already supported. M0 establishes deterministic canonical semantics and records that:

- Codex is the first public M1 adapter candidate.
- Jcode is a private experimental comparison target only.
- exact authentication-route proof is unresolved for both targets without isolated target inspection
- security-sensitive fields remain non-applicable unless M1 proves equivalent or stronger enforcement
- no target artifact, launch recipe, or live-state mutation is part of M0

This satisfies the M0 evidence requirement by classifying unknowns and unsupported mappings rather than filling them from memory or weakening the portable policy.

## Evidence references

- Local binaries: `codex --version`, `codex --help`, `jcode --version`, `jcode profile --help` on 2026-09-20.
- OpenAI Codex source documentation: configuration loader order and `ConfigToml` types in [`openai/codex`](https://github.com/openai/codex/tree/main/codex-rs/config/src).
- Bundled Jcode documentation for build `ed9b93b89`: `README.md` named-session-profile section and `docs/WRAPPERS.md` profile inspection section.
- Design gate: `openclaw-wpii`, `Unified coding-agent profiles: support matrix and product architecture`, retained in Ariel's OpenClaw workspace at `/home/ari/.openclaw/docs/research/general/2026-09-20-unified-coding-profile-mangos.md`. The Bead ID is the durable cross-repository reference.

## Deferred checks

- Run Codex parsing/effective-config inspection only in an isolated synthetic `CODEX_HOME` after the command's effects are bounded.
- Run Jcode `profile show/current/resolve` only against an isolated synthetic config and private socket.
- Do not read live agent homes, credentials, sessions, global configuration values, or provider endpoints.
- Do not launch target sessions, hooks, extensions, children, or provider requests for M0 evidence.
- Claude Code, Pi, Oh My Pi, OpenClaw, and Hermes remain roadmap research, not M0 targets.
- The placeholder `profilemango.dev` schema identifier is not a claim that the domain is registered or controlled. Rename it before public release if ownership is unavailable.
