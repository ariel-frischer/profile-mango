# profile-mango 🥭

**A CLI that sets up all your coding agents from one profile: models, effort, and instructions, installed into Claude Code, Codex, OpenCode, and more.**

Choose your models and effort in one place. `mango` writes them into each
agent's own config, usually as a named profile you can switch to. It shows you
the plan first, backs up what it changes, and can undo it.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/profile-mango/main/install.sh | sh
```

Or with Go:

```bash
go install github.com/ariel-frischer/profile-mango/cmd/mango@latest
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

The preview ends with the exact command to apply it. Afterwards:

```bash
codex --profile default        # each agent gets a profile you can switch to
mango status                   # which profile each agent runs
mango undo --target codex      # preview putting Codex back
```

## Change your model

```bash
mango route list                                     # routes and their models
mango route set local --model gpt-6-sol --effort medium
mango use default                                    # apply to every managed agent
```

## How it works

- **Profile:** what you want, independent of any agent: a route, optional
  subagent roles, and global instruction files. One YAML file per profile.
- **Bindings:** which provider, model, and effort each route uses, optionally
  per agent or role. They stay on your machine and never hold credentials.
- **Install:** writes what each agent supports into that agent's own config.
  Your default settings stay untouched unless you pass `--default` or run
  `mango use`. Anything an agent can't take is listed in the plan and skipped.

## Supported agents

| Agent | Installs |
| --- | --- |
| Claude Code | Model and effort as a named profile; `CLAUDE.md`; role subagents |
| Codex | Provider, model, and effort as a named profile; `AGENTS.md`; role subagents |
| OpenCode | Model, effort, and instructions as a named agent; `AGENTS.md`; role subagents |
| Pi | Provider, model, and thinking level; `~/AGENTS.md` |
| Oh My Pi | Model roles with per-role effort as a profile file; `AGENTS.md`, `RULES.md`; role subagents |
| OpenClaw | Model and thinking level as a named profile |
| Hermes | Provider, model, and effort as a named profile |

Each agent is tested against a specific version; newer versions still install,
with a warning. Credentials, sessions, plugins, and MCP are never read or
written. Details per agent: [agents](docs/public/agents.md).

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

`mango use daily-driver` then writes the role files (for example
`~/.codex/agents/research.toml`) and the global files, backing up anything it
replaces. Full example: [`examples/profiles/daily-driver`](examples/profiles/daily-driver/profile.yaml).

## Documentation

- [Concepts](docs/public/concepts.md): profiles, bindings, and what install writes, skips, and undoes.
- [Command-line details](docs/public/cli.md): plans, consent, target selection, and output.
- [Profile and bindings reference](docs/public/profile-reference.md): every field.
- [Agents](docs/public/agents.md): per-agent settings, paths, tested versions, and live-test safety.
- [Examples](examples/README.md), [roadmap](ROADMAP.md), [contributing](CONTRIBUTING.md), [all docs](docs/index.md).

## License

[MIT](LICENSE)
