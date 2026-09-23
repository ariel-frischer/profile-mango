---
name: profile-mango
description: >
  Use the profile-mango CLI to scaffold, author, validate, preview, and plan
  narrow installations of portable coding-agent profiles and machine-local
  route bindings. Preserve its fail-closed safety and evidence boundaries.
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
identity. Create and validate profile packages, inspect inert previews, and plan
only the version-qualified installation subsets below. No agent supports full
Mango profile installation yet.

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
profile-mango init ./my-profiles     # explicit package in a new directory
```

`init` creates only absent paths and refuses to overwrite existing files:

```text
profiles/default/profile.yaml
bindings/local.example.yaml
bindings/local.yaml       # a create-only copy of the example, gitignored
bindings/.gitignore
```

The effective global home is `--home`, then `$PROFILE_MANGO_HOME`, then
`~/.profile-mango`. `profile-mango home` prints it without creating it. A named
profile lives at `profiles/<name>/profile.yaml`; `init` creates only `default`.
Instruction and skill files are optional resources that you create separately.
Their references are relative to the package/resource root, not to the profile
YAML: `instructions/AGENTS.md` means `<root>/instructions/AGENTS.md`. Explicit
project render/install inputs require `--profiles`, `--resource-root`, and
`--bindings` together; global inputs default to the effective home.

`init` already creates the ignored machine-local binding, so the global
package validates the starter profile with no manual copy:

```bash
profile-mango validate ~/.profile-mango/profiles/default/profile.yaml \
  --bindings ~/.profile-mango/bindings/local.yaml
```

For an explicit project package, run the same workflow relative to its root:

```bash
profile-mango validate profiles/default/profile.yaml \
  --bindings bindings/local.yaml
```

If `bindings/local.yaml` is missing (for example, deleted after init), the
missing-bindings error from `validate`, `render`, and `install` names the
exact fix: copy `local.example.yaml` back over it, or run `profile-mango init`
again in a fresh package.

Use `--json` when another tool or agent will consume the validation result.

## Optional repository examples

When the user asks for an existing setup to use or adapt, look first at
`examples/portable/README.md` relative to the profile-mango repository root,
**if that package is present in the checkout**. It has settings-only examples
for Claude Code, OpenCode, Pi, Oh My Pi, OpenClaw, Hermes, and bounded Codex
settings; and separate coding, review, and documentation workflows with original
AGENTS.md instructions and skills. `examples/jcode-like/README.md` is an older
preview-only reference. Neither package is the default: do not copy,
select, or install a sample unless requested. Prefer the user's existing profile
package when one exists. Copy an example as a **whole package** to a new path
before editing it; resource paths are relative to its root. Check its dated
model bindings against the exact target version and account availability.
Settings-only installation is limited to the qualified fields below, and
workflow examples with required permissions/tools/resources remain preview-only.
If `examples/portable/` is absent, say so rather than assuming it is on `main`.

## Authoring rules

A richer package might look like this (the files beyond the starter scaffold are
user-created):

```text
~/.profile-mango/
├── profiles/
│   └── review/
│       └── profile.yaml
├── instructions/
│   └── AGENTS.md
├── skills/
│   └── review/
│       └── SKILL.md
└── bindings/
    ├── local.example.yaml
    ├── local.yaml          # machine-local, ignored
    └── .gitignore
```

`profiles/review/profile.yaml` contains the policy and *paths* to resources.
Every field sits at the top level:

```yaml
description: Read-only code review
route: local
permissions:
  mode: read-only
  network: deny
  shell: deny
tools:
  allow: [read, search]
  deny: [write, edit, shell, deploy]
instructions:
  append:
    - instructions/AGENTS.md
skills:
  - skills/review/SKILL.md
```

`name` is optional and defaults to the folder name (`review` here); if present
it must match the folder. Other fields are `description`, `labels`, `extends`,
`route`, `permissions`, `tools`, `instructions`, and `skills`. The older
`apiVersion`/`kind`/`metadata`/`spec` wrapper with `spec.routeRef` still parses
but emits a `profile.legacy_format` deprecation warning; move the fields to the
top level and rename `routeRef` to `route`.

`instructions/AGENTS.md` is ordinary Markdown, for example:

```markdown
# Review instructions

Inspect the change and its tests. Report concrete findings with file and line
references. Do not modify files.
```

`skills/review/SKILL.md` can contain target-consumable skill instructions:

```markdown
---
name: review
description: Review code without modifying it
---

# Review

Check correctness, regressions, and test coverage. Report actionable findings.
```

The name `AGENTS.md` does not automatically install a native agent instruction
file: Mango reads it only because the YAML refers to it. Likewise,
`skills/review/SKILL.md` is a separately authored skill file, not inline YAML.
The `tools` allow/deny lists and `permissions` are YAML policy fields, not files
in the folder. This richer example is **not** fully installable on current targets:
required permissions, tools, instructions, or skills that a target cannot
preserve block installation. For a minimal installable settings profile, use the
a profile containing only `route: local`, then
check the plan for the exact target and version. Profile names belong to Mango;
installing one does not automatically create or activate a native named profile.

With multiple profiles, each folder has its own `profile.yaml`; resources may be
private to that folder or shared. For example:

```text
~/.profile-mango/
├── profiles/
│   ├── base/   (profile.yaml, AGENTS.md)
│   ├── daily/  (profile.yaml, AGENTS.md)
│   └── review/ (profile.yaml, AGENTS.md)
├── skills/
│   ├── coding/SKILL.md
│   └── review/SKILL.md
└── bindings/local.yaml
```

`profiles/base/profile.yaml` can reference `profiles/base/AGENTS.md` and define
`route: local`. A daily profile can use `extends: base`, append
`profiles/daily/AGENTS.md`, and list `skills/coding/SKILL.md`; review can do the
same with its own files and tools policy. Paths are still relative to the *home*
(resource root), even when the Markdown sits next to a profile YAML. Inheritance
appends instruction paths in parent-then-child order. A child's `skills` list
replaces the parent's if present; omit it to inherit, or use `skills: []` to
clear it. Parent and child must not resolve to the same instruction path twice.

The local binding identifies a route, never credentials:

```yaml
routes:
  local:
    provider: openai
    model: gpt-6-sol
    effort: high
    # transport: native     # default when omitted
    # authentication: oauth # default when omitted
    targets:              # optional explicit per-agent overrides
      claude-code:
        provider: anthropic
        model: claude-sonnet-5
