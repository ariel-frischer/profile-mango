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
this guide does not add a product schema or CLI contract. No public target adapter
is shipped today.

## Keep useful configuration references locally

The planned location is `docs/dev/agents/`, maintained by **ap-8vz**. It is tracked
contributor documentation, not private material. Cover Codex, Claude Code, Pi and
Oh My Pi as separately qualified variants, OpenClaw, Hermes, and experimental-only
Ariel Jcode. Keep experimental details out of public support promises.

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

## Refresh deliberately, not automatically

**ap-a07** owns a later on-demand local workflow or skill:

1. Compare the recorded sources and releases with upstream using ordinary document
   fetches, not target inference, paid APIs or installed-agent upgrades.
2. Report unchanged sources, meaningful configuration changes, new releases,
   relocated sources and fetch failures separately. A failed fetch is not unchanged.
   Hash differences flag investigation; cosmetic page churn is not a compatibility
   regression, and equal docs hashes do not prove unchanged target behavior.
3. Identify affected capabilities and the smallest relevant no-inference probes.
   Leave uncertain effects explicitly unknown.
4. Review source/reference changes and retain old provenance. Update local references
   intentionally, then promote support evidence only after the appropriate checks.

No automatic personal-config edits, adapter rewrites, support-range expansion or
background scheduler. Start with an operator-invoked check before adding support
for a new target version and before releases that change compatibility claims.
Occasional maintenance checks are useful, but no fixed cadence is required yet.

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
