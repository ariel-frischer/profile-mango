<div align="center">

**agent-profile**

Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

</div>

## Install

**Go install**:

```bash
go install gitlab.com/ariel-frischer/agent-profile/cmd/agent-profile@latest
```

**Go get** (library):

```bash
go get gitlab.com/ariel-frischer/agent-profile
```

**From source**:

```bash
git clone https://gitlab.com/demo/agent-profile.git
cd agent-profile
make build    # Binary at bin/agent-profile
make bin      # Alias for build
```

## Usage

```bash
agent-profile --help
```

## Library Usage

```go
import "gitlab.com/ariel-frischer/agent-profile/pkg/agentprofile"
```

### AI Agent Skill

This project ships a [SKILL.md](.skills/default/SKILL.md) following the [Agent Skills open standard](https://agentskills.io). Install it so your coding agent knows all commands and options.

**Quick install with [`skills`](https://skills.sh) CLI** (by Vercel Labs):

```bash
npx skills add demo/agent-profile
```

<details>
<summary><strong>Manual install</strong></summary>

**Claude Code** — Skills live in `~/.claude/skills/` (global) or `.claude/skills/` (project-local).

```bash
# Global — available in all projects
mkdir -p ~/.claude/skills/agent-profile
curl -fsSL https://raw.githubusercontent.com/demo/agent-profile/main/.skills/default/SKILL.md \
  -o ~/.claude/skills/agent-profile/SKILL.md

# Project-local — checked into this repo only
mkdir -p .claude/skills/agent-profile
curl -fsSL https://raw.githubusercontent.com/demo/agent-profile/main/.skills/default/SKILL.md \
  -o .claude/skills/agent-profile/SKILL.md
```

**Codex CLI** — reads skills from `~/.codex/skills/` (global) or `.codex/skills/` (project-local).

```bash
# Global
mkdir -p ~/.codex/skills/agent-profile
curl -fsSL https://raw.githubusercontent.com/demo/agent-profile/main/.skills/default/SKILL.md \
  -o ~/.codex/skills/agent-profile/SKILL.md

# Project-local
mkdir -p .codex/skills/agent-profile
curl -fsSL https://raw.githubusercontent.com/demo/agent-profile/main/.skills/default/SKILL.md \
  -o .codex/skills/agent-profile/SKILL.md
```

Or pass directly: `codex --instructions .skills/default/SKILL.md`

</details>

## Development

```bash
make build          # Build binary
make bin            # Alias for build
make install-global # Alias for go-install
make test           # Run tests
make lint           # Run linters
make format         # Format code
```

## Shell Completion

```bash
# Bash
source <(agent-profile completion bash)

# Zsh
source <(agent-profile completion zsh)

# Fish
agent-profile completion fish > ~/.config/fish/completions/agent-profile.fish
```

## License
[MIT](LICENSE)
