---
name: profile-mango
description: >
  Use the mango CLI (profile-mango compatibility alias) to scaffold, author,
  validate, preview, and plan narrow installs of portable agent profiles;
  preserve its safety boundary.
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

`mango` separates portable coding-agent intent from machine-local route
identity: author/validate profiles, render inert previews, and plan only the
version-qualified installs below. Install the CLI first if not on PATH:

```bash
git clone git@gitlab.com:ariel-frischer/profile-mango.git
cd profile-mango && make deps && make install && mango version
```

From a source checkout, `make link-skill` links the global
`~/.agents/skills/profile-mango/SKILL.md` to this tracked skill in the primary
checkout. It backs up any existing global entry and is safe to run again.

## Scaffold a package

```bash
mango init                # global package at ~/.profile-mango
mango init ./my-profiles  # explicit package in a new directory
```

`init` creates only absent paths, never overwriting: `profiles/default/profile.yaml`,
`bindings/local.example.yaml`, a gitignored `bindings/local.yaml` copy, and
`bindings/.gitignore`. Home precedence is `--home`, `$PROFILE_MANGO_HOME`,
then `~/.profile-mango` (`mango home` prints it). Explicit project
`render`/`install` inputs (`--profiles`/`--resource-root`/`--bindings`) are
all-or-none; else they default from the home.

## Author a profile

Every field sits at the top level of `profiles/<name>/profile.yaml`: `description`,
`labels`, `extends`, `route`, `permissions`, `tools`, `instructions`, `skills`.
A child's `instructions` append after the parent's; `skills` replaces unless omitted.
`labels` is a string map (`labels: {team: core}`), not a list; a wrong shape
fails with its path, e.g. `labels: expected a map of string keys to string
values (e.g. labels: {team: core}), got a list`.

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

Effort is applied per agent: Codex `model_reasoning_effort` (none..xhigh;
other values block), Claude Code `effortLevel` (low/medium/high/xhigh), OpenCode agent `variant`, Oh My Pi
`:effort` selector suffix. A value an agent cannot take shows as
`effort <v>: NOT APPLIED (<reason>)` in the plan and a JSON
`skippedRequirements` entry; it is never dropped silently.

Optional `roles` under a route bind extra model roles (Oh My Pi built-ins:
`task`, `plan`, `slow`, `smol`, `tiny`, `commit`, `advisor`, `vision`); each
installs as `modelRoles.<role>: provider/model:effort`. Other agents list
roles as skipped:

```yaml
    roles:
      plan: {provider: anthropic, model: claude-opus-5-5, effort: high}
      smol: {provider: opencode-go, model: gpt-6-luna, effort: high}
```

Optional `globalInstructions` owns whole global instruction files per agent
(file name to package resource). Qualified files only: Claude Code `CLAUDE.md`,
Codex `AGENTS.md`, Oh My Pi `AGENTS.md`/`RULES.md`, OpenCode `AGENTS.md`;
another name blocks, other agents skip (`--strict` blocks). They are written
beside the config only by `use`, `install --default`, or agents without named
profiles:

```yaml
globalInstructions:
  oh-my-pi: {AGENTS.md: instructions/work/AGENTS.md, RULES.md: instructions/work/RULES.md}
```

## Validate, preview, check readiness

```bash
mango validate <profile.yaml> --bindings <file> --json
mango doctor --json
mango render <profile-name> --profiles ./profiles --resource-root . \
  --bindings ./bindings/local.yaml --target <name> --target-version <exact> \
  --out ./preview --preview --json
```

`validate` checks one profile's syntax and route binding offline; parent
resolution and resources are only checked by `render`/`install` with a
package root. `doctor` is read-only; its `PLAN` column is exactly what a plain
`mango install <profile> --target <name>` would plan, and `IN RANGE` compares
the detected binary to the tested range (tested version up to the next minor):

```text
TARGET               BINARY  DETECTED               IN RANGE  CONFIG                                   PLAN
claude-code@2.1.278  claude  2.1.281 (Claude Code)  yes       /home/u/.claude/settings.json (exists)   ready
oh-my-pi@18.2.6      omp     omp/18.2.11            yes       /home/u/.omp/agent/config.yml (exists)   ready
next: mango install default --target claude-code
```

`render` (alias `preview`) writes only beneath a new `--out` directory. Full
profiles are never directly applicable (`render.json` keeps
`applicable: false`); a staged `--preview` prints the blockers as `warning`
lines, ends with `preview staged in <out>; applicable to <target>@<version>:
no (...)`, and exits 0. Without `--preview` it writes nothing and exits nonzero.

## Plan, apply, undo

A preview's `applicable: false` does not mean the agent can't be installed:
`install` applies the narrow per-agent subset in `docs/public/agents.md` (e.g.
the Claude Code and Oh My Pi model settings). A `ready` plan is installable;
use `install` rather than hand-writing agent config. An installed version
outside the tested range still plans, with a warning. `install` always plans
first, then repeats the same inputs to apply:

```bash
mango install <profile-name> --target <name>
mango install <profile-name> --target <name> --apply --yes --expect-plan <planID>
mango undo --target <name>   # add --apply --yes --expect-plan <id> to apply
```

Agents with named profiles, native or emulated, get the profile under its own
name and leave the agent's default settings alone; the plan prints
`use it: <command>` (Codex: `codex --profile <profile-name>`, from
`$CODEX_HOME/<profile-name>.config.toml`; OpenClaw: `openclaw --profile <profile-name>`,
from `~/.openclaw-<profile-name>/openclaw.json`; Claude Code, which has no native
profiles: `claude --settings ~/.claude/profiles/<profile-name>.json`; OpenCode:
`opencode --agent <profile-name>`, from `agents/<profile-name>.md` beside `opencode.json`;
Hermes: `hermes -p <profile-name>`, from `<hermes-home>/profiles/<profile-name>/config.yaml`).
Add `--default` to also write the agent's default settings. Agents without
profiles are installed as their default settings, with a plan note saying so.
Without `--config`, plans use the target's documented default config path.
Review the resolved path, field diff, skipped requirements, and plan ID
before applying. An existing config is adopted with a backup; a
Mango-installed value the user later edited needs `--override`, and
`--non-interactive` never grants consent by itself. `undo` (alias `restore`)
restores the latest committed install, or an explicit `--original-plan <id>`.

`mango use <profile>` switches every managed agent (or `--target`/`--all`) to
the profile as its default, with the same plan-then-`--apply --yes
--expect-plan <id>` consent. Global files the new profile does not write are
released: deleted if profile-mango created them, restored from the backup if
adopted. `mango status [--json]` is read-only: per agent the recorded profile,
each owned file `in-sync`/`edited`/`missing`, and `sources:
current`/`changed`/`unknown` against the current profile files and bindings.

```bash
mango status --json
mango use <profile-name>            # add --apply --yes --expect-plan <id>
```

## Safety boundary

- `init`, `home`, `validate`, and `render` never inspect agent homes, read
  credentials, call providers, or launch target agents.
- `install`/`undo` touch only the planned config path and adjacent
  profile-mango manifests, backups, journals, and locks; never auth stores,
  sessions, plugins, MCP, providers, or the network. Unqualified targets and
  unknown required properties block installation rather than being dropped.
- Known requirements a target cannot install (permissions, tools,
  instructions, skills, roles, unsupported effort) are skipped, not applied: the plan lists them under
  each target (`not installed for this agent: ...`, JSON
  `skippedRequirements`). Pass `--strict` to block instead.
- `mango agents check` is a separate network drift check; skip it
  when offline operation is required.
