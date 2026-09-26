# Agent configuration validation and reference maintenance

This is the agreed development strategy, not a shipped validator, reference pack,
or compatibility claim. Follow the [constitution](../../.autospec/constitution.yaml)
and [project scope](project-scope.md). Actual observations belong in the
[target evidence ledger](target-evidence.md).

## Validate configuration without inference charges

Use the smallest useful checks, in this order:

| Layer | What it establishes | What it does not establish |
| --- | --- | --- |
| Schema, golden rendering and negative fixtures | Our parser/renderer handles known inputs deterministically and rejects invalid mappings | The target accepts or enforces the output |
| Native config validation | The pinned target parser accepts the generated configuration | Every key is used, or the selected profile takes effect |
| Native effective-config inspection | Selected settings and precedence resolve as expected, where the target exposes them | Runtime permissions, authentication route or model behavior are enforced |
| Negative and isolation checks | Invalid inputs fail as expected, overrides are understood and unrelated state is preserved | Unobserved runtime guarantees |

Prefer an agent's documented config-check, doctor, or effective-config command
when its effects are understood. A command called `doctor` may inspect credentials,
contact services or offer repairs. A startup flag may continue into a real session.
Neither is safe merely because its name sounds diagnostic. Check the command's
version-qualified documentation/source before running it. Do not invent a common
command across agents, or treat a successful exit as proof that unknown keys were
not silently ignored. Use sentinel values and deliberately invalid cases where
safe to demonstrate that the intended configuration was actually consumed.

Run native probes in synthetic homes and projects with a sanitized environment,
no credentials, blocked network access, bounded execution time, and private sockets
where required. Isolate all relevant configuration discovery paths, not just `HOME`;
account for XDG, project, system and environment layers. If a layer cannot be
isolated or inspected safely, record the limitation and stop the affected claim.
Do not reuse personal homes, sessions, daemons, hooks, extensions or child agents.
Disable update checks and telemetry through verified mechanisms where available.
Network blocking prevents outbound calls but does not authorize unsafe startup or
prove the command made no attempt to contact a service.

If the target offers no safe native inspector, retain schema/source/golden evidence
and mark native acceptance or effective state **unverified**. Do not replace the
missing check with a billable prompt. Explicitly authorized runtime observation is
a separate evidence level and must never be implied by these no-inference checks.

## Repository probe harness

[`scripts/agent-config-probe.sh`](../../scripts/agent-config-probe.sh) retains the
smallest repeatable native probe used for the Codex `0.154.0` build , and the Jcode fork
custom Jcode fork build. It requires explicit direct executable paths through
`PROFILE_MANGO_CODEX_BIN` and `PROFILE_MANGO_JCODE_BIN`; it never resolves a
launcher or reads a personal target home. Each invocation creates synthetic
`HOME`, `XDG_CONFIG_HOME`, `CODEX_HOME`, custom-fork Jcode config, and private
socket paths, clears the child environment with `env -i`, requests update and
telemetry suppression through the target's documented flags/environment, unshares
the network with bubblewrap, and uses a bounded timeout. It runs only
version/help, Codex feature-list, and custom-fork profile inspection commands.

Run it only with exact direct binaries and a task-owned probe root:

```bash
PROFILE_MANGO_CODEX_BIN=/path/to/codex \
PROFILE_MANGO_JCODE_BIN=/path/to/jcode-fork-custom \
PROFILE_MANGO_PROBE_ROOT="$PWD/.probe" \
./scripts/agent-config-probe.sh
```

For Codex, the probe hashes the resolved direct executable and rejects any
binary other than the pinned `codex-cli 0.154.0` SHA-256 before running target
commands. Its native consumption check is intentionally limited to the safe
`features list` inspector: positive, negative, malformed, and feature-only
runtime-override cases are asserted. Route, authentication, project/profile
precedence, permission/tool enforcement, and instruction/skill delivery remain
explicit gaps unless a version-qualified safe inspector is established.

For the experimental Jcode fork path, the probe likewise hashes the direct executable
and rejects any build other than `jcode v0.83.909-dev (ca8017a3a)` with SHA-256
`392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992` before
running the retained synthetic profile commands. This gate qualifies the
fixture identity only. The commands still do not establish runtime enforcement,
authentication, delivery, precedence, child overrides, hooks, extensions, MCP,
or upstream Jcode compatibility.

The custom-fork profile fixtures and commands are not evidence for upstream
Jcode, and successful parsing does not establish authentication, permissions,
network policy, tool enforcement, or runtime behavior. The experimental
`jcode-fork` adapter reuses this retained exact-build evidence and adds only
offline deterministic renderer, strict projection-boundary, resource, and
secret-exclusion tests. It remains non-applicable and does not justify a new
native probe, a target-home read, or a support claim. The current session Jcode
binary must never substitute for the retained exact artifact.

