---
name: profile-mango
description: >
  Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.
license: MIT
compatibility:
  - Claude Code
  - Cursor
  - Codex
  - Gemini CLI
  - VS Code
metadata:
  author: Ariel Frischer
  version: 0.0.1
  tags: go, cli, library
allowed-tools: Bash Read Write Edit
---

# profile-mango

Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.

## Commands

```bash
profile-mango --help              # Show available commands
profile-mango version             # Show version info
profile-mango v                   # Alias for version
profile-mango completion bash     # Shell completion: bash|zsh|fish|powershell
profile-mango --config ./config.yaml config path
profile-mango config init         # Create user config
profile-mango config get <key>    # Read a config value
profile-mango config set <key> <value>
profile-mango config toggle <key> # Toggle a boolean value
profile-mango config keys         # List configurable keys
```

## Library Usage

```go
import "gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
```

## Project Structure

Includes: Makefile, CI pipelines, assets/ directory, .gitignore, LICENSE, SECURITY.md, CHANGELOG.yaml + CHANGELOG.md.
Agent guidance lives in `AGENTS.md`.
Test fixtures live in `pkg/profilemango/testdata/`.

## Development

```bash
make build          # Build binary
make bin            # Alias for build
make install-global # Alias for go-install
make test           # Run tests
make lint           # Run linters
make format         # Format code
make prep-release VERSION=v0.1.0 # Validate release without publishing
```
Use `chlog add ...`, `chlog sync`, and `chlog check` for changelog updates.
