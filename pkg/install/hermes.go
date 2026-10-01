package install

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
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

// CheckNamedProfileHome requires configPath to be <hermes-home>/config.yaml: `hermes -p
// <name>` looks under that home whatever config profile-mango was pointed at. The
// "default" profile is configPath itself, so it needs no home.
func (hermesAdapter) CheckNamedProfileHome(configPath, name string, env PathEnv) error {
	if hermes.IsDefaultProfile(name) {
		return nil
	}
	if !env.enabled() {
		return fmt.Errorf("hermes -p %s reads <hermes-home>/profiles/%s/config.yaml, but the Hermes home cannot be resolved here", name, name)
	}
	home, err := hermesProfilesRoot(env)
	if err != nil {
		return fmt.Errorf("resolve the Hermes home for hermes -p %s: %w", name, err)
	}
	if want := filepath.Join(home, "config.yaml"); filepath.Clean(configPath) != want {
		return fmt.Errorf("hermes -p %s reads %s/profiles/%s/config.yaml, which can only be derived from %s, not %s; set HERMES_HOME to that config's directory or omit --config", name, home, name, want, configPath)
	}
	return nil
}

// hermesProfilesRoot mirrors hermes_cli/profiles.py:1789-1806 resolve_profile_env: the
// root is HERMES_HOME, or its grandparent when it names a profiles/<name> directory,
// else ~/.hermes.
func hermesProfilesRoot(env PathEnv) (string, error) {
	home, err := env.dirOrHome("HERMES_HOME", ".hermes")
	if err != nil {
		return "", err
	}
	if parent := filepath.Dir(home); filepath.Base(parent) == "profiles" {
		return filepath.Dir(parent), nil
	}
	return home, nil
}

func (hermesAdapter) NamedProfileUse(name string) string { return "hermes -p " + name }

func (hermesAdapter) OwnedFieldValues(content []byte, fields []string) (map[string]FieldChange, error) {
	values, err := hermes.OwnedConfigValues(content)
	if err != nil {
		return nil, err
	}
	result := make(map[string]FieldChange, len(values))
	for _, field := range fields {
		name := strings.TrimPrefix(field, "config.")
		if _, suffix, found := strings.Cut(field, ".config."); found {
			name = suffix
		}
		if value, found := values[name]; found {
			result[field] = FieldChange{Path: field, Before: value}
		}
	}
	return result, nil
}

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
	patch.Files = append(patch.Files, FilePatch{Path: file.path, Content: result.Content, Fields: fieldNames(fields), LiveFields: true})
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
