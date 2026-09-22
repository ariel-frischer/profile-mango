# M0 offline validation

M0 has automated evidence for clean local installation, installed-binary smoke
behavior, and all four Draft 2020-12 JSON Schema contracts. The checks are
reproducible and require no credentials, target agents, providers, or model calls.

## Clean-install and CLI evidence

Run the installed-binary suite from the repository root:

```bash
go test ./integration -run TestCleanOfflineInstallAndInstalledBinary -count=1 -v
```

The test creates temporary `HOME`, `XDG_CONFIG_HOME`, `GOCACHE`, and `GOBIN`
directories. It sets `GOTOOLCHAIN=local` and `GOPROXY=off`, reuses the already
populated `GOMODCACHE`, runs local `go build` and `go install`, and then invokes
only the installed `mango` binary. It verifies:

- root help, plain version output, and the effective profile-home path without
  creating it;
- JSON validation success for the route-only and constrained read-only fixtures;
- JSON validation failure for the unsupported fixture, with one stable
  `yaml.strict` diagnostic.

The suite does not read a personal agent home or start an agent/provider process.
It exercises only these CLI paths; it does not prove that unrelated binary paths
contain no subprocess support or that an operating-system network sandbox is in
effect.

## Schema evidence

Run the schema suite from the repository root:

```bash
go test ./schemas -count=1 -v
```

The suite compiles the profile, bindings, plan, and manifest schemas as Draft
2020-12 and checks a positive and negative instance for each. Every schema and
relative filename URI is registered from checked-in JSON before compilation. An
empty external URL loader makes an unresolved reference fail instead of reading a
file or fetching the network.

## Interpretation boundary

Passing canonical parsing or schema validation establishes only that an input
conforms to the M0 contract. The constrained read-only fixture remains valid
canonical input, but this evidence does not establish target applicability, delivery,
or enforcement. The separate Codex preview renderer remains explicitly
non-applicable, and required unsupported or unknown target capabilities still block
future application as described in the [target evidence ledger](target-evidence.md).
