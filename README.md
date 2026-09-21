# profile-mango

<div align="center">

<pre>
█▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█
</pre>

**Define portable coding-agent behavior once, then see exactly what each target can preserve.**

</div>

`profile-mango` is a strict profile format, Go library, and CLI for portable
coding-agent behavior. It separates shared intent from machine-local routes,
resolves profiles deterministically, and fails closed when a target cannot prove
that it preserves a requirement.

> **Status:** private prototype. Shipped validation and M1 project scaffolding
> are offline and pure. Codex CLI `0.154.0`, Oh My Pi `18.2.6`, OpenClaw
> `2026.9.5`, and Hermes Agent `0.21.3` have exact-version, explicitly inert
> preview renderers, but no generated output is applicable or installed because
> authentication, delivery, precedence, and enforcement remain unverified. Oh
> My Pi is blocked by its missing standalone native addon and unsafe
> config-inspector effects. OpenClaw is blocked by source-only runtime evidence
> and an unsafe config-inspection path. Hermes is blocked by its exact Python
> build requirement, missing runtime artifact, and unsafe inspector paths.

## Install

Build and install from source:

```bash
git clone git@gitlab.com:ariel-frischer/profile-mango.git
cd profile-mango
make install
make install-global
profile-mango version
```

The repository also contains a checksum-verifying private release installer for
explicitly authorized GitLab release access. It does not edit shell profiles:

```bash
./install.sh
```

Set `PROFILE_MANGO_VERSION` to select a release and
`PROFILE_MANGO_INSTALL_DIR` to change the default `~/.local/bin` destination.

## Usage

Create a deterministic starter package in the global profile home, or choose an
explicit project destination:

```console
$ profile-mango init
█▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█
Created profile scaffold in /home/alice/.profile-mango

$ profile-mango init .
█▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█
Created profile scaffold in .

$ profile-mango init ./my-profile-project
█▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█
Created profile scaffold in my-profile-project
```

The command creates only absent paths and never overwrites existing files. The
same package layout is used beneath the global home or an explicit destination:

```text
profiles/default/profile.yaml
bindings/local.example.yaml
bindings/.gitignore
```

The example binding records route identity only. Credentials remain target-owned,
and `bindings/local.yaml` is ignored for machine-local values. Copy the example
before rendering from the global home:

```bash
cp ~/.profile-mango/bindings/local.example.yaml \
   ~/.profile-mango/bindings/local.yaml
```

The default home is `~/.profile-mango` on Linux and macOS and
`%USERPROFILE%\.profile-mango` on Windows. Override the whole package with the
root `--home` flag or `PROFILE_MANGO_HOME`; the flag wins. `profile-mango home`
prints the effective absolute path without creating it. Instructions and skills
may live under `instructions/` and `skills/` in the same package.

Validate a tracked example entirely offline:

```console
$ profile-mango validate \
    pkg/profilemango/testdata/fixtures/route-only/profile.yaml \
    --bindings pkg/profilemango/testdata/fixtures/bindings.yaml
route-only is valid
```

Use `--json` for a stable machine-readable result:

```bash
profile-mango validate profile.yaml --bindings local-bindings.yaml --json
```

A profile carries portable intent while a local binding identifies a route
without containing credentials:

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

```yaml
routes:
  research-primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
```

## Features

- **Strict contracts:** unknown keys, duplicate keys, nulls, unsupported
  versions, missing parents, cycles, and escaping resource paths fail closed.
- **Deterministic resolution:** one-parent inheritance, explicit merge rules,
  closed tool allowlists, and deny-wins behavior produce reproducible results.
- **Deterministic scaffolding:** `init` creates the global home package, while
  `init .` or `init <directory>` creates an explicit project package without
  overwriting existing files.
- **Portable resources:** instructions and skills are resolved and hashed without
  reading outside the explicitly supplied resource root.
- **Machine-local routing:** bindings describe provider, transport,
  authentication mode, model, and effort, never tokens or credentials.
- **Inspectability:** stable diagnostics and versioned profile, binding, plan,
  render, and ownership-manifest schemas make limitations visible.
- **Offline operation:** validation does not inspect agent homes, run target
  subprocesses, access credentials, call providers, or use the network.

## Why this exists

Agent configuration mixes three different concerns:

1. **Portable intent:** permissions, tools, instructions, skills, and route
   requirements that should retain meaning across targets.
2. **Target delivery:** syntax, file locations, precedence, and resource
   placement that differ between agents.
3. **Enforcement evidence:** what the tested target version can actually prove,
   including authentication routes and bypass surfaces.

Treating syntax translation as compatibility hides meaningful gaps. A read-only
instruction is not the same as enforced read-only access, and selecting a model
does not prove which authentication route will be used. `profile-mango` keeps
those distinctions explicit and blocks required properties that are unsupported
or unknown.

## Inert Codex preview

