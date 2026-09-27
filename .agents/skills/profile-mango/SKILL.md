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
  version: 0.0.5
  tags: profile-mango, coding-agents, profiles, cli, yaml, validation
allowed-tools: Bash Read Write Edit
---

# profile-mango

`mango` separates portable coding-agent intent from machine-local route
identity: author/validate profiles, render inert previews, and plan only the
version-qualified installs below. Install the CLI first if not on PATH:

```bash
go install github.com/ariel-frischer/profile-mango/cmd/mango@latest
# or a checksum-verified release binary:
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/profile-mango/main/install.sh | sh
mango version
```

Install or update this skill with
`npx skills add ariel-frischer/profile-mango --skill profile-mango -g`.

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

Each `skills` entry is a skill folder's `SKILL.md` (frontmatter `description`
required; `name`, if set, equals the folder name). A vendored copy adds
`source: {repo, commit, path?, sha256}`, where `sha256` is the folder's tree
digest; a mismatch fails validation and names the actual digest.

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

Roles use a fixed portable vocabulary: `worker` (implementation subagents),
`planner` (planning/architecture), `research` (read-only scouting), `tiny`
(small mechanical tasks, commit messages). Any other name fails; old Oh My Pi
slot names fail with a hint (`smol` -> `research`, `task` -> `worker`,
`plan`/`slow` -> `planner`, `commit` -> `tiny`). Optional `roles` under a route
bind a model per role. Oh My Pi installs them as `modelRoles.<slot>:
provider/model:effort` (worker: task; planner: plan, slow; research: smol; tiny:
tiny, commit) and the optional route `subagentMaxEffort` as `task.maxEffort`
(a cap on caller-requested per-spawn effort); other agents skip
`subagentMaxEffort` and install bound roles only through role subagent files
(below; `--strict` blocks skips):

```yaml
    subagentMaxEffort: high
    roles:
      planner: {provider: anthropic, model: claude-opus-5-5, effort: medium}
      research: {provider: opencode-go, model: gpt-6-luna, effort: high}
```

To change an existing route, use `mango route set` instead of hand-editing the
bindings file: it edits only the affected lines (comments survive), creates a
missing `targets.<agent>`/`roles.<role>` entry, validates like install, and
writes nothing when invalid or unchanged. `mango route unset` removes fields and
prunes emptied entries. An effort no agent uses only warns. Then apply with
`mango use <profile>`.

```bash
mango route list                                   # routes + profiles using each
mango route show sol --target oh-my-pi [--json]    # as written + effective route
mango route set sol --target oh-my-pi --effort medium --dry-run
mango route set opus55 --role research --model gpt-6-luna
mango route set luna --target codex --role research --provider openai  # targets.codex.roles.research
mango route unset sol --target oh-my-pi effort     # provider|model|effort|subagent-max-effort
```

`targets.<agent>.roles.<role>: {provider?, model?, effort?}` changes a role the
base `roles` already binds for one agent (e.g. ChatGPT OAuth is `openai-codex`
on Oh My Pi but `openai` on Codex); an unbound role fails. Agents filter role providers:
Claude Code role files take only `anthropic` models and Codex only its OpenAI
provider; another provider leaves that role file without a model (the agent
default) and the plan says why.

Profile `roles.<role>: {description, instructions?}` describes each role
(`instructions` is a package resource path; a child's role replaces the
parent's same-name role). When the profile is the agent's default (`--default`
or `mango use`) each role becomes a whole-file owned subagent file carrying the
bound model/effort: Codex `~/.codex/agents/<role>.toml`, OpenCode
`~/.config/opencode/agents/<role>.md` (`mode: subagent`, `variant`), Oh My Pi
`~/.omp/agent/agents/<role>.md` (`model: "@<slot>"`), Claude Code
`~/.claude/agents/<role>.md` (anthropic only). Named-only installs list them as
`role-definitions` (subagent files are global); Pi, Hermes, and OpenClaw always
skip them; `--strict` blocks. A role effort the agent cannot write shows as
`effort <v> (role <r>): NOT APPLIED`; a bound role with no declared profile role
installs only on Oh My Pi and is otherwise skipped as `roles`. A same-name
unmanaged file (e.g. a hand-written `~/.codex/agents/planner.toml`) is adopted
with a backup and replaced whole, so native-only fields such as Codex
`sandbox_mode` are lost: check the plan's `adopt` rows first.

Optional `globalInstructions` owns whole global instruction files per agent
(file name to package resource, or a non-empty list of resources joined in
order with one blank line, e.g. shared core plus a per-agent part). Qualified files only: Claude Code `CLAUDE.md`,
Codex `AGENTS.md`, Oh My Pi `AGENTS.md`/`RULES.md`, OpenCode `AGENTS.md`;
another name blocks, other agents skip (`--strict` blocks). They are written
beside the config only by `use`, `install --default`, or agents without named
profiles:

```yaml
globalInstructions:
  oh-my-pi: {AGENTS.md: [instructions/shared/core.md, instructions/work/omp.md], RULES.md: instructions/work/RULES.md}
  home: {AGENTS.md: instructions/work/home-AGENTS.md}   # ~/AGENTS.md, owned via Pi
```

`home` accepts only `AGENTS.md` and is written by the Pi target (the one agent
that reads it in every folder under your home); installs without Pi skip it.
Optional `agentFiles` ships native subagent files verbatim into the agent's
role-file folder (`agents/`), for frontmatter mango does not model or to
override a bundled agent by name (Oh My Pi loads `~/.omp/agent/agents/scout.md`
over its bundled `scout`):

```yaml
agentFiles:
  oh-my-pi: {scout.md: agents/omp/scout.md}
  codex: {reviewer.toml: agents/codex/reviewer.toml}
```

Only role-file agents accept them (Codex `.toml`; Claude Code, OpenCode, Oh My
Pi `.md`; a wrong extension blocks); other agents and named-only installs skip
them as `agentFiles` (`--strict` blocks). A name matching a profile role
(`research.md` with `roles.research`) fails validation. Ownership matches global
files: adopt with backup, `agent-file` kind in `mango status`, edits need
`--override`, `use` releases, `undo` restores.

A full example with all four roles and global files:
`examples/profiles/daily-driver/profile.yaml` in the source checkout.

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
oh-my-pi@18.3.2      omp     omp/18.3.4             yes       /home/u/.omp/agent/config.yml (exists)   ready
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
Hermes: `hermes -p <profile-name>`, from `<hermes-home>/profiles/<profile-name>/config.yaml`;
Oh My Pi, emulated: `omp --config ~/.omp/agent/profiles/<profile-name>.yml`).
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
Each agent is labeled at its installed version (same `--version` probe as
install plans), e.g. `codex@0.155.1 (tested 0.157.1)` when it differs from the
tested adapter version; `--json` adds `versionCheck`
(`binary`, `qualified`, `range`, `detected`, `status`).

```bash
mango status --json
mango use <profile-name>            # add --apply --yes --expect-plan <id>
```

`mango use` only covers agents already managed (see `status`). To bring a new
agent under a profile, first run `mango install <profile> --default --target
<name>`. With `use` or `--default`, Claude Code, Codex, Oh My Pi, and OpenCode
copy each skill folder whole into their skills folder (Codex:
`~/.agents/skills`); named profiles list skills as not installed, and switching
to a profile without a skill removes its files or restores the replaced ones.
OpenClaw copies skills beside every config it writes, named profiles included
(`~/.openclaw-<name>/skills`); `mango install <p> --target openclaw --agent
openclaw=<id>` also sets that existing agent's `agents.entries.<id>.skills`
allowlist, which later installs keep and a switch gives back.

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
