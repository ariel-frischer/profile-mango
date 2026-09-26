# Documentation index

## User docs

- [`public/concepts.md`](public/concepts.md) - Profiles, bindings, routes, and target overrides; what install writes, skips, backs up, and undoes; tested version ranges.
- [`public/profile-reference.md`](public/profile-reference.md) - Every profile and bindings field, inheritance rules, defaults, `targets` overrides, and the older wrapped format.
- [`public/agents.md`](public/agents.md) - Per-agent cost/authentication preflight, installed settings, default paths, tested versions and ranges, and caveats.
- [`../examples/README.md`](../examples/README.md) - Three credential-free workflow profiles (coding, review, docs-research) with original instructions, skills, and a `targets`-override binding example.
- [Roadmap](../ROADMAP.md) - Evidence-gated product direction for target applicability, profile assignments, profile-scoped adapters, agent roles, MCP, and hooks.

## Developer docs

### Scope and governance

- [Project constitution](../.autospec/constitution.yaml) - Durable principles and lightweight governance for relevant project work.
- [`dev/project-scope.md`](dev/project-scope.md) - North star, current M0, intended MVP targets, boundaries, success criteria, and proposed defaults.

### Architecture

- [`architecture.md`](architecture.md) - Shared canonical, render, install, filesystem, and CLI ownership with parallel target-install integration boundaries.
- [`dev/adapter-architecture.md`](dev/adapter-architecture.md) - Target-neutral render boundary, explicit dispatch, and inert adapter failure behavior.

### Target evidence and validation

- [`dev/target-evidence.md`](dev/target-evidence.md) - Version-qualified native/installer evidence, installed-version policy, default config destinations, limited OpenCode named-agent delivery, support boundaries, and deferred verification for public targets.
- [`dev/agent-validation.md`](dev/agent-validation.md) - No-inference native checks, version-qualified support, local references, and the read-only upstream source drift check.
- [`dev/m0-validation.md`](dev/m0-validation.md) - Reproducible offline clean-install, installed-binary, and Draft 2020-12 contract evidence for M0.
- [`Hermes native installer evidence`](../pkg/adapters/hermes/hermes_native_evidence.md) - Exact imported-module hashes, isolated generated-patch consumption, and runtime limitations for Hermes 0.21.3.
- [`OpenClaw native field evidence`](../pkg/adapters/openclaw/testdata/openclaw-native-field-consumption.evidence.json) - Exact imported-source hashes, model/thinking getter results, and override/fallback limits.

### Agent configuration references

- [`dev/agents/README.md`](dev/agents/README.md) - Version-qualified configuration references and provenance for intended targets.
- [`dev/agents/claude-code.md`](dev/agents/claude-code.md) - Exact Claude Code v2.1.278 artifact provenance, mutable-documentation boundary, inert preview syntax, and bounded model-only native installation evidence.
- [`dev/agents/codex.md`](dev/agents/codex.md) - Codex 0.157.1 configuration and precedence, inert preview boundary, and bounded three-setting installation.
- [`dev/agents/pi.md`](dev/agents/pi.md) - Exact Pi v0.87.1 source/package provenance, inert preview syntax, static effect review, and bounded settings-module installation evidence.
- [`dev/agents/oh-my-pi.md`](dev/agents/oh-my-pi.md) - Exact Oh My Pi v18.3.2 source/release evidence (v18.2.6 addon build), default model-role native getter qualification, source-reviewed per-role selectors, and broader blocked capabilities.
- [`dev/agents/openclaw.md`](dev/agents/openclaw.md) - Exact OpenClaw v2026.9.5 source evidence, JSON5 preview syntax, and blocked native inspection boundary.
- [`dev/agents/hermes.md`](dev/agents/hermes.md) - Exact Hermes Agent v0.21.3 source evidence, YAML preview syntax, and bounded native config-merge installation qualification.
- [`dev/agents/opencode.md`](dev/agents/opencode.md) - Exact OpenCode v1.18.31 model/one-skill and named primary/subagent installation evidence, isolated generated-definition consumption, and runtime limits.

### Research

- [`research/2026-09-23-pi-omp-named-profiles.md`](research/2026-09-23-pi-omp-named-profiles.md) - Pinned-source evidence on named-profile options for Pi and Oh My Pi: OMP `--config` overlay is viable, Pi has none short of relocating its agent dir.