[`scripts/opencode-config-probe.sh`](../../scripts/opencode-config-probe.sh)
separately validates OpenCode `1.18.31`. It requires direct absolute paths to the
exact extracted binary and a Profile Mango JSONC candidate through
`PROFILE_MANGO_OPENCODE_BIN` and `PROFILE_MANGO_OPENCODE_CANDIDATE`. The script
rejects any binary other than SHA-256
`f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11`.
It clears the environment, isolates all HOME/XDG/config/data/cache/state,
managed-config, database, project, and temporary paths, supplies empty auth
content, disables project config, model fetching, auto-update, pruning, default
plugins, external plugins, and LSP downloads, blocks network access, and uses a
bounded timeout. It never invokes the default TUI or `run` command.

The probe checks version/help, native JSONC acceptance, merged model output,
inline-over-global precedence, explicit custom-file consumption and preservation,
malformed input, unknown-key behavior, and scratch-state writes. It inventories
state, creates an independent backup, restores after success or failure, and requires the
restored recursive type/mode/size/SHA-256 inventory to match the baseline. The
result establishes exact-version candidate parsing and partial merged-state and
precedence behavior. It does not establish zero-write inspection, general
per-field provenance, production installation, authentication, delivery,
permissions, tools, plugins, MCP, provider behavior, or runtime enforcement.

OpenClaw `v2026.9.5` and Hermes Agent `0.21.3` are intentionally not added to
this native probe harness.
Source review of `openclaw.mjs`, the `config validate --json` handler, config
snapshot/plugin metadata loading, and state-read helpers did not establish a
bounded no-write inspector. The candidate command can resolve dotenv and config
environment substitutions, resolve `$include` inputs, discover installed
plugin/workspace state, inspect SQLite state, and run migration-capable
normalization. The source also contains recovery-capable helpers, although this
handler does not pass an explicit suspicious-recovery opt-in. `OPENCLAW_CONFIG_READONLY=1`
protects the target config policy but does not prove those other effects are
absent. No OpenClaw command was run; its adapter records source/build-input
evidence and keeps native acceptance, effective state, delivery, precedence,
authentication, permissions, tools, skills, and runtime enforcement blocked.

Hermes source release `v2026.9.14` was reviewed before considering
`hermes config get model --json`, `hermes status`, or `hermes profile show`.
The launcher loads `HERMES_HOME/.env` and project dotenv data, parses effective
configuration, configures logging, and may create or secure the Hermes home.
The candidate commands read credential/auth, profile, plugin, gateway, skills,
context, and session state; `status --deep` can contact a provider endpoint and
inspect a local socket. No bounded credential-free, no-write execution was
established, so no Hermes command was run and the probe harness was not
broadened. Smart approvals are excluded because they invoke an auxiliary model
and do not cover file writes.

Claude Code `v2.1.278` has a separate opt-in
`scripts/claudecode-config-probe.sh` for the exact Linux x64 ELF. It uses cleared
environment, hidden personal homes/sockets, isolated network/PID/IPC namespaces,
synthetic writable state, and timeouts. A `--bare --settings` sentinel is consumed
before no-auth termination. This is a bounded-write model-consumption probe, not
a no-write inspector or authenticated session. `doctor` and `/status` remain
excluded. Full effective state, precedence, route/authentication,
permissions/tools, CLAUDE.md, skills, and runtime enforcement remain blocked.

## Record support by version and capability

Keep these facts distinct in the reference/evidence records:

- **Documentation basis:** release/tag/commit covered, source URL and retrieval date;
  label an unversioned page as unversioned rather than guessing a release.
- **Installed version:** the inspected binary/build, not proof of compatibility.
- **Tested version:** exact binary/build, platform, relevant environment, fixture or
  adapter revision, command, date and observed result.
- **Supported capability:** the tested subset and evidence level, known limitations,
  and links to reproducible checks. Experimental targets remain explicitly separate.

Start with exact tested versions. Do not infer supported ranges from semantic
versioning, a newer release, or unchanged documentation. A new release begins as
untested, not automatically supported or known broken. Failed probes should identify
the affected capability and version rather than invalidating unrelated evidence.
Use ranges only when an explicit test policy and evidence justify them. Existing
required-but-unknown properties continue to block applicability.

