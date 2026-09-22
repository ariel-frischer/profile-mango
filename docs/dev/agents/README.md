# Agent configuration references

These contributor references summarize configuration surfaces for the intended
MVP targets and Ariel's experimental-only custom Jcode fork. They are documentation
bases for isolated probes and adapter work, not a compatibility matrix. Codex,
Claude Code, Pi, Oh My Pi, OpenClaw, Hermes, and OpenCode now have exact-version,
preview-only renderers;
no target has an applicable installer or enforcement claim.

The [source manifest](sources.json) records deterministic provenance and keeps
four states separate:

- **Documentation basis** says what official or local source was reviewed.
- **Installed observation** belongs in the
  [target evidence ledger](../target-evidence.md), when one exists.
- **Tested evidence** requires an isolated, version-qualified check.
- **Supported capability** requires evidence for the exact claimed capability.

Documentation alone never advances the latter three states. Mutable pages are
marked unversioned, release pins do not imply that those pages describe the
release, and an untested newer release is neither supported nor known broken.
Recorded hashes identify the 2026-09-20 retrieval snapshots; source bodies are
not vendored, so later drift work must fetch and hash them again. Pinned Git tags
and commits are reproducible locators, not proof of target behavior.

## Targets

- [Codex](codex.md) — exact Codex 0.154.0 inert preview renderer; native applicability remains blocked.
- [Claude Code](claude-code.md): exact v2.1.278 inert preview renderer plus narrow model-only installer; explicit settings-file native consumption verified, full applicability remains blocked.
- [Pi](pi.md): exact `0.86.1` inert preview plus three-default settings installer; native settings-module evidence only, full applicability remains blocked.
- [Oh My Pi](oh-my-pi.md) — intended MVP target variant; qualified independently.
- [OpenClaw](openclaw.md) — exact v2026.9.5 inert preview renderer; native applicability remains blocked.
- [Hermes](hermes.md) — exact Hermes Agent 0.21.3 inert preview renderer plus three-field installer with isolated native config-merge evidence; full startup/auth/enforcement remain blocked.
- [OpenCode](opencode.md) — exact OpenCode 1.18.31 inert JSONC preview renderer plus bounded model and one-skill installation; isolated merged-config and skill-body discovery evidence does not prove full runtime enforcement.
- [Ariel custom Jcode fork](jcode.md) — experimental-only local comparison target with an exact-build inert preview renderer; native applicability remains blocked.

Pi native commands were not executed because its startup and metadata paths were
not accepted as bounded. OpenCode `1.18.31` was executed only through the
approved synthetic, network-blocked, non-TUI probe documented in the
[validation strategy](../agent-validation.md). That evidence proves the exact
JSONC model candidate parses and appears in merged output, while write-free
inspection, production installation, authentication, delivery, and enforcement
remain unverified. Record executed results only in the
[target evidence ledger](../target-evidence.md).
