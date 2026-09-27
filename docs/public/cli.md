# Command-line details

Run `mango <command> --help` for every flag. This page covers the behavior that
is shared across commands.

## Plans and consent

`install`, `use`, and `undo` always print a plan first and write nothing. The
plan ends with the exact apply command, including its plan ID:

```bash
mango install default --all --apply --yes --expect-plan <plan-id>
```

On a terminal, `--apply` alone asks for y/N confirmation instead.
`--non-interactive` never grants consent by itself.

A plan shows, per agent: status, resolved destination, installed model and
effort when known, the command to use the profile, file and field effects,
every skipped requirement, and warnings. It ends with target and file counts.
`--verbose` adds full version and diagnostic detail (plus undo hashes). `undo`
also shows its redacted file diff and a per-target apply command.

- `--all` selects every supported agent and skips those whose config folder
  does not exist.
- An existing config is adopted with a backup. A Mango-installed value you
  edited later needs `--override`.
- `--strict` blocks instead of skipping requirements an agent cannot install.
- `--json` is the stable machine-readable contract.

## Selecting agents

`-t` is short for `--target`. `install`, `undo`, `doctor`, and `agents check`
accept comma-separated or repeated targets; duplicates are ignored:

```bash
mango install default -t codex,opencode
mango doctor -t codex -t opencode
```

- `render` (alias `preview`) accepts the same syntax but needs exactly one
  target and one `--out` directory.
- A multi-target `undo` previews one plan per target (`--json` returns an
  array). Apply each target separately with its own `--expect-plan` ID; one
  consent cannot partially undo several agents.
- `--config target=path`, `--manifest target=path`, and `--agent target=value`
  are repeatable mappings; commas inside their values are not split.
- `agents check` selects documentation sources, not installable versions.
  Checking a source is not a claim of version compatibility.

## Changing routes

`mango route set <route> --model <m> --effort <e>` edits
`~/.profile-mango/bindings/local.yaml` in place, keeps comments, validates the
result, and prints the diff. Add `--target <agent>` or `--role <role>` to change
one agent or role; `mango route unset` removes fields. It then prints, per
profile using the route, the `mango install <profile> --target <agent>...`
command that re-applies it only to agents already recorded on that profile. See
[editing routes](profile-reference.md#editing-routes).

## Output

Colors appear only on terminals. Disable them with `--no-color`, `NO_COLOR`, or
`TERM=dumb`.

## Command name

The command is `mango`. The release installer and `make install` also install
`profile-mango` as an alias; use it if MangoWC's unrelated `mango` compositor
CLI comes first on your `PATH`.
