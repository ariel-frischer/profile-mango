# Profile and bindings reference

A profile package has this layout. `mango init` creates it in
`~/.profile-mango` (or `--home`, or `$PROFILE_MANGO_HOME`). `init <dir>`
creates it in a new folder instead.

```text
profiles/<name>/profile.yaml   one folder per profile
bindings/local.yaml            your routes (gitignored)
bindings/local.example.yaml    a shareable copy
instructions/, skills/         optional files that profiles refer to
```

JSON Schemas: [profile](../../schemas/profile.schema.json) and
[bindings](../../schemas/bindings.schema.json).

## Profile fields

Every field sits at the top level of `profile.yaml`, and all of them are optional.

```yaml
description: Read-only code review
extends: workflow-base
route: local
permissions: {mode: read-only, network: deny, shell: deny}
tools: {allow: [read, search], deny: [write, edit, shell]}
instructions: {append: [instructions/review/AGENTS.md]}
skills: [skills/review/SKILL.md]
```

| Field | Meaning | With `extends` |
| --- | --- | --- |
| `name` | Defaults to the folder name. If set, it must match. Lowercase letters, digits, and `-`. | Not inherited |
| `description` | Free text. | Child wins if set |
| `labels` | Map of string keys to string values, e.g. `labels: {team: core}`. A list such as `labels: [a, b]` is rejected. | Merged, child keys win |
| `extends` | Parent profile name. | — |
| `route` | Route key in the bindings file. | Child wins if set |
| `permissions.mode` | `read-only`, `workspace-write`, or `unrestricted` | Per field, child wins |
| `permissions.network`, `permissions.shell` | `allow`, `deny`, or `unmanaged` | Per field, child wins |
| `tools.allow`, `tools.deny` | Tool name lists. A denied name is removed from `allow`. | Each list replaces the parent's if set |
| `instructions.append` | Instruction files, relative to the package root. | Appended after the parent's |
| `skills` | `SKILL.md` paths, relative to the package root. | Replaces the parent's if set |
| `globalInstructions` | Whole global instruction files per agent, see below. | A child's agent entry replaces the parent's; `{}` clears it |
| `roles` | Portable role definitions, see below. | A child's role replaces the parent's same-name role; other parent roles are inherited |

Unknown fields are rejected. A field with the wrong YAML shape fails with its
path and the expected shape, e.g. `labels: expected a map of string keys to
string values (e.g. labels: {team: core}), got a list`. Most agents can't install permissions, tools,
instructions, or skills yet. See [agents](agents.md) for what each one takes.

### `globalInstructions`

`globalInstructions` maps an agent name to the global instruction files that
profile owns, each a file name and a Markdown resource relative to the package
root. The installer writes the whole file beside the agent's config.

```yaml
globalInstructions:
  codex: {AGENTS.md: instructions/work/AGENTS.md}
  oh-my-pi:
    AGENTS.md: instructions/work/AGENTS.md
    RULES.md: instructions/work/RULES.md
  claude-code: {CLAUDE.md: instructions/work/CLAUDE.md}
  opencode: {AGENTS.md: instructions/work/AGENTS.md}
```

Only files an agent is documented to read are accepted: Claude Code
`CLAUDE.md` (`~/.claude`), Codex `AGENTS.md` (`$CODEX_HOME`), Oh My Pi
`AGENTS.md` and `RULES.md` (`~/.omp/agent`), and OpenCode `AGENTS.md`
(`~/.config/opencode`). Another file name for one of those agents blocks the
install. Other agents list `globalInstructions` under "not installed for this
agent"; `install --strict` blocks them instead. A global file changes every
profile of that agent, so it is written only by `mango use`, by `install
--default`, or by agents without named profiles.

An existing file is adopted with a create-only backup. A file edited after the
install shows as `edited` in `mango status` and needs `--override`. When
`mango use` switches to a profile without that file, a file profile-mango
created is deleted and an adopted file gets its original bytes back from the
backup. `mango undo` reverses each step.

The `home` key owns files in your home directory rather than an agent's
config folder. Only `AGENTS.md` is accepted:

```yaml
globalInstructions:
  home: {AGENTS.md: instructions/work/home-AGENTS.md}
```

Pi owns `~/AGENTS.md`, because it is the one supported agent that reads it in
every folder under your home directory; the plan shows it as `~/AGENTS.md`
under Pi with a warning that other agents share it. An install without Pi lists
it under "not installed for this agent"; `--strict` blocks. `mango use`,
`mango status`, and `mango undo` treat it like any other global file.

For Oh My Pi, `mango use` also gives back model role slots and
`task.maxEffort`: a slot or setting the new route no longer sets returns to the
value it had before profile-mango first wrote it, or is removed if it did not
exist.

### `roles`

`roles` describes what each portable role is for. The names are fixed:

| Role | Used for |
| --- | --- |
| `worker` | Implementation subagents |
| `planner` | Planning and architecture |
| `research` | Read-only exploration and scouting |
| `tiny` | Small mechanical tasks and commit messages |

```yaml
roles:
  worker: {description: Implements one scoped change}
  research:
    description: Read-only codebase scout
    instructions: instructions/roles/research.md
```

Each role needs a non-empty `description`. `instructions` is optional and must
be a path under the package root. Another role name fails with the list of
valid names; an Oh My Pi slot name such as `smol` also names the portable role
to use instead. The model for each role comes from the route's `roles` in the
bindings file.

When the profile is an agent's default (`install --default` or `mango use`),
each role becomes one subagent file, with the bound model and effort when the
route binds the role:

