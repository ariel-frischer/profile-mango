package install

import (
	"fmt"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type piAdapter struct{}

func (piAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         pi.TargetName,
		Version:        pi.TargetVersion,
		AdapterVersion: pi.InstallAdapterVersion,
		EvidenceSHA256: pi.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact Pi 0.87.1 settings-module consumption is qualified for three top-level route defaults",
	}
}

func (piAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validatePiProfile(input); err != nil {
		return Patch{}, err
	}
	settings, err := pi.PatchSettings(input.Config.Content, input.Config.Exists, input.Route)
	if err != nil {
		return Patch{}, err
	}
	fields := make([]FieldChange, 0, len(settings.Fields))
	fileFields := make([]string, 0, len(settings.Fields))
	for _, field := range settings.Fields {
		path := "config." + field.Path
		fields = append(fields, FieldChange{Path: path, Before: field.Before, After: field.After})
		fileFields = append(fileFields, path)
	}
	patch := Patch{
		Files:           []FilePatch{{Content: settings.Content, Fields: fileFields, LiveFields: true}},
		Fields:          fields,
		OverrideAllowed: true,
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "pi.install.settings_only", "target.config", "three top-level route defaults are applied: defaultProvider, defaultModel, and defaultThinkingLevel; project .pi/settings.json overrides the global file, while credentials, authentication identity, project trust, resources, extensions, tools, permissions, delivery, and runtime enforcement remain target-owned or unverified", 0, 0)
	return patch, nil
}

func validatePiProfile(input AdapterInput) error {
	if err := validatePiTarget(input.Target); err != nil {
		return err
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("pi permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("pi tool requirements remain install-blocking")
	}
	return validatePiDelivery(input)
}

func validatePiTarget(target Target) error {
	if target.Name != pi.TargetName || target.Version != pi.TargetVersion {
		return fmt.Errorf("pi install adapter requires exact target %s@%s", pi.TargetName, pi.TargetVersion)
	}
	return nil
}

func validatePiDelivery(input AdapterInput) error {
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("pi instruction and skill delivery remains install-blocking")
	}
	return nil
}
