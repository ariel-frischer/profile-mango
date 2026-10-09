# AGENTS.md

## Project Context

profile-mango: Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.

- **Language:** Go
- **Module:** `github.com/ariel-frischer/profile-mango`
- **Layout:** CLI + library
- **Env prefix:** `PROFILE_MANGO`

## Constitution and Scope

Consult and follow [`.autospec/constitution.yaml`](.autospec/constitution.yaml) for relevant planning, implementation, review, and documentation. Use [`docs/dev/project-scope.md`](docs/dev/project-scope.md) to distinguish shipped capabilities, intended targets, and proposed defaults. Check affected principles proportionally: documentation normally needs factual/link/scope checks, while code, contracts, and compatibility claims need relevant tests and evidence. Surface conflicts explicitly rather than silently bypassing the constitution.

For target configuration work, consult the version-qualified [`docs/dev/agents/`](docs/dev/agents/README.md) references before fresh research. Treat their documentation provenance separately from installed observations, tested evidence, and supported capabilities.

## Command Map

```bash
make help           # List Make targets
make deps           # Download Go modules
make install        # Install mango (+ profile-mango alias) to ~/.local/bin (PROFILE_MANGO_INSTALL_DIR overrides)
make build          # Build ./bin/mango with version ldflags
make bin            # Alias for build
make run            # Run ./cmd/mango
make go-install     # Compatibility alias for installing mango
make install-global # Alias for go-install
make test           # Run tests
make test-v         # Run tests verbosely
make test-coverage  # Run tests with race detector and coverage
make lint           # Run linters (golangci-lint, fallback go vet)
make format         # Run go fmt ./...
make clean          # Remove build artifacts
make worktree BRANCH=agent/name   # Create an isolated agent worktree
make worktree-clean              # Clean merged agent worktrees
make prep-release VERSION=v0.1.0  # Run release preflight
make release VERSION=v0.1.0       # Alias for prep-release
```

CLI smoke checks:

```bash
go run ./cmd/mango --help
go run ./cmd/mango version
go run ./cmd/mango home
go run ./cmd/mango validate <profile.yaml> [--bindings <local.yaml>] [--json]
go run ./cmd/mango status [--json]          # read-only: managed profile per agent, drift
go run ./cmd/mango use <profile>             # plan only; sandbox HOME before --apply
go run ./cmd/mango route list [--json]      # routes; route set/unset edit bindings in place
```

## File Layout

```
cmd/mango/          # CLI entry point (cobra)
  main.go             # binary entry point
  root.go             # root command, persistent flags, command wiring
  version.go          # version subcommand
  validate.go         # offline strict profile/binding validation
  help.go             # custom help formatting
  home.go             # effective global profile-home inspection
internal/
  profilehome/        # --home/env/user-home resolution
  version/            # version info injected via ldflags
pkg/profilemango/      # pure canonical domain, parsing, resolution, resource hashing
  testdata/fixtures/  # route-only, constrained, and unsupported fixtures
  testdata/golden/    # deterministic resolved output
pkg/adapters/          # per-agent renderers and native installers
pkg/install/           # plan, apply, undo, and manifest state machine
schemas/               # versioned profile, binding, plan, and manifest contracts
docs/dev/              # target evidence and support boundaries
docs/public/           # user-facing documentation
assets/                # demo content (GIFs, screenshots)
.github/workflows/     # GitHub Actions CI (ci.yml) + tag release (release.yml)
.goreleaser.yaml       # GitHub release build config
CHANGELOG.yaml        # changelog source
CHANGELOG.md          # generated changelog output
.chlog.yaml           # changelog config
```

## Profile Home Behavior

- The default application home is `<user-home>/.profile-mango` on every OS.
- Path priority: root `--home`, then `$PROFILE_MANGO_HOME`, then the default.
- `mango home` prints the absolute effective path without creating it.
- Root `init` scaffolds the effective home; `init .` or `init <directory>` scaffolds an explicit project package.
- Render inputs default to `<home>/profiles`, `<home>`, and `<home>/bindings/local.yaml`. Explicit project rendering must provide `--profiles`, `--resource-root`, and `--bindings` together.
- The binary never creates the home implicitly during installation or read-only commands.

## Target Safety Boundary

