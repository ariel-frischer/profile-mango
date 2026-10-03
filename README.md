# profile-mango 🥭

**A CLI that sets up all your coding agents from one profile: models, effort, and instructions, installed into Claude Code, Codex, OpenCode, and more.**

Choose your models and effort in one place. `mango` writes them into each
agent's own config, usually as a named profile you can switch to. It shows you
the plan first, backs up what it changes, and can undo it.


https://github.com/user-attachments/assets/844c25fc-582e-4c18-9654-4949b437ce3c


## Install

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/profile-mango/main/install.sh | sh
```

Or with Go:

```bash
go install github.com/ariel-frischer/profile-mango/cmd/mango@latest   # Go 1.25.5+
```

Optional: let your coding agent drive `mango` for you with the agent skill:

```bash
npx skills add ariel-frischer/profile-mango --skill profile-mango -g
```

## Quickstart

```bash
mango init                     # starter profile in ~/.profile-mango
mango doctor                   # which agents you have
mango install default --all    # preview the changes; nothing is written yet
```

The preview ends with the exact command to apply it. Pi, OpenClaw and Hermes have
no separate `default` profile, so applying `default` edits their main config
(backed up; `mango undo` restores it). Afterwards:

```bash
codex --profile default        # each agent gets a profile you can switch to
mango status                   # which profile each agent runs
mango undo --target codex      # preview putting Codex back
```

## Change your model

```bash
mango route list                                     # routes and their models
mango route set local --model gpt-6-sol --effort medium
mango use default                                    # preview switching every managed agent; run the printed apply command
```

## What's a profile?

One YAML file describing how you want your agents set up. It bundles:

- **Model route:** which model and effort to use
- **Subagent roles:** worker, planner, research, tiny, each with its own model,
  plus native agent files shipped as-is
- **Global instructions:** whole files like `AGENTS.md` and `CLAUDE.md`
- **Skills:** skill folders (`SKILL.md` and its files) copied into each agent's
  skill directory
- **Permissions and tools:** sandbox, network, shell, and tool allow/deny rules
  (declared portably; no agent installs these yet)

A profile can inherit a parent's settings with `extends`. The actual provider and
model names live in a local bindings file, so profiles stay shareable and never
hold credentials. `mango` writes what each agent supports and lists the rest as
skipped.

## Supported agents

| Agent | Installs |
| --- | --- |
| Claude Code | Model and effort as a named profile; `CLAUDE.md`; role subagents; skills |
| Codex | Provider, model, and effort as a named profile; `AGENTS.md`; role subagents; skills |
| OpenCode | Model, effort, and instructions as a named agent; `AGENTS.md`; role subagents; skills |
| Pi | Provider, model, and thinking level; `~/AGENTS.md` |
| Oh My Pi | Model roles with per-role effort as a profile file; `AGENTS.md`, `RULES.md`; role subagents; skills |
| OpenClaw | Model and thinking level as a named profile; skills and per-agent skill allowlist |
| Hermes | Provider, model, and effort as a named profile |

Each agent is tested against a specific version; newer versions still install,
with a warning. Credential stores, sessions, plugins, and MCP are left alone;
config backups are full copies, so they include any keys stored inline in those
files. Details per agent: [agents](docs/public/agents.md).

## Roles and global instructions

A profile can declare subagent roles and own whole global instruction files.
Bindings pick each role's model, so the profile stays shareable:

```yaml
# profiles/daily-driver/profile.yaml
globalInstructions:
  codex: {AGENTS.md: instructions/global/AGENTS.md}
  claude-code: {CLAUDE.md: instructions/global/AGENTS.md}
roles:
  worker: {description: Implements one scoped change, instructions: instructions/roles/worker.md}
  research: {description: Read-only codebase scout}

# bindings/local.yaml
routes:
  openai:
    provider: openai
    model: gpt-6-sol
    effort: high
    roles:
      research: {provider: openai, model: gpt-6-luna, effort: medium}
```

`mango use daily-driver` then plans the role files (apply to write them; for example
`~/.codex/agents/research.toml`) and the global files, backing up anything it
replaces. `agentFiles` also ships native subagent files as-is, for example
`agentFiles: {oh-my-pi: {scout.md: agents/omp/scout.md}}` replaces Oh My Pi's
bundled `scout` agent. Full example: [`examples/profiles/daily-driver`](examples/profiles/daily-driver/profile.yaml).

### Switch Mango routes inside Oh My Pi

With omp 18.6.0, `mango use <profile> --target oh-my-pi` installs a
`mango-<route>` model preset for every binding route in global `config.yml`.
Run `/modelpreset switch mango-opus55` in omp, or use Ctrl+←/→ in
`/models` → Roles. Presets swap model roles and default thinking only:
the applied profile's task effort cap, instructions, skills and agents stay
in place. A full profile switch still uses `mango use`.
Overlay-only installs keep presets in the overlay rather than changing the
global file. See [preset ownership and overlay limits](docs/public/agents.md#model-presets-omp-1860).


## What it isn't

`mango` is not a unified configuration manager for every agent setting. It
covers only the portable profile slice: model routes, roles, global
instructions, and skills. MCP servers, credentials, sessions, plugins, keybindings,
themes, and other agent-specific settings stay in each agent's own config, and
`mango` leaves them untouched.

## Documentation

- [Concepts](docs/public/concepts.md): profiles, bindings, and what install writes, skips, and undoes.
- [Command-line details](docs/public/cli.md): plans, consent, target selection, and output.
- [Profile and bindings reference](docs/public/profile-reference.md): every field.
- [Agents](docs/public/agents.md): per-agent settings, paths, tested versions, and live-test safety.
- [Examples](examples/README.md), [roadmap](ROADMAP.md), [contributing](CONTRIBUTING.md), [all docs](docs/index.md).

## License

[MIT](LICENSE)
