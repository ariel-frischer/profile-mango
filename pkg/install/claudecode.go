package install

import (
	"fmt"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
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
		Reason:         "exact Claude Code 2.1.278 consumes top-level model and effortLevel settings from an explicit --settings file and settings.json; install writes them to a Mango-owned emulated profile file by default, and to settings.json only with --default; all other profile effects remain blocked",
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
	profilePatch, err := claudecode.PatchSettings(nil, input.Route)
	if err != nil {
		return Patch{}, err
	}
	var prior claudecode.SettingsPatch
	if current, err := claudecode.PatchSettings(input.NamedFile.Content, input.Route); err == nil {
		prior = current
	}
	file := FilePatch{Path: claudecode.ProfileFileName(input.Install.ProfileName), Content: profilePatch.Content, Fields: []string{"profile.model"}}
	patch := Patch{Fields: []FieldChange{{Path: "profile.model", Before: prior.ModelBefore, After: profilePatch.ModelAfter}}, OverrideAllowed: true}
	if profilePatch.EffortAfter != "" {
		file.Fields = append(file.Fields, "profile.effortLevel")
		patch.Fields = append(patch.Fields, FieldChange{Path: "profile.effortLevel", Before: prior.EffortBefore, After: profilePatch.EffortAfter})
	}
	patch.Files = []FilePatch{file}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "claudecode.install.named_profile_settings_only", "target.profile", "only model and a supported effortLevel are written to the emulated profile file, used with claude --settings <path>; provider, transport, authentication, permissions, tools, instructions, skills, plugins, hooks, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
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

// planClaudeCodeConfig patches only the top-level model and a supported effortLevel
// in the main settings file.
func planClaudeCodeConfig(source []byte, route profilemango.RouteBinding) (Patch, error) {
	settings, err := claudecode.PatchSettings(source, route)
	if err != nil {
		return Patch{}, err
	}
	file := FilePatch{Content: settings.Content, Fields: []string{"config.model"}, LiveFields: true}
	patch := Patch{Fields: []FieldChange{{Path: "config.model", Before: settings.ModelBefore, After: settings.ModelAfter}}, OverrideAllowed: true}
	if settings.EffortAfter != "" {
		file.Fields = append(file.Fields, "config.effortLevel")
		patch.Fields = append(patch.Fields, FieldChange{Path: "config.effortLevel", Before: settings.EffortBefore, After: settings.EffortAfter})
	}
	patch.Files = []FilePatch{file}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "claudecode.install.settings_only", "target.config", "only top-level model and a supported effortLevel are applied; project/local settings, --effort, CLAUDE_CODE_EFFORT_LEVEL and per-model modelSettings can override effort; provider, transport, authentication, permissions, tools, instructions, skills, plugins, hooks, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// EffortSupport lets planning list an unsupported effort as not applied.
func (claudeCodeAdapter) EffortSupport(effort string) (bool, string) {
	return claudecode.EffortSupport(effort)
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

// NamedProfileUse shows the default profile location until planning resolves the file.
func (claudeCodeAdapter) NamedProfileUse(name string) string {
	return "claude --settings ~/.claude/" + claudecode.ProfileFileName(name)
}

// NamedProfileUseFile starts Claude Code with the resolved profile file.
func (claudeCodeAdapter) NamedProfileUseFile(path string) string {
	return "claude --settings " + path
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