Use one small source/version manifest and the existing evidence ledger rather than
competing compatibility tables. The reference-pack task decides the minimal format;
this guide does not add a product schema or CLI contract. No full-profile public
target adapter is shipped today; narrow install subsets are qualified separately. Claude Code, Codex, Pi, Oh My Pi, OpenClaw, and
Hermes preview rendering remain non-applicable. Claude Code has only bounded
explicit-file model consumption evidence, not full startup/effective-state proof. Pi is blocked
on startup settings/auth/model/session effects, migrations, project and extension
discovery, package/update subprocesses, network-capable model/catalog paths, and
writes that were not proven bounded. Oh My Pi's separate narrow installer now has
exact source-native addon and read-only getter evidence, but its standalone startup,
general config inspection, authentication, precedence, and enforcement remain blocked.
OpenClaw is blocked on its missing runtime
artifact and unaccepted config inspection path; Hermes is blocked on its Python
runtime/build requirement and unsafe candidate inspector paths.

## Keep useful configuration references locally

The planned location is `docs/dev/agents/`, maintained by **ap-8vz**. It is tracked
contributor documentation, not private material. Cover Codex, Claude Code, Pi and
Oh My Pi as separately qualified variants, OpenClaw, Hermes, , and the Jcode fork
experimental-only custom Jcode fork. Keep experimental details out of public
support promises.

Each concise reference should cover the configuration formats and paths, selection
and precedence, model/provider/authentication/effort fields, permissions and tools,
instructions and skills, override surfaces, and safe validation commands or gaps.
Include pinned official source locators where available, official latest-doc links,
retrieval dates, applicability and a reproducible source revision/hash where useful.
Distinguish documented facts from observed behavior. Summarize relevant material
and use only permitted excerpts rather than copying entire documentation sites.

Agents should consult these references first once available. Fresh local evidence
should eliminate routine repeated research, not prohibit a targeted official lookup
when the installed version differs, a source is stale or the local summary is
insufficient. The populated [agent configuration references](agents/README.md)
are indexed in `docs/index.md` and linked from the agent instructions. They remain
documentation provenance, not executed evidence or a compatibility claim.

## On-demand source drift check

From the repository root, run the read-only check when reviewing upstream changes:

```bash
make check-agent-sources
# or select one target and request structured output
go run ./cmd/mango agents check --target codex --json
```

The command reads [`agents/sources.json`](agents/sources.json), fetches only
HTTP(S) sources, bounds request time, response size, and redirects, and never
writes the manifest, references, evidence ledger, adapter mappings, installed
agents, configurations, credentials, or support records. Use `--manifest` to
check an explicitly selected copy and `--target` for one manifest target.
It prints each pending and completed source to stderr so a slow request does
not look idle. The final text or `--json` report remains on stdout. Sources
are checked sequentially, with a 10-second timeout per request by default;
use `--timeout` and `--target` to bound a focused run.

Each source is reported as one of these states:

- **unchanged** — the fetched body matches an existing manifest hash. This is
  source evidence only, not target compatibility evidence.
- **changed** — a hashed body differs. Content drift is not a compatibility
  regression or support promotion; review the diff and run targeted native
  probes.
- **unversioned** — the source is reachable but has no manifest hash. The
  report preserves its target version context without claiming unchanged
  content.
- **unavailable** — the request failed or returned a non-success status. It is
  never treated as unchanged, and the command exits nonzero when any source is
  unavailable.
- **relocated** — a redirect was followed or surfaced. Review the redirect and
  source diff before changing provenance.

Non-public local snapshot locators are reported as `not_checked`; the command
does not inspect personal agent repositories or homes. A successful check does
not promote a release to supported or tested status. To refresh intentionally,
review the source diff and exact version/revision, update the reference pack in
a separate reviewed change, retain prior provenance/evidence in history and the
ledger, and run the relevant no-inference probes before making any support
claim. A manually chosen cadence such as after an upstream release or monthly
is sufficient; this project does not install a scheduler or background service.

## Work ownership and order

| Bead | Responsibility | Prerequisite |
| --- | --- | --- |
| **ap-8vz**: Versioned local configuration references | Curated configuration knowledge and source/version provenance | First reference work |
| **ap-3kw**: Codex and experimental Jcode compatibility probes | Executed, isolated no-inference evidence | ap-8vz |
| **ap-3pa**: First version-qualified Codex adapter | Deterministic rendering and applicability diagnostics | ap-3kw and existing independent M0 validation ap-sha |
| **ap-a07**: On-demand documentation/release drift check | Later reference refresh and revalidation guidance | ap-8vz; does not block adapter work |

**ap-6fu** remains the multi-target MVP epic. Other target probes follow the same
pattern when scoped, without pretending Codex/Jcode evidence covers them. This
coordination update is **ap-4me** and does not execute the future work above.
