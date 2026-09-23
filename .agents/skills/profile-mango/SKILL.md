---
name: profile-mango
description: >
  Use the profile-mango CLI to scaffold, author, validate, preview, and plan
  narrow installs of portable agent profiles; preserve its fail-closed safety.
license: MIT
compatibility:
  - Codex
  - OpenCode
  - Pi
  - Cursor
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
identity: author/validate profiles, render inert previews, and plan only the
version-qualified installs below. Install the CLI first if not on PATH:

```bash
git clone git@gitlab.com:ariel-frischer/profile-mango.git
cd profile-mango && make deps && make install && profile-mango version
```

## Scaffold a package

```bash
profile-mango init                # global package at ~/.profile-mango
profile-mango init ./my-profiles  # explicit package in a new directory
```

`init` creates only absent paths, never overwriting: `profiles/default/profile.yaml`,
`bindings/local.example.yaml`, a gitignored `bindings/local.yaml` copy, and
`bindings/.gitignore`. Home precedence is `--home`, `$PROFILE_MANGO_HOME`,
then `~/.profile-mango` (`profile-mango home` prints it). Explicit project
`render`/`install` inputs (`--profiles`/`--resource-root`/`--bindings`) are
all-or-none; else they default from the home.

## Author a profile

Every field sits at the top level of `profiles/<name>/profile.yaml`: `description`,
`labels`, `extends`, `route`, `permissions`, `tools`, `instructions`, `skills`.
A child's `instructions` append after the parent's; `skills` replaces unless omitted.

```yaml
description: Read-only code review
route: local
permissions: {mode: read-only, network: deny, shell: deny}
tools: {allow: [read, search], deny: [write, edit, shell, deploy]}
instructions: {append: [instructions/AGENTS.md]}
skills: [skills/review/SKILL.md]
```

`bindings/local.yaml` names a route, never credentials. `provider`/`model`/`effort`
are required; an optional `targets` map gives named agents (`claude-code`,
`codex`, `hermes`, `oh-my-pi`, `openclaw`, `opencode`, `pi`) override fields
on top of the base route, so one profile drives several agents at once:

```yaml
routes:
  local:
    provider: openai
    model: gpt-6-sol
    effort: high
    targets:
      claude-code: {provider: anthropic, model: claude-sonnet-5}
```

## Validate, preview, check readiness

```bash
profile-mango validate <profile.yaml> --bindings <file> --json
profile-mango doctor --json
profile-mango render <profile-name> --profiles ./profiles --resource-root . \
  --bindings ./bindings/local.yaml --target <name> --target-version <exact> \
  --out ./preview --preview --json
```

`validate` checks one profile's syntax and route binding offline; parent
resolution and resources are only checked by `render`/`install` with a
package root. `doctor` reports each target's command, version, path, and
readiness, read-only. `render` (alias `preview`) only writes beneath the new
`--out` directory over `claude-code`, `codex`, `pi`, `oh-my-pi`, `openclaw`,
`hermes`, `opencode`; every renderer today reports `applicable: false`.

## Plan, apply, undo

Only exact versions/subsets in `docs/dev/target-evidence.md` are installable
(e.g. Claude Code `2.1.278` main-config model only). `install` always plans
first, then repeats the same inputs to apply:

```bash
profile-mango install <profile-name> --target <name>
profile-mango install <profile-name> --target <name> --apply --yes --expect-plan <planID>
profile-mango undo --target <name>   # add --apply --yes --expect-plan <id> to apply
```

Without `--config`, plans use the target's documented default config path.
Review the resolved path, field diff, and plan ID before applying; unowned
or externally edited files need `--override` (no general force path) and
`--non-interactive` never grants consent by itself. `undo` (alias `restore`)
restores the latest committed install, or an explicit `--original-plan <id>`.

## Safety boundary

- `init`, `home`, `validate`, and `render` never inspect agent homes, read
  credentials, call providers, or launch target agents.
- `install`/`undo` touch only the planned config path and adjacent
  profile-mango manifests, backups, journals, and locks; never auth stores,
  sessions, plugins, MCP, providers, or the network. Unqualified targets and
  unknown required properties block installation rather than being dropped.
- `profile-mango agents check` is a separate network drift check; skip it
  when offline operation is required.
