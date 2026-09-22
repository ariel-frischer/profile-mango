# profile-mango

<div align="center">

<pre>
█▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█
</pre>

**Unified Agent Profiles**

</div>

`profile-mango` is a strict profile format, Go library, and CLI for portable
coding-agent behavior. It separates shared intent from machine-local routes,
resolves profiles deterministically, and fails closed when a target cannot prove
that it preserves a requirement.

Profile Mango manages **profiles**, not every setting exposed by every agent. A
profile contains reusable behavior and resources that should travel between coding
agents. Adapters use native configuration only where needed to deliver that profile
and make unsupported or target-specific differences visible. They are not intended
to become general-purpose configuration managers for Codex, Claude Code, OpenClaw,
Hermes, or other targets.

## What is a profile?

A profile is a user-owned bundle of the behavior and resources you want to carry
between coding agents:

- **Instructions:** reusable agent guidance, including files such as `AGENTS.md`.
- **Skills:** `SKILL.md` packages and their supporting resources.
- **Policy:** permissions, tool allow/deny rules, and other behavioral requirements.
- **Routing intent:** provider, transport, authentication mode, model, and effort,
  kept separate from target-owned credentials.

Instead of rewriting the same setup in Codex TOML, Claude Code JSON, Pi settings,
OpenClaw JSON5, Hermes YAML, OpenCode JSONC, and every other agent-specific format, you define the
shared intent once and select a named profile. Target adapters translate the
parts each agent understands and report anything they cannot faithfully deliver
or enforce. The intended workflow is to switch profiles across agents without
maintaining a separate copy of your instructions, skills, and policy for each one.

Settings with no meaningful role in a portable coding-agent profile remain owned by
the target. This includes unrelated channels, gateways, UI preferences, telemetry,
updates, session databases, and other product-specific operations. Credentials and
authentication stores always remain target-owned.

The current prototype validates and resolves these bundles, produces deterministic,
inspectable inert previews, and can transactionally apply only the exact OpenCode
`1.18.31` top-level `model` field at one explicit path. Every other production target
and every broader OpenCode profile capability remain install-blocked.

> **Status:** private prototype. Validation and project scaffolding are offline.
> Exact-version renderers remain inert. Installation supports only the qualified
> target-specific subsets listed below, with explicit paths, deterministic diffs,
> hash-bound consent, default backups, stale checks, rollback, and guarded recovery.
> Full-profile authentication, delivery, permissions, and enforcement remain
> unverified. Native qualification uses isolated disposable state, never personal
> global agent configuration or authenticated provider requests. See the
> [evidence ledger](docs/dev/target-evidence.md) for precise versions and limits.

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

### Install the agent skill

The bundled [`profile-mango` agent skill](.agents/skills/profile-mango/SKILL.md) teaches
coding agents the safe scaffolding, profile authoring, validation, and inert
preview workflow. Install it with the same one-liner used by other skills-based
CLI repositories:

```bash
npx skills add ariel-frischer/profile-mango
```

The repository is still private, so the GitHub shorthand above becomes usable
after `ariel-frischer/profile-mango` is published publicly. Until then, the
repository-owned skill remains available directly from
`.agents/skills/profile-mango/SKILL.md` in an authorized checkout.

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

## Plan-first install boundary

`install` accepts repeatable exact `target@version` selections or `--all`, resolves
the same explicit profile inputs as `render`, and emits a deterministic plan ID.
Only these exact target subsets are currently installable. Other targets and
unsupported required permissions, tools, instructions, or skills remain blocked.

| Target | Installed fields | Native evidence boundary |
| --- | --- | --- |
| OpenCode `1.18.31` | Top-level `model` | Exact native merged config, limited precedence |
| Claude Code `2.1.278` | Top-level `model` | Exact ELF explicit settings-file consumption before no-auth termination |
| Pi `0.86.1` | `defaultProvider`, `defaultModel`, `defaultThinkingLevel` | Exact settings-module getters and project-over-global precedence only |

```bash
profile-mango install route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target opencode@1.18.31 \
  --config-path opencode=/explicit/disposable/opencode.jsonc \
  --override --json
```

Review the plan, then repeat the same explicit inputs with
`--apply --yes --expect-plan <planID>`. `--override` is accepted only because this
adapter replaces or inserts the single top-level model field from the current
snapshot. Comments, unknown keys, unrelated bytes, modes, provider options, and
credential-shaped target-owned state are preserved.

Claude Code uses `--target claude-code@2.1.278` and
`--config-path claude-code=/explicit/disposable/settings.json`, with an Anthropic
native-route binding. Only strict JSON `model` is changed. Exact native evidence
covers explicit `--settings` file consumption and no-auth early termination, not
default-path precedence, authentication, effort, or full-profile activation.

The application engine supports destination-bound consent, create-only backups,
stale snapshot rejection, ownership manifests, atomic per-file replacement,
journals, rollback, and guarded recovery. A non-interactive apply requires
`--apply --yes --expect-plan <sha256>`. There is no unbound force path. OpenCode
model application does not run OpenCode or inspect auth stores, sessions, plugins,
MCP, providers, or the network. Any live configuration path still requires separate
path-specific user approval after a disposable backup/restore rehearsal.

All commands accept the global `--non-interactive` flag. It disables prompts, not
safety checks or consent requirements. Interactive install apply prompts only on
a real terminal and accepts `y` or `yes`; an empty answer declines. JSON output,
redirected input, and `--non-interactive` never prompt and require explicit
`--yes --expect-plan <planID>` to apply.

