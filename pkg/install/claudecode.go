package install

import (
	"fmt"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type claudeCodeAdapter struct{}

func (claudeCodeAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         claudecode.TargetName,
		Version:        claudecode.TargetVersion,
		AdapterVersion: claudecode.AdapterVersion,
		EvidenceSHA256: claudecode.NativeBinarySHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact Claude Code 2.1.278 consumes one top-level model setting at an explicit path; all other profile effects remain blocked",
	}
}

func (claudeCodeAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateClaudeCodeProfile(input); err != nil {
		return Patch{}, err
	}
	modelPatch, err := claudecode.PatchModel(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{
		Files:           []FilePatch{{Content: modelPatch.Content, Fields: []string{"config.model"}}},
		Fields:          []FieldChange{{Path: "config.model", Before: modelPatch.Before, After: modelPatch.After}},
		OverrideAllowed: true,
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "claudecode.install.model_only", "target.config.model", "only the top-level model setting is applied; provider, transport, authentication, effort, permissions, tools, instructions, skills, plugins, hooks, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

func validateClaudeCodeProfile(input AdapterInput) error {
	if input.Target.Name != claudecode.TargetName || input.Target.Version != claudecode.TargetVersion {
		return fmt.Errorf("Claude Code install adapter requires exact target %s@%s", claudecode.TargetName, claudecode.TargetVersion)
	}
	if input.Config.Exists && len(input.Config.Content) == 0 {
		return fmt.Errorf("Claude Code settings file is empty")
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("Claude Code permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("Claude Code tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("Claude Code instruction and skill delivery remains install-blocking")
	}
	return nil
}
