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
		Reason:         "exact Claude Code 2.1.278 consumes one top-level model setting from an explicit --settings file; install writes it to a Mango-owned emulated profile file by default, and to settings.json only with --default; all other profile effects remain blocked",
	}
}

// Plan writes the model to a Mango-owned emulated profile file (Claude Code has no
// native named profiles) used with `claude --settings <path>`, and additionally
// patches settings.json when --default is set. A named agent destination, which
// carries no install mode, patches settings.json directly instead.
func (claudeCodeAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateClaudeCodeProfile(input); err != nil {
		return Patch{}, err
	}
	if input.Install.Mode != InstallModeNamedProfile {
		return planClaudeCodeConfig(input.Config.Content, input.Route)
	}
	profilePatch, err := claudecode.PatchModel(nil, input.Route)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{
		Files:           []FilePatch{{Path: claudecode.ProfileFileName(input.Install.ProfileName), Content: profilePatch.Content, Fields: []string{"profile.model"}}},
		Fields:          []FieldChange{{Path: "profile.model", After: profilePatch.After}},
		OverrideAllowed: true,
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "claudecode.install.named_profile_model_only", "target.profile.model", "only the model setting is written to the emulated profile file, used with claude --settings <path>; provider, transport, authentication, effort, permissions, tools, instructions, skills, plugins, hooks, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	if !input.Install.SetsDefault {
		return patch, nil
	}
	configPatch, err := planClaudeCodeConfig(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	patch.Files = append(patch.Files, configPatch.Files...)
	patch.Fields = append(patch.Fields, configPatch.Fields...)
	patch.Diagnostics = append(patch.Diagnostics, configPatch.Diagnostics...)
	return patch, nil
}

// planClaudeCodeConfig patches only the top-level model in the main settings file.
func planClaudeCodeConfig(source []byte, route profilemango.RouteBinding) (Patch, error) {
	modelPatch, err := claudecode.PatchModel(source, route)
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

// NamedProfileFile is the Mango-owned emulated profile file Claude Code has no
// native equivalent of; --settings <path> can point at any file, so this is a
// profile-mango convention, not observed target behavior.
func (claudeCodeAdapter) NamedProfileFile(name string) (string, error) {
	if !claudecode.ValidProfileName(name) {
		return "", fmt.Errorf("claude code profile names may use only letters, digits, '_' or '-'")
	}
	return claudecode.ProfileFileName(name), nil
}

// NamedProfileUse shows the documented default profile location. It receives only
// the profile name, not the resolved config path, so an install with an explicit
// --config still prints this default-location form.
func (claudeCodeAdapter) NamedProfileUse(name string) string {
	return "claude --settings ~/.claude/" + claudecode.ProfileFileName(name)
}

func validateClaudeCodeProfile(input AdapterInput) error {
	if input.Target.Name != claudecode.TargetName || input.Target.Version != claudecode.TargetVersion {
		return fmt.Errorf("claude code install adapter requires exact target %s@%s", claudecode.TargetName, claudecode.TargetVersion)
	}
	if input.Config.Exists && len(input.Config.Content) == 0 {
		return fmt.Errorf("claude code settings file is empty")
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("claude code permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("claude code tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("claude code instruction and skill delivery remains install-blocking")
	}
	return nil
}
