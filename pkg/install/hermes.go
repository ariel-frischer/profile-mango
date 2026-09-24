package install

import (
	"fmt"
	"path/filepath"

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
		Reason:         "exact Hermes 0.21.3 config YAML fields are consumed by a pinned native loader module in an isolated effective-merge probe; startup, authentication, provider calls, and runtime enforcement remain unverified; pinned source selects <hermes-home>/profiles/<name>/config.yaml for hermes -p <name>, so install writes that file and the default config only with --default",
	}
}

func (hermesAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateHermesProfile(input); err != nil {
		return Patch{}, err
	}
	files, err := hermesFiles(input)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{OverrideAllowed: true}
	for _, file := range files {
		if err := addHermesFile(&patch, file, input.Route); err != nil {
			return Patch{}, err
		}
	}
	if len(files) > 0 && files[0].path != "" {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "hermes.install.profile_state_separate", "target.config", "the profile config is profiles/"+input.Install.ProfileName+"/config.yaml under the Hermes home; hermes -p "+input.Install.ProfileName+" reads that separate profile directory, so memories, sessions, skills, and other state from the default profile are not copied and remain target-owned", 0, 0)
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "hermes.install.config_only", "target.config", "only model.provider, model.default, and agent.reasoning_effort are installable; authentication, delivery, permissions, tools, skills, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// hermesFile is one Hermes config.yaml to patch; an empty path is the main config.
type hermesFile struct {
	path, prefix string
	content      []byte
}

// hermesFiles writes the named profile's config.yaml, plus the main config with
// --default. The "default" profile is the main config itself (hermes.ProfileConfigPath),
// so that case writes only the main config even when the install mode is named.
func hermesFiles(input AdapterInput) ([]hermesFile, error) {
	main := hermesFile{prefix: "config.", content: input.Config.Content}
	if input.Install.Mode != InstallModeNamedProfile {
		return []hermesFile{main}, nil
	}
	name := input.Install.ProfileName
	path, err := hermes.ProfileConfigPath(input.ConfigPath, name)
	if err != nil {
		return nil, err
	}
	if path == filepath.Clean(input.ConfigPath) {
		return []hermesFile{main}, nil
	}
	named := hermesFile{path: path, prefix: name + ".config.", content: input.NamedFile.Content}
	if input.Install.SetsDefault {
		return []hermesFile{named, main}, nil
	}
	return []hermesFile{named}, nil
}

// NamedProfilePath is the config.yaml `hermes -p <name>` reads under the Hermes home.
func (hermesAdapter) NamedProfilePath(configPath, name string) (string, error) {
	return hermes.ProfileConfigPath(configPath, name)
}

func (hermesAdapter) NamedProfileUse(name string) string { return "hermes -p " + name }

func addHermesFile(patch *Patch, file hermesFile, route profilemango.RouteBinding) error {
	result, err := hermes.PatchConfig(file.content, route)
	if err != nil {
		if file.path != "" {
			return fmt.Errorf("%s: %w", file.path, err)
		}
		return err
	}
	fields := []FieldChange{
		{Path: file.prefix + "model.provider", Before: result.Before["model.provider"], After: route.Provider},
		{Path: file.prefix + "model.default", Before: result.Before["model.default"], After: route.Model},
		{Path: file.prefix + "agent.reasoning_effort", Before: result.Before["agent.reasoning_effort"], After: route.Effort},
	}
	patch.Files = append(patch.Files, FilePatch{Path: file.path, Content: result.Content, Fields: fieldNames(fields)})
	patch.Fields = append(patch.Fields, fields...)
	return nil
}

func validateHermesProfile(input AdapterInput) error {
	if input.Target.Name != hermes.TargetName || input.Target.Version != hermes.TargetVersion {
		return fmt.Errorf("hermes install adapter requires exact target %s@%s", hermes.TargetName, hermes.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("hermes permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("hermes tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("hermes instruction, skill, and resource delivery remains install-blocking")
	}
	return nil
}
