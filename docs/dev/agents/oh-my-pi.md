# Oh My Pi configuration reference

**Reference date:** 2026-09-20. **Documentation pin:** release `v18.2.6`,
commit `78b753124d11f8dd3ae73e2524125890ff7c977e`. **Status:** intended MVP target
variant; not installed, tested, or supported by profile-mango, and no adapter
ships. Oh My Pi is qualified independently from [Pi](pi.md).

## Configuration and precedence

Pinned documentation describes YAML at `~/.omp/agent/config.yml` and
`<cwd>/.omp/config.yml`, plus legacy JSON migration. Precedence is built-ins,
global configuration, project configuration, repeated CLI overlays, then runtime
or environment overrides. Objects deep-merge and arrays replace.

Custom `models.yml`, provider URL/API/auth metadata, separate credential
resolution, and role-specific provider/model/effort selection are documented.
Portable profiles must not embed credentials or infer that a selected provider
proves the authentication route.

## Permissions, instructions, and skills

Approval mode and per-tool allow, deny, and prompt rules are documented, with
deny winning. Approval is not assumed to be process or filesystem containment.
Oh My Pi has native and compatibility context-file discovery and explicit skill
source precedence; every relevant override layer needs exact-version testing.

## Candidate inspection and gaps

`omp config list --json` is documented to show merged values with credential
redaction, and `omp config path` locates sources. These are candidates only and
were not executed. Arbitrary `config get` can return requested secrets unmasked
and must not be used as a general inspector. A future probe must isolate project
and agent directories, sanitize environment variables, block network access, and
verify redaction with synthetic secrets before retaining output.

Pinned sources: [settings][settings], [providers][providers], [models][models],
[approval mode][approval], [context files][context], and [skills][skills].

[settings]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/settings.md
[providers]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/providers.md
[models]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/models.md
[approval]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/approval-mode.md
[context]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/context-files.md
[skills]: https://github.com/can1357/oh-my-pi/blob/v18.2.6/docs/skills.md
