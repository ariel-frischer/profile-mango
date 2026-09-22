package install

import (
	"fmt"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type openClawAdapter struct{}

func (openClawAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         openclaw.TargetName,
		Version:        openclaw.TargetVersion,
		AdapterVersion: openclaw.AdapterVersion,
		EvidenceSHA256: openclaw.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact OpenClaw 2026.9.5 model and thinking-default fields are qualified by source-native resolver/getter evidence; fallback and user model-override limits remain explicit",
	}
}

func (openClawAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateOpenClawProfile(input); err != nil {
		return Patch{}, err
	}
	configPatch, err := openclaw.PatchConfig(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	fields := []FieldChange{
		{Path: "config.agents.defaults.model.primary", Before: configPatch.BeforeModel, After: configPatch.AfterModel},
		{Path: "config.agents.defaults.thinkingDefault", Before: configPatch.BeforeThinking, After: configPatch.AfterThinking},
	}
	patch := Patch{
		Files:           []FilePatch{{Content: configPatch.Content, Fields: fieldNames(fields)}},
		Fields:          fields,
		OverrideAllowed: true,
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "openclaw.install.model_thinking_only", "target.config", "only agents.defaults.model.primary and agents.defaults.thinkingDefault are applied; authentication, fallbacks, provider options, permissions, tools, instructions, skills, plugins, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

func validateOpenClawProfile(input AdapterInput) error {
	if input.Target.Name != openclaw.TargetName || input.Target.Version != openclaw.TargetVersion {
		return fmt.Errorf("OpenClaw install adapter requires exact target %s@%s", openclaw.TargetName, openclaw.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("OpenClaw permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("OpenClaw tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("OpenClaw instruction and skill delivery remains install-blocking")
	}
	return nil
}
