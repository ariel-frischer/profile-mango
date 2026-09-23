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
		Installable:    true,
		Status:         StatusReady,
		Reason:         "settings-only: isolated installed Codex 0.154.0 binary consumed root model_provider, model, and model_reasoning_effort; trusted project and runtime overrides can shadow root settings; OAuth identity, delivery, runtime enforcement, and full-profile applicability remain unverified",
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
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.route_fields_source_qualified", "target.config", "only model_provider, model, and model_reasoning_effort are installed; isolated installed Codex 0.154.0 binary consumed these root fields, but OAuth identity, delivery, runtime enforcement, and full-profile applicability remain unverified", 0, 0)
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.auth_unmanaged", "route.authentication", "authentication remains unmanaged and target-owned: run codex login status locally to distinguish API-key from ChatGPT login, but ChatGPT status also includes externally supplied tokens and does not prove exact OAuth; do not share credentials or status output containing key fragments", 0, 0)
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