- Canonical parsing, validation, and rendering remain offline and pure: no target homes, credentials, subprocesses, providers, or network access.
- Only the exact installation subsets in `docs/dev/target-evidence.md` may modify target config: an explicit `--config` path or, when omitted, the target's documented default user path listed there. Every write keeps the displayed plan diff and resolved path, create-only backups, hash/drift checks, ownership manifests, and interactive or `--yes --expect-plan` consent.
- Tests and native probes use synthetic or disposable state and must never resolve to the real user home.
- Installers must not read auth stores or touch sessions, plugins, MCP, providers, or the network. Unknown required properties and unqualified targets remain blocked. Known requirements a target cannot install (permissions, tools, instructions, skills, or a route effort it cannot write) are skipped and listed per target in the plan (JSON `skippedRequirements`); `install --strict` blocks on them instead.
- Native qualification is opt-in, exact-artifact, network/PID/IPC-isolated and timeout-bounded. Hide personal homes and sockets; writes may occur only inside the disposable sandbox.
- The experimental Jcode fork (`jcode-fork`) is experimental developer evidence, not a supported product target or README promise.
- Unknown keys in canonical input, duplicate keys, nulls, unsupported versions, missing parents, cycles, and escaping resource paths fail closed. Target-owned unrelated configuration remains preserved.

## Testing Guidance

- Prefer table tests with `map[string]struct{}` for command behavior.
- Toolchain versions are pinned in `mise.toml`; run `mise install` before canonical lint/build checks.
- Run `chlog check` after changelog edits.
- Run `make test` for normal validation; use `make test-coverage` when touching shared packages.
- Smoke-test generated command paths with `go run ./cmd/mango ...` before release work.
- Keep fixtures in `pkg/profilemango/testdata/` and avoid depending on the caller's working directory.

## Release And Changelog

- Release flow is `make prep-release VERSION=vX.Y.Z`, which runs `scripts/release.sh`; the pushed `v*` tag triggers `.github/workflows/release.yml` (GoReleaser GitHub release). Set `RELEASE_REMOTE=<name>` when the GitHub remote is not `origin`.
- The release preflight (`make test`, `make lint`) must cover every `.github/workflows/ci.yml` check, so a failing CI check blocks tagging. When adding a CI step, add the same check to a Make target the preflight runs.
- `BUILD_VERSION`, `COMMIT`, and `BUILD_DATE` are injected through Makefile ldflags.
- Changelog source is `CHANGELOG.yaml`; regenerate `CHANGELOG.md` with `chlog sync`.
- Use `chlog add <category> "message"` for unreleased entries when possible.

## Git Rules

- Check `git status --short` before editing and before committing.
- Keep writes scoped to files relevant to the task; do not clean up unrelated work.
- Do not use `git stash`, `git reset --hard`, or broad checkout/revert commands in a shared worktree.
- Stage explicit files only, not `git add .` or `git add -A`.
- Run focused tests for the files you touched, then the relevant Make targets before handoff.
- Before pushing, inspect outgoing commits, fetch the remote base, require it to be an ancestor of `HEAD`, and never rewrite history.
- After a merged task's worktree is validated, clean it up only if its HEAD is reachable from the fetched remote base and no valuable untracked data remains; use ordinary `git worktree remove`.

## Coding Standards

- Functions under 40 lines
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Map-based table tests: `map[string]struct{}`
- Accept interfaces, return concrete types

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:970c3bf2 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Agent Context Profiles

The managed Beads block is task-tracking guidance, not permission to override repository, user, or orchestrator instructions.

- **Conservative (default)**: Use `bd` for task tracking. Do not run git commits, git pushes, or Dolt remote sync unless explicitly asked. At handoff, report changed files, validation, and suggested next commands.
- **Minimal**: Keep tool instruction files as pointers to `bd prime`; use the same conservative git policy unless active instructions say otherwise.
- **Team-maintainer**: Only when the repository explicitly opts in, agents may close beads, run quality gates, commit, and push as part of session close. A current "do not commit" or "do not push" instruction still wins.

## Session Completion

This protocol applies when ending a Beads implementation workflow. It is subordinate to explicit user, repository, and orchestrator instructions.

1. **File issues for remaining work** - Create beads for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Handle git/sync by active profile**: report status and proposed commands at handoff; commit and push only when authorized by the active profile or the current user request.
5. **Hand off** - Summarize changes, validation, issue status, and any blocked sync/commit/push step

**Critical rules:**
- Explicit user or orchestrator instructions override this Beads block.
- Do not commit or push without clear authority from the active profile or the current user request.
- If a required sync or push is blocked, stop and report the exact command and error.
<!-- END BEADS INTEGRATION -->

Maintainer-private workflow rules that extend this file live in `AGENTS.dev.md`
and are stripped from public publication.
