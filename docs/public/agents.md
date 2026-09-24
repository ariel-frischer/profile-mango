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

- **Installs:** the top-level `model` (as `provider/model`) in `opencode.json`.
  If only `opencode.jsonc` exists, that file is used; if both exist, install
  blocks.
- **One skill:** a profile with exactly one skill gets it copied to a
  `SKILL.md` next to the config, and that folder is added to `skills.paths`.
  Removing the skill from the profile later removes only an unchanged skill
  file that profile-mango created. Other skills in those folders can still be
  found, so this is not an allowlist.
- **Named agents:** `--agent opencode=primary:<name>` or
  `subagent:<name>`, with `--config opencode=<path>` ending in
  `agents/<name>.md`, writes a named agent definition with the route's model
  and the profile's instructions in order. The instructions replace that
  agent's built-in prompt. A primary agent becomes selectable, not active, and
  a subagent becomes available for delegation. Neither enforces permissions.
  Named agents have no default path.
- **Caveats:** effort and provider options are not set. Edited or unowned skill
  and agent files are never overwritten, even with `--override`.
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

- **Installs:** `agents.defaults.model.primary` (as `provider/model`) and
  `agents.defaults.thinkingDefault` in `openclaw.json`.
- **Caveats:** fallbacks, per-agent overrides, and provider options are left
  alone. A per-agent model setting can still win over the default.
- [Reference](../dev/agents/openclaw.md)

## Hermes

- **Installs:** `model.provider`, `model.default`, and `agent.reasoning_effort`
  in `config.yaml`.
- [Reference](../dev/agents/hermes.md)

## Not supported

Other agents are not supported yet; see the [roadmap](../../ROADMAP.md).
