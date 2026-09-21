# ap-i2v agent skill plan

## Goal

Replace the generated placeholder skill with one canonical distributable
`profile-mango` skill and make the README the primary installation entry point.

## Requirements and checks

| Requirement | Implementation | Check |
| --- | --- | --- |
| Canonical package | Store `SKILL.md` at `.agents/skills/profile-mango/`, where the current skills CLI discovers it without `--full-depth` | Run local `npx skills add` discovery |
| Accurate workflow | Cover install checks, scaffolding, bindings, validation, previews, and errors | Compare every command with current CLI help and execute representative offline commands |
| Safety boundary | Separate offline commands, inert previews, and the networked source-drift check | Review against the constitution and project scope |
| README entry point | Add the familiar `npx skills add ariel-frischer/profile-mango` one-liner | Validate README structure and links |
| Private repository state | State that the one-liner becomes usable after public GitHub publication | Confirm no remote, package, or release is created |

## Non-goals

- Publishing or creating a repository, package, release, or remote.
- Claiming that any target preview is applicable or installed.
- Adding new CLI behavior or changing profile contracts.
