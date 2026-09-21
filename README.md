# profile-mango

Define portable coding-agent behavior once, resolve it deterministically, and know which properties a target can or cannot preserve.

> **Status:** private prototype. M0 validation remains offline and pure. An exact-version Codex 0.154.0 adapter can now produce an explicitly inert preview, but no generated output is applicable or installed because authentication, delivery, and enforcement remain unverified.

## Why

Coding-agent configuration mixes portable intent with target-specific syntax, authentication routes, and enforcement boundaries. `profile-mango` provides a strict canonical model for the portable part while failing closed on ambiguous input.

M0 includes:

- strict `PolicyProfile` and local route-binding parsing
- stable, field-aware diagnostics
- one-parent inheritance with explicit merge rules
- closed tool allowlists and deny-wins resolution
- safe instruction and skill resource hashing
- versioned plan and ownership-manifest contracts
- JSON Schemas and positive, constrained, and unsupported fixtures
- an offline `validate` command

M0 intentionally excludes applicable adapters, application, import, drift repair, credentials, provider calls, target-home inspection, roles, MCP projection, and identity management. The later Codex preview renderer writes only to an explicit staging directory and does not weaken those safety boundaries.

## Support policy

Codex is the first intended public adapter target. The shipped Codex 0.154.0 boundary is preview-only: it has version-qualified evidence and golden rendering tests, always reports current profiles as non-applicable, and emits no active target configuration.

Experimental comparison targets are documented only in developer evidence. They are not part of the public compatibility promise.

## Profile

```yaml
apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: research
spec:
  routeRef: research-primary
  permissions:
    mode: read-only
    network: allow
    shell: deny
  tools:
    allow: [read, search, web]
    deny: [write, edit, deploy]
  instructions:
    append:
      - instructions/system.md
      - instructions/research.md
  skills:
    - skills/research/SKILL.md
```

Machine-local bindings contain route identity, never tokens:

```yaml
routes:
  research-primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
```

## CLI

```bash
make build

./bin/profile-mango validate profiles/research/profile.yaml \
  --bindings bindings/local.yaml

./bin/profile-mango validate profiles/research/profile.yaml --json

# Writes only an inert preview to a new explicit directory, then exits nonzero
# because current authentication, delivery, and enforcement requirements block applicability.
./bin/profile-mango render research \
  --profiles profiles \
  --resource-root . \
  --bindings bindings/local.yaml \
  --target codex \
  --target-version 0.154.0 \
  --out ./codex-preview \
  --preview --json
```

Validation and preview rendering read only explicitly supplied files. They do not inspect agent homes, resolve credentials, run subprocesses, or use the network. Preview output is conspicuously inert and is never installed automatically.

## Library

```go
profile, diagnostics := profilemango.ParseProfile(profileYAML)
resolved, diagnostics := profilemango.Resolve(profiles, "research")
resources, diagnostics := profilemango.DigestResources(packageRoot, resolved)
```

See [`schemas/`](schemas/) for the versioned contracts and [`docs/dev/target-evidence.md`](docs/dev/target-evidence.md) for the evidence boundary.

## Development

```bash
make test
make test-coverage
make lint
make format
make build
```

## License

[MIT](LICENSE)
