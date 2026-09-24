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
- Install exact OpenCode 1.18.31 named primary and subagent definitions with model and ordered instructions at explicit owned paths, qualified by isolated native consumption
- Codex 0.154.0 settings-only installation of source-qualified root provider, model, and high effort at explicit paths with consent, backups, preserved authentication, and full-profile warnings
- Guarded public Codex 0.154.0 restore preview and hash-bound apply for adjacent committed install journals, original backups, and ownership manifests
- Add copyable agent-oriented profile examples with original workflow instructions, skills, and version-scoped model bindings
- Per-agent route overrides in bindings (routes.<name>.targets) so one profile installs across agents with different providers
- preview alias for render
- doctor command: read-only report of installed agents, detected vs qualified versions, default config paths, and what a profile would install
- undo command (restore kept as alias) reverses the latest install for any installable target
- install and doctor check the installed agent version against a tested range and warn outside it

### Changed

- Successful profile-mango init output now shows the ANSI README logo
- Zero-argument init and omitted render repository inputs now use the effective global profile home; explicit project render inputs must be supplied together
- Renamed the repository, Go module, CLI, package, schema namespace, and configuration identity to profile-mango
- GitLab pipelines now run only when a human starts them from the web UI
- Human-readable CLI output now uses semantic terminal colors with automatic non-TTY suppression and --no-color/NO_COLOR opt-outs, while JSON, completion, and path output remain ANSI-free
- Add global --non-interactive consent handling and emit one JSON apply report instead of a concatenated plan/report stream.
- Renamed the installed executable and user-facing CLI command to profile-mango while preserving profile-mango project, module, home, and environment identities
- Clarify Codex 0.154.0 settings-only installation warnings with isolated installed-binary field and precedence evidence; OAuth and runtime behavior remain unverified
- Install and restore accept a bare target name that resolves to its single qualified version; install uses one `--config target[@version]=path` flag that also selects its target, with `--config-path` kept as a hidden deprecated alias
- install uses each agent's documented default config path when --config is omitted, still behind the plan, backups, drift checks, and confirmation
- Profiles use a flat YAML format (name, description, extends, route, permissions, tools, instructions, skills); the old apiVersion/kind/metadata/spec form still loads with a deprecation warning
- Routes default transport to native and authentication to oauth
- Plain-language help text; the agents maintenance command and the experimental ariel-jcode target are hidden from help
- A first install into an existing, unowned agent config adopts it with a mandatory backup instead of requiring --override; plans end with the exact apply command
- Examples consolidated into examples/ (coding, review, docs-research) with a per-agent bindings example; the bundled agent skill is shorter and its compatibility list is corrected
- install --all skips agents that are not installed (listed as skipped); a named target with a missing config folder gets a clear message
- install applies the supported subset of a profile by default and lists every skipped requirement per agent; --strict restores blocking; init starter bindings include a Claude Code override
- User-first README and new docs/public pages (concepts, profile reference, agents); docs index split into user and developer sections
- The curl installer one-liner and install.sh release defaults now point at GitHub
- Codex install now writes a native named profile ($CODEX_HOME/<name>.config.toml, used with codex --profile <name>) and leaves config.toml unchanged; --default also writes the root settings; plans and JSON report the install mode and use command, and agents without profiles get a note
- Install creates missing private parent directories for a new file (undo removes the file and leaves the directory), and named profiles may live in a nested or absolute path
- OpenClaw install now writes a native named profile (<home>/.openclaw-<name>/openclaw.json, used with openclaw --profile <name>) and leaves the default config unchanged; --default also writes it; a main config outside <home>/.openclaw/openclaw.json blocks named install; the profile named default patches the default config
- Claude Code install now writes an emulated named profile (~/.claude/profiles/<name>.json with the model, used with claude --settings <file>) and leaves settings.json unchanged; --default also writes settings.json
- OpenCode install without --agent now writes a named primary agent (agents/<name>.md beside opencode.json, with model and instructions, used with opencode --agent <name>) and leaves opencode.json unchanged; skills are skipped with a reason unless --default, which also keeps the main-config model and skill write
- Hermes install now writes a native named profile (<hermes-home>/profiles/<name>/config.yaml, used with hermes -p <name>) and leaves config.yaml unchanged; --default also writes it; the profile named default patches the default config
- The CLI command is now mango (make build gives bin/mango; the installer, make install and releases ship mango plus a profile-mango compatibility alias); the ~/.profile-mango home, PROFILE_MANGO_HOME and on-disk .profile-mango.* files are unchanged

### Removed

- Removed the unused private-prototype user configuration commands and root `--config` override

### Fixed

- CI formatting checks inspect tracked Go sources without scanning the restored module cache
- Cross-target inert preview acceptance, early experimental identity reporting, and race-safe no-replace staging
- Quote the OpenCode JSONC model key after exact 1.18.31 native parsing rejected the prior candidate
- Apply all selected installation targets in one filesystem transaction and reject altered recovery backups before restoring files.
- Show sanitized field-level before/after changes in human install plans before consent, without changing JSON plan output.
- Report install preflight rejection as not-attempted while preserving unchanged targets as noop in library and single-object CLI reports.
- Clean up verified preparation-owned install backups after backup or initial journal failure, preserving unrelated artifacts and same-plan retryability.
- Reconcile only unchanged owned OpenCode skills and Mango-introduced discovery paths on profile omission, with guarded delete recovery and no adoption of identical unowned files
- init now scaffolds a usable bindings/local.yaml with a current model, and missing-bindings errors name the fix
- Command tests no longer leak flag state between tests
- undo works for every target of a multi-target install
- Show live stderr progress while checking agent sources and probing installed agent versions without changing JSON reports

## [0.0.1] - 2026-01-01

### Added

- Initial project scaffolding

[Unreleased]: https://gitlab.com/ariel-frischer/profile-mango/-/compare/v0.0.1...HEAD
