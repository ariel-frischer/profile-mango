package install

import (
	"fmt"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
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
		Reason:         "settings-only: model_provider, model, and model_reasoning_effort install as a native profile file used with codex --profile <name>, and into config.toml only with --default; the isolated installed Codex 0.157.1 binary loaded these fields from both; trusted project, runtime, and managed requirements.toml provider overrides can shadow them; OAuth identity, delivery, runtime enforcement, and full-profile applicability remain unverified",
	}
}

func (codexAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateCodexProfile(input); err != nil {
		return Patch{}, err
	}
	name := input.Install.ProfileName
	if err := codex.CheckBaseConfig(input.Config.Content, name); err != nil {
		return Patch{}, err
	}
	profilePatch, err := codex.PatchConfig(input.NamedFile.Content, input.Route)
	if err != nil {
		return Patch{}, fmt.Errorf("%s: %w", codex.ProfileFileName(name), err)
	}
	patch := Patch{OverrideAllowed: true}
	addCodexFile(&patch, codex.ProfileFileName(name), name+".config.", profilePatch)
	if input.Install.SetsDefault {
		rootPatch, err := codex.PatchConfig(input.Config.Content, input.Route)
		if err != nil {
			return Patch{}, err
		}
		addCodexFile(&patch, "", "config.", rootPatch)
	}
	addCodexDiagnostics(&patch)
	return patch, nil
}

// NamedProfileFile is the Codex 0.157.1 profile layer that `codex --profile <name>` reads.
func (codexAdapter) NamedProfileFile(name string) (string, error) {
	if !codex.ValidProfileName(name) {
		return "", fmt.Errorf("codex profile names may use only letters, digits, '_' or '-'")
	}
	return codex.ProfileFileName(name), nil
}

func (codexAdapter) NamedProfileUse(name string) string { return "codex --profile " + name }

func addCodexFile(patch *Patch, path, prefix string, configPatch codex.ConfigPatch) {
	file := FilePatch{Path: path, Content: configPatch.Content}
	for _, field := range configPatch.Fields {
		file.Fields = append(file.Fields, prefix+field.Key)
		patch.Fields = append(patch.Fields, FieldChange{Path: prefix + field.Key, Before: field.Before, After: field.After})
	}
	patch.Files = append(patch.Files, file)
}

func addCodexDiagnostics(patch *Patch) {
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.route_fields_source_qualified", "target.config", "only model_provider, model, and model_reasoning_effort are installed; the isolated installed Codex 0.157.1 binary loaded these fields from config.toml and from a --profile file, but OAuth identity, delivery, runtime enforcement, and full-profile applicability remain unverified", 0, 0)
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.auth_unmanaged", "route.authentication", "authentication remains unmanaged and target-owned: run codex login status locally to distinguish API-key from ChatGPT login, but ChatGPT status also includes externally supplied tokens and does not prove exact OAuth; do not share credentials or status output containing key fragments", 0, 0)
	patch.Diagnostics.Add(profilemango.SeverityWarning, "codex.install.precedence_bounded", "target.config", "this patch changes only the named profile file, plus config.toml with --default, after rejecting legacy profile and provider shadow state; project-local layers, runtime overrides, and managed requirements are not inspected or controlled", 0, 0)
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
