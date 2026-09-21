package install

import (
	"fmt"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type openCodeAdapter struct{}

func (openCodeAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         opencode.TargetName,
		Version:        opencode.TargetVersion,
		AdapterVersion: opencode.AdapterVersion,
		EvidenceSHA256: opencode.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact OpenCode 1.18.31 top-level model field is installable at one explicit path",
	}
}

func (openCodeAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateOpenCodeProfile(input); err != nil {
		return Patch{}, err
	}
	modelPatch, err := opencode.PatchModel(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{
		Files:           []FilePatch{{Content: modelPatch.Content, Fields: []string{"config.model"}}},
		Fields:          []FieldChange{{Path: "config.model", Before: modelPatch.Before, After: modelPatch.After}},
		OverrideAllowed: true,
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.model_only", "target.config.model", "only the exact top-level model field is applied; authentication, effort, provider options, permissions, tools, instructions, skills, plugins, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

func validateOpenCodeProfile(input AdapterInput) error {
	if input.Target.Name != opencode.TargetName || input.Target.Version != opencode.TargetVersion {
		return fmt.Errorf("OpenCode install adapter requires exact target %s@%s", opencode.TargetName, opencode.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("OpenCode permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("OpenCode tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("OpenCode instruction and skill delivery remains install-blocking")
	}
	return nil
}