```

Each target gets the base route with any fields from `targets.<target>` applied,
so one profile can install across agents that need different providers. Only
agent names (`claude-code`, `codex`, `hermes`, `oh-my-pi`, `openclaw`,
`opencode`, `pi`, `ariel-jcode`) are valid keys, and each override must set at
least one field. An unknown key fails validation. Routes without `targets`
behave as before. `provider`, `model`, and `effort` are required; `transport`
defaults to `native` and `authentication` to `oauth` when omitted, and a target
override that omits them inherits the base values.

Keep `bindings/local.yaml` untracked. Credentials remain owned by the target
agent and must not be copied into profiles, bindings, generated artifacts, or
diagnostics.

## Validation and errors

`validate <profile.yaml> --bindings <file>` checks one profile's syntax and its
route binding offline. Unknown or duplicate keys, null values, and unsupported
versions block it. Parent resolution, inheritance cycles, and referenced resource
files are checked when rendering or planning installation with a package root;
do not treat a passing single-file validation as proof they exist.

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

All target renderers remain non-applicable previews. Expect blocking diagnostics
and a nonzero exit status even when preview artifacts are written. The separate
installer can apply only the qualified subsets below. Preview syntax and native
config consumption do not prove authentication, full delivery, or policy
enforcement.

## Plan-first install

Only these exact versions and subsets are qualified for installation:

| Target | Installable subset |
| --- | --- |
| Claude Code `2.1.278` | Main-config model only |
| OpenCode `1.18.31` | Main-config model, optionally one owned `SKILL.md` and discovery path; alternatively an explicit named primary/subagent Markdown definition with model and ordered instructions |
| Pi `0.86.1` | Provider, model, thinking level |
| Oh My Pi `18.2.6` | Default model role and thinking level |
| OpenClaw `2026.9.5` | Default agent model and thinking level |
| Hermes `0.21.3` | Provider, default model, reasoning effort |
| Codex `0.154.0` | Root `model_provider`, `model`, and `high` reasoning effort only; settings, not authentication or full-profile installation |

Consult the repository README and `docs/dev/target-evidence.md` for exact field,
route, and precedence limitations before planning. For example, OpenCode
main-config installation requires a route-only or single-skill profile and one
config path. Without `--config`, `--target <name>` plans against the target's
documented default user config (for example `~/.codex/config.toml` or
`${XDG_CONFIG_HOME:-~/.config}/opencode/opencode.json`); the human plan prints
`config: <path> (default)`. See `docs/dev/target-evidence.md#default-config-destinations-2026-09-22`
for every target's default and relocation variable. Always plan first and review:

```bash
profile-mango install <profile-name> --target opencode
profile-mango install <profile-name> \
  --profiles ./profiles \
  --resource-root . \
  --bindings ./bindings/local.yaml \
  --config opencode=/explicit/path/opencode.jsonc \
  --override --json
```

`--config <name>[@<version>]=<path>` overrides the default and selects its
target, so `--target` is only needed to plan a target without a config entry. A bare name
resolves to that target's single qualified version; an explicit unqualified
`@version` still blocks. `--config-path` is a deprecated hidden alias for
`--config`. For Codex, use `--target codex` (default
`$CODEX_HOME/config.toml` or `~/.codex/config.toml`) or `--config codex=<path>`,
an OpenAI/native/OAuth route, and `effort: high`. Treat this as **settings only**, not successful OAuth
or full profile installation. The plan warns that `codex login status`
distinguishes stored API-key from ChatGPT modes but cannot prove exact OAuth.
Do not run it on another user's behalf or share status output containing key
fragments.

Review the destination digest, field diff, file hashes, and plan ID. Apply only by
repeating the exact inputs with `--apply --yes --expect-plan <planID>`. Existing
unowned or externally edited files require `--override`; there is no general force
path. OpenCode skill files cannot override unowned or externally edited resources,
even with `--override`. Its directory-wide discovery is not an exclusive allowlist.
Default paths are the user's live agent config: confirm the printed `config:`
path and field diff before applying, and keep the create-only backup.

For a distinct named OpenCode agent definition, use an instruction-only profile
with a native route and pass `--agent opencode@1.18.31=primary:mango-review` (or
`subagent:mango-review`) plus
`--config opencode@1.18.31=<opencode-config-dir>/agents/mango-review.md` (named
definitions have no default path).
The parent `agents` directory must already exist. This writes a custom Markdown
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
  the planned config (explicit `--config` or the documented default user path)
  and adjacent Profile Mango manifests, backups, journals, and locks.
  It must not read auth stores, sessions, plugins, MCP, providers, or the network.
  Apply only after the user reviews the plan's resolved path and diff.
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
