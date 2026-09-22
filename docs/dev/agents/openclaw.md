# OpenClaw configuration reference

**Reference date:** 2026-09-21. **Documentation pin:** release `v2026.9.5`,
commit `ec9c1a13db8938e5a3eaa51fca2e981cde2395a9`. **Status:** profile-mango
ships an exact-version inert preview renderer and a bounded two-default installer.
Source-native consumption was qualified on 2026-09-22. Full startup, authentication,
and policy enforcement remain blocked.

## Exact source observation

The task-owned qualification checkout verified the annotated tag and commit above.
The deterministic source archive SHA-256 is
`0e15e679795134cf7d488302f2bdaf0682ad4413e19a7f5c6cc22584f03d02a4`. Build-input
hashes are `package.json`
`16a72a5768e629f703a89b27417feb05d4c4ece7df816c5d1364d202c5a2e619`,
`pnpm-lock.yaml`
`2415d6f2835843db64b1f4eadd676fa64c2fc9f5ddd868e73ae3deb2f0ed1b78`, and
`scripts/build-all.mts`
`65f6726bbc3b5e18f02d268458b73648da5da5a0ddfb5dd5eca6e3b9a8984e17`. The
review platform was Linux x86_64 with Node `v24.21.0` and PNPM `10.28.2`.

The immutable checkout had no `node_modules` or `dist/entry.js`. No dependency
install or OpenClaw build was run, so there is no produced runtime artifact hash.
The adapter's evidence hash is the source archive hash, not a claim that an
executable target build was produced.

## Configuration and precedence

Pinned documentation and source describe strict JSON5 at
`~/.openclaw/openclaw.json`, relocatable with `OPENCLAW_CONFIG_PATH`; invalid or
unknown configuration fails startup, and `$include` can split configuration.
Model configuration supports `agents.defaults.model.primary`, ordered
`agents.defaults.model.fallbacks`, per-agent overrides, model allowlists,
provider-local auth profile rotation, and `agents.defaults.thinkingDefault`.
These documented fields establish candidate syntax only. They do not prove which
route, credential, override, or effective value wins in a future run.

The inert adapter emits only:

```json5
{
  agents: {
    defaults: {
      model: { primary: "provider/model", fallbacks: [] },
      thinkingDefault: "high",
    },
  },
}
```

It never emits credentials, auth profile references, provider URLs, workspace or
state paths, plugin configuration, active `openclaw.json` paths, or permission
policy as if it were enforced. Candidates are written as
`preview/<profile>.config.json5.preview` only when the caller explicitly requests
an inert preview.

## Policies, instructions, and skills

Tool policy combines general and provider profiles, allow/deny rules, sender
policy, plugins, and sandbox gates; documentation says deny wins. The agent
workspace is not itself a sandbox, and instruction files are guidance rather than
enforcement. Skills can come from project, user/state, and bundled sources. The
adapter therefore reports route/authentication, delivery, precedence, permission,
tool, and runtime-enforcement gaps instead of claiming equivalence.

## Candidate inspection and safety boundary

The pinned source review covered `openclaw.mjs`,
`src/cli/program/preaction.ts`, `src/cli/config-cli.ts`,
`src/config/io.snapshot.ts`, `src/config/io.context.ts`,
`src/config/io.plugin-metadata.ts`, and
`src/state/openclaw-state-db-readonly.ts`.

`OPENCLAW_CONFIG_READONLY=1 openclaw config validate --json` is documented as a
read-only `openclaw.json` validation command, but it was **not executed**.
The exact handler reads config snapshots with plugin metadata while `observe:false`
only suppresses config observation. The reviewed path can load dotenv and config
environment substitutions, resolve `$include` files, discover installed plugin
indexes and agent workspaces, inspect state databases, and run migration-capable
normalization. The source also contains recovery-capable helpers, although this
handler does not pass an explicit suspicious-recovery opt-in. Network blocking
and synthetic homes do not by themselves prove that these local effects are
absent. No OpenClaw command, gateway, daemon, session, provider,
credential, plugin, update, or native probe was run.

Consequently, the adapter reports these independent evidence levels as blocking:

- config syntax fidelity is source-grounded, but native config acceptance is unverified
- merged effective state and per-field provenance are unverified
- config and runtime precedence are unverified
- route authentication identity and credential handling are unverified
- permissions, tools, plugins, instructions, skills, delivery, and runtime enforcement are unverified

A future probe must re-review the exact source, use a direct exact artifact, isolate
all config/state/workspace/include paths, sanitize the environment, use synthetic
secret sentinels, block the network, bound execution, and avoid gateway/session
startup. See the [target evidence ledger](../target-evidence.md).

## Pinned sources

[configuration]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/configuration.md
[models]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/concepts/models.md
[agent-models]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/config-agents/models.md
[secrets]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/config-secrets-env.md
[tools]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/config-tools/tool-policy.md
[workspace]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/concepts/agent-workspace.md
[skills]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/tools/skills.md
[cli]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/cli/config.md


## Bounded installation qualification, 2026-09-22

The exact `2026.9.5` installer applies only `agents.defaults.model.primary` and
`agents.defaults.thinkingDefault`, preserving unrelated JSON5 bytes, comments,
fallbacks and target-owned state. Native source getters consumed both fields,
and demonstrated per-agent overrides and fallback limits. Root independently
reran the probe with hash-gated source modules and actual compiled-CLI output.
Network/PID/IPC isolation, cleared environment, dropped capabilities, allowlisted
runtime/source mounts, no broad `/etc` mount, hidden homes/sockets and a timeout
bounded the probe. No full startup, credentials, providers, gateway, sessions,
hooks, plugins, MCP, or policy enforcement were exercised.

The codeload gzip hash differs from the retained archive despite matching exact
peeled commit/tree and imported-source hashes. Both archive hashes remain recorded
in the [native evidence fixture](../../../pkg/adapters/openclaw/testdata/openclaw-native-field-consumption.evidence.json).
The [probe](../../../pkg/adapters/openclaw/testdata/openclaw-native-field-consumption-probe.mjs)
accepts an optional generated-config path; otherwise it uses a synthetic fixture.
Earlier M0 inspection observations above are historical, not current install gates.
