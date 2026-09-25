package install

import (
	"fmt"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ohMyPiRolePriorPrefix marks, on the config's manifest entry, the selector a
// non-default role had before profile-mango first wrote it ("" = absent), so a
// later route without that role gives it back.
const ohMyPiRolePriorPrefix = "ohmypi-role-prior:"

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
	priors := ohMyPiRolePriors(input.Ownership, input.ConfigPath)
	content, released, err := ohmypi.ReleaseRoles(configPatch.Content, releasedOhMyPiRoles(priors, configPatch.Roles))
	if err != nil {
		return Patch{}, err
	}
	file := FilePatch{Content: content}
	var fields []FieldChange
	for _, role := range configPatch.Roles {
		path := "config.modelRoles." + role.Role
		file.Fields = append(file.Fields, path)
		fields = append(fields, FieldChange{Path: path, Before: role.Before, After: role.After})
		file.Ownership = append(file.Ownership, ohMyPiRoleMarker(priors, role))
	}
	for _, role := range released {
		fields = append(fields, FieldChange{Path: "config.modelRoles." + role.Role, Before: role.Before, After: role.After})
	}
	file.Ownership = compactStrings(file.Ownership)
	patch := Patch{Files: []FilePatch{file}, Fields: fields, OverrideAllowed: true}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.route_fields_only", "target.config", "only modelRoles selectors are applied; their :effort suffixes and non-default roles are source-reviewed, not natively observed; defaultThinkingLevel, authentication, provider options, permissions, tools, instructions, skills, precedence, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// ohMyPiRolePriors reads the recorded pre-install selector of every owned role.
func ohMyPiRolePriors(ownership Manifest, configPath string) map[string]string {
	priors := map[string]string{}
	for _, file := range ownership.Files {
		if file.Path != configPath {
			continue
		}
		for _, field := range file.Fields {
			if role, prior, ok := strings.Cut(strings.TrimPrefix(field, ohMyPiRolePriorPrefix), "="); ok && strings.HasPrefix(field, ohMyPiRolePriorPrefix) {
				priors[role] = prior
			}
		}
	}
	return priors
}

// releasedOhMyPiRoles keeps the priors of recorded roles the new route no longer sets.
func releasedOhMyPiRoles(priors map[string]string, planned []ohmypi.RoleChange) map[string]string {
	released := make(map[string]string, len(priors))
	for role, prior := range priors {
		released[role] = prior
	}
	for _, role := range planned {
		delete(released, role.Role)
	}
	return released
}

// ohMyPiRoleMarker records a non-default role's pre-install selector, keeping the
// first recorded value while the role stays owned. The default role is never released.
func ohMyPiRoleMarker(priors map[string]string, role ohmypi.RoleChange) string {
	if role.Role == profilemango.ReservedRoleDefault {
		return ""
	}
	prior, recorded := priors[role.Role]
	if !recorded {
		prior = role.Before
	}
	return ohMyPiRolePriorPrefix + role.Role + "=" + prior
}

func compactStrings(values []string) []string {
	result := values[:0]
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
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
