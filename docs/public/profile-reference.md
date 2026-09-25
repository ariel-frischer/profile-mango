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

For Oh My Pi, `mango use` also gives back model roles: a role the new route
no longer binds returns to the value it had before profile-mango first wrote
it, or is removed if it did not exist.

### Older wrapped format

Profiles written as `apiVersion` / `kind` / `metadata` / `spec` (with
`spec.routeRef`) still load, with a deprecation warning. Move the fields to the
top level and rename `routeRef` to `route`.

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
| `roles` | no | | Extra named roles with their own model, see below |

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

`roles` gives named roles, such as a planner or a fast helper, their own
`provider`, `model`, and optional `effort`. The route's own fields are the
default role, so `default` is not allowed as a role name. Role names are
lowercase kebab-case, and `targets` overrides never change roles.

```yaml
routes:
  local:
    provider: anthropic
    model: claude-opus-5-5
    effort: medium
    roles:
      plan:
        provider: anthropic
        model: claude-opus-5-5
        effort: high
      smol:
        provider: opencode-go
        model: gpt-6-luna
        effort: high
```

Only Oh My Pi installs roles today. It writes each role as
`modelRoles.<role>: provider/model:effort` and accepts only its built-in roles
(`advisor`, `commit`, `plan`, `slow`, `smol`, `task`, `tiny`, `vision`); any
other role name blocks the Oh My Pi install. Other agents install only the
default route and list `roles` under "not installed for this agent";
`install --strict` blocks them instead.

## Checking a profile

```bash
mango validate ~/.profile-mango/profiles/default/profile.yaml \
  --bindings ~/.profile-mango/bindings/local.yaml
```

`validate` checks one file and its route. It does not follow `extends` or open
referenced files. `install`, `doctor`, and `preview` resolve the whole package.
