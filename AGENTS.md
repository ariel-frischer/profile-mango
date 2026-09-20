# AGENTS.md

## Project Context

agent-profile: Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.

- **Language:** Go
- **Module:** `gitlab.com/ariel-frischer/agent-profile`
- **Layout:** CLI + library
- **Env prefix:** `AGENT_PROFILE`

## Command Map

```bash
make help           # List Make targets
make install        # Download Go modules
make build          # Build ./bin/agent-profile with version ldflags
make bin            # Alias for build
make run            # Run ./cmd/agent-profile
make go-install     # Install agent-profile to GOPATH/bin
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
go run ./cmd/agent-profile --help
go run ./cmd/agent-profile version
go run ./cmd/agent-profile config keys
```

## File Layout

```
cmd/agent-profile/      # CLI entry point (cobra)
  main.go             # binary entry point
  root.go             # root command, persistent flags, command wiring
  version.go          # version subcommand
  config.go           # user config subcommands
  ui.go               # terminal output helpers
  help.go             # custom help formatting
internal/
  version/            # version info injected via ldflags
  config/             # YAML config load/save/path helpers
pkg/agentprofile/      # public library package
  testdata/           # test fixtures
assets/               # demo content (GIFs, screenshots)
.gitlab-ci.yml        # GitLab CI + release
CHANGELOG.yaml        # changelog source
CHANGELOG.md          # generated changelog output
.chlog.yaml           # changelog config
```

## Config Behavior

- User config lives at `~/.config/agent-profile/config.yaml` by default.
- Path priority: root `--config`, then `$AGENT_PROFILE_CONFIG`, then the default path.
- Config commands: `init`, `show`, `path`, `edit`, `get`, `set`, `toggle`, `keys`.
- Missing config files load as empty config; CLI flags should still win over config defaults.


## Testing Guidance

- Prefer table tests with `map[string]struct{}` for command/config behavior.
- Run `chlog check` after changelog edits.
- Run `make test` for normal validation; use `make test-coverage` when touching shared packages.
- Smoke-test generated command paths with `go run ./cmd/agent-profile ...` before release work.
- Keep fixtures in `pkg/agentprofile/testdata/` and avoid depending on the caller's working directory.

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
