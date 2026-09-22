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
		Reason:         "no safe credential-free exact-release probe proves model_provider, model, or model_reasoning_effort consumption, effective route or precedence, or OAuth identity. Native evidence is limited to parser acceptance and unrelated features/apps output",
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
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.route_parse_only", "target.config", "the pinned Codex build accepted the route-shaped TOML and typed route fields in an isolated non-session probe; that parsing evidence does not prove effective provider/model/effort values or OAuth route identity, and permissions, tools, instructions, skills, precedence, delivery, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

func validateCodexProfile(input AdapterInput) error {
	if input.Target.Name != codex.TargetName || input.Target.Version != codex.TargetVersion {
		return fmt.Errorf("Codex install adapter requires exact target %s@%s", codex.TargetName, codex.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("Codex permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("Codex tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("Codex instruction and skill delivery remains install-blocking")
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
