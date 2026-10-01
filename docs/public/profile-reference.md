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
| `skills` | Skill folders, each a `SKILL.md` path relative to the package root, optionally with its `source`, see below. | Replaces the parent's if set |
| `globalInstructions` | Whole global instruction files per agent, each one resource or a list of fragments, see below. | A child's agent entry replaces the parent's; `{}` clears it |
| `agentFiles` | Whole native subagent files per agent, see below. | A child's agent entry replaces the parent's; `{}` clears it |
| `roles` | Portable role definitions, see below. | A child's role replaces the parent's same-name role; other parent roles are inherited |

Unknown fields are rejected. A field with the wrong YAML shape fails with its
path and the expected shape, e.g. `labels: expected a map of string keys to
string values (e.g. labels: {team: core}), got a list`. Most agents can't install permissions, tools,
instructions, or skills yet. See [agents](agents.md) for what each one takes.

### `skills`

Each `skills` entry names a skill folder's `SKILL.md`, either as a string or,
for a skill copied from another repository, with its source:

```yaml
skills:
  - skills/review/SKILL.md
  - path: skills/pdf/SKILL.md
    source:
      repo: https://github.com/anthropics/skills
      commit: 0123456789abcdef0123456789abcdef01234567
      path: skills/pdf            # optional, the folder within repo
      sha256: <tree digest of skills/pdf>
```

The folder name is the skill name: 1-64 lowercase letters, digits, or hyphens,
unique within the profile. `SKILL.md` needs YAML frontmatter with a non-empty
`description`; a `name`, when set, must equal the folder name. The whole folder
is installed: every regular file under it, at most 256, with no symlinks, and
each file keeps execute permission (`0755`) or is written `0644`. Agents that
take skills install them with `mango use` or `--default`; see
[agents](agents.md) for where each one puts them.

