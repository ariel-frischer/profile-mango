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
  edited later needs `--override`. In a shared config, edits outside the
  Mango-owned fields (other keys, or the agent re-serializing the file) do
  not; see [concepts](concepts.md#what-install-does).
- `--strict` blocks instead of skipping requirements an agent cannot install.
- `--json` is the stable machine-readable contract.

## Status

`mango status [--json]` is read-only. Per agent it shows the recorded profile,
each owned file's state, and whether the profile's sources changed since the
install. Each agent is labeled at its installed version, found the same way as
the plan's version warnings. When that differs from the version profile-mango
was tested with, the label names both, such as `codex@0.155.1 (tested
0.157.1)`. An agent not on `PATH` shows `codex (tested 0.157.1; codex not found
on PATH)`. `--json` adds a `versionCheck` object per agent with `binary`,
`qualified`, `range`, `detected`, and `status` (`in-range`, `out-of-range`,
`not-found`, or `unknown`).

Status also compares each skill of an agent's default profile with its folder
in the global skills directory (`~/.agents/skills`, or `--global-skills
<dir>`). The profile's copy is canonical. A folder that differs is listed with
the differing files and the newer side by file modification time
(`profile`, `global`, or `unknown`); `--json` puts these rows under `skills`.
A global copy that matches the profile, or its rendering of `{{route.…}}`
placeholders for any agent, is not drift, and an absent folder is not listed.
When the global copy is newer, port its edits into the profile's folder;
mango never copies an installed skill back into the profile.

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
result, and prints the diff. Add `--target <agent>`, `--role <role>`, or both to
change one agent, one role, or one role for one agent; `mango route unset`
removes fields. It then prints, per
profile using the route, the `mango install <profile> --target <agent>...`
command that re-applies it only to agents already recorded on that profile. A
changed provider or model also lists, under `Still names <old>:`, each
`file:line` in those profiles and their resources that still spells the old
value. See [editing routes](profile-reference.md#editing-routes).

## Output

Colors appear only on terminals. Disable them with `--no-color`, `NO_COLOR`, or
`TERM=dumb`.

## Command name

The command is `mango`. The release installer and `make install` also install
`profile-mango` as an alias; use it if MangoWC's unrelated `mango` compositor
CLI comes first on your `PATH`.
