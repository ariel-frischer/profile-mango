# Pi configuration reference

**Reference date:** 2026-09-20. **Documentation pin:** release `v0.86.1`,
commit `13cbf77df2396303013a41646bcfa77b4271ae56`. **Status:** intended MVP target
variant; not installed, tested, or supported by profile-mango, and no adapter
ships. Pi is qualified independently from [Oh My Pi](oh-my-pi.md).

## Configuration and precedence

Pinned documentation describes JSON settings at `~/.pi/agent/settings.json` and
project `.pi/settings.json`. Project settings override global settings; nested
objects merge and arrays such as `defaultTools` replace. Provider, model, and
thinking level have settings and CLI surfaces. Authentication resolution may use
a CLI API key, `auth.json`, environment variables, or `models.json`; portable
profiles must never copy credentials or infer the effective route.

## Tools, instructions, and skills

Pi documents read, write, edit, and shell tools. Its security guide does not
present a permission popup or built-in sandbox boundary; trusting a project is
not containment. Instructions can come from global and ancestor/project
`AGENTS.md` or `CLAUDE.md`; an `AGENTS.override.md` in the same directory replaces
the normal candidate. Skill discovery and instruction delivery are not evidence
of access enforcement.

## Candidate inspection and gaps

Version/help, model listing, and interactive settings surfaces are documented,
but no noninteractive command is established that emits a complete redacted
effective configuration with provenance or performs standalone schema validation.
A future probe must inspect source and effects first, then use isolated user and
project paths with no credentials or provider access. The absence of a safe
inspector remains an explicit evidence gap.

Pinned sources: [settings][settings], [providers][providers], [models][models],
[security][security], [skills][skills], and [coding-agent README][readme].

[settings]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/settings.md
[providers]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/providers.md
[models]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/models.md
[security]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/security.md
[skills]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/docs/skills.md
[readme]: https://github.com/earendil-works/pi/blob/v0.86.1/packages/coding-agent/README.md
