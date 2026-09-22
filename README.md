# profile-mango 🥭

**Your coding-agent setup, without the copy-paste.**

Switching agents shouldn't mean rewriting your instructions, copying skills, and
setting up the same preferences again. Profile Mango aims to keep those choices in
one reusable profile, while making it clear what each agent can actually support.

A profile is a reusable bundle of coding-agent preferences:

- **Instructions and skills:** how you want your agent to work.
- **Permissions:** which tools and actions it should be allowed to use.
- **Model preferences:** which provider, model, and effort level to use.

**Profile Mango unifies profiles, not entire agent configurations.** It manages
native settings where needed to support a profile, leaving the rest to each agent.

Your credentials stay with your agents, not in your profiles. Agent-specific
settings such as themes and session history stay outside Profile Mango too.

## What works today?

**This is a private prototype. Full-profile installation is not available for any
agent yet**, even if it has its own native named profiles.

You can create profiles, check them for errors offline, and generate previews.
For the exact versions below, `profile-mango install` can also apply a limited set of
settings to an explicitly chosen agent configuration file. It does not launch an
agent, authenticate, or switch a running conversation.

| Agent | Native named profiles | Main-agent settings installable from a Mango profile | Explicit named agent destination |
| --- | --- | --- | --- |
| Claude Code `2.1.278` | Not established* | Model only | Not qualified |
| OpenCode `1.18.31` | Not established* | Model, optionally one `SKILL.md` and its discovery path | Named primary or subagent Markdown definition with model and ordered instructions |
| Pi `0.86.1` | Not established* | Provider, model, and thinking level | Not qualified |
| Oh My Pi `18.2.6` | Not established* | Default model role and thinking level | Not qualified |
| OpenClaw `2026.9.5` | Yes, separate config/state | Default agent model and thinking level | Not qualified |
| Hermes `0.21.3` | Yes, separate config/state | Provider, default model, and reasoning effort | Not qualified |
| Codex `0.154.0` | Yes, named config presets | Preview only; installation remains blocked | Separately in progress |

\*Our [versioned references](docs/dev/agents/README.md) do not establish an
equivalent native named-profile feature for these agents. Mango installation does
not depend on one. Native profiles are not proof of full Mango profile support.

Unsupported permissions, tool rules, and resource requirements block installation
rather than being silently dropped. Read the plan's warnings: installing a model
does not verify authentication or enforce every route preference. OpenCode named
definitions deliver a custom prompt, not permission enforcement or verified
delegation. Use disposable test configuration for now; live paths require
separate, path-specific approval.
See [agent versions and limitations](docs/dev/target-evidence.md) for the exact
fields, native evidence, and precedence limits.

## How does installation work without native profiles?

Profile Mango owns the profile names. It translates the selected profile into the
supported settings in the destination agent's main configuration, preserving
unrelated settings and credentials. The agent does not need a native named-profile
feature for this.

Installing starts with a diff and a plan. Applying requires confirmation, creates
backups by default, and checks that the files have not changed since planning.
A profile name in Mango does not by itself create a native agent profile or
subagent. For OpenCode `1.18.31`, explicitly select `--agent
opencode@1.18.31=primary:mango-review` or `subagent:mango-review` with a
caller-supplied `--config-path` ending in `agents/mango-review.md`. This writes
a named definition and adjacent ownership manifest, not the main config. A
primary is selectable by OpenCode, not automatically activated; a subagent is
eligible for delegation, not proof of runtime orchestration. The generated
instructions replace that named agent's stock prompt.

To switch settings, run `profile-mango install <other-profile>` with the same target,
configuration path, and ownership manifest, then review and approve the new plan.
Unchanged Mango-owned files can be updated directly; unowned or edited files need
an adapter-approved `--override`. Non-interactive apply requires
`--apply --yes --expect-plan <planID>` from that new plan.

This is not a full reset to the new profile. Switching or omitting the one
installed OpenCode skill removes only a clean Mango-owned `SKILL.md` and a
Mango-introduced discovery path; ambiguous legacy paths are preserved with a
warning. Edited or unowned skill files cannot be deleted or overridden.

## Install

You'll need Go and Make. The pinned development tools are listed in
[`mise.toml`](mise.toml). Repository access is required while the project is private.

```bash
git clone git@gitlab.com:ariel-frischer/profile-mango.git
cd profile-mango
make deps
make install
profile-mango version
```

The binary is called `profile-mango` and goes into your Go bin directory, usually
`~/go/bin`. Make sure it is on your `PATH`.

## Try it

Create a starter profile in a new project folder:

```bash
profile-mango init ./my-profiles
```

Check it using the included example model settings:

```bash
profile-mango validate ./my-profiles/profiles/default/profile.yaml \
  --bindings ./my-profiles/bindings/local.example.yaml
```

Edit `profiles/default/profile.yaml` to change the profile. The separate
`bindings/local.example.yaml` file describes the provider and model, not passwords
or tokens. Neither command changes your agent configuration.

Prefer a shared location? Run `profile-mango init` without a directory to use
`~/.profile-mango`. Existing files are never overwritten.

For installation flags and consent requirements, run `profile-mango install --help`.
The bundled [agent skill](.agents/skills/profile-mango/SKILL.md) can also guide a
coding agent through authoring profiles and planning supported installations.

## Documentation

- [Example profiles](examples/jcode-like/README.md): starting points for daily work, reviews, and research; not fully installable yet.
- [Roadmap](ROADMAP.md): where the project is heading.
- [Technical documentation](docs/index.md): formats, architecture, and agent compatibility evidence.
- [Contributing](CONTRIBUTING.md): development setup and checks.

## License

[MIT](LICENSE)
