# OpenClaw configuration reference

**Reference date:** 2026-09-20. **Documentation pin:** release `v2026.9.5`,
commit `ec9c1a13db8938e5a3eaa51fca2e981cde2395a9`. **Status:** intended MVP target;
not installed, tested, or supported by profile-mango, and no adapter ships.

## Configuration and precedence

Pinned documentation describes strict JSON5 at `~/.openclaw/openclaw.json`,
relocatable with `OPENCLAW_CONFIG_PATH`; invalid or unknown configuration fails
startup, and `$include` can split configuration. Model configuration supports
primary and fallback selection, agent overrides, allowlists, provider-local auth
profile rotation, and thinking defaults. These documented surfaces do not prove
which route or credential wins in a future isolated run.

## Policies, instructions, and skills

Tool policy combines general and provider profiles, allow/deny rules, sender
policy, and sandbox gates; documentation says deny wins. The agent workspace is
not itself a sandbox, and instruction files are guidance rather than enforcement.
Skills can come from project, user/state, and bundled sources. Later probes must
account for all applicable layers and bypass surfaces.

## Candidate inspection and gaps

`OPENCLAW_CONFIG_READONLY=1 openclaw config validate --json` is a candidate
read-only validation command. Config-schema access and redacted reads are also
documented, but no command is established that emits every merged runtime default
with per-key provenance. These candidates were not executed. A future probe must
verify read-only behavior at the pinned revision, isolate configuration and state,
use synthetic credentials, block network access, and avoid gateway/session startup.

Pinned sources: [configuration][configuration], [models][models],
[agent models][agent-models], [secrets][secrets], [tool policy][tools],
[workspace][workspace], [skills][skills], and [config CLI][cli].

[configuration]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/configuration.md
[models]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/concepts/models.md
[agent-models]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/config-agents/models.md
[secrets]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/config-secrets-env.md
[tools]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/gateway/config-tools/tool-policy.md
[workspace]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/concepts/agent-workspace.md
[skills]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/tools/skills.md
[cli]: https://github.com/openclaw/openclaw/blob/v2026.9.5/docs/cli/config.md
