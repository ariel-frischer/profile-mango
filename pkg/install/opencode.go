package install

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
	"gopkg.in/yaml.v3"
)

type openCodeAdapter struct{}

const openCodeSkillsPathOwnership = "opencode.skills.paths.profile-mango"

func (openCodeAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         opencode.TargetName,
		Version:        opencode.TargetVersion,
		AdapterVersion: opencode.AdapterVersion,
		EvidenceSHA256: opencode.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact OpenCode 1.18.31 top-level model and one target-owned skill are installable at one explicit path",
	}
}

func (openCodeAdapter) Plan(input AdapterInput) (Patch, error) {
	skill, err := validateOpenCodeProfile(input)
	if err != nil {
		return Patch{}, err
	}
	skillPaths := []string(nil)
	if skill != nil {
		skillPaths = []string{skillConfigDir(input.ConfigPath)}
	}
	configPatch, err := opencode.PatchConfig(input.Config.Content, input.Route, skillPaths)
	if err != nil {
		return Patch{}, err
	}
	fields := []FieldChange{{Path: "config.model", Before: configPatch.ModelBefore, After: configPatch.ModelAfter}}
	configFile := FilePatch{Content: configPatch.Content, Fields: []string{"config.model"}}
	files := []FilePatch{configFile}
	patch := Patch{Files: files, Fields: fields, OverrideAllowed: true}
	if skill != nil {
		files[0].Fields = append(files[0].Fields, "config.skills.paths")
		if len(configPatch.SkillsAdded) > 0 || manifestOwnsField(input.Ownership, input.ConfigPath, openCodeSkillsPathOwnership) {
			files[0].Ownership = []string{openCodeSkillsPathOwnership}
		}
		patch.Fields = append(patch.Fields, FieldChange{
			Path:      "config.skills.paths",
			Before:    strings.Join(configPatch.SkillsBefore, ","),
			After:     strings.Join(configPatch.SkillsAfter, ","),
			Sensitive: true,
		})
		patch.Files = append(patch.Files, FilePatch{Path: "SKILL.md", Content: skill.Content, Fields: []string{"resources.skills"}, NoOverride: true})
	} else {
		legacyPath := openCodeLegacySkillPath(input.ConfigPath)
		if manifestOwnsField(input.Ownership, input.ConfigPath, openCodeSkillsPathOwnership) {
			configPatch, err = opencode.PatchConfigRemoveSkillPath(input.Config.Content, input.Route, skillConfigDir(input.ConfigPath))
			if err != nil {
				return Patch{}, err
			}
			files[0].Content = configPatch.Content
			if configPatch.SkillsRemoved {
				files[0].Fields = append(files[0].Fields, "config.skills.paths")
				patch.Fields = append(patch.Fields, FieldChange{
					Path:      "config.skills.paths",
					Before:    strings.Join(configPatch.SkillsBefore, ","),
					After:     strings.Join(configPatch.SkillsAfter, ","),
					Sensitive: true,
				})
			}
		} else if manifestHasFile(input.Ownership, legacyPath) {
			patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.skills_path_preserved", "target.config.skills.paths", "existing skills.paths entry has no profile-mango provenance marker and is preserved", 0, 0)
		}
		if manifestHasFile(input.Ownership, legacyPath) {
			patch.Files = append(patch.Files, FilePatch{Path: "SKILL.md", Delete: true, Fields: []string{"resources.skills"}, NoOverride: true})
		}
	}
	if skill == nil {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.model_only", "target.config.model", "only the exact top-level model field is applied; authentication, effort, provider options, permissions, tools, instructions, skills, plugins, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	} else {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.skills_narrow", "target.config.skills.paths", "one skill is copied beside the explicit config and exposed through a directory-wide skills.paths entry; other skills in that directory may also be discovered, so this is not an exclusive allowlist; instructions, additional skills, permissions, tools, authentication, effort, plugins, MCP, and runtime enforcement remain unmanaged", 0, 0)
	}
	return patch, nil
}

