package install

import (
	"fmt"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type ohMyPiAdapter struct{}

func (ohMyPiAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         ohmypi.TargetName,
		Version:        ohmypi.TargetVersion,
		AdapterVersion: ohmypi.AdapterVersion,
		EvidenceSHA256: ohmypi.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact Oh My Pi 18.2.6 source-native Settings.loadReadOnly evidence qualifies only modelRoles.default and defaultThinkingLevel; authentication, provider options, permissions, tools, instructions, skills, precedence, and runtime enforcement remain unmanaged",
	}
}

func (ohMyPiAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateOhMyPiProfile(input); err != nil {
		return Patch{}, err
	}
	configPatch, err := ohmypi.PatchConfig(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{
		Files: []FilePatch{{Content: configPatch.Content, Fields: []string{
			"config.modelRoles.default",
			"config.defaultThinkingLevel",
		}}},
		Fields: []FieldChange{
			{Path: "config.modelRoles.default", Before: configPatch.BeforeModel, After: configPatch.AfterModel},
			{Path: "config.defaultThinkingLevel", Before: configPatch.BeforeThinkingLevel, After: configPatch.AfterThinkingLevel},
		},
		OverrideAllowed: true,
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.route_fields_only", "target.config", "only modelRoles.default and defaultThinkingLevel are applied; authentication, provider options, permissions, tools, instructions, skills, precedence, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

func validateOhMyPiProfile(input AdapterInput) error {
	if input.Target.Name != ohmypi.TargetName || input.Target.Version != ohmypi.TargetVersion {
		return fmt.Errorf("Oh My Pi install adapter requires exact target %s@%s", ohmypi.TargetName, ohmypi.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("Oh My Pi permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("Oh My Pi tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("Oh My Pi instruction and skill delivery remains install-blocking")
	}
	return nil
}