The folder in your package is the canonical copy; installed folders are
output and are never copied back. An install refuses to replace a skill file
mango does not own when that file differs and is newer than the package copy,
unless you pass `--override` (it is backed up first); an older one is adopted
with a backup. `mango status` lists skills whose global copy differs and which
side is newer; see [status](cli.md#status).

`source` records where a vendored skill came from; `repo` is a URL and
`commit` the full 40-character commit. Nothing is downloaded: copy the folder
into your package yourself, then pin `sha256`, the folder's tree digest. The
tree digest is the SHA-256 of one line per file, sorted by path:
`<path relative to the folder>` NUL `<0755 or 0644>` NUL `<file SHA-256 in hex>`
newline. Every `mango install` and `mango use` plan recomputes it and fails if
a file changed, naming the digest it found, so the quickest way to pin a new
copy is to set any 64-hex placeholder and copy the digest from the error
(`mango validate` checks only the entry's shape, not the files). The digest
covers the files as vendored; `{{route.…}}` placeholders are rendered only in
the installed copies. To compute
it yourself, from inside the skill folder:

```sh
find . -type f | sed 's|^\./||' | LC_ALL=C sort | while IFS= read -r f; do
  if [ -x "$f" ]; then m=0755; else m=0644; fi
  printf '%s\0%s\0%s\n' "$f" "$m" "$(sha256sum < "$f" | cut -d' ' -f1)"
done | sha256sum | cut -d' ' -f1
```

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

A file can also be a list of resources, so agents share fragments. The
installed file is the fragments in order: each one except the last loses its
trailing newlines and is followed by one blank line, and the last is written as
is. A one-item list installs the same bytes as the single path.

```yaml
globalInstructions:
  oh-my-pi:
    AGENTS.md: [instructions/shared/core.md, instructions/work/omp.md]
  codex:
    AGENTS.md: [instructions/shared/core.md, instructions/work/codex.md]
```

An empty list, or an empty or null item, fails validation with its path, such
as `globalInstructions.codex.AGENTS.md[1]`. A missing fragment fails the plan
with its path. `{{route.…}}` placeholders resolve in each fragment, and
`mango status` reports the sources changed when any fragment changes. The
installed file is still one owned file.

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

### `agentFiles`

`agentFiles` ships agent definitions in an agent's own format, including
frontmatter mango does not model. Each entry maps a file name to a resource
relative to the package root; the installer copies the bytes unchanged, apart
from [route placeholders](#route-placeholders), into the folder that holds the
agent's role files:

```yaml
agentFiles:
  oh-my-pi:
    scout.md: agents/omp/scout.md    # ~/.omp/agent/agents/scout.md
  codex:
    reviewer.toml: agents/codex/reviewer.toml
```

Only the agents in the role file table above accept them, with the same
extension: `.toml` for Codex, `.md` for the others. The wrong extension blocks
the install. A file name must be plain (no `/`). A name that matches an agent's
bundled subagent overrides it where the agent lets user files win; Oh My Pi,
for example, loads `~/.omp/agent/agents/scout.md` over its bundled `scout`. A
name that matches a role in the same profile (`research.md` with
`roles.research`) fails validation, since both would write the same file.

Ownership matches global instruction files: an existing file is adopted with a
create-only backup, an edited file shows as `edited` with kind `agent-file` in
`mango status` and needs `--override`, `mango use` of a profile without the
file deletes it or restores the original, and `mango undo` reverses each step.
Agent files are global, so a named-only install lists them as `agentFiles`
under "not installed for this agent"; Pi, Hermes, OpenClaw, and the jcode fork
always skip them, and `install --strict` blocks instead.

### Route placeholders

Resources mango delivers (`instructions`, the text files in each `skills`
folder, `globalInstructions`, `agentFiles`, and role `instructions`) may name a
route field instead of repeating a provider or model:

```markdown
Delegate reviews to {{route.opus55.roles.research.model}}.
You run on {{route.opus55.provider}}/{{route.opus55.model}}.
```

`{{route.<name>.<field>}}` and `{{route.<name>.roles.<role>.<field>}}` take a
field from `provider`, `model`, `effort`, `transport`, and `authentication`.
Each agent gets the value from its own effective route, so a `targets` override
for Codex changes only the Codex copy. A role sets only `provider`, `model`, and
optionally `effort`; naming another role field, or an effort the role does not
set, is an error. Any route in the bindings file may be named, not only the
profile's own.

Only text starting `{{route.` is read, and a placeholder must close with `}}` on
the same line. Everything else, including other `{{…}}`, stays literal; there
are no conditionals, loops, or escapes. An unknown route, role, or field, or a
malformed placeholder, stops `install`, `use`, `doctor`, and `render` with the
resource path, line, and reason (`resource.route_placeholder_invalid`).
`mango validate` does not open resources, so it does not check them.

Hashes, plans, and `mango status` drift compare the rendered bytes, so an
unchanged install stays clean, and changing a route field a placeholder names
shows the file as drift until you apply it again.

### Older wrapped format

Profiles written as `apiVersion` / `kind` / `metadata` / `spec` (with
`spec.routeRef`) still load, with a deprecation warning. Move the fields to the
top level and rename `routeRef` to `route`.

The wrapped format does not accept `roles` or `agentFiles`.

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
`transport`, `authentication`, and `roles`. That agent gets the base route with only
those fields replaced, and other agents use the base route. This is how one
profile drives several agents with different providers.

`targets.<agent>.roles.<role>` sets one or more of `provider`, `model`, and
`effort` for a role the route already binds under [`roles`](#roles); that
agent's role keeps the other base fields. A role missing from the base `roles`
fails with its path. For example, the same ChatGPT subscription is provider
`openai-codex` in Oh My Pi but `openai` in Codex:

```yaml
routes:
  luna:
    provider: openai-codex
    model: gpt-6-sol
    effort: high
    roles:
      research:
        provider: openai-codex
        model: gpt-6-luna
    targets:
      codex:
        provider: openai
        roles:
          research:
            provider: openai
```

Some agents need specific route values. For example, Claude Code installs only
with an `anthropic` provider, and Codex only with `openai`, native transport,
OAuth, and an effort from `none` to `xhigh`. The plan says why when a route doesn't fit. See [agents](agents.md).

### `roles`

`roles` gives portable roles (`worker`, `planner`, `research`, `tiny`) their
own `provider`, `model`, and optional `effort`. The route's own fields are the
default model, and `targets.<agent>.roles` changes a role for one agent (see
[`targets` overrides](#targets-overrides)). Any other role name
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
mango route set luna --target codex --role research --provider openai
mango route set opus55 --subagent-max-effort high  # whole route only
mango route unset sol --target oh-my-pi effort     # fields: provider, model, effort, subagent-max-effort
```

`set` changes only the affected lines and adds a missing `targets.<agent>`,
`roles.<role>`, or, with both `--target` and `--role`,
`targets.<agent>.roles.<role>` entry. `unset` removes an entry, and then each
enclosing `roles`, target, or `targets` map, once it is empty. Both check the
edited file the way `install` does and write nothing if it is invalid or
unchanged. An effort no supported agent uses prints a warning, since
each agent checks its own efforts at install. They print the diff and the
profiles using the route; agent config never changes until you apply one. After
writing, they read the ownership manifests at the default config paths (read-only,
as `mango status` does) and print, for each profile, one
`mango install <profile> --target <a> --target <b>` naming only the agents whose
recorded profile is that one, with `--default` where it is their default. Agents
on other profiles are never listed, so following the hint cannot switch them. A
profile no agent records gets the adopt form
`mango install <profile> --target <target>` instead. `mango status` names its
changed-sources fix the same way, for that one agent; when the profile dropped a
file the agent still owns, it suggests `mango use <profile> --target <agent>`,
which gives the file back.

When a change replaces a provider or model value, `set` and `unset` (dry runs
included) also list where the profiles using the route still spell the old
value: their `profile.yaml` files and every resource they deliver, as
`file:line` relative to the package root (`--resource-root`, default the profile
home). Text inside `{{route.…}}` placeholders is ignored, and a longer name that
starts with the old value, such as `gpt-6-sol-mini` for `gpt-6-sol`, does not
count:

```text
Still names gpt-6-sol:
  instructions/daily.md:2
  profiles/daily/profile.yaml:2
```

Replace those with [route placeholders](#route-placeholders) so the next change
needs no hand edits.
New routes are still added by hand.

## Checking a profile

```bash
mango validate ~/.profile-mango/profiles/default/profile.yaml \
  --bindings ~/.profile-mango/bindings/local.yaml
```

`validate` checks one file and its route. It does not follow `extends` or open
referenced files. `install`, `doctor`, and `preview` resolve the whole package.
