# Concepts

profile-mango keeps what you want from a coding agent in one place and writes
the parts each agent supports into that agent's own config file.

```text
profiles/review/profile.yaml         bindings/local.yaml
  description, route: local  ──────▶ routes.local: provider, model, effort
  permissions, tools,                  targets:
  instructions, skills                   claude-code: provider, model   (override)
            │                                  │
            └──────────────┬───────────────────┘
                           ▼
          mango install review --all
                           │  plan → consent → backup → write
        ┌──────────────────┼──────────────────────┐
        ▼                  ▼                      ▼
 ~/.claude/settings.json  ~/.codex/config.toml   ...each installed agent
```

## Profile

A profile is portable intent: a description, the name of a route, and optionally
roles, global instruction files, permissions, tools, instructions, and skills. It
lives at `profiles/<name>/profile.yaml`, and its name is the folder name. A
profile can `extend` another one. Profiles hold no credentials and no
machine-specific model IDs, so you can share them.

Role names are identifiers matching `^[a-z][a-z0-9-]{0,62}$`; `default` is
reserved for the route's own model. Choose names such as `coder` and `reviewer`,
or semantic names: `worker` (implementation), `planner` (planning), `research`
(read-only exploration), and `tiny` (small mechanical tasks). A profile
describes what each role does; a route's `roles:` picks its model and effort.
Codex, OpenCode, Oh My Pi, and Claude Code get one subagent file per declared
role when the profile is their default. Oh My Pi also supports binding-only
roles: model slots without generated agent files. See the
[profile reference](profile-reference.md#roles).

## Bindings, routes, and target overrides

A route is the model choice a profile points at. Routes live in a separate local
bindings file, `bindings/local.yaml`, which `init` creates and gitignores. Each
route needs a `provider`, `model`, and `effort`.

Agents don't all use the same provider. Under a route, `targets:` gives one
agent its own fields. That agent gets the base route with only those fields
replaced. The starter bindings from `init` already do this for Claude Code:

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

Bindings name a route. They never contain API keys, tokens, or account IDs.
Sign-in stays with each agent.

`mango route set` and `mango route unset` change a route's fields in place,
keeping comments; see [Editing routes](profile-reference.md#editing-routes).

## What install does

`mango install <profile>` works in two steps:

1. **Plan.** It resolves the profile and route for each target and prints a plan:
   the config path, a field-by-field diff, warnings, and a plan ID. Nothing is
   written. The plan ends with the exact command to apply it.
   The compact human view keeps every skipped requirement and the concrete
   destinations and field effects visible. Use `--verbose` for full version and
   diagnostic detail, or `--json` for the deterministic machine-readable plan.
   Undo previews retain the redacted file diff in the default view.
2. **Apply.** `--apply` asks y/N on a terminal. In scripts, use
   `--apply --yes --expect-plan <plan-id>`. If a file changed since the plan was
   made, the apply fails and writes nothing.

Choose targets with `--all` (every supported agent that is installed; others
are listed as skipped) or `--target <name>` (repeatable). Without `--config`,
each agent's standard user config path is used. Pass `--config <name>=<path>`
to write somewhere else.

**Named profiles:** for an agent with its own named profiles (Codex today),
install writes the profile under its name and leaves the agent's default
settings alone, so several profiles can sit side by side. The plan shows how to
start it, for example `use it: codex --profile review`. Add `--default` to also
write the agent's default settings. An agent without named profiles gets the
settings as its default, and the plan says so with a `note:` line. In JSON each
target has `install.mode` (`named-profile` or `default-config`), plus
`install.profileName`, `install.useCommand`, and `install.setsDefault` for a
named profile.

**What it writes:** only the settings that agent supports (see
[agents](agents.md)), plus an ownership manifest and an install journal next to
the config. Other settings in the file are kept as they are.

**What it skips:** profile requirements an agent can't install, such as
permissions, tools, instructions, or skills on most agents. They are listed per
agent as `not installed for this agent: ...` (JSON `skippedRequirements`) and
are never claimed as applied. A route effort an agent cannot write is shown as
`effort <value>: NOT APPLIED (<reason>)` (JSON `skippedRequirements` entry
`effort`). `--strict` blocks the plan instead. Unknown
profile fields and unsupported targets always block.

**What it backs up:** before changing an existing file, install makes a
create-only backup. An existing config that profile-mango doesn't own yet is
adopted on the first install. The plan shows `adopt`, and the backup is
mandatory, so `--no-backup` blocks adoption. A file you edited after a
profile-mango install is protected; `--override` replaces it only where that
agent allows it.

**Where history lives:** backups and the journals undo reads are kept in
mango's state folder, not beside your agent configs: `$PROFILE_MANGO_STATE_DIR`,
else `$XDG_STATE_HOME/profile-mango`, else `~/.local/state/profile-mango`
(`mango home --state` prints it). Only the ownership manifest
(`<config>.profile-mango.manifest.json`) stays next to the config. Backups and
journals written beside configs by earlier releases are still read by undo and
release; mango never moves or deletes them, so remove them by hand once you no
longer need them.

**Shared configs:** in a config profile-mango patches field by field (such as
Oh My Pi `config.yml` or Codex `config.toml`), the manifest records a hash of
each value it wrote. Only a changed owned value counts as your edit. Other
keys, comments, or the agent re-serializing the file (for example `omp config
set` unquoting values) do not, so a changed profile applies without
`--override` and keeps those edits; `mango status` shows the file as
`other-edits`. A manifest from an earlier mango release has no per-field
hashes, so after such a file is edited any profile change still needs
`--override` once; every install that writes the file records the hashes.

**What it never touches:** credentials and auth stores, sessions, plugins, MCP
servers, providers, and the network. Installing a model does not sign you in
or check that the model is available to your account.

## Undo

`mango undo --target <name>` (alias `restore`) previews reversing the
latest install for that target, with a diff and a new undo plan ID. Apply it
with `--apply`, or `--apply --yes --expect-plan <undo-plan-id>`. Each target of
a multi-target install is undone separately.

- An existing config comes back byte-for-byte from its backup.
- A config or profile file the install created is removed. Undoing a named
  profile install removes that profile only; earlier installs stay until you
  undo them too.
- A config edited after the install is refused unless `--override` discards
  those edits.
- Undo needs the install's backup, so it can't reverse an install made with
  `--no-backup`, or one whose history you removed from the state folder.

Use `--original-plan <id>` to undo a specific earlier install.

## Tested versions

Agents update themselves, so the installed version often differs from the one
mango was tested with. Each agent has a tested range: from the tested
version up to, but not including, the next minor release. `doctor` shows
whether each installed agent is in range. `install` prints the installed version
and warns when it is outside the range, missing, or unreadable. The plan still
proceeds, but the settings are written as they were for the tested version, and
nothing is claimed about the newer binary.

## Checking and previewing

- `mango validate <profile.yaml> --bindings <file>` checks one profile
  and its route offline.
- `mango doctor` lists agents, versions, config paths, and whether a
  profile would install. Its plan column is the status a plain
  `mango install <profile> --target <agent>` would plan. It writes nothing.
- `mango preview` (alias of `render`) writes an inert preview of a
  profile for one exact agent version into a new `--out` folder, without
  touching the agent's real files. Once staged it exits 0 and prints why
  the preview is not directly applicable as `warning` lines; `render.json`
  keeps `applicable: false` and the full diagnostics.
