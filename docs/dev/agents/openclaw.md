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

**Native named profiles:** the pinned source at
`ec9c1a13db8938e5a3eaa51fca2e981cde2395a9` maps `openclaw --profile <name>` to
`<home>/.openclaw-<name>/openclaw.json`. profile-mango installs named profiles
there; see [Named-profile installation](#named-profile-installation-2026-09-23)
for the file and line evidence. These are config/state profiles, distinct from
provider-local authentication profiles.

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

## Named-profile installation, 2026-09-23

`mango install <profile> --target openclaw` writes the two qualified
fields to the config that `openclaw --profile <profile>` reads, and leaves
the default config alone unless you pass `--default`. The plan prints
`use it: openclaw --profile <profile>`. Evidence is **source-only**: the pinned
files below were fetched read-only from GitHub raw at commit
`ec9c1a13db8938e5a3eaa51fca2e981cde2395a9` on 2026-09-23. The locally installed
npm package is `2026.9.4`, not the pinned version, so it was not read or run as
evidence, and no native `--profile` probe was run.

| Pinned file | Lines | Observation |
| --- | --- | --- |
| [`src/entry.ts`](https://github.com/openclaw/openclaw/blob/ec9c1a13db8938e5a3eaa51fca2e981cde2395a9/src/entry.ts) | 131-134, 194-203 | Root `--profile` is parsed and applied to the environment before any command runs. |
| [`src/cli/profile.ts`](https://github.com/openclaw/openclaw/blob/ec9c1a13db8938e5a3eaa51fca2e981cde2395a9/src/cli/profile.ts) | 41-65 | `--profile <name>` must pass `isValidProfileName`, and it can't be combined with `--dev`. |
| same | 89-95 | The selected state directory comes from `resolveProfileStateDir(profile, env, homedir)`. |
| same | 110-118, 135-147 | `OPENCLAW_STATE_DIR` becomes the profile state directory, and `OPENCLAW_CONFIG_PATH` becomes `<stateDir>/openclaw.json`. An existing `OPENCLAW_CONFIG_PATH` or `OPENCLAW_STATE_DIR` is kept unless it is the inherited profile's own canonical path. |
| [`src/cli/profile-utils.ts`](https://github.com/openclaw/openclaw/blob/ec9c1a13db8938e5a3eaa51fca2e981cde2395a9/src/cli/profile-utils.ts) | 6-14 | Names match `^[a-z0-9][a-z0-9_-]{0,63}$`, case-insensitively. |
| same | 31-42 | The state directory is `<home>/.openclaw-<name>`, and `default` (any case) maps to `<home>/.openclaw`. |
| [`src/infra/home-dir.ts`](https://github.com/openclaw/openclaw/blob/ec9c1a13db8938e5a3eaa51fca2e981cde2395a9/src/infra/home-dir.ts) and [`packages/normalization-core/src/home-dir.ts`](https://github.com/openclaw/openclaw/blob/ec9c1a13db8938e5a3eaa51fca2e981cde2395a9/packages/normalization-core/src/home-dir.ts) | 14-25; 28-35, 45-53 | `<home>` is `OPENCLAW_HOME` when set, otherwise `HOME`, then `USERPROFILE`, then the OS home. |
| [`src/config/paths.ts`](https://github.com/openclaw/openclaw/blob/ec9c1a13db8938e5a3eaa51fca2e981cde2395a9/src/config/paths.ts) | 222-230, 266-274 | Config resolution uses `OPENCLAW_CONFIG_PATH` directly, so the profile reads exactly `<home>/.openclaw-<name>/openclaw.json`. |

Full fetched-file SHA-256 values: `entry.ts`
`7714d6416147764e2499ef6b7cc694cd553b820c04b12a057611efa0445f615d`,
`profile.ts` `969d6172f0ab4f4bcd1a3541bfc1bc0d9e80f9900f8055aa1b29dcedeb861dbe`,
`profile-utils.ts` `f622c1c8922ffd91fa9813373f4a60c85b1faabb8b2d5e0961310d3455586e11`,
`infra/home-dir.ts` `6c850a1712a5e33fdacdc6885a876843da2d854c3a5968e7396fd41d3684b184`,
`normalization-core/src/home-dir.ts` `680f0632da8ad6a2cc74d8daf7ed89163f8cf7aad4bd29a5725134b0da03eec7`,
`config/paths.ts` `9b97d69001407f6fed83e018ce04d379c0262806d2c1be7012d197bf0356cf0e`.

**Path rule.** profile-mango derives the profile config from the main config
path. `<dir>/.openclaw/openclaw.json` gives `<dir>/.openclaw-<name>/openclaw.json`.
That is correct when `<dir>` is OpenClaw's effective home: the default
`~/.openclaw/openclaw.json`, or an explicit `--config` or `OPENCLAW_CONFIG_PATH`
under the home that OpenClaw will use. Any other main config path is blocked with
`install.named_profile_path_unsafe`, because a relocated config says nothing
about where `--profile` looks. profile-mango doesn't read `OPENCLAW_HOME`. If you
set it, pass `--config $OPENCLAW_HOME/.openclaw/openclaw.json`.

**`default` profile.** OpenClaw maps `--profile default` to the default state
directory and keeps an existing `OPENCLAW_CONFIG_PATH`. So installing a profile
named `default` patches the main config in place, and no second file is written.

**Separate state.** A new profile directory has no authentication, sessions,
workspace, or plugins. profile-mango doesn't copy them, and the plan says so
(`openclaw.install.profile_state_separate`). An `OPENCLAW_CONFIG_PATH` or
`OPENCLAW_STATE_DIR` already exported to another location still wins over
`--profile`.

**Validation.** Unit, command, and compiled-binary integration tests cover these
cases in synthetic homes: create, noop reinstall, `--default`, `default`-name,
blocked relocated paths, and undo. A sandbox run of the built binary in a scratch
`HOME`, with `PROFILE_MANGO_HOME` inside it, installed `coding`. It created
`~/.openclaw-coding/openclaw.json` and left `~/.openclaw/openclaw.json`
byte-identical (SHA-256 before and after). It then applied `--default`, and two
undos restored the original default config and removed the profile config.