func manifestHasFile(manifest Manifest, path string) bool {
	for _, file := range manifest.Files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func manifestOwnsField(manifest Manifest, path, field string) bool {
	for _, file := range manifest.Files {
		if file.Path != path {
			continue
		}
		for _, candidate := range file.Fields {
			if candidate == field {
				return true
			}
		}
	}
	return false
}

func openCodeLegacySkillPath(configPath string) string {
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		absolute = configPath
	}
	return filepath.Clean(filepath.Join(filepath.Dir(absolute), "SKILL.md"))
}

func validateOpenCodeProfile(input AdapterInput) (*render.Resource, error) {
	if err := validateOpenCodeBase(input); err != nil {
		return nil, err
	}
	if len(input.Profile.Skills) == 0 {
		if len(input.Resources) > 0 {
			return nil, fmt.Errorf("OpenCode resources are not referenced by an installable profile field")
		}
		return nil, nil
	}
	return validateOpenCodeSkillResource(input)
}

func validateOpenCodeBase(input AdapterInput) error {
	if input.Target.Name != opencode.TargetName || input.Target.Version != opencode.TargetVersion {
		return fmt.Errorf("OpenCode install adapter requires exact target %s@%s", opencode.TargetName, opencode.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("OpenCode permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("OpenCode tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 {
		return fmt.Errorf("OpenCode instruction delivery remains install-blocking")
	}
	return nil
}

func validateOpenCodeSkillResource(input AdapterInput) (*render.Resource, error) {
	if len(input.Profile.Skills) != 1 {
		return nil, fmt.Errorf("OpenCode install supports exactly one skill resource")
	}
	if len(input.Resources) != 1 {
		return nil, fmt.Errorf("OpenCode install requires exactly one matching skill resource")
	}
	resource := input.Resources[0]
	if resource.Digest.Kind != "skill" || resource.Digest.Path != input.Profile.Skills[0] || filepath.Base(resource.Digest.Path) != "SKILL.md" {
		return nil, fmt.Errorf("OpenCode install requires one canonical skills/**/SKILL.md resource")
	}
	if err := validateOpenCodeSkill(resource.Content); err != nil {
		return nil, err
	}
	if expected := render.ResourceFromContent(resource.Digest.Path, resource.Digest.Kind, resource.Content).Digest; expected != resource.Digest {
		return nil, fmt.Errorf("OpenCode skill resource digest does not match its content")
	}
	if strings.TrimSpace(input.ConfigPath) == "" {
		return nil, fmt.Errorf("OpenCode skill install requires an explicit config path")
	}
	configPath, err := filepath.Abs(input.ConfigPath)
	if err != nil || filepath.Base(configPath) == "SKILL.md" {
		return nil, fmt.Errorf("OpenCode skill install requires a config path beside a distinct SKILL.md")
	}
	return &resource, nil
}

func skillConfigDir(configPath string) string {
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		absolute = configPath
	}
	return filepath.ToSlash(filepath.Dir(absolute))
}

func validateOpenCodeSkill(content []byte) error {
	if !utf8.Valid(content) {
		return fmt.Errorf("OpenCode skill resource is not valid UTF-8")
	}
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return fmt.Errorf("OpenCode skill resource requires YAML frontmatter")
	}
	end := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			end = index
			break
		}
	}
	if end < 0 {
		return fmt.Errorf("OpenCode skill resource has unterminated YAML frontmatter")
	}
	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &metadata); err != nil {
		return fmt.Errorf("parse OpenCode skill frontmatter: %w", err)
	}
	if !validOpenCodeSkillName(metadata.Name) || strings.TrimSpace(metadata.Description) == "" {
		return fmt.Errorf("OpenCode skill resource requires a valid name and description")
	}
	return nil
}

func validOpenCodeSkillName(value string) bool {
	if value == "" || len(value) > 64 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, character := range value[1:] {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}
