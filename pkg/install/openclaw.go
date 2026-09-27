package install

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
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
		Reason:         "exact OpenClaw 2026.9.5 model and thinking-default fields are qualified by source-native resolver/getter evidence; fallback and user model-override limits remain explicit; pinned source selects <home>/.openclaw-<name>/openclaw.json for openclaw --profile <name>, so install writes that file and the default config only with --default",
	}
}

func (openClawAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateOpenClawProfile(input); err != nil {
		return Patch{}, err
	}
	files, err := openClawFiles(input)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{OverrideAllowed: true}
	for _, file := range files {
		if err := addOpenClawFile(&patch, file, input.Route); err != nil {
			return Patch{}, err
		}
	}
	if len(files) > 0 && files[0].path != "" {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "openclaw.install.profile_state_separate", "target.config", "the profile config is "+filepath.Join(filepath.Base(filepath.Dir(files[0].path)), "openclaw.json")+" beside the .openclaw state directory; openclaw --profile "+input.Install.ProfileName+" uses that separate state directory, so authentication, sessions, and other state from the default profile are not copied and remain target-owned, and an OPENCLAW_CONFIG_PATH or OPENCLAW_STATE_DIR already set to another location keeps taking precedence", 0, 0)
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "openclaw.install.model_thinking_only", "target.config", "only agents.defaults.model.primary and agents.defaults.thinkingDefault are applied; authentication, fallbacks, provider options, permissions, tools, instructions, skills, plugins, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// openClawFile is one OpenClaw config to patch; an empty path is the main config.
type openClawFile struct {
	path, prefix string
	content      []byte
}

// openClawFiles writes the named profile's config, plus the main config with --default.
// The "default" profile is the main config, and default-config mode writes only it.
func openClawFiles(input AdapterInput) ([]openClawFile, error) {
	main := openClawFile{prefix: "config.", content: input.Config.Content}
	if input.Install.Mode != InstallModeNamedProfile {
		return []openClawFile{main}, nil
	}
	name := input.Install.ProfileName
	path, err := openclaw.ProfileConfigPath(input.ConfigPath, name)
	if err != nil {
		return nil, err
	}
	if path == filepath.Clean(input.ConfigPath) {
		return []openClawFile{main}, nil
	}
	named := openClawFile{path: path, prefix: name + ".config.", content: input.NamedFile.Content}
	if input.Install.SetsDefault {
		return []openClawFile{named, main}, nil
	}
	return []openClawFile{named}, nil
}

// NamedProfilePath is the config file `openclaw --profile <name>` reads beside the default state directory.
func (openClawAdapter) NamedProfilePath(configPath, name string) (string, error) {
	return openclaw.ProfileConfigPath(configPath, name)
}

func (openClawAdapter) NamedProfileUse(name string) string { return "openclaw --profile " + name }

func addOpenClawFile(patch *Patch, file openClawFile, route profilemango.RouteBinding) error {
	configPatch, err := openclaw.PatchConfig(file.content, route)
	if err != nil {
		if file.path != "" {
			return fmt.Errorf("%s: %w", file.path, err)
		}
		return err
	}
	fields := []FieldChange{
		{Path: file.prefix + "agents.defaults.model.primary", Before: configPatch.BeforeModel, After: configPatch.AfterModel},
		{Path: file.prefix + "agents.defaults.thinkingDefault", Before: configPatch.BeforeThinking, After: configPatch.AfterThinking},
	}
	patch.Files = append(patch.Files, FilePatch{Path: file.path, Content: configPatch.Content, Fields: fieldNames(fields), LiveFields: true})
	patch.Fields = append(patch.Fields, fields...)
	return nil
}

func validateOpenClawProfile(input AdapterInput) error {
	if input.Target.Name != openclaw.TargetName || input.Target.Version != openclaw.TargetVersion {
		return fmt.Errorf("openclaw install adapter requires exact target %s@%s", openclaw.TargetName, openclaw.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("openclaw permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("openclaw tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("openclaw instruction and skill delivery remains install-blocking")
	}
	return nil
}
