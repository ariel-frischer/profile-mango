# Changelog

All notable changes to profile-mango will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- M0 strict profile contracts, deterministic resolution, resource hashing, schemas, fixtures, and offline validation
- Version-qualified target evidence with explicit public-support and applicability boundaries
- Version-qualified local agent configuration references with deterministic source provenance
- Automated offline clean-install and Draft 2020-12 schema validation evidence for M0
- Exact Codex 0.154.0 inert preview rendering with fail-closed applicability diagnostics and explicit staging
- Exact Oh My Pi 18.2.6 inert preview rendering with source/build blockers, deterministic YAML candidates, and explicit fail-closed dispatch
- Exact OpenClaw 2026.9.5 inert preview rendering with source/archive provenance, deterministic JSON5 candidates, and explicit config-inspection and applicability blockers
- Exact Hermes Agent 0.21.3 inert preview rendering for source release v2026.9.14 with source/build provenance, deterministic YAML candidates, and explicit inspector and applicability blockers
- Exact Claude Code 2.1.278 inert preview rendering with immutable npm artifact provenance, deterministic JSON candidates, and explicit native inspection and applicability blockers
- Exact Pi 0.86.1 inert JSON settings preview rendering with immutable source/package provenance, deterministic candidates, and explicit native inspection and applicability blockers
- Exact OpenCode 1.18.31 inert JSONC preview rendering with immutable release/source provenance, source-grounded model syntax, and explicit native and install applicability blockers
- Mock-only install planning and application mechanics with deterministic plan IDs, hash-bound consent, synthetic backups, stale checks, atomic replacement, ownership evidence, and production target gates
- Experimental-only Ariel custom Jcode fork inert TOML preview rendering pinned to the exact tested 0.83.909-dev (ca8017a3a) build and SHA-256, with strict projection and non-applicability diagnostics
- Target-neutral render report and resource boundary shared by inert target adapters
- Deterministic profile scaffolding with profile-mango init
- A user-owned global profile home with cross-platform `~/.profile-mango` semantics, explicit overrides, and read-only inspection
- Add an isolated OpenCode 1.18.31 native config probe with backup restoration and representative portable profiles validated across every public adapter
- Lossless transactional application of the OpenCode 1.18.31 top-level model field at one explicit destination, with destination-bound consent, preservation, backups, journals, stale checks, and guarded recovery
- Install the exact Claude Code 2.1.278 model field and Pi 0.86.1 route-default settings with native consumption evidence and transactional safeguards.
- Install one qualified OpenCode 1.18.31 skill with native discovery evidence, lossless skills.paths edits, and non-overridable resource ownership protection
- Install qualified Hermes 0.21.3 model and reasoning config fields with exact isolated native merge evidence and transactional safeguards
- Install qualified OpenClaw 2026.9.5 model and thinking defaults with exact source-native consumption and transactional safeguards
- Exact Oh My Pi 18.2.6 two-field YAML installation with source-native getter qualification, lossless patches, backups and guarded transactions

### Changed

- Successful profile-mango init output now shows the ANSI README logo
- Zero-argument init and omitted render repository inputs now use the effective global profile home; explicit project render inputs must be supplied together
- Renamed the repository, Go module, CLI, package, schema namespace, and configuration identity to profile-mango
- GitLab pipelines now run only when a human starts them from the web UI
- Human-readable CLI output now uses semantic terminal colors with automatic non-TTY suppression and --no-color/NO_COLOR opt-outs, while JSON, completion, and path output remain ANSI-free
- Add global --non-interactive consent handling and emit one JSON apply report instead of a concatenated plan/report stream.
- Renamed the installed executable and user-facing CLI command to mango while preserving profile-mango project, module, home, and environment identities

### Removed

- Removed the unused private-prototype user configuration commands and root `--config` override

### Fixed

- CI formatting checks inspect tracked Go sources without scanning the restored module cache
- Cross-target inert preview acceptance, early experimental identity reporting, and race-safe no-replace staging
- Quote the OpenCode JSONC model key after exact 1.18.31 native parsing rejected the prior candidate
- Apply all selected installation targets in one filesystem transaction and reject altered recovery backups before restoring files.
- Show sanitized field-level before/after changes in human install plans before consent, without changing JSON plan output.

## [0.0.1] - 2026-01-01

### Added

- Initial project scaffolding

[Unreleased]: https://gitlab.com/ariel-frischer/profile-mango/-/compare/v0.0.1...HEAD
