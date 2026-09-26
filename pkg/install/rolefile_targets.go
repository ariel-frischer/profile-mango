package install

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// Codex rust-v0.154.0 discovers <config dir>/agents/**/*.toml role files
// (codex-rs/agent-roles/src/loader.rs, discovery.rs); see docs/dev/agents/codex.md.
func (codexAdapter) roleFilePath(role string) string { return "agents/" + role + ".toml" }

func (codexAdapter) roleFileModel(_ string, route profilemango.RoleRoute) (string, string) {
	if route.Provider != codex.RoleProvider {
		return "", "Codex role files select only the built-in " + codex.RoleProvider + " provider, not " + route.Provider
	}
	return route.Model, ""
}

func (codexAdapter) roleEffortSupport(effort string) (bool, string) {
	return codex.RoleEffortSupport(effort)
}

func (codexAdapter) renderRoleFile(file roleFile) ([]byte, error) {
	return codex.RenderRoleFile(codex.RoleFile(file))
}

// OpenCode 1.18.31 discovers <config dir>/{agent,agents}/**/*.md, named by path
// (packages/opencode/src/config/agent.ts); see docs/dev/agents/opencode.md.
func (openCodeAdapter) roleFilePath(role string) string { return "agents/" + role + ".md" }

func (openCodeAdapter) roleFileModel(_ string, route profilemango.RoleRoute) (string, string) {
	model, err := opencode.RoleModel(route.Provider, route.Model)
	if err != nil {
		return "", err.Error()
	}
	return model, ""
}

func (openCodeAdapter) roleEffortSupport(effort string) (bool, string) {
	return opencode.EffortSupport(effort)
}

func (openCodeAdapter) renderRoleFile(file roleFile) ([]byte, error) {
	if err := requireRoleText("OpenCode", file); err != nil {
		return nil, err
	}
	return markdownAgent([][2]string{{"description", file.Description}, {"mode", "subagent"}, {"model", file.Model}, {"variant", file.Effort}}, file.Instructions), nil
}

// Oh My Pi 18.2.6 discovers user agents in ~/.omp/agent/agents/*.md with required name
// and description (task/discovery.ts, discovery/helpers.ts); see docs/dev/agents/oh-my-pi.md.
func (ohMyPiAdapter) roleFilePath(role string) string { return "agents/" + role + ".md" }

// roleFileModel names the role's modelRoles slot as an @ alias, so the subagent resolves
// the provider/model:effort selector the same install writes to that slot.
func (ohMyPiAdapter) roleFileModel(role string, _ profilemango.RoleRoute) (string, string) {
	slots := ohmypi.RoleSlots[role]
	if len(slots) == 0 {
		return "", "no Oh My Pi modelRoles slot carries role " + role
	}
	return "@" + slots[0], ""
}

// roleEffortSupport accepts every effort: it travels in the slot selector the alias
// resolves, which the modelRoles install validates.
func (ohMyPiAdapter) roleEffortSupport(string) (bool, string) { return true, "" }

func (ohMyPiAdapter) renderRoleFile(file roleFile) ([]byte, error) {
	if err := requireRoleText("Oh My Pi", file); err != nil {
		return nil, err
	}
	return markdownAgent([][2]string{{"name", file.Name}, {"description", file.Description}, {"model", file.Model}}, file.Instructions), nil
}

// Claude Code reads ~/.claude/agents/*.md with required name and description, model,
// and effort (installed 2.1.280/2.1.281 binaries); see docs/dev/agents/claude-code.md.
func (claudeCodeAdapter) roleFilePath(role string) string { return "agents/" + role + ".md" }

func (claudeCodeAdapter) roleFileModel(_ string, route profilemango.RoleRoute) (string, string) {
	if route.Provider != "anthropic" {
		return "", "Claude Code agent files select only anthropic models, not provider " + route.Provider
	}
	return route.Model, ""
}

func (claudeCodeAdapter) roleEffortSupport(effort string) (bool, string) {
	return claudecode.AgentEffortSupport(effort)
}

func (claudeCodeAdapter) renderRoleFile(file roleFile) ([]byte, error) {
	if err := requireRoleText("Claude Code", file); err != nil {
		return nil, err
	}
	if file.Model != "" && strings.TrimSpace(file.Model) != file.Model {
		return nil, fmt.Errorf("claude-code role %q model has surrounding whitespace", file.Name)
	}
	return markdownAgent([][2]string{{"name", file.Name}, {"description", file.Description}, {"model", file.Model}, {"effort", file.Effort}}, file.Instructions), nil
}
