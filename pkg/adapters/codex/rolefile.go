package codex

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// RoleProvider is the one provider a role file model may name: the built-in
// openai provider the default route install also qualifies.
const RoleProvider = "openai"

// RoleFile is one portable role rendered as a rust-v0.154.0 agent role file,
// discovered from <config dir>/agents/*.toml (codex-rs/agent-roles/src/loader.rs
// discover_agent_roles_in_dir). Model and Effort are empty when the role inherits.
type RoleFile struct {
	Name         string
	Description  string
	Instructions string
	Model        string
	Effort       string
}

// roleFileTOML fixes the key order. name, description, and developer_instructions
// are required for discovered files (agent_role_config.rs); the rest is a config layer.
type roleFileTOML struct {
	Name                  string `toml:"name"`
	Description           string `toml:"description"`
	DeveloperInstructions string `toml:"developer_instructions,multiline"`
	ModelProvider         string `toml:"model_provider,omitempty"`
	Model                 string `toml:"model,omitempty"`
	ModelReasoningEffort  string `toml:"model_reasoning_effort,omitempty"`
}

// RoleEffortSupport reports whether a role effort is writable as model_reasoning_effort.
func RoleEffortSupport(effort string) (bool, string) {
	if slices.Contains(installEfforts, effort) {
		return true, ""
	}
	return false, "Codex " + TargetVersion + " model_reasoning_effort is installed only as " + strings.Join(installEfforts, ", ")
}

// RenderRoleFile encodes role as TOML and checks that it decodes back to the same values.
func RenderRoleFile(role RoleFile) ([]byte, error) {
	value := roleFileTOML{Name: role.Name, Description: strings.TrimSpace(role.Description), DeveloperInstructions: strings.TrimSpace(role.Instructions), Model: role.Model, ModelReasoningEffort: role.Effort}
	if value.Name == "" || value.Description == "" || value.DeveloperInstructions == "" {
		return nil, fmt.Errorf("codex role file %q requires name, description, and developer_instructions", role.Name)
	}
	if value.Model != "" {
		if !validCodexValue(value.Model) {
			return nil, fmt.Errorf("codex role %q model must be a non-empty safe string", role.Name)
		}
		value.ModelProvider = RoleProvider
	}
	if supported, reason := RoleEffortSupport(value.ModelReasoningEffort); value.ModelReasoningEffort != "" && !supported {
		return nil, fmt.Errorf("codex role %q: %s", role.Name, reason)
	}
	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(value); err != nil {
		return nil, fmt.Errorf("encode codex role %q: %w", role.Name, err)
	}
	var decoded roleFileTOML
	if _, err := toml.Decode(buffer.String(), &decoded); err != nil || decoded != value {
		return nil, fmt.Errorf("codex role %q does not round-trip as TOML", role.Name)
	}
	return buffer.Bytes(), nil
}
