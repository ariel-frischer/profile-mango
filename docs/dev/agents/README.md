# Agent configuration references

These contributor references summarize configuration surfaces for the intended
MVP targets and Ariel's experimental-only custom Jcode fork. They are documentation
bases for isolated probes and adapter work, not a compatibility matrix. Codex,
Oh My Pi, OpenClaw, and Hermes now have exact-version, preview-only renderers;
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
- [Claude Code](claude-code.md) — intended MVP target; untested.
- [Pi](pi.md) — intended MVP target variant; qualified independently.
- [Oh My Pi](oh-my-pi.md) — intended MVP target variant; qualified independently.
- [OpenClaw](openclaw.md) — exact v2026.9.5 inert preview renderer; native applicability remains blocked.
- [Hermes](hermes.md) — exact Hermes Agent 0.21.3 inert preview renderer; native applicability remains blocked.
- [Ariel custom Jcode fork](jcode.md) — experimental-only local comparison target.

Candidate inspection commands below have not been executed for this reference
pack. Before a future probe, verify the exact target revision and command effects,
then use a synthetic home/project, sanitized environment, no real credentials,
blocked network, bounded execution, and private sockets where relevant. Follow
the [validation strategy](../agent-validation.md); record executed results only
in the [target evidence ledger](../target-evidence.md).
