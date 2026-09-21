# Ariel custom Jcode fork experimental configuration reference

**Reference date:** 2026-09-20. **Local source snapshot:** version `0.83.0`,
branch `dev`, commit `ed9b93b894454619f73ccddd24c2ff7a3c98ddc3`. No canonical
public release was established. **Status:** experimental-only local comparison
target; not an intended supported MVP target, public compatibility promise, or
supported capability. An exact-build inert `ariel-jcode` preview adapter exists,
but it is not public support.

This reference covers only Ariel's custom Jcode fork. The named profile commands
and `[profiles.<name>]` configuration described below are fork-specific
observations and must not be generalized to upstream or otherwise independently
distributed Jcode. An inert `ariel-jcode` renderer exists for this exact tested
build, but it is experimental-only and not a supported public target.

## Configuration and precedence

The Ariel custom-fork snapshot defines runtime TOML at `~/.jcode/config.toml`,
base provider fields, and named `[profiles.<name>]` entries. Provider, model,
reasoning effort, tool, skill, and additive-instruction settings are candidate
comparison concepts for this fork only.
Credential stores remain target-owned and out of scope. Some normal loading paths
can fall back after parse errors, so a later probe must use strict secret-free
profile resolution instead of assuming every entry point fails closed.

Source locators at the pinned snapshot:

- `crates/jcode-config-types/src/lib.rs` — configuration types.
- `src/cli/profile.rs` — secret-free profile inspection.
- `docs/SYSTEM_PROMPT_CONFIG.md` — instruction layering.

## Exact tested build and adapter boundary

The retained isolated synthetic probe tested only `jcode v0.83.909-dev
(ca8017a3a)`, commit `ca8017a3a`, with SHA-256
`392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992`. The
source documentation snapshot `ed9b93b894454619f73ccddd24c2ff7a3c98ddc3` is
separate provenance and is not proven equivalent to that build.

The `pkg/adapters/arieljcode` renderer uses the explicit target identity
`ariel-jcode` and produces only a deterministic inert TOML preview. It projects
the exact-build-observed provider, model, reasoning effort, closed tool
selectors, an observed `none` tool profile for canonical closed allowlists,
empty-skill mode, canonical skill selectors, and instruction presence/character
count metadata. It never emits route authentication, provider profiles,
credentials, `agents_md_path`, arbitrary target keys, or an active target file.
Canonical input has no target-owned skill exclusion or non-empty skill-mode
field, so those fields are deliberately omitted rather than guessed.

The target parser accepted and omitted an unknown profile key in the retained
probe. Adapter-side projection is therefore an explicit allowlist over canonical
fields, and unsupported canonical policy shapes fail closed. This adapter remains
non-applicable because authentication, discovery, delivery, precedence, child
overrides, hooks, extensions, MCP, and runtime enforcement are unverified.

## Candidate inspection and gaps

The local custom-fork source documents `jcode version` and these custom-fork
commands as candidate inspection surfaces:

- `jcode profile list`
- `jcode profile show <name>`
- `jcode profile resolve <name> --json`
- `jcode profile current`
- `jcode provider current --json`
- `jcode model list --provider`

The retained isolated exact-build probe actually executed only `jcode --version`,
`jcode profile --help`, and the synthetic `profile list`, `profile show`,
`profile current`, and `profile resolve` commands. It did not execute
`jcode provider current --json` or `jcode model list --provider`; those remain
documented candidates, not executed evidence. No live profile, target home,
credential store, shared socket, session, or provider state was inspected.

There is no retained runtime evidence here for exact skill order, auth-store
behavior, tool enforcement, or installed/source parity. A future experimental
probe must use synthetic configuration, a private socket, no real credentials,
blocked provider access, and no session, hook, extension, or child-agent launch.
Results remain experimental developer evidence and cannot establish upstream or
public Jcode support.
