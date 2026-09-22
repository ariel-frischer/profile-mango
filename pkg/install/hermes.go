package install

import (
	"fmt"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type hermesAdapter struct{}

func (hermesAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         hermes.TargetName,
		Version:        hermes.TargetVersion,
		AdapterVersion: hermes.InstallAdapterVersion,
		EvidenceSHA256: hermes.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact Hermes 0.21.3 config YAML fields are consumed by a pinned native loader module in an isolated effective-merge probe; startup, authentication, provider calls, and runtime enforcement remain unverified",
	}
}

func (hermesAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateHermesProfile(input); err != nil {
		return Patch{}, err
	}
	result, err := hermes.PatchConfig(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	fields := []FieldChange{
		{Path: "model.provider", Before: result.Before["model.provider"], After: input.Route.Provider},
		{Path: "model.default", Before: result.Before["model.default"], After: input.Route.Model},
		{Path: "agent.reasoning_effort", Before: result.Before["agent.reasoning_effort"], After: input.Route.Effort},
	}
	patch := Patch{
		Files:           []FilePatch{{Content: result.Content, Fields: fieldNames(fields)}},
		Fields:          fields,
		OverrideAllowed: true,
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "hermes.install.config_only", "target.config", "only model.provider, model.default, and agent.reasoning_effort are installable; authentication, delivery, permissions, tools, skills, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

func validateHermesProfile(input AdapterInput) error {
	if input.Target.Name != hermes.TargetName || input.Target.Version != hermes.TargetVersion {
		return fmt.Errorf("Hermes install adapter requires exact target %s@%s", hermes.TargetName, hermes.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("Hermes permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("Hermes tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("Hermes instruction, skill, and resource delivery remains install-blocking")
	}
	return nil
}
