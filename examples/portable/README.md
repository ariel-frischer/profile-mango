# Portable example package

**Examples, not installed agent configurations.** Copy this entire directory, including `profiles/`, `bindings/`, `instructions/`, and `skills/`. Paths inside profiles resolve relative to the package root. The `AGENTS.md` files here are source resources, **not** automatically installed native agent instruction files. Eleven target-oriented profiles are intentionally **settings-only**. Four workflow profiles (`workflow-base`, `coding`, `review`, `docs-research`) show portable instructions and three original skill packs. Workflow resources and permission rules are for offline inspection, not currently installable as full profiles.

```text
portable/
├── profiles/
│   ├── claude-code/         (profile.yaml in every directory below)
│   ├── claude-code-daily/
│   ├── opencode/
│   ├── opencode-focused/
│   ├── codex/
│   ├── codex-focused/
│   ├── codex-hard/
│   ├── pi/
│   ├── oh-my-pi/
│   ├── openclaw/
│   ├── hermes/
│   ├── workflow-base/
│   ├── coding/
│   ├── review/
│   └── docs-research/
├── instructions/
│   ├── AGENTS.md
│   ├── coding/AGENTS.md
│   ├── review/AGENTS.md
│   └── docs-research/AGENTS.md
├── skills/
│   ├── code-review/SKILL.md
│   ├── test-driven-coding/SKILL.md
│   └── docs-research/SKILL.md
└── bindings/
    ├── local.example.yaml
    ├── latest-models.example.yaml
    └── .gitignore
```

The additional `claude-code-daily`, `opencode-focused`, `codex-focused`, and
`codex-hard` profiles select alternate route keys from the same bindings.
`workflow-base` contributes shared instructions. Its children append their own
AGENTS.md-style source and select a skill. The `tools` and `permissions` rules
live in each profile YAML; they are not hidden in those Markdown files.

## Start locally

From the repository root (or replace `examples/portable` with the path to your copied package):

```bash
cp -R examples/portable ./my-agent-profiles   # choose a new, absent destination
cp ./my-agent-profiles/bindings/local.example.yaml ./my-agent-profiles/bindings/local.yaml
# Edit local.yaml: choose your actual provider, bare model ID, transport, auth mode, and effort.
profile-mango validate ./my-agent-profiles/profiles/coding/profile.yaml \
  --bindings ./my-agent-profiles/bindings/local.yaml
```

`validate` checks the individual profile and binding syntax, **not** the existence of parents or referenced resources. To resolve the package and inspect candidate resources, use an offline render. The following example uses an explicit new staging directory and does not access an agent home:

```bash
profile-mango render coding \
  --profiles ./my-agent-profiles/profiles \
  --resource-root ./my-agent-profiles \
  --bindings ./my-agent-profiles/bindings/local.yaml \
  --target codex --target-version 0.154.0 \
  --out ./my-coding-preview --preview --json
```

This preview **writes inert staging artifacts but exits nonzero** because Codex applicability is blocked. Inspect `my-coding-preview/render.json`, the candidate preview, and copied resources. Pick a fresh `--out` directory for another run. Omit `--preview` if you want diagnostics without any output. A missing parent or resource must fail rather than be silently skipped.

For a bounded settings-only **plan**, use a disposable empty config path and explicit inputs, for example:

```bash
profile-mango install pi \
  --profiles ./my-agent-profiles/profiles \
  --resource-root ./my-agent-profiles \
  --bindings ./my-agent-profiles/bindings/local.yaml \
  --target pi@0.86.1 \
  --config-path pi=./synthetic-pi-settings.json --json
```

This command plans only. Do not add `--apply`, `--yes`, or an actual agent config path without reviewing the exact plan and obtaining path-specific approval. For other targets, use the exact version and appropriate target config format from the [evidence ledger](../../docs/dev/target-evidence.md), never infer the destination from this package.

For Codex `0.154.0`, the `codex`, `codex-focused`, and `codex-hard` profiles can
also plan the three qualified root settings with `--target codex@0.154.0` and
`--config-path codex=./synthetic-codex-config.toml` in place of Pi's target and
path above. The route must remain OpenAI/native/OAuth with `high` effort. This
does not prove that a suggested model is available, which authentication method
Codex will use, or that higher-precedence configuration will preserve the route.

## Profiles and boundaries

