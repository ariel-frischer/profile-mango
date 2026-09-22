# profile-mango

**Your coding-agent setup, without the copy-paste.**

Switching agents shouldn't mean rewriting your instructions, copying skills, and
setting up the same preferences again. Profile Mango aims to keep those choices in
one reusable profile, while making it clear what each agent can actually support.

A profile brings together:

- **Instructions and skills:** how you want your agent to work.
- **Permissions:** which tools and actions it should be allowed to use.
- **Model preferences:** which provider, model, and effort level to use.

Your credentials stay with your agents, not in your profiles. Agent-specific
settings such as themes and session history stay outside Profile Mango too.

## What works today?

**This is a private prototype, not yet a full cross-agent profile switcher.**

You can create profiles, check them for errors offline, and generate previews of
agent configuration without changing your agents. Previews show the gaps rather
than pretending every setting works everywhere.

Full-profile installation is not available. The only supported configuration
change is the `model` setting for **OpenCode 1.18.31**, at an explicitly chosen
file, after reviewing a plan, with backups and rollback. Use disposable test
configuration for now.

See [agent versions and limitations](docs/dev/target-evidence.md) for the details.

## Install

You'll need Go and Make. The pinned development tools are listed in
[`mise.toml`](mise.toml). Repository access is required while the project is private.

```bash
git clone git@gitlab.com:ariel-frischer/profile-mango.git
cd profile-mango
make install
make install-global
profile-mango version
```

The binary goes into your Go bin directory, usually `~/go/bin`. Make sure it is
on your `PATH`.

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

For more commands, run `profile-mango --help`. The bundled
[agent skill](.agents/skills/profile-mango/SKILL.md) can also guide a coding agent
through creating profiles and generating previews.

## Documentation

- [Example profiles](examples/jcode-like/README.md): starting points for daily work, reviews, and research.
- [Roadmap](ROADMAP.md): where the project is heading.
- [Technical documentation](docs/index.md): formats, architecture, and agent compatibility evidence.
- [Contributing](CONTRIBUTING.md): development setup and checks.

## License

[MIT](LICENSE)
