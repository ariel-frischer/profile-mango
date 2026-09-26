# profile-mango 🥭

**Set up your coding agents once: one profile, installed into Claude Code, Codex, OpenCode, and more.**

## What is a profile?

- **Profile:** bundled agentic configuration. A profile describes portable intent: a description, a route name, optional roles (worker, planner, research, tiny), global instruction files (`globalInstructions`), and later permissions, instructions, and skills. Each profile is a flat YAML file inside its own folder.
- **Bindings:** which provider, model, and effort each route uses, with optional per-agent overrides and a model per role. They stay local and never hold credentials.
- **Install:** writes the settings each agent supports into that agent's own config. Where the agent has named profiles, native (Codex, OpenClaw, Hermes, OpenCode agents) or emulated (Claude Code), the profile is installed under its own name and your default settings are left alone unless you pass `--default`. It shows a plan first, backs up what it changes, and can be undone.

## Install

```bash
go install gitlab.com/ariel-frischer/profile-mango/cmd/profile-mango@latest
```

Once the first release is tagged, you can also install a checksum-verified binary
from GitHub releases:

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/profile-mango/main/install.sh | sh
```

Install the agent skill so your coding agent can drive `mango` for you:

```bash
npx skills add ariel-frischer/profile-mango --skill profile-mango -g
```

## Quickstart

```bash
mango init                      # starter profile and bindings in ~/.profile-mango
mango doctor                    # which agents are installed, versions, config paths
mango install default --all     # show the plan; nothing is written yet
mango install default --all --apply --yes --expect-plan <plan-id>
codex --profile default         # Codex gets a named profile; the plan prints this line
mango undo --target codex       # preview reversing that install
mango undo --target codex --apply --yes --expect-plan <undo-plan-id>
mango use <profile>             # switch managed agents to another profile (same plan/apply)
mango status                    # which profile each agent runs, and edited or stale files
mango route list                # routes, their models, and the profiles using them
mango route set local --effort medium   # change a route in bindings/local.yaml
```

The plan ends with the exact apply command, including its plan ID. On a terminal,
`--apply` alone asks for y/N confirmation instead. `--all` skips agents whose
config folder does not exist, and an existing config is adopted with a backup.
Human plans show each agent's status, resolved destination, installed model and
effort when known, use command, file and field effects, every skipped requirement,
and actionable warnings. They end with target and file counts. Add `--verbose`
for full version and diagnostic detail (plus undo hashes). `undo` also shows its
redacted file diff by default and prints a per-target apply command. `--json`
remains the unchanged machine-readable contract. Colors appear only on terminals
and can be disabled with `--no-color`, `NO_COLOR`, or `TERM=dumb`.

Use `-t` as shorthand for `--target`. Install, undo, doctor, and `agents check`
accept comma-separated or repeated target flags, for example
`mango install default -t codex,opencode` or
`mango doctor -t codex -t opencode`. Repeated entries are deduplicated.
`render`/`preview` accept the same selector syntax but require exactly one
unique target and one `--out` directory. Multi-target `undo` previews one plan
per target (`--json` returns an array); apply each target separately with its
own `--expect-plan` ID. A shared consent cannot partially undo multiple agents.
`--config target=path`, `--manifest target=path`, and `--agent target=value`
remain repeatable mappings; commas inside their values are not split.
`agents check` selects documentation-source IDs, not version-qualified adapter
installations; checking a source is not a claim of version compatibility.

The command is `mango`; `profile-mango` keeps working as a compatibility alias
installed alongside it. If you already use MangoWC's unrelated `mango`
compositor CLI, keep invoking `profile-mango` (or adjust `PATH` ordering) to
avoid the name collision.

To change the model, run `mango route set <route> --model <m>` (add
`--target <agent>` or `--role <role>` to change one agent or role). It edits
`~/.profile-mango/bindings/local.yaml` in place, keeps comments, checks the
result, and prints the diff; then run `mango use <profile>` to apply it. See the
[profile reference](docs/public/profile-reference.md).

## Check an installed profile

Repeat `mango install <name> -t codex,opencode,openclaw,hermes` without `--apply`:
`unchanged` confirms the installed files match the plan, not that an agent used
that route. The plan prints each agent's selection command. For `sol-daily`:

**Warning: a smoke test can incur paid API charges.** Mango does not check your
sign-in, API keys, effective configuration, or billing. Before running a test,
follow the [per-agent cost and authentication preflight](docs/public/agents.md#before-a-live-test).
If you cannot verify the route and who pays for it, do not send a prompt.

```bash
codex --profile sol-daily
opencode run --agent sol-daily "Reply with exactly OK."
hermes -p sol-daily -z "Reply with exactly OK."
openclaw --profile sol-daily agent --local --message "Reply with exactly OK." --timeout 30
```

The last three make a model call and may create a session. A Codex prompt does too.
A reply alone does not prove which model or authentication route served it; check the
agent's own model/session details. Do not override model or thinking while testing.
OpenClaw and Hermes named profiles may need separate sign-in. Version warnings
mean native behavior is unqualified. An effort an agent cannot apply is shown as
`effort <value>: NOT APPLIED`.

## Roles and global instructions

A profile can declare subagent **roles** (`worker`, `planner`, `research`,
`tiny`) and own whole **global instruction files**. Bindings pick each role's
model, so the profile stays shareable:

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

`mango use daily-driver` then writes `~/.codex/agents/research.toml`,
`~/.claude/agents/worker.md`, OpenCode and Oh My Pi `agents/<role>.md`, and the
global files, backing up anything it replaces. See
[`examples/profiles/daily-driver`](examples/profiles/daily-driver/profile.yaml),
[`examples/bindings/local.example.yaml`](examples/bindings/local.example.yaml),
and the [profile reference](docs/public/profile-reference.md#roles).

## What installs today

Today mainly the model and route settings install. OpenCode installs as a named
agent by default, with `--default` also patching the main config and one skill.
`globalInstructions` owns whole global instruction files (`AGENTS.md`, `CLAUDE.md`,
Oh My Pi `RULES.md`, and `~/AGENTS.md` through Pi). With `--default` or
`mango use`, profile roles become subagent files for Claude Code, Codex,
OpenCode, and Oh My Pi.
Anything else an agent can't take, such as instructions,
permissions, tools, or skills, is listed in the plan as
`not installed for this agent: ...` and skipped. Pass `--strict` to block
instead. Real delivery of those is on the [roadmap](ROADMAP.md).

| Agent | Installs | Details |
| --- | --- | --- |
| Claude Code | Model, effort, as an emulated named profile; `CLAUDE.md`; role subagents | [agents](docs/public/agents.md#claude-code) |
| Codex | Provider, model, effort, as a named profile; `AGENTS.md`; role subagents | [agents](docs/public/agents.md#codex) |
| OpenCode | Model, effort variant, and instructions, as a named agent; `AGENTS.md`; role subagents | [agents](docs/public/agents.md#opencode) |
| Pi | Provider, model, thinking level; `~/AGENTS.md` | [agents](docs/public/agents.md#pi) |
| Oh My Pi | Model roles with per-role effort, subagent effort cap, as an `omp --config` profile file; `AGENTS.md`, `RULES.md`; role subagents | [agents](docs/public/agents.md#oh-my-pi) |
| OpenClaw | Model, thinking level, as a named profile | [agents](docs/public/agents.md#openclaw) |
| Hermes | Provider, model, effort, as a named profile | [agents](docs/public/agents.md#hermes) |

Each agent has a tested version range. Outside it, install still works but warns.
Credentials, sessions, plugins, and MCP are never read or written.
Named OpenClaw and Hermes profiles use separate target-owned state that may
require signing in there. The plan cannot determine whether either is signed in.

## Documentation

- [Concepts](docs/public/concepts.md): profiles, bindings, routes, and what install writes, skips, backs up, and undoes.
- [Profile and bindings reference](docs/public/profile-reference.md): every field and its default.
- [Agents](docs/public/agents.md): per-agent settings, default paths, tested versions, and caveats.
- [Example profiles](examples/README.md): coding, review, and docs-research workflows.
- [All docs](docs/index.md), including developer evidence and architecture.
- [Roadmap](ROADMAP.md) and [Contributing](CONTRIBUTING.md).

## License

[MIT](LICENSE)
