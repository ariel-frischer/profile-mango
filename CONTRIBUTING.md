# Contributing to profile-mango

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/ariel-frischer/profile-mango.git
cd profile-mango
make deps      # Download dependencies
make install   # Install mango (+ profile-mango alias) to ~/.local/bin
make build     # Build bin/mango
make test      # Run tests
```

## Development

```bash
make build     # Build to bin/mango
make test      # Run all tests
make lint      # Run linters
make format    # Format code
```

To make your global agent skill follow this checkout while you edit it, run
`make link-skill`. It symlinks `~/.agents/skills/profile-mango/SKILL.md` to the
primary checkout and backs up any existing file first.

## Pull Requests

1. Fork the repo and create your branch from `main`
2. Add tests for any new functionality
3. Ensure `make test` and `make lint` pass
4. Update `CHANGELOG.yaml` with your changes

## Reporting Issues

Use [GitHub issues](https://github.com/ariel-frischer/profile-mango/issues). Include:
- What you expected vs what happened
- Steps to reproduce
- `mango version` output
- OS and architecture

## Code Style

- Functions under 40 lines
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Table-driven tests with `map[string]struct{}`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
