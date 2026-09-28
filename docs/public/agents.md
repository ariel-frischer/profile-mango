# Agents

What `mango install` writes for each agent, where, and for which
versions. Everything not listed for an agent is skipped and shown in the plan as
`not installed for this agent: ...`, or blocks with `--strict`. No agent gets
authentication, permissions, tools, plugins, MCP, or runtime enforcement.

Each agent has one tested version. The tested range runs from that version up
to the next minor release. Outside the range, or when the agent isn't found,
install warns but still plans. See [concepts](concepts.md#tested-versions).
`--config <agent>=<path>` replaces any default path below.

## Before a live test

**A working install does not mean a free or OAuth-backed model call.** Mango
writes settings, not credentials, and cannot verify the effective account or
billing route. Before sending any prompt, check the plan's `use it:` command and
destination, the agent's selected provider/model, its active authentication
method, any environment/project/CLI overrides or fallbacks, and the provider's
billing terms or spend limit. Inspect credential status privately: never paste
tokens, full auth files, or unredacted status output. If any of these are unclear,
skip the live test. OAuth or a subscription in one agent does not establish how
another agent is billed. A successful reply still does not prove its billing route.

| Agent | Check before sending a prompt |
| --- | --- |
| [Codex](#codex) | Inspect `<profile>.config.toml` and use `codex --profile <profile>`. Check `codex login status` yourself, but its ChatGPT result cannot prove exact OAuth or rule out external tokens; project config and `-c` can override the model. Do not share status output. |
| [OpenCode](#opencode) | Inspect `agents/<profile>.md` and select it with `--agent`. Check the actual provider account and billing separately; project, explicit, inline, and managed config may override the route. Effort is set as the agent's model `variant` and applies only if that model has a variant of that name; Mango does not authenticate OpenCode. |
| [OpenClaw](#openclaw) | Inspect `~/.openclaw-<profile>/openclaw.json`. Verify profile-specific sign-in, `OPENCLAW_CONFIG_PATH`/`OPENCLAW_STATE_DIR`, per-agent model overrides and fallbacks, then the account's billing. The default profile's credentials are not copied. |
| [Hermes](#hermes) | Inspect `~/.hermes/profiles/<profile>/config.yaml`. Verify sign-in for that profile, CLI/environment overrides and provider fallbacks, then the account's billing. The default profile's credentials are not copied. |
| [Claude Code](#claude-code) | Check the plan's `--settings` path, other active settings layers, the agent's login method and account billing. Mango installs only the model and effort. |
| [Pi](#pi) | Check global and project settings for the effective provider/model, then the agent's active credentials and provider billing. Project settings can win. |
| [Oh My Pi](#oh-my-pi) | Inspect `profiles/<profile>.yml` and use `omp --config <path>`. Check the default model role and any other role selected for the test, then active credentials and provider billing. |

To confirm the files match the profile without calling a model, repeat the
install without `--apply`: `unchanged` means the installed files match the plan,
not that an agent used that route. After the checks above, a minimal live test
for a profile named `sol-daily`:

```bash
codex --profile sol-daily
opencode run --agent sol-daily "Reply with exactly OK."
hermes -p sol-daily -z "Reply with exactly OK."
openclaw --profile sol-daily agent --local --message "Reply with exactly OK." --timeout 30
```

Each prompt makes a model call and may create a session. A reply does not
prove which model served it; check the agent's own model or session details, and
do not override model or thinking while testing. An effort an agent cannot
apply shows in the plan as `effort <value>: NOT APPLIED`.

| Agent | Tested version | Tested range | Default config path |
| --- | --- | --- | --- |
| Claude Code | `2.1.278` | `>=2.1.278 <2.2.0` | `~/.claude/profiles/<profile>.json` next to `~/.claude/settings.json` |
| Codex | `0.157.1` | `>=0.157.1 <0.158.0` | `$CODEX_HOME/<profile>.config.toml` next to `$CODEX_HOME/config.toml`, default `~/.codex/` |
| OpenCode | `1.18.31` | `>=1.18.31 <1.19.0` | `${XDG_CONFIG_HOME:-~/.config}/opencode/opencode.json` |
| Pi | `0.87.1` | `>=0.87.1 <0.88.0` | `~/.pi/agent/settings.json` (or `$PI_CODING_AGENT_DIR`) |
| Oh My Pi | `18.3.2` | `>=18.3.2 <18.4.0` | `~/.omp/agent/config.yml` |
| OpenClaw | `2026.9.5` | `>=2026.9.5 <2026.10.0` | `~/.openclaw/openclaw.json` (or `$OPENCLAW_CONFIG_PATH`) |
| Hermes | `0.21.3` | `>=0.21.3 <0.22.0` | `~/.hermes/config.yaml` (or `$HERMES_HOME`) |

The evidence behind every row is in the developer
[target evidence ledger](../dev/target-evidence.md).

## Claude Code

- **Installs:** the top-level `model` and, for `low`, `medium`, `high`, or
  `xhigh` effort, `effortLevel` in an emulated named profile:
  `profiles/<profile>.json` next to `settings.json`. Start it with
  `claude --settings <path>`, printed by the plan as `use it:` (the file
  actually written, also beside an explicit `--config`). `settings.json`
  is not changed unless you pass `--default`, which also writes these there
  so plain `claude` uses them. Claude Code has no native named profiles; this is
  a profile-mango file-placement convention over the qualified `--settings`
  consumption.
- **Route:** needs provider `anthropic` and native transport. The `init`
  starter bindings already include a `claude-code` override for this.
- **Caveats:** provider and sign-in are not set. Other efforts (such as `max`)
  are shown in the plan as `effort <value>: NOT APPLIED`. Project, managed, or
  command-line settings, `--effort`, and `CLAUDE_CODE_EFFORT_LEVEL` can override
  the user file. An existing `settings.json`
  must not be empty.
- **Skills:** with `mango use` or `--default`, each `skills` folder is
  copied whole to `~/.claude/skills/<name>/`. A named profile lists skills
  as not installed. Removing a skill later deletes the files profile-mango
  wrote, or restores a file it replaced.
- **Global instructions:** `globalInstructions` can own `~/.claude/CLAUDE.md`
  (documentation-backed only), written with `mango use` or `--default`.
- **Roles:** with `mango use` or `--default`, each profile role becomes
  `~/.claude/agents/<role>.md` with its bound `model` (provider `anthropic`
  only) and `effort` (`low` to `max`). Checked against installed `2.1.280` and
  `2.1.281` binaries, not `2.1.278` itself.
- **Agent files:** `agentFiles.claude-code` copies native `.md` subagent files
  unchanged to `~/.claude/agents/`, with `mango use` or `--default`.
- [Reference](../dev/agents/claude-code.md)

## Codex

- **Installs:** a native Codex profile: `model_provider`, `model`, and
  `model_reasoning_effort` in `<profile>.config.toml` next to `config.toml`.
  Start it with `codex --profile <profile>`. `config.toml` is not changed
  unless you pass `--default`, which also writes the same three settings there
  so plain `codex` uses them. Each profile you install gets its own file.
- **Blocks:** a legacy `profile = ...` line, or a `[profiles.<profile>]` table
  with the same name, in `config.toml`. Codex 0.157.1 refuses to start with
  either one, so remove or move it first. Other `[profiles.*]` tables are kept.
- **Route:** needs provider `openai`, native transport, OAuth, and effort
  `none`, `minimal`, `low`, `medium`, `high`, or `xhigh`. Any other route blocks.
- **Caveats:** profile-mango never writes or reads login data, and installing
  doesn't prove you're signed in with OAuth. Run `codex login status` yourself.
  It tells an API key apart from a ChatGPT login, but not Codex-managed OAuth
  from externally supplied tokens. Don't share its output. Trusted project
  config and `-c` overrides can replace these values at run time, and so can
  an admin-managed `requirements.toml` provider setting. Codex
  treats a `--profile` name with no file as empty, so a mistyped name silently
  runs your default settings. Model availability and effort enforcement haven't
  been verified.
- **Skills:** with `mango use` or `--default`, each `skills` folder is
  copied whole to `~/.agents/skills/<name>/` (under your home directory, not
  `CODEX_HOME`; Oh My Pi and OpenCode read it too). A named profile lists
  skills as not installed. Removing a skill later deletes the files
  profile-mango wrote, or restores a file it replaced.
- **Global instructions:** `globalInstructions` can own `$CODEX_HOME/AGENTS.md`,
  written with `mango use` or `--default`. An existing `AGENTS.override.md`
  is read instead of it.
- **Roles:** with `mango use` or `--default`, each profile role becomes
  `~/.codex/agents/<role>.toml` with `developer_instructions` and its bound
  model and effort (`none` to `xhigh`; others are `NOT APPLIED`).
- **Agent files:** `agentFiles.codex` copies native `.toml` role files
  unchanged to `~/.codex/agents/`, with `mango use` or `--default`.
- [Reference](../dev/agents/codex.md)

## OpenCode

- **Installs:** a named primary agent definition, `agents/<profile>.md` next
  to `opencode.json` (or `opencode.jsonc`), with the route's model and the
  profile's instructions in order, and the effort as the agent's model
  `variant`. The instructions replace that agent's
  built-in prompt. Start it with `opencode --agent <profile>`. `opencode.json`
  is not changed unless you pass `--default`.
- **`--default` also writes:** the top-level `model` (as `provider/model`)
  in `opencode.json`. It also removes the `SKILL.md` and `skills.paths`
  entry that older profile-mango versions wrote next to the config.
- **Explicit agents:** `--agent opencode=primary:<name>` or
  `subagent:<name>`, with `--config opencode=<path>` ending in
  `agents/<name>.md`, writes the same kind of definition at that path
  instead of the default one. A primary agent becomes selectable, not
  active, and a subagent becomes available for delegation. Neither enforces
  permissions.
- **Caveats:** OpenCode applies the effort `variant` only when the agent's
  model defines a variant of that name, and otherwise silently uses the model
  default; the `--default` top-level model carries no effort. Efforts with no
  built-in variant (such as `ultra`) are shown as `effort <value>: NOT APPLIED`.
  Provider options are not set. Edited or unowned agent files are never
  overwritten, even with `--override`.
- **Skills:** with `mango use` or `--default`, each `skills` folder is
  copied whole to `~/.config/opencode/skills/<name>/`; every skill's
  `SKILL.md` must set `name`. A named profile lists skills as not installed.
  Removing a skill later deletes the files profile-mango wrote, or restores
  a file it replaced.
- **Global instructions:** `globalInstructions` can own
  `~/.config/opencode/AGENTS.md`, written with `mango use` or `--default`.
  OpenCode also reads `~/.claude/CLAUDE.md`.
- **Roles:** with `mango use` or `--default`, each profile role becomes a
  subagent, `~/.config/opencode/agents/<role>.md`, with its bound model and
  effort `variant`.
- **Agent files:** `agentFiles.opencode` copies native `.md` agent files
  unchanged to `~/.config/opencode/agents/`, with `mango use` or `--default`.
- [Reference](../dev/agents/opencode.md)

## Pi

- **Installs:** `defaultProvider`, `defaultModel`, and `defaultThinkingLevel` in
  the global `settings.json`.
- **Caveats:** a project `.pi/settings.json` overrides the global file.
- **Global instructions:** `globalInstructions.home` can own `~/AGENTS.md`,
  written by every Pi install (Pi has no named profiles). Pi reads it in every
  folder under your home directory; an existing `~/AGENTS.override.md` is read
  instead. The file is shared: Codex, OpenCode, Oh My Pi, and Claude Code read
  it only in some folders (for example outside a Git repository), so
  profile-mango writes it only when Pi is part of the install.
- [Reference](../dev/agents/pi.md)

## Oh My Pi

- **Installs:** `modelRoles.default` as `provider/model:effort`, plus the
  model slots of each bound portable role (`worker`: `task`; `planner`:
  `plan`, `slow`; `research`: `smol`; `tiny`: `tiny`, `commit`), and
  `task.maxEffort` from `subagentMaxEffort`, in an emulated named profile:
  the overlay `~/.omp/agent/profiles/<profile>.yml`. Start it with
  `omp --config <path>`, printed by the plan as `use it:`. `config.yml` is not
  changed unless you pass `--default`, which also writes these there so plain
  `omp` uses them. Effort stays on each selector; `defaultThinkingLevel`, which
  applies to models you pick by hand, is left alone. Native `omp --profile`
  moves sign-in and sessions too, so it is not used.
- **Roles:** with `--default` or `mango use`, each profile role becomes
  `~/.omp/agent/agents/<role>.md`, whose `model: "@<slot>"` resolves the slot
  above, so model and effort come from the active config.
- **Agent files:** `agentFiles.oh-my-pi` copies native `.md` agent files
  unchanged to `~/.omp/agent/agents/`, with `mango use` or `--default`. A
  file named like a bundled agent, such as `scout.md`, replaces it.
- **Skills:** with `mango use` or `--default`, each `skills` folder is
  copied whole to `~/.omp/agent/skills/<name>/`. A named profile lists
  skills as not installed. Removing a skill later deletes the files
  profile-mango wrote, or restores a file it replaced.
- **Global instructions:** `globalInstructions` can own `AGENTS.md` and
  `RULES.md` in `~/.omp/agent`, written with `mango use` or `--default`.
- **Caveats:** the overlay wins over global and project config; slots it does
  not set still come from them, and command-line `--model`/`--thinking` still
  win. Omp refuses to start if the `--config` file is missing, so undoing the
  profile breaks aliases that point at it. Unbound model slots (such as
  `memory`, `image`, `web`, `speech`, `dictation`, and `judge`) and provider
  options are left alone; an unset `memory` slot falls back to `tiny`. A legacy
  `providers.tinyModel` in the same file is put in front of the `tiny` slot
  when Oh My Pi loads it. In `config.yml`, a slot or `task.maxEffort` a later
  profile no longer sets gets its pre-install value back, or is removed if
  profile-mango added it.
- [Reference](../dev/agents/oh-my-pi.md)

## OpenClaw

- **Installs:** a native OpenClaw profile: `agents.defaults.model.primary`
  (as `provider/model`) and `agents.defaults.thinkingDefault` in
  `~/.openclaw-<profile>/openclaw.json`. Start it with
  `openclaw --profile <profile>`. `~/.openclaw/openclaw.json` is not changed
  unless you pass `--default`, which also writes the same two settings there.
  A profile named `default` is OpenClaw's default config, so it is written in
  place.
- **Skills:** each `skills` folder is copied whole to the `skills` folder of
  every config written: `~/.openclaw-<profile>/skills/<name>/`, and
  `~/.openclaw/skills/<name>/` with `--default` or `mango use`. Removing a
  skill later deletes the files profile-mango wrote, or restores a file it
  replaced.
- **Agent allowlist:** `--agent openclaw=<id>` also sets
  `agents.entries.<id>.skills` to the profile's skill names, which replaces
  `agents.defaults.skills` for that agent. The agent must already be defined
  in a config the install writes (the main config needs `--default`).
  Later installs keep that allowlist in step; `--agent` with another id, or a
  profile without skills, puts the previous list back.
- **Blocks:** named installs when the main config isn't at
  `<home>/.openclaw/openclaw.json` (a relocated `--config` or
  `OPENCLAW_CONFIG_PATH`). The profile location follows OpenClaw's home, not
  the config file. If you set `OPENCLAW_HOME`, pass
  `--config $OPENCLAW_HOME/.openclaw/openclaw.json`.
- **Caveats:** a new profile has its own state folder, with no sign-in,
  sessions, or plugins from your default profile. profile-mango doesn't copy
  them. An exported `OPENCLAW_CONFIG_PATH` or `OPENCLAW_STATE_DIR` still wins
  over `--profile`, and skills are then read from that state folder. Fallbacks,
  other per-agent overrides, and provider options are left alone. A per-agent
  model setting can still win over the default.
- [Reference](../dev/agents/openclaw.md)

## Hermes

- **Installs:** a native Hermes profile: `model.provider`, `model.default`,
  and `agent.reasoning_effort` in
  `<hermes-home>/profiles/<profile>/config.yaml`. Start it with
  `hermes -p <profile>`. `~/.hermes/config.yaml` is not changed unless you
  pass `--default`, which also writes the same three settings there. A
  profile named `default` is Hermes' own default config, so it is written in
  place.
- **Caveats:** a new profile has its own state directory, with no memories,
  sessions, skills, or credentials from your default profile. profile-mango
  doesn't copy them. Reserved Hermes profile names (`hermes`, `test`, `tmp`,
  `root`, `sudo`) are blocked, matching what `hermes -p` itself refuses.
  `hermes -p` only looks under `$HERMES_HOME` (or `~/.hermes`), so a
  `--config` elsewhere is blocked for any profile other than `default`. Undo
  removes the profile directory again if install created it and it is empty.
- [Reference](../dev/agents/hermes.md)

## Not supported

Other agents are not supported yet; see the [roadmap](../../ROADMAP.md).

`~/AGENTS.md` is written only through Pi, see [Pi](#pi). Without Pi in the
install, other agents list it under "not installed for this agent".
