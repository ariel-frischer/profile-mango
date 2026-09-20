# CLAUDE.md

## Project: agent-profile

Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.

## Commands

```bash
make build          # Build binary
make test           # Run tests
make lint           # Run linters
make format         # Format code
```

## Architecture

```
cmd/agent-profile/      # CLI entry point (cobra)
internal/version/    # Version info (ldflags)
pkg/agentprofile/      # Public library package
  testdata/          # Test fixtures
assets/              # Demo content (GIFs, screenshots)
```

## Coding Standards

- Functions under 40 lines
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Map-based table tests: `map[string]struct{}`
- Accept interfaces, return concrete types
