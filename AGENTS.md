# AGENTS.md

## Project Context

profile-mango: Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.

- **Language:** Go
- **Module:** `gitlab.com/ariel-frischer/profile-mango`
- **Layout:** CLI + library
- **Env prefix:** `PROFILE_MANGO`

## Constitution and Scope

Consult and follow [`.autospec/constitution.yaml`](.autospec/constitution.yaml) for relevant planning, implementation, review, and documentation. Use [`docs/dev/project-scope.md`](docs/dev/project-scope.md) to distinguish shipped capabilities, intended targets, and proposed defaults. Check affected principles proportionally: documentation normally needs factual/link/scope checks, while code, contracts, and compatibility claims need relevant tests and evidence. Surface conflicts explicitly rather than silently bypassing the constitution.

For target configuration work, consult the version-qualified
[`docs/dev/agents/`](docs/dev/agents/README.md) references before fresh research.
Treat their documentation provenance separately from installed observations,
tested evidence, and supported capabilities.

## Command Map

```bash
make help           # List Make targets
make install        # Download Go modules
make build          # Build ./bin/profile-mango with version ldflags
make bin            # Alias for build
make run            # Run ./cmd/profile-mango
make go-install     # Install profile-mango to GOPATH/bin
make install-global # Alias for go-install
make test           # Run tests
make test-v         # Run tests verbosely
make test-coverage  # Run tests with race detector and coverage
make lint           # Run linters (golangci-lint, fallback go vet)
make format         # Run go fmt ./...
make clean          # Remove build artifacts
make prep-release VERSION=v0.1.0  # Run release preflight
make release VERSION=v0.1.0       # Alias for prep-release
```
CLI smoke checks:

```bash
go run ./cmd/profile-mango --help
go run ./cmd/profile-mango version
go run ./cmd/profile-mango home
go run ./cmd/profile-mango validate <profile.yaml> [--bindings <local.yaml>] [--json]
```

## File Layout

```
cmd/profile-mango/      # CLI entry point (cobra)
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
schemas/               # versioned profile, binding, plan, and manifest contracts
docs/dev/              # target evidence and support boundaries
assets/               # demo content (GIFs, screenshots)
.gitlab-ci.yml        # GitLab CI + release
CHANGELOG.yaml        # changelog source
CHANGELOG.md          # generated changelog output
.chlog.yaml           # changelog config
```

## Profile Home Behavior

- The default application home is `<user-home>/.profile-mango` on every OS.
- Path priority: root `--home`, then `$PROFILE_MANGO_HOME`, then the default.
- `profile-mango home` prints the absolute effective path without creating it.
- Root `init` scaffolds the effective home; `init .` or `init <directory>` scaffolds an explicit project package.
- Render inputs default to `<home>/profiles`, `<home>`, and `<home>/bindings/local.yaml`. Explicit project rendering must provide `--profiles`, `--resource-root`, and `--bindings` together.
- The binary never creates the home implicitly during installation or read-only commands.

## M0 Safety Boundary

- M0 is offline and pure: no target homes, credentials, subprocesses, providers, or network access.
- Codex is the first intended public adapter target; only an exact-version inert preview renderer exists, and native applicability remains blocked.
- Jcode is developer-only experimental evidence, not a supported product target or README promise.
- Unknown keys, duplicate keys, nulls, unsupported versions, missing parents, cycles, and escaping resource paths fail closed.

## Testing Guidance

- Prefer table tests with `map[string]struct{}` for command behavior.
- Toolchain versions are pinned in `mise.toml`; run `mise install` before canonical lint/build checks.
- Run `chlog check` after changelog edits.
- Run `make test` for normal validation; use `make test-coverage` when touching shared packages.
- Smoke-test generated command paths with `go run ./cmd/profile-mango ...` before release work.
- Keep fixtures in `pkg/profilemango/testdata/` and avoid depending on the caller's working directory.

## Release And Changelog

- Release flow is `make prep-release VERSION=vX.Y.Z`, which runs `scripts/release.sh`.
- `BUILD_VERSION`, `COMMIT`, and `BUILD_DATE` are injected through Makefile ldflags.
- Changelog source is `CHANGELOG.yaml`; regenerate `CHANGELOG.md` with `chlog sync`.
- Use `chlog add <category> "message"` for unreleased entries when possible.


## Multi-Agent Git Rules

- Check `git status --short` before editing and before committing.
- Keep writes scoped to files relevant to the task; do not clean up unrelated work.
- Do not use `git stash`, `git reset --hard`, or broad checkout/revert commands in a shared worktree.
- Stage explicit files only, not `git add .` or `git add -A`.
- Run focused tests for the files you touched, then the relevant Make targets before handoff.

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
4. **Handle git/sync by active profile**:
   ```bash
   # Conservative/minimal/default: report status and proposed commands; wait for approval.
   git status

   # Team-maintainer opt-in only, unless current instructions forbid it:
   git pull --rebase
   bd dolt push
   git push
   git status
   ```
5. **Hand off** - Summarize changes, validation, issue status, and any blocked sync/commit/push step

**Critical rules:**
- Explicit user or orchestrator instructions override this Beads block.
- Do not commit or push without clear authority from the active profile or the current user request.
- If a required sync or push is blocked, stop and report the exact command and error.
<!-- END BEADS INTEGRATION -->

<!-- BEGIN BEADS CODEX SETUP: generated by bd setup codex -->
## Beads Issue Tracker

Use Beads (`bd`) for durable task tracking in repositories that include it. Use the `beads` skill at `.agents/skills/beads/SKILL.md` (project install) or `~/.agents/skills/beads/SKILL.md` (global install) for Beads workflow guidance, then use the `bd` CLI for issue operations.

### Quick Reference

```bash
bd ready                # Find available work
bd show <id>            # View issue details
bd update <id> --claim  # Claim work
bd close <id>           # Complete work
bd prime                # Refresh Beads context
```

### Rules

- Use `bd` for all task tracking; do not create markdown TODO lists.
- Run `bd prime` when Beads context is missing or stale. Codex 0.129.0+ can load Beads context automatically through native hooks; use `/hooks` to inspect or toggle them.
- Keep persistent project memory in Beads via `bd remember`; do not create ad hoc memory files.

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.
<!-- END BEADS CODEX SETUP -->