| Profile | Pinned target | Example intention | Qualified subset at that version |
| --- | --- | --- | --- |
| `claude-code`, `claude-code-daily` | Claude Code `2.1.278` | Demanding Fable 5.1 or daily Sonnet 5 | Explicit-file model field only |
| `opencode`, `opencode-focused` | OpenCode `1.18.31` | General or focused model tier | Main model field (one skill separately possible) |
| `codex`, `codex-focused`, `codex-hard` | Codex CLI `0.154.0` | General, focused, or hardest-work candidate model | Root OpenAI provider, model, and `high` effort only, with exact route and explicit path |
| `pi` | Pi `0.86.1` | Provider/model/thinking defaults | Three settings fields |
| `oh-my-pi` | Oh My Pi `18.2.6` | Default model role/thinking | Two settings fields |
| `openclaw` | OpenClaw `2026.9.5` | Default agent model/thinking | Two settings fields |
| `hermes` | Hermes `0.21.3` | Provider/model/reasoning defaults | Three settings fields |
| `workflow-base`, `coding`, `review`, `docs-research` | Any listed preview renderer | Original AGENTS.md-style guidance and skills | **Preview only**, not full-policy installation |

Names like `claude-code` and `pi` are **Mango profile names**, not native named agents. A main-settings install does not create or activate a native named agent. Only the listed fields have bounded installation evidence. Instructions are not permission enforcement. Skills are not an exclusive allowlist. No profile here configures a credential, starts a provider session, proves model availability, sets an effective auth route, or guarantees runtime tool restrictions. Target-owned settings and higher-precedence overrides may change the actual result. See the [root support table](../../README.md#what-works-today) and [target evidence](../../docs/dev/target-evidence.md).

## Bindings and models

`bindings/local.example.yaml` is a credential-free **starting example**, not a verified model/client matrix. It offers Fable 5.1 for demanding Claude work, Sonnet 5 for daily Claude work, and illustrative GPT-5.6 Terra, GPT-5.6 Luna, and GPT-5.6 Sol routes for general, focused, and harder OpenAI work. Claude Code's current [model configuration](https://code.claude.com/docs/en/model-config) lists Fable 5.1 as requiring `2.1.257` or later, below the pinned `2.1.278`, but this is documentation rather than a release-specific model-call test. Edit the copy for your own target, account, catalog and route. OpenCode and OpenClaw map `provider` plus `/` plus the **bare** `model` to native model names. Pi maps effort to its thinking level. Other targets may interpret or omit provider, transport, authentication, and effort differently. A valid binding is not proof that a target can authenticate or call that model. Never place API keys, tokens, account identifiers, or provider URLs in these files.

`bindings/latest-models.example.yaml` is **separate and opt-in**. Its catalog suggestions dated **2026-09-22** pair Fable 5.1 with demanding Claude work, Sonnet 5 with daily Claude work, GPT-6 Sol with general coding, GPT-6 Luna with focused work, and GPT-6 Astra with the hardest end-to-end work. These are task-size suggestions, not universal performance rankings. Current catalog availability does **not** prove old-client compatibility: Claude Code's current [model configuration](https://code.claude.com/docs/en/model-config) says **Opus 5.5** (Anthropic's current default recommendation) **requires Claude Code 2.1.280 or later**, newer than Mango's qualified 2.1.278, so it is deliberately **not** in the copyable bindings. Codex's bounded installer can write a safe model string, but [current Codex model recommendations](https://developers.openai.com/codex/models.md) do not prove GPT-6 availability on Codex 0.154.0. Do not swap in the latest binding and install by default. Verify exact target version, provider catalog/access, native model syntax, auth route and effort handling in isolated state first.

## Sources and provenance

Reviewed as of **2026-09-22**. Official mutable model/config pages describe *current documentation*, not the pinned release's tested behavior. Local exact-version references and the [evidence ledger](../../docs/dev/target-evidence.md) describe qualified subsets separately.

- [Anthropic Claude models](https://platform.claude.com/docs/en/about-claude/models/overview), [Claude Code model configuration](https://code.claude.com/docs/en/model-config)
- [OpenAI Codex models](https://developers.openai.com/codex/models.md), [OpenAI API model catalog](https://developers.openai.com/api/docs/models.md), [OpenAI latest-model guidance](https://developers.openai.com/api/docs/guides/latest-model.md)
- [OpenCode model configuration](https://opencode.ai/docs/models/), [local target references](../../docs/dev/agents/README.md)
- [AGENTS.md convention](https://agents.md/), [Agent Skills specification](https://agentskills.io/specification). The instruction and skill text in this package is original, not copied from third-party packs.
