package install

import (
	"fmt"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type codexAdapter struct{}

func (codexAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         codex.TargetName,
		Version:        codex.TargetVersion,
		AdapterVersion: codex.AdapterVersion,
		EvidenceSHA256: codex.EvidenceSHA256,
		Installable:    false,
		Status:         StatusBlocked,
		Reason:         "source-native exact-release evidence consumes model_provider, model, and model_reasoning_effort from actual PatchConfig output, but installed-binary equivalence, project/runtime precedence, active-profile selection, and OAuth identity remain unverified",
	}
}

func (codexAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateCodexProfile(input); err != nil {
		return Patch{}, err
	}
	configPatch, err := codex.PatchConfig(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{OverrideAllowed: true}
	patch.Files = []FilePatch{{Content: configPatch.Content, Fields: codexFieldNames(configPatch.Fields)}}
	for _, field := range configPatch.Fields {
		patch.Fields = append(patch.Fields, FieldChange{Path: "config." + field.Key, Before: field.Before, After: field.After})
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.route_fields_source_qualified", "target.config", "the pinned Codex source resolver consumed the actual PatchConfig route fields in an isolated non-session probe; installed-binary equivalence and OAuth route identity remain unverified, and permissions, tools, instructions, skills, delivery, and runtime enforcement remain unmanaged", 0, 0)
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.precedence_bounded", "target.config", "this patch changes only the supplied root config document after rejecting active profiles and provider shadow state; project-local layers and runtime overrides are not inspected or controlled", 0, 0)
	return patch, nil
}

func validateCodexProfile(input AdapterInput) error {
	if input.Target.Name != codex.TargetName || input.Target.Version != codex.TargetVersion {
		return fmt.Errorf("codex install adapter requires exact target %s@%s", codex.TargetName, codex.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("codex permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("codex tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("codex instruction and skill delivery remains install-blocking")
	}
	return nil
}

func codexFieldNames(fields []codex.ConfigFieldChange) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, "config."+field.Key)
	}
	return names
}