JSON planning emits one `Plan` object. A ready JSON apply now emits one
`ApplyReport` object, including a failure report when the apply engine fails,
rather than concatenating the plan and report. Automation that consumed the old
two-object stream must read a single report instead. A blocked apply emits its
blocked plan and exits nonzero. Invalid flags or declined consent can fail before
any report is produced. Always check the exit status as well as the JSON status.

## Representative profiles

[`examples/jcode-like/`](examples/jcode-like/) contains credential-free `base`,
`daily`, `review`, and `research` profiles modeled on common named Jcode workflows
without reading or copying live Jcode configuration. Installed-binary integration
tests validate each profile and render it through every public adapter. Every
render remains inert. Install plans remain blocked for all representative profiles
that carry instructions, skills, permissions, or tools; only compatible route-only profiles can use the narrow install subsets.

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

## Inert Pi preview

The same explicit package can render a deterministic Pi `0.86.1` JSON settings
candidate. The output is under `preview/`, never `~/.pi/agent`, and the command
exits nonzero because the report is non-applicable:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target pi \
  --target-version 0.86.1 \
  --out ./pi-preview \
  --preview --json
```

The candidate contains only the exact source-grounded `defaultProvider`,
`defaultModel`, and `defaultThinkingLevel` settings keys. It emits no credentials,
auth-file data, model-store data, provider URLs, sessions, or active target files.
Pi is qualified independently from Oh My Pi. Its immutable source/package hashes,
static startup effect review, and blocked native command boundary are recorded in
the [Pi configuration reference](docs/dev/agents/pi.md) and [target evidence
ledger](docs/dev/target-evidence.md).

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

## Inert Claude Code preview

The same explicit package can render a deterministic Claude Code `2.1.278`
candidate. The output is a documentation-context JSON settings preview under
`preview/`, never `~/.claude`, and the command exits nonzero because the report
is non-applicable:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target claude-code \
  --target-version 2.1.278 \
  --out ./claude-code-preview \
  --preview --json
```

The candidate contains only the route model. It emits no provider, transport,
authentication, effort, credentials, active target files, or enforcement claims.
The immutable npm package hashes, release commit, launcher effect review, and
blocked native command boundary are recorded in the [target evidence ledger](docs/dev/target-evidence.md).

## Inert OpenCode preview

OpenCode `1.18.31` has an exact-release inert JSONC renderer. It emits only the
`model` candidate in `provider/model` form under `preview/`. The exact binary
natively accepted that corrected candidate and emitted the model in merged config,
but rendering still exits nonzero because installation, authentication identity,
effort, full precedence/provenance, delivery, permissions, tools, plugins, MCP,
and enforcement remain blocked:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target opencode \
  --target-version 1.18.31 \
  --out ./opencode-preview \
  --preview --json
```

The render command itself does not execute OpenCode, read `~/.config/opencode`,
inspect `~/.local/share/opencode/auth.json`, or install the candidate. Separately,
the approved `scripts/opencode-config-probe.sh` ran only the exact direct binary's
version, help, and `debug config` paths inside synthetic scratch with blocked
network and no TUI/session/provider/credentials. It observed target-owned scratch
writes, restored an independent backup, and verified the restored inventory
byte-for-byte. See the [target evidence ledger](docs/dev/target-evidence.md).

## Experimental Ariel custom Jcode fork preview

The explicit `ariel-jcode` target is an **experimental-only** renderer for
Ariel's custom Jcode fork build `jcode v0.83.909-dev (ca8017a3a)`. It is not
upstream Jcode, is not a supported public target, and is not included in the
support list below. The candidate is deterministic TOML syntax under `preview/`
and the command exits nonzero because the report remains non-applicable:

```bash
profile-mango render route-only \
  --profiles pkg/profilemango/testdata/fixtures \
  --resource-root pkg/profilemango/testdata \
  --bindings pkg/profilemango/testdata/fixtures/bindings.yaml \
  --target ariel-jcode \
  --target-version '0.83.909-dev (ca8017a3a)' \
  --out ./ariel-jcode-preview \
  --preview --json
```

The preview uses only retained exact-build synthetic profile-resolution evidence
for provider/model/effort, observed tool selectors, skill selectors, and
instruction metadata. It never reads a Jcode home or emits credentials,
provider profiles, arbitrary target keys, or active target files. The source
documentation snapshot and all native applicability gaps remain separate in the
[target evidence ledger](docs/dev/target-evidence.md).

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

Codex is the first intended public adapter target. The shipped Claude Code
`2.1.278`, Codex `0.154.0`, Pi `0.86.1`, Oh My Pi `18.2.6`, OpenClaw `2026.9.5`,
and Hermes `0.21.3` boundaries are preview-only and always
report current profiles as non-applicable. No applicable adapter, installation
engine, import, drift repair, credential handling, provider call, target-home
inspection, role projection, MCP projection, or identity management is shipped.
User-level preference storage is not shipped; the application home stores profile
packages and route identity only.

Other intended targets and experimental comparison evidence are documented as
roadmap or developer material, not as current compatibility claims. See the
[roadmap](ROADMAP.md) for product direction and
[project scope](docs/dev/project-scope.md) for the precise current boundary.

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

- [Roadmap](ROADMAP.md)
- [Documentation index](docs/index.md)
- [Project scope and goals](docs/dev/project-scope.md)
- [M0 validation evidence](docs/dev/m0-validation.md)
- [Target evidence ledger](docs/dev/target-evidence.md)
- [Adapter architecture](docs/dev/adapter-architecture.md)
- [Pi configuration reference](docs/dev/agents/pi.md)
- [Oh My Pi configuration reference](docs/dev/agents/oh-my-pi.md)
- [Agent reference pack](docs/dev/agents/README.md)
- [Changelog](CHANGELOG.md)

## License

[MIT](LICENSE)
