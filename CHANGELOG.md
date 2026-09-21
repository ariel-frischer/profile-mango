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
- Experimental-only Ariel custom Jcode fork inert TOML preview rendering pinned to the exact tested 0.83.909-dev (ca8017a3a) build and SHA-256, with strict projection and non-applicability diagnostics
- Target-neutral render report and resource boundary shared by inert target adapters
- Deterministic profile scaffolding with profile-mango init
- A user-owned global profile home with cross-platform `~/.profile-mango` semantics, explicit overrides, and read-only inspection

### Changed

- Successful profile-mango init output now shows the ANSI README logo
- Zero-argument init and omitted render repository inputs now use the effective global profile home; explicit project render inputs must be supplied together
- Renamed the repository, Go module, CLI, package, schema namespace, and configuration identity to profile-mango
- GitLab pipelines now run only when a human starts them from the web UI

### Removed

- Removed the unused private-prototype user configuration commands and root `--config` override

### Fixed

- CI formatting checks inspect tracked Go sources without scanning the restored module cache
- Cross-target inert preview acceptance, early experimental identity reporting, and race-safe no-replace staging

## [0.0.1] - 2026-01-01

### Added

- Initial project scaffolding

[Unreleased]: https://gitlab.com/ariel-frischer/profile-mango/-/compare/v0.0.1...HEAD
