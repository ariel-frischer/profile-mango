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
profile-mango init ./my-profiles # explicit package in a new directory
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

All current target renderers remain non-applicable previews. Expect blocking
diagnostics and a nonzero exit status even when preview artifacts are written.
OpenCode `1.18.31` separately supports lossless transactional application of
top-level `model` and optionally one `SKILL.md` plus `skills.paths` at one explicit
path. A distinct named primary/subagent definition can receive model and ordered
instructions. Codex `0.154.0` separately supports only root `model_provider`,
`model`, and `model_reasoning_effort = "high"`. Neither makes its renderer or full
profile applicable. Never present installed settings as authentication, full
delivery, or policy enforcement.

## Plan-first install

Only the exact subsets in the README install table and target evidence ledger
are installable. Other targets remain blocked. OpenCode main-config installation
requires exact version `1.18.31`, a route-only or single-skill profile, and one
explicit config path. Plan first against synthetic or separately approved
disposable state:

```bash
profile-mango install <profile-name> \
  --profiles ./profiles \
  --resource-root . \
  --bindings ./bindings/local.yaml \
  --target opencode@1.18.31 \
  --config-path opencode=/explicit/disposable/opencode.jsonc \
  --override --json
```

For Codex, use `--target codex@0.154.0` with
`--config-path codex=/explicit/disposable/config.toml`, an OpenAI/native/OAuth route, and
`effort: high`. Treat this as **settings only**, not successful OAuth or full
profile installation. The plan warns that `codex login status` distinguishes
stored API-key from ChatGPT modes but cannot prove exact OAuth. Do not run it
on another user's behalf or share status output containing key fragments.

Review the destination digest, field diff, file hashes, and plan ID. Apply only by
repeating the exact inputs with `--apply --yes --expect-plan <planID>`. Existing
unowned or externally edited files require `--override`; there is no general force
path. OpenCode skill files cannot override unowned or externally edited resources,
even with `--override`. Its directory-wide discovery is not an exclusive allowlist.
Never use a live path without new path-specific user approval after a verified
disposable backup/restore rehearsal.

For a distinct named OpenCode agent definition, use an instruction-only profile
with a native route and pass `--agent opencode@1.18.31=primary:mango-review` (or
`subagent:mango-review`) plus `--config-path
opencode@1.18.31=/explicit/disposable/opencode/agents/mango-review.md`. The
parent `agents` directory must already exist. This writes a custom Markdown
prompt and adjacent ownership manifest, not the main JSONC config or a native
named-profile preset. A primary is not activated by installation and a subagent
is not proven delegated. Required permissions, tools and skills block. Unowned
or edited definitions never permit override, including identical bytes. On
profile omission, only clean owned skills and Mango-introduced discovery paths
are removed; ambiguous legacy paths remain with a warning.

Use global `--non-interactive` for automation. It never grants consent: apply
still needs `--yes --expect-plan <planID>`. JSON and redirected input never prompt.
Plan-only JSON is one `Plan`; ready apply JSON is one `ApplyReport`, including
engine failure reports, not a concatenated plan/report stream. Blocked apply emits
its blocked plan and exits nonzero. Check exit status even when JSON is present.

## Safety boundary

- `init`, `home`, `validate`, and `render` do not inspect agent homes, access
  credentials, call providers, or launch target agents.
- `install` may apply only qualified target-specific subsets. It may touch only
  explicit config paths and adjacent Profile Mango manifests, backups, journals,
  and locks.
  It must not read auth stores, sessions, plugins, MCP, providers, or the network.
  Do not point it at a live config without new path-specific user approval.
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
make deps
make install
profile-mango version
```
