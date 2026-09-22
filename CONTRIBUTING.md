# Contributing to profile-mango

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://gitlab.com/ariel-frischer/profile-mango.git
cd profile-mango
make deps      # Download dependencies
make install   # Install profile-mango
make build     # Build bin/profile-mango
make test      # Run tests
```

## Development

```bash
make build     # Build to bin/profile-mango
make test      # Run all tests
make lint      # Run linters
make format    # Format code
```

## Pull Requests

1. Fork the repo and create your branch from `main`
2. Add tests for any new functionality
3. Ensure `make test` and `make lint` pass
4. Update `CHANGELOG.yaml` with your changes

## Reporting Issues

Use [GitLab issues](https://gitlab.com/ariel-frischer/profile-mango/issues). Include:
- What you expected vs what happened
- Steps to reproduce
- `profile-mango version` output
- OS and architecture

## Code Style

- Functions under 40 lines
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Table-driven tests with `map[string]struct{}`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
