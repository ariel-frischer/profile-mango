# profile-mango 🥭

**Set up your coding agents once: one profile, installed into Claude Code, Codex, OpenCode, and more.**

## What is a profile?

- **Profile:** portable intent, such as a description, a route name, and later permissions, instructions, and skills. It is a flat YAML file named by its folder.
- **Bindings:** which provider, model, and effort each route uses, with optional per-agent overrides. They stay local and never hold credentials.
- **Install:** writes the settings each agent supports into that agent's own config file. It shows a plan first, backs up what it changes, and can be undone.

## Install

```bash
go install gitlab.com/ariel-frischer/profile-mango/cmd/profile-mango@latest
```

Once the first release is tagged, you can also install a checksum-verified binary
from GitLab releases:

```bash
curl -fsSL https://gitlab.com/ariel-frischer/profile-mango/-/raw/main/install.sh | sh
```

## Quickstart

```bash
profile-mango init                      # starter profile and bindings in ~/.profile-mango
profile-mango doctor                    # which agents are installed, versions, config paths
profile-mango install default --all     # show the plan; nothing is written yet
profile-mango install default --all --apply --yes --expect-plan <plan-id>
profile-mango undo --target codex       # preview reversing that install
profile-mango undo --target codex --apply --yes --expect-plan <undo-plan-id>
```

The plan ends with the exact apply command, including its plan ID. On a terminal,
`--apply` alone asks for y/N confirmation instead. `--all` skips agents that are
not installed, and an existing config is adopted with a backup.

To change the model, edit `~/.profile-mango/bindings/local.yaml`. See the
[profile reference](docs/public/profile-reference.md).

## What installs today

Today mainly the model and route settings install. OpenCode can also take one
skill and named agents. Anything else an agent can't take, such as instructions,
permissions, tools, or skills, is listed in the plan as
`not installed for this agent: ...` and skipped. Pass `--strict` to block
instead. Real delivery of those is on the [roadmap](ROADMAP.md).

| Agent | Installs | Details |
| --- | --- | --- |
| Claude Code | Model | [agents](docs/public/agents.md#claude-code) |
| Codex | Provider, model, effort | [agents](docs/public/agents.md#codex) |
| OpenCode | Model, one skill, named agents | [agents](docs/public/agents.md#opencode) |
| Pi | Provider, model, thinking level | [agents](docs/public/agents.md#pi) |
| Oh My Pi | Model, thinking level | [agents](docs/public/agents.md#oh-my-pi) |
| OpenClaw | Model, thinking level | [agents](docs/public/agents.md#openclaw) |
| Hermes | Provider, model, effort | [agents](docs/public/agents.md#hermes) |

Each agent has a tested version range. Outside it, install still works but warns.
Credentials, sessions, plugins, and MCP are never read or written.

## Documentation

- [Concepts](docs/public/concepts.md): profiles, bindings, routes, and what install writes, skips, backs up, and undoes.
- [Profile and bindings reference](docs/public/profile-reference.md): every field and its default.
- [Agents](docs/public/agents.md): per-agent settings, default paths, tested versions, and caveats.
- [Example profiles](examples/README.md): coding, review, and docs-research workflows.
- [All docs](docs/index.md), including developer evidence and architecture.
- [Roadmap](ROADMAP.md) and [Contributing](CONTRIBUTING.md).

## License

[MIT](LICENSE)
