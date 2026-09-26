# Example profiles

**Examples, not installed agent configurations.** Copy this whole directory —
`profiles/`, `bindings/`, `instructions/`, and `skills/` — before editing;
resource paths inside a profile resolve relative to the package root.

```text
examples/
├── profiles/
│   ├── workflow-base/     (shared route + instructions; extended, not installed directly)
│   ├── coding/            (workspace-write, test-driven coding)
│   ├── review/            (read-only code review)
│   ├── docs-research/     (read-only, network-off documentation research)
│   └── daily-driver/      (all four roles + global instruction files)
├── instructions/          AGENTS.md files the profiles above append;
│                          roles/ and global/ hold role prompts and global files
├── skills/                one original SKILL.md per workflow profile
└── bindings/
    ├── local.example.yaml (copy to local.yaml; `targets` override, role models)
    └── .gitignore
```

`daily-driver` shows roles and `globalInstructions`: install it with
`mango use daily-driver` (or `install --default`) and each role becomes a
subagent file, while the global files replace the agents' own `AGENTS.md` /
`CLAUDE.md` after a backup. `coding`, `review`, and `docs-research` declare
smaller role sets.

`coding`, `review`, and `docs-research` all `extends: workflow-base` and select
their own `permissions`, `tools`, `instructions`, and `skills`. None is fully
installable today: those requirements exceed every target's qualified subset.
`install` applies the supported subset (the route's model settings) and lists
the rest per target as `not installed for this agent: ...`; add `--strict` to
block instead. `render` previews stay non-applicable. Use them to see profile
structure, inheritance, and route/target-override syntax, not as a full policy.

## Start locally

```bash
cp -R examples ./my-agent-profiles   # choose a new, absent destination
cp ./my-agent-profiles/bindings/local.example.yaml ./my-agent-profiles/bindings/local.yaml
# Edit local.yaml: your real provider, bare model ID, effort, and any per-agent targets.
mango validate ./my-agent-profiles/profiles/coding/profile.yaml \
  --bindings ./my-agent-profiles/bindings/local.yaml
```

`validate` checks one profile's syntax and route binding, not parent
resolution or referenced resources. To resolve the whole package, render it
into a new staging directory instead:

```bash
mango render coding \
  --profiles ./my-agent-profiles/profiles \
  --resource-root ./my-agent-profiles \
  --bindings ./my-agent-profiles/bindings/local.yaml \
  --target codex --target-version 0.157.1 \
  --out ./my-coding-preview --preview --json
```

This writes inert staging artifacts and exits nonzero because these workflow
profiles are not applicable on any current target. Pick a fresh `--out` for
another run.

## Bindings and `targets` overrides

`bindings/local.example.yaml` defines one route (`openai`) with a `targets`
map that gives `claude-code` and `opencode` their own provider/model. Each
listed agent gets the base route with only its override fields applied, so
one profile can drive several agents with different models from a single
route key. Its `roles` give `research` and `tiny` a cheaper model; `worker`
and `planner` fall back to the agent's default model. Never place API keys, tokens, or account identifiers in this file.

See [agents](../docs/public/agents.md) and the
[target evidence](../docs/dev/target-evidence.md) for exactly which fields
each target version can install.
