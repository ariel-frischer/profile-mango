# Agents

What `profile-mango install` writes for each agent, where, and for which
versions. Everything not listed for an agent is skipped and shown in the plan as
`not installed for this agent: ...`, or blocks with `--strict`. No agent gets
authentication, permissions, tools, plugins, MCP, or runtime enforcement.

Each agent has one tested version. The tested range runs from that version up
to the next minor release. Outside the range, or when the agent isn't found,
install warns but still plans. See [concepts](concepts.md#tested-versions).
`--config <agent>=<path>` replaces any default path below.

| Agent | Tested version | Tested range | Default config path |
| --- | --- | --- | --- |
| Claude Code | `2.1.278` | `>=2.1.278 <2.2.0` | `~/.claude/profiles/<profile>.json` next to `~/.claude/settings.json` |
| Codex | `0.154.0` | `>=0.154.0 <0.155.0` | `$CODEX_HOME/<profile>.config.toml` next to `$CODEX_HOME/config.toml`, default `~/.codex/` |
| OpenCode | `1.18.31` | `>=1.18.31 <1.19.0` | `${XDG_CONFIG_HOME:-~/.config}/opencode/opencode.json` |
| Pi | `0.86.1` | `>=0.86.1 <0.87.0` | `~/.pi/agent/settings.json` (or `$PI_CODING_AGENT_DIR`) |
| Oh My Pi | `18.2.6` | `>=18.2.6 <18.3.0` | `~/.omp/agent/config.yml` |
| OpenClaw | `2026.9.5` | `>=2026.9.5 <2026.10.0` | `~/.openclaw/openclaw.json` (or `$OPENCLAW_CONFIG_PATH`) |
| Hermes | `0.21.3` | `>=0.21.3 <0.22.0` | `~/.hermes/config.yaml` (or `$HERMES_HOME`) |

The evidence behind every row is in the developer
[target evidence ledger](../dev/target-evidence.md).

## Claude Code

- **Installs:** the top-level `model` in an emulated named profile:
  `profiles/<profile>.json` next to `settings.json`. Start it with
  `claude --settings <path>`, printed by the plan as `use it:`. `settings.json`
  is not changed unless you pass `--default`, which also writes `model` there
  so plain `claude` uses it. Claude Code has no native named profiles; this is
  a profile-mango file-placement convention over the qualified `--settings`
  consumption.
- **Route:** needs provider `anthropic` and native transport. The `init`
  starter bindings already include a `claude-code` override for this.
- **Caveats:** effort, provider, and sign-in are not set. Project, managed, or
  command-line settings can override the user file. An existing `settings.json`
  must not be empty.
- [Reference](../dev/agents/claude-code.md)

## Codex

- **Installs:** a native Codex profile: `model_provider`, `model`, and
  `model_reasoning_effort` in `<profile>.config.toml` next to `config.toml`.
  Start it with `codex --profile <profile>`. `config.toml` is not changed
  unless you pass `--default`, which also writes the same three settings there
  so plain `codex` uses them. Each profile you install gets its own file.
- **Blocks:** a legacy `profile = ...` line, or a `[profiles.<profile>]` table
  with the same name, in `config.toml`. Codex 0.154.0 refuses to start with
  either one, so remove or move it first. Other `[profiles.*]` tables are kept.
- **Route:** needs provider `openai`, `high` effort, native transport, and
  OAuth. Any other route blocks.
- **Caveats:** profile-mango never writes or reads login data, and installing
  doesn't prove you're signed in with OAuth. Run `codex login status` yourself.
  It tells an API key apart from a ChatGPT login, but not Codex-managed OAuth
  from externally supplied tokens. Don't share its output. Trusted project
  config and `-c` overrides can replace these values at run time. Codex
  treats a `--profile` name with no file as empty, so a mistyped name silently
  runs your default settings. Model availability and effort enforcement haven't
  been verified.
- [Reference](../dev/agents/codex.md)

## OpenCode

- **Installs:** a named primary agent definition, `agents/<profile>.md` next
  to `opencode.json` (or `opencode.jsonc`), with the route's model and the
  profile's instructions in order. The instructions replace that agent's
  built-in prompt. Start it with `opencode --agent <profile>`. `opencode.json`
  is not changed unless you pass `--default`.
- **`--default` also writes:** the top-level `model` (as `provider/model`)
  in `opencode.json`, plus, if the profile has exactly one skill, a
  `SKILL.md` copied next to the config with that folder added to
  `skills.paths`. Skills are a directory-wide setting in OpenCode, not
  per-agent, which is why they only install alongside `--default`. Removing
  the skill from the profile later removes only an unchanged skill file
  profile-mango created; other skills in that folder can still be found, so
  this is not an allowlist.
- **Explicit agents:** `--agent opencode=primary:<name>` or
  `subagent:<name>`, with `--config opencode=<path>` ending in
  `agents/<name>.md`, writes the same kind of definition at that path
  instead of the default one. A primary agent becomes selectable, not
  active, and a subagent becomes available for delegation. Neither enforces
  permissions.
- **Caveats:** effort and provider options are not set. Edited or unowned
  skill and agent files are never overwritten, even with `--override`.
- [Reference](../dev/agents/opencode.md)

## Pi

- **Installs:** `defaultProvider`, `defaultModel`, and `defaultThinkingLevel` in
  the global `settings.json`.
- **Caveats:** a project `.pi/settings.json` overrides the global file.
- [Reference](../dev/agents/pi.md)

## Oh My Pi

- **Installs:** `modelRoles.default` (as `provider/model`) and
  `defaultThinkingLevel` in `config.yml`.
- **Caveats:** other model roles and provider options are left alone.
- [Reference](../dev/agents/oh-my-pi.md)

## OpenClaw

- **Installs:** a native OpenClaw profile: `agents.defaults.model.primary`
  (as `provider/model`) and `agents.defaults.thinkingDefault` in
  `~/.openclaw-<profile>/openclaw.json`. Start it with
  `openclaw --profile <profile>`. `~/.openclaw/openclaw.json` is not changed
  unless you pass `--default`, which also writes the same two settings there.
  A profile named `default` is OpenClaw's default config, so it is written in
  place.
- **Blocks:** named installs when the main config isn't at
  `<home>/.openclaw/openclaw.json` (a relocated `--config` or
  `OPENCLAW_CONFIG_PATH`). The profile location follows OpenClaw's home, not
  the config file. If you set `OPENCLAW_HOME`, pass
  `--config $OPENCLAW_HOME/.openclaw/openclaw.json`.
- **Caveats:** a new profile has its own state folder, with no sign-in,
  sessions, or plugins from your default profile. profile-mango doesn't copy
  them. An exported `OPENCLAW_CONFIG_PATH` or `OPENCLAW_STATE_DIR` still wins
  over `--profile`. Fallbacks, per-agent overrides, and provider options are
  left alone. A per-agent model setting can still win over the default.
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
- [Reference](../dev/agents/hermes.md)

## Not supported

Other agents are not supported yet; see the [roadmap](../../ROADMAP.md).