The current Codex adapter renders only a candidate into a new explicit staging
directory. Render input defaults come from the effective profile home. Supplying
one project input flag requires all three of `--profiles`, `--resource-root`, and
`--bindings`, preventing accidental global/project mixing. This verified fixture
command uses an explicit project package, writes preview artifacts, reports
blocking diagnostics, and exits nonzero because the output is not applicable:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target codex \
  --target-version 0.154.0 \
  --out ./codex-preview \
  --preview --json
```

The command never installs the preview or modifies Codex configuration. See the
[target evidence ledger](docs/dev/target-evidence.md) for the exact evidence
level and unresolved boundaries.

## Inert Oh My Pi preview

The same explicit package can render a deterministic Oh My Pi `18.2.6` candidate.
The output is YAML syntax under `preview/`, never `~/.omp`, and the command exits
nonzero because the report is non-applicable:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target oh-my-pi \
  --target-version 18.2.6 \
  --out ./oh-my-pi-preview \
  --preview --json
```

The Oh My Pi adapter emits no credentials or active target-home files. Its source
version, build blocker, config-inspector effects, and capability gaps are recorded
in the [target evidence ledger](docs/dev/target-evidence.md).

## Inert OpenClaw preview

The same explicit package can render a deterministic OpenClaw `2026.9.5` JSON5
candidate. The output is under `preview/`, never `~/.openclaw`, and the command
exits nonzero because the report is non-applicable:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target openclaw \
  --target-version 2026.9.5 \
  --out ./openclaw-preview \
  --preview --json
```

The OpenClaw candidate contains only source-grounded `agents.defaults.model` and
`agents.defaults.thinkingDefault` fields. It emits no credentials, auth profile
references, active target paths, or enforcement claims. The exact source/archive
hashes, config command effect review, and blocked native acceptance boundary are
recorded in the [target evidence ledger](docs/dev/target-evidence.md).

## Inert Hermes preview

The same explicit package can render a deterministic Hermes Agent `0.21.3`
candidate from source release `v2026.9.14`. The output is YAML syntax under
`preview/`, never `~/.hermes`, and the command exits nonzero because the report
is non-applicable:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target hermes \
  --target-version 0.21.3 \
  --out ./hermes-preview \
  --preview --json
```

The Hermes candidate contains only source-grounded `model.provider`,
`model.default`, and `agent.reasoning_effort` fields. It emits no credentials,
auth state, memory/session paths, or active target files. The exact source
hashes, entrypoint effect review, Python requirement, and blocked native
acceptance boundary are recorded in the [target evidence ledger](docs/dev/target-evidence.md).

## Commands

```text
profile-mango agents      Inspect documented agent sources without changing local state
profile-mango completion  Generate shell completion scripts
profile-mango home        Print the effective profile package home
profile-mango init        Create a deterministic starter profile package
profile-mango render      Render an inert, version-qualified candidate for a known target
profile-mango validate    Validate one PolicyProfile offline
profile-mango version     Display version information
```

## Library

The canonical domain is available as a Go library:

```go
profile, diagnostics := profilemango.ParseProfile(profileYAML)
resolved, diagnostics := profilemango.Resolve(profiles, "research")
resources, diagnostics := profilemango.DigestResources(packageRoot, resolved)
```

Module path:

```text
gitlab.com/ariel-frischer/profile-mango/pkg/profilemango
```

Versioned JSON Schemas live in [`schemas/`](schemas/).

## Support boundary

Codex is the first intended public adapter target. The shipped Codex `0.154.0`,
Oh My Pi `18.2.6`, and OpenClaw `2026.9.5` boundaries are preview-only and always
report current profiles as non-applicable. No applicable adapter, installation
engine, import, drift repair, credential handling, provider call, target-home
inspection, role projection, MCP projection, or identity management is shipped.
User-level preference storage is not shipped; the application home stores profile
packages and route identity only.

Other intended targets and experimental comparison evidence are documented as
roadmap or developer material, not as current compatibility claims. See
[project scope](docs/dev/project-scope.md) for the precise distinction.

## Development

The project uses Go `1.25.5`, pinned with [`mise.toml`](mise.toml).

```bash
mise install
make install
make test
make test-coverage
make lint
make format
make build
```

Useful CLI smoke checks:

```bash
go run ./cmd/profile-mango --help
go run ./cmd/profile-mango version
go run ./cmd/profile-mango home
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidance and
[SECURITY.md](SECURITY.md) for reporting security issues.

## Documentation

- [Documentation index](docs/index.md)
- [Project scope and goals](docs/dev/project-scope.md)
- [M0 validation evidence](docs/dev/m0-validation.md)
- [Target evidence ledger](docs/dev/target-evidence.md)
- [Adapter architecture](docs/dev/adapter-architecture.md)
- [Oh My Pi configuration reference](docs/dev/agents/oh-my-pi.md)
- [Agent reference pack](docs/dev/agents/README.md)
- [Changelog](CHANGELOG.md)

## License

[MIT](LICENSE)
