---
name: polish
description: Validate changelog, docs, Go code, fixtures, schemas, and offline CLI behavior after changes.
---

# Polish profile-mango

Run from the repository root after code or documentation changes:

```bash
chlog sync
chlog check
make format
make test
make test-coverage
make lint
make build
go run ./cmd/mango --help
go run ./cmd/mango validate pkg/profilemango/testdata/fixtures/route-only/profile.yaml \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml --json
git diff --check
```

Confirm README, `docs/index.md`, `docs/dev/target-evidence.md`, schemas, fixtures, `AGENTS.md`, and `CLAUDE.md` still describe the implemented M0 boundary. Validation must remain offline and must not inspect target homes, credentials, or providers.
