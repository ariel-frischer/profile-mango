# Ariel custom Jcode fork experimental configuration reference

**Reference date:** 2026-09-20. **Local source snapshot:** version `0.83.0`,
branch `dev`, commit `ed9b93b894454619f73ccddd24c2ff7a3c98ddc3`. No canonical
public release was established. **Status:** experimental-only local comparison
target; not an intended supported MVP target, public compatibility promise, or
supported capability. No adapter ships.

This reference covers only Ariel's custom Jcode fork. The named profile commands
and `[profiles.<name>]` configuration described below are fork-specific
observations and must not be generalized to upstream or otherwise independently
distributed Jcode.

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

## Candidate inspection and gaps

The local custom-fork source documents `jcode version` and these custom-fork
commands:

- `jcode profile list`
- `jcode profile show <name>`
- `jcode profile resolve <name> --json`
- `jcode profile current`
- `jcode provider current --json`
- `jcode model list --provider`

These commands were not executed for this pack.

There is no retained runtime evidence here for exact skill order, auth-store
behavior, tool enforcement, or installed/source parity. A future experimental
probe must use synthetic configuration, a private socket, no real credentials,
blocked provider access, and no session, hook, extension, or child-agent launch.
Results remain experimental developer evidence and cannot establish upstream or
public Jcode support.