| Agent | Role file |
| --- | --- |
| Codex | `~/.codex/agents/<role>.toml` |
| OpenCode | `~/.config/opencode/agents/<role>.md` |
| Oh My Pi | `~/.omp/agent/agents/<role>.md`, model `@<slot>` |
| Claude Code | `~/.claude/agents/<role>.md` |

The file body is the role's `instructions`, or its description. Mango owns each
file: an existing one is backed up and adopted, `mango use` removes or restores
files the next profile lacks, and `undo` restores the originals. A named-only
install lists the roles as `role-definitions` under "not installed for this
agent", because subagent files are global. Pi, Hermes, and OpenClaw always
skip them. `install --strict` blocks skipped roles.

### Older wrapped format

Profiles written as `apiVersion` / `kind` / `metadata` / `spec` (with
`spec.routeRef`) still load, with a deprecation warning. Move the fields to the
top level and rename `routeRef` to `route`.

The wrapped format does not accept `roles`.

## Bindings fields

`bindings/local.yaml` maps route keys to a model choice. It never holds
credentials.

```yaml
routes:
  local:
    provider: openai
    model: gpt-6-sol
    effort: high
    targets:
      claude-code:
        provider: anthropic
        model: claude-sonnet-5
```

| Field | Required | Default | Meaning |
| --- | --- | --- | --- |
| `provider` | yes | | Provider ID, such as `openai` or `anthropic` |
| `model` | yes | | Bare model ID |
| `effort` | yes | | Reasoning effort or thinking level, such as `high` |
| `transport` | no | `native` | How the agent reaches the provider |
| `authentication` | no | `oauth` | How the agent signs in; credentials stay with the agent |
| `targets` | no | | Per-agent overrides, see below |
| `roles` | no | | A model per portable role, see below |
| `subagentMaxEffort` | no | | Highest effort a subagent spawn may request, see below |

### `targets` overrides

`targets` maps an agent name (`claude-code`, `codex`, `hermes`, `oh-my-pi`,
`openclaw`, `opencode`, `pi`) to one or more of `provider`, `model`, `effort`,
`transport`, and `authentication`. That agent gets the base route with only
those fields replaced, and other agents use the base route. This is how one
profile drives several agents with different providers.

Some agents need specific route values. For example, Claude Code installs only
with an `anthropic` provider, and Codex only with `openai`, native transport,
OAuth, and an effort from `none` to `xhigh`. The plan says why when a route doesn't fit. See [agents](agents.md).

### `roles`

`roles` gives portable roles (`worker`, `planner`, `research`, `tiny`) their
own `provider`, `model`, and optional `effort`. The route's own fields are the
default model, and `targets` overrides never change roles. Any other role name
fails with its path. The older Oh My Pi slot names fail with a hint:
`task` becomes `worker`, `plan` and `slow` become `planner`, `smol` becomes
`research`, and `commit` becomes `tiny`. `advisor` and `vision` have no
portable role.

```yaml
routes:
  local:
    provider: anthropic
    model: claude-opus-5-5
    effort: medium
    subagentMaxEffort: high
    roles:
      planner:
        provider: anthropic
        model: claude-opus-5-5
        effort: high
      research:
        provider: opencode-go
        model: gpt-6-luna
        effort: high
```

Oh My Pi writes each bound role into its model slots as
`modelRoles.<slot>: provider/model:effort`:

| Role | Oh My Pi slots |
| --- | --- |
| `worker` | `task` |
| `planner` | `plan`, `slow` |
| `research` | `smol` |
| `tiny` | `tiny`, `commit` |

Codex, OpenCode, and Claude Code install a bound role's model and effort only
inside that role's subagent file, so the profile must also declare the role and
be the agent's default (see [profile `roles`](#roles)). Other bound roles are
listed as `roles` under "not installed for this agent", and an effort the agent
cannot write shows as `effort <value> (role <role>): NOT APPLIED`;
`install --strict` blocks them instead.

### `subagentMaxEffort`

`subagentMaxEffort` caps the effort one subagent spawn may request: `minimal`,
`low`, `medium`, `high`, `xhigh`, or `max`. Oh My Pi installs it as
`task.maxEffort`, which lowers an effort the caller passes for a spawn to the
cap. It does not raise the effort of spawns that pass none, and it does not
change the role models. Other agents list it under "not installed for this
agent"; `install --strict` blocks it. `targets` overrides never change it.

### Editing routes

`mango route` edits existing routes without hand-editing YAML:

```bash
mango route list                                   # routes and the profiles using each
mango route show sol --target oh-my-pi             # as written, and what that agent gets
mango route set sol --target oh-my-pi --effort medium [--dry-run]
mango route set opus55 --role research --model gpt-6-luna
mango route set opus55 --subagent-max-effort high  # whole route only
mango route unset sol --target oh-my-pi effort     # fields: provider, model, effort, subagent-max-effort
```

`set` changes only the affected lines and adds a missing `targets.<agent>` or
`roles.<role>` entry. `unset` removes an entry, and then its `targets` or
`roles` map, once it is empty. Both check the edited file the way `install`
does and write nothing if it is invalid or unchanged; `--target` and `--role`
cannot be combined. They print the diff, then the profiles to apply with
`mango use` or `mango install`; agent config never changes until you do.
New routes are still added by hand.

## Checking a profile

```bash
mango validate ~/.profile-mango/profiles/default/profile.yaml \
  --bindings ~/.profile-mango/bindings/local.yaml
```

`validate` checks one file and its route. It does not follow `extends` or open
referenced files. `install`, `doctor`, and `preview` resolve the whole package.
