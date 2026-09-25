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
		Reason:         "exact Oh My Pi 18.2.6 source-native Settings.loadReadOnly evidence qualifies only modelRoles selectors (default plus built-in roles, each with its own :effort suffix); authentication, provider options, permissions, tools, instructions, skills, precedence, and runtime enforcement remain unmanaged",
	}
}

// SupportedRequirements reports that Oh My Pi installs per-role routes as modelRoles selectors.
func (ohMyPiAdapter) SupportedRequirements(AgentDestination, bool) []string {
	return []string{RequirementRoles}
}

func (ohMyPiAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateOhMyPiProfile(input); err != nil {
		return Patch{}, err
	}
	configPatch, err := ohmypi.PatchConfig(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	names := make([]string, 0, len(configPatch.Roles))
	fields := make([]FieldChange, 0, len(configPatch.Roles))
	for _, role := range configPatch.Roles {
		path := "config.modelRoles." + role.Role
		names = append(names, path)
		fields = append(fields, FieldChange{Path: path, Before: role.Before, After: role.After})
	}
	patch := Patch{Files: []FilePatch{{Content: configPatch.Content, Fields: names}}, Fields: fields, OverrideAllowed: true}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.route_fields_only", "target.config", "only modelRoles selectors are applied; their :effort suffixes and non-default roles are source-reviewed, not natively observed; defaultThinkingLevel, authentication, provider options, permissions, tools, instructions, skills, precedence, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

func validateOhMyPiProfile(input AdapterInput) error {
	if input.Target.Name != ohmypi.TargetName || input.Target.Version != ohmypi.TargetVersion {
		return fmt.Errorf("oh my pi install adapter requires exact target %s@%s", ohmypi.TargetName, ohmypi.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("oh my pi permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("oh my pi tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("oh my pi instruction and skill delivery remains install-blocking")
	}
	return nil
}
