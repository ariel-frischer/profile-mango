# Claude Code configuration reference

**Reference date:** 2026-09-20. **Documentation:** official mutable,
unversioned pages. **Release context:** `v2.1.278`, commit
`bf7d404e26a5fb6167d21b46c93a2bf6c22ab274`. The release pin does not version
the documentation. **Status:** intended MVP target; not installed, tested, or
supported by profile-mango, and no adapter ships.

## Configuration and precedence

Claude Code documents strict JSON settings in user `~/.claude/settings.json`,
shared project `.claude/settings.json`, project-local
`.claude/settings.local.json`, and managed settings. Documented precedence is
managed, command line, local project, shared project, then user, while some
collections merge. Verify that behavior against the exact future test build.

Model selection can come from CLI, settings, environment, and interactive
selection. Authentication may use Anthropic or documented cloud-provider routes.
Effort has CLI, environment, settings, and model-specific surfaces. These facts
do not prove an effective route or authentication mode.

## Permissions, instructions, and skills

Permissions include allow, ask, and deny rules plus multiple permission modes;
the documentation says deny is evaluated before ask and allow. A prompt or rule
match is not assumed to be containment, and enforcement remains untested.

User and project `CLAUDE.md` files compose through a hierarchy, with some
subdirectory context loaded lazily. Skills use `SKILL.md` in personal, project,
managed, and plugin locations. Later adapter work must account for every relevant
override and discovery layer.

## Candidate inspection and gaps

`/status` and `claude doctor` are documented, but neither is established as a
noninteractive, complete, secret-free effective-config report with per-key
provenance. The published [settings schema][schema] may lag the CLI. A future
probe must first bound command effects, then use an isolated home/project, no
credentials, blocked network, and no session startup.

Primary sources: [settings][settings], [model configuration][model],
[CLI reference][cli], [authentication][auth], [permissions][permissions],
[memory][memory], [skills][skills], and the separate [release context][release].

[settings]: https://code.claude.com/docs/en/settings
[model]: https://code.claude.com/docs/en/model-config
[cli]: https://code.claude.com/docs/en/cli-reference
[auth]: https://code.claude.com/docs/en/authentication
[permissions]: https://code.claude.com/docs/en/permissions
[memory]: https://code.claude.com/docs/en/memory
[skills]: https://code.claude.com/docs/en/skills
[schema]: https://json.schemastore.org/claude-code-settings.json
[release]: https://github.com/anthropics/claude-code/releases/tag/v2.1.278
