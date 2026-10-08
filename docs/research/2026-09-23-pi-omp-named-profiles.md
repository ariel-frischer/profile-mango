# Named-profile options for Pi and Oh My Pi

**Question (Bead ap-vjc.6, 2026-09-23):** can `profile-mango install <name>` create a selectable profile in Pi 0.86.1 or Oh My Pi (OMP) 18.2.6, instead of falling back to the default config?

**Sources (fetched read-only at the pinned commits on 2026-09-23):**
- OMP `v18.2.6`, commit [`78b7531`](https://github.com/can1357/oh-my-pi/tree/78b753124d11f8dd3ae73e2524125890ff7c977e)
- Pi `v0.86.1`, commit [`13cbf77`](https://github.com/badlogic/pi-mono/tree/13cbf77df2396303013a41646bcfa77b4271ae56)

The installed local packages (Pi 0.87.1, OMP 18.2.11) do not match these pins, so they were not used as evidence. No binary was run.

## Summary

| Agent | Mechanism | Verdict |
|---|---|---|
| OMP | `omp --config <file>` overlay holding `modelRoles.default` and `defaultThinkingLevel` | **Viable.** An emulated profile, like Claude Code's `--settings`; follow-up Bead ap-vjc.7 |
| OMP | Native `--profile <name>` | Works, but moves the whole agent home, including auth and sessions, which is outside the installer's safety boundary |
| OMP | Agent files in `~/.omp/agent/agents/*.md` | Apply only to subagents started through the `task` tool, not to the main session |
| Pi | None short of `PI_CODING_AGENT_DIR` | **None.** Keep the default-config fallback |

## Oh My Pi

### `--config` overlay (evidence)
- `packages/coding-agent/src/cli/flag-tables.ts:117-119`: `--config` can be repeated and each value is appended to `result.config`.
- `docs/settings.md:21-22, 96-102`: settings load in this order, later layers winning: built-in defaults, global config, project config, `--config` overlays, runtime flags (`--model`, `--thinking`, ...).
- `docs/settings.md:259-274`: `--config` works with the default launch command, `acp`, and `models`. Paths are relative to the working directory, with `~` expanded. A missing file, invalid YAML, or a non-mapping is a hard error, with no silent fallback.
- `packages/coding-agent/src/config/settings.ts:603-605, 1957-1998`: overlays come from `PI_CONFIG_FILES` plus `--config` and are loaded strictly. `config-file.ts:140-157` requires a `.yml`, `.yaml`, `.json`, or `.jsonc` extension.
- `docs/settings.md:368, 410`: the schemas for `modelRoles` (built-in role `default`) and `defaultThinkingLevel` (`minimal|low|medium|high|xhigh|max|auto`).

**Inference:** a Mango-owned whole file (for example `~/.omp/agent/profiles/<name>.yml`) with just `modelRoles.default` and `defaultThinkingLevel` would give `use it: omp --config ~/.omp/agent/profiles/<name>.yml`. This has not been checked against a native binary yet. The follow-up has to qualify it the same way the existing two-field getter evidence was qualified.

**Limitations:** project config can't override the overlay, since overlays load after it. Runtime flags still win. A hard error on a missing file means that removing the profile breaks any alias that points at it.

### Native `--profile` (evidence, not recommended)
- `packages/coding-agent/src/cli/profile-bootstrap.ts:77-229` removes `--profile` from the arguments before any config loads. `packages/utils/src/dirs.ts:110-132, 328-360` moves the agent directory to `~/.omp/profiles/<name>/agent`. `profile-alias.ts:285, 345-380` writes a shell alias into the user's shell rc.
- This moves `auth.json`, sessions, and the rest along with settings. A new profile therefore starts with no credentials, and the installer would have to touch auth and session storage, which the target safety boundary forbids.

### Subagent definitions (evidence)
- Discovery searches project `.omp/agents`, then user `~/.omp/agent/agents`, then extension `agents/` roots, Claude marketplace plugin agents, and finally the bundled agents. The first match by exact name wins (`packages/coding-agent/src/task/discovery.ts:5-6, 34, 91-99, 117, 149`; `docs/task-agent-discovery.md`).
- Front matter includes `model` (a prioritized list; `@role` aliases resolve through `modelRoles`) and `thinkingLevel` (`packages/coding-agent/src/discovery/helpers.ts:277-375`).
- The model picks one through the `task` tool's `agent` field inside a session, and nothing selects one at launch. That could be useful later for delivering subagent presets, but it can't carry the main session profile.

## Pi
- `packages/coding-agent/src/config.ts:507-578`: `PI_CODING_AGENT_DIR` moves `settings.json`, `auth.json`, `sessions/`, `prompts/`, `models.json`, and the rest, confirming [`pi.md`](../dev/agents/pi.md).
- `packages/coding-agent/src/cli/args.ts:104-156, 278-321`: `--provider`, `--model`, and `--thinking` apply to a single run and are never saved. There is no `--config` or `--settings` file flag.
- Prompt templates (`docs/prompt-templates.md`) only have `description` and `argument-hint` front matter, with no model or thinking settings. `PI_PROVIDER` and `PI_MODEL` only report the active model to subprocesses. Extensions run as code with full process permissions and stay owned by the target.
- **Inference:** a shell alias around the per-run flags (`pi --provider X --model Y --thinking Z`) would act like a profile, but Mango would own a shell rc edit or a wrapper script instead of an agent config file. That is outside the current installer boundary.

## Recommendation
- **OMP:** implement the `--config` overlay as an emulated named profile (ap-vjc.7), with native qualification of the overlay first. Leave native `--profile` and subagent files alone.
- **Pi:** no change. The default-config fallback with its note stays the only safe mechanism.
