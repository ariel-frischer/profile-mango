---
name: agent-profile
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

# agent-profile

Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.

## Commands

```bash
agent-profile --help              # Show available commands
agent-profile version             # Show version info
agent-profile v                   # Alias for version
agent-profile completion bash     # Shell completion: bash|zsh|fish|powershell
agent-profile --config ./config.yaml config path
agent-profile config init         # Create user config
agent-profile config get <key>    # Read a config value
agent-profile config set <key> <value>
agent-profile config toggle <key> # Toggle a boolean value
agent-profile config keys         # List configurable keys
```

## Library Usage

```go
import "gitlab.com/ariel-frischer/agent-profile/pkg/agentprofile"
```

## Project Structure

Includes: Makefile, CI pipelines, assets/ directory, .gitignore, LICENSE, SECURITY.md, CHANGELOG.yaml + CHANGELOG.md.
Agent guidance lives in `AGENTS.md`.
Test fixtures live in `pkg/agentprofile/testdata/`.

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
