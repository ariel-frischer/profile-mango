# CLAUDE.md

## Project: profile-mango

Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.

## Constitution and Scope

Consult and follow [`.autospec/constitution.yaml`](.autospec/constitution.yaml) for relevant planning, implementation, review, and documentation. Use [`docs/dev/project-scope.md`](docs/dev/project-scope.md) to distinguish shipped capabilities, intended targets, and proposed defaults. Check affected principles proportionally: documentation normally needs factual/link/scope checks, while code, contracts, and compatibility claims need relevant tests and evidence. Surface conflicts explicitly rather than silently bypassing the constitution.

For target configuration work, consult the version-qualified
[`docs/dev/agents/`](docs/dev/agents/README.md) references before fresh research.
Treat their documentation provenance separately from installed observations,
tested evidence, and supported capabilities.

## Commands

```bash
make build          # Build binary
make deps           # Download dependencies
make install        # Install profile-mango
make test           # Run tests
make lint           # Run linters
make format         # Format code
```

## Architecture

```
cmd/profile-mango/             # CLI entry point (cobra)
  home.go             # effective profile-home inspection
  validate.go         # offline strict validation
internal/profilehome/  # --home/env/user-home resolution
internal/version/      # Version info (ldflags)
pkg/profilemango/      # pure domain, parser, resolver, resource digests
schemas/               # versioned contracts
docs/dev/              # target evidence and support policy
assets/                # Demo content (GIFs, screenshots)
```

## Profile Home Behavior

- The default application home is `<user-home>/.profile-mango` on every OS.
- Precedence is root `--home`, then `$PROFILE_MANGO_HOME`, then the default.
- `home` is read-only; root `init` scaffolds the home, while an explicit directory creates a project package.
- Render inputs default coherently from the home; explicit project inputs are an all-or-none trio.

## Coding Standards

- Functions under 40 lines
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Map-based table tests: `map[string]struct{}`
- Accept interfaces, return concrete types

## Target Safety Boundary

The canonical core, validation, and rendering remain offline and pure. Only exact installation subsets in `docs/dev/target-evidence.md` may modify target config: an explicit `--config` path or, when omitted, the target's documented default user path listed there. Every write keeps the displayed plan diff and resolved path, create-only backups, hash/drift checks, ownership manifests, and interactive or `--yes --expect-plan` consent. Tests and native probes use synthetic or disposable state and must never resolve to the real user home. Installers must not read auth stores or touch sessions, plugins, MCP, providers, or the network. Unqualified targets and unknown required properties remain blocked. Known requirements a target cannot install (permissions, tools, instructions, skills) are skipped and listed per target in the plan; `install --strict` blocks on them instead. Native qualification is opt-in, exact-artifact, network/PID/IPC-isolated, timeout-bounded, and must hide personal homes and sockets. Jcode remains experimental developer evidence, not a supported public target.

## Post-Feature Checklist

Always invoke the `/polish` skill after changes and before handoff.

After a task is merged, pushed, and validated, automatically clean up its owned worktree. Verify the current HEAD is reachable from the fetched remote base, no agent still uses it, and staged, unstaged, untracked, and ignored contents contain no valuable data; preserve reports outside the worktree. Use ordinary `git worktree remove` without force and delete its local branch only if `git branch -d` permits it. Leave active, dirty, unmerged, or uncertain worktrees intact and report why.


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:6cd5cc61 -->
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
   git push
   git status
   ```
5. **Hand off** - Summarize changes, validation, issue status, and any blocked sync/commit/push step

**Critical rules:**
- Explicit user or orchestrator instructions override this Beads block.
- Do not commit or push without clear authority from the active profile or the current user request.
- If a required sync or push is blocked, stop and report the exact command and error.
<!-- END BEADS INTEGRATION -->
