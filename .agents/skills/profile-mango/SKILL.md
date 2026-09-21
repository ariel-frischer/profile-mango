---
name: profile-mango
description: >
  Use the profile-mango CLI to scaffold, author, validate, and inspect portable
  coding-agent profiles and machine-local route bindings. Use when working with
  PolicyProfile YAML, local bindings, offline validation, or inert target
  previews. Preserve its fail-closed safety and evidence boundaries.
license: MIT
compatibility:
  - Claude Code
  - Cursor
  - Codex
  - Gemini CLI
  - VS Code
metadata:
  author: Ariel Frischer
  version: 0.0.1
  tags: profile-mango, coding-agents, profiles, cli, yaml, validation
allowed-tools: Bash Read Write Edit
---

# profile-mango

`profile-mango` separates portable coding-agent intent from machine-local route
identity. Use it to create and validate strict profile packages, then inspect
explicitly inert target previews without modifying an agent's configuration.

## Check the CLI

```bash
profile-mango version
profile-mango --help
```

The skill is guidance. The `profile-mango` binary must be installed separately.

## Core workflow

Choose one package location:

```bash
profile-mango init                  # global package at ~/.profile-mango
profile-mango init .                # explicit package in the current directory
profile-mango init ./agent-profiles # explicit package in a new directory
```

`init` creates only absent paths and refuses to overwrite existing files:

```text
profiles/default/profile.yaml
bindings/local.example.yaml
bindings/.gitignore
```

For the global package, create the ignored machine-local binding and validate
the starter profile:

```bash
cp ~/.profile-mango/bindings/local.example.yaml \
   ~/.profile-mango/bindings/local.yaml
profile-mango validate ~/.profile-mango/profiles/default/profile.yaml \
  --bindings ~/.profile-mango/bindings/local.yaml
```

For an explicit project package, run the same workflow relative to its root:

```bash
cp bindings/local.example.yaml bindings/local.yaml
profile-mango validate profiles/default/profile.yaml \
  --bindings bindings/local.yaml
```

Use `--json` when another tool or agent will consume the validation result.

## Authoring rules

A profile contains portable intent:

```yaml
apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: research
spec:
  routeRef: local
  permissions:
    mode: read-only
    network: allow
    shell: deny
  tools:
    allow: [read, search, web]
    deny: [write, edit, deploy]
  instructions:
    append:
      - instructions/system.md
  skills:
    - skills/research/SKILL.md
```

The local binding identifies a route, never credentials:

```yaml
routes:
  local:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
```

Keep `bindings/local.yaml` untracked. Credentials remain owned by the target
agent and must not be copied into profiles, bindings, generated artifacts, or
diagnostics.

## Validation and errors

Validation is strict and offline. Unknown or duplicate keys, null values,
unsupported versions, missing parents, inheritance cycles, missing routes, and
resource paths that escape the selected root are blocking errors.

When validation fails:

1. Read the diagnostic code and field path.
2. Fix the source profile, parent profile, binding, or referenced resource.
3. Re-run `validate` with the same explicit inputs.
4. Do not weaken a required permission, tool, authentication route, or transport
   merely to make validation pass.

## Inert target previews

Rendering writes only to a new explicit staging directory. It does not install
or apply target configuration. Use an exact documented target version and pass
all three project input flags together:

```bash
profile-mango render <profile-name> \
  --profiles ./profiles \
  --resource-root . \
  --bindings ./bindings/local.yaml \
  --target <target> \
  --target-version <exact-version> \
  --out ./preview-output \
  --preview --json
```

Current target names are `claude-code`, `codex`, `pi`, `oh-my-pi`, `openclaw`,
`hermes`, and `opencode`. `ariel-jcode` is experimental-only for Ariel's custom Jcode fork,
not upstream Jcode or a supported public target.

All current production target renderers are non-applicable previews. Expect blocking
diagnostics and a nonzero exit status even when preview artifacts are written.
A mock-only install engine may plan blocked production targets and exercise apply
semantics through fake adapters and synthetic temporary files. It must not read or
modify global agent configuration. OpenCode `1.18.31` has separate approved
isolated evidence that the exact JSONC model candidate parses and appears in merged
output; this does not make the renderer applicable or the installer production-
ready. Never present preview syntax, native parsing, or mock application evidence
as installed or enforced.

## Plan-first install

Production targets currently support deterministic blocked planning only. Use exact
target versions and explicit project inputs when inspecting the plan:

```bash
profile-mango install <profile-name> \
  --profiles ./profiles \
  --resource-root . \
  --bindings ./bindings/local.yaml \
  --target codex@0.154.0 \
  --target opencode@1.18.31 \
  --json
```

Expect a nonzero exit and `blocked` status before any target configuration read.
Do not add `--apply` for a production target. Hash-bound apply, backups, stale
checks, journals, and recovery are tested only with fake adapters and synthetic
temporary files.

## Safety boundary

- `init`, `home`, `validate`, and `render` do not inspect agent homes, access
  credentials, call providers, or launch target agents.
- `install` must remain plan-only for production targets in the current build.
  Successful mutation tests use fake adapters and `t.TempDir` state only; do not
  point it at a real Codex, Claude Code, OpenCode, Jcode, or other agent path.
- `render` writes only beneath the new path supplied by `--out` and never applies
  the candidate.
- `scripts/opencode-config-probe.sh` is an opt-in exact-binary developer probe. Run
  it only with explicit authorization, synthetic scratch state, blocked network,
  no TUI/session/provider credentials, and retained backup/restore evidence.
- `profile-mango agents check` is separate: it performs an explicit network drift
  check against documented sources. Do not run it when offline operation is
  required.
- Unknown or unsupported target behavior remains a blocker, not an invitation to
  infer compatibility.

## Installing the CLI from source

```bash
git clone git@gitlab.com:ariel-frischer/profile-mango.git
cd profile-mango
make install
make install-global
profile-mango version
```
