package install

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
	"gopkg.in/yaml.v3"
)

type openCodeAdapter struct{}

const openCodeSkillsPathOwnership = "opencode.skills.paths.profile-mango"

// openCodeReservedAgentNames are OpenCode 1.18.31 built-in agents a named definition cannot reuse.
var openCodeReservedAgentNames = []string{"build", "plan", "general", "explore", "compaction", "title", "summary"}

func (openCodeAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         opencode.TargetName,
		Version:        opencode.TargetVersion,
		AdapterVersion: opencode.AdapterVersion,
		EvidenceSHA256: opencode.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "an install without --agent writes a Mango-owned primary agent definition (model and instructions) to agents/<profile>.md beside the config, prints its use-it command, and leaves the main config untouched unless --default, which also writes the exact top-level model field and one owned skill into that main config; an explicit --agent opencode@1.18.31=primary:name or subagent:name writes the same kind of definition at an explicit --config path instead",
	}
}

func (openCodeAdapter) Plan(input AdapterInput) (Patch, error) {
	if !input.Agent.Empty() {
		if input.Profile.Permissions != nil || input.Profile.Tools != nil || len(input.Profile.Skills) > 0 {
			return Patch{}, fmt.Errorf("OpenCode named agent permissions, tools, and skills remain unqualified; remove these requirements or use a supported destination")
		}
		return planOpenCodeAgent(input, input.Agent.Mode, "")
	}
	return planOpenCodeDefaultAgent(input)
}

// planOpenCodeDefaultAgent renders the Mango-owned primary agent for an install without
// --agent, and, with --default, also patches the main config exactly as before.
func planOpenCodeDefaultAgent(input AdapterInput) (Patch, error) {
	if len(input.Profile.Skills) > 0 && !input.Install.SetsDefault {
		return Patch{}, fmt.Errorf("OpenCode skills are global (skills.paths), not per-agent; install with --default to also install one skill and the top-level model, or remove this requirement")
	}
	filePath, err := (openCodeAdapter{}).NamedProfileFile(input.Install.ProfileName)
	if err != nil {
		return Patch{}, err
	}
	patch, err := planOpenCodeAgent(input, "primary", filePath)
	if err != nil {
		return Patch{}, err
	}
	if !input.Install.SetsDefault {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.skills_global", "target.profile.skills", "OpenCode skills are global (skills.paths), not per-agent; install with --default to also install one skill and the top-level model into the main config", 0, 0)
		return patch, nil
	}
	mainPatch, err := planOpenCodeMainConfig(input)
	if err != nil {
		return Patch{}, err
	}
	patch.Files = append(patch.Files, mainPatch.Files...)
	patch.Fields = append(patch.Fields, mainPatch.Fields...)
	patch.Diagnostics = append(patch.Diagnostics, mainPatch.Diagnostics...)
	patch.OverrideAllowed = patch.OverrideAllowed || mainPatch.OverrideAllowed
	return patch, nil
}

// planOpenCodeAgent renders the shared primary/subagent Markdown definition at filePath
// ("" writes directly to the explicit --config path).
func planOpenCodeAgent(input AdapterInput, mode, filePath string) (Patch, error) {
	if input.Target.Name != opencode.TargetName || input.Target.Version != opencode.TargetVersion {
		return Patch{}, fmt.Errorf("OpenCode install adapter requires exact target %s@%s", opencode.TargetName, opencode.TargetVersion)
	}
	if input.Profile.Permissions != nil || input.Profile.Tools != nil {
		return Patch{}, fmt.Errorf("OpenCode named agent permissions and tools remain unqualified; remove these requirements or use a supported destination")
	}
	contentByPath := make(map[string][]byte, len(input.Resources))
	for _, resource := range input.Resources {
		if resource.Digest.Kind == "instruction" {
			contentByPath[resource.Digest.Path] = resource.Content
		}
	}
	instructions := make([]string, 0, len(input.Profile.Instructions))
	for _, path := range input.Profile.Instructions {
		content, found := contentByPath[path]
		if !found {
			return Patch{}, fmt.Errorf("OpenCode named agent instruction %q is missing", path)
		}
		instructions = append(instructions, string(content))
	}
	definition, err := opencode.AgentDefinition(input.Route, mode, instructions)
	if err != nil {
		return Patch{}, err
	}
	file := FilePatch{Path: filePath, Content: definition, Fields: []string{"agent.mode", "agent.model", "agent.instructions"}, NoOverride: true}
	patch := Patch{Fields: []FieldChange{{Path: "agent.mode", After: mode}, {Path: "agent.model", After: input.Route.Provider + "/" + input.Route.Model}}}
	if supported, _ := opencode.EffortSupport(input.Route.Effort); supported {
		file.Fields = append(file.Fields, "agent.variant")
		patch.Fields = append(patch.Fields, FieldChange{Path: "agent.variant", After: input.Route.Effort})
		patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.effort_variant", "agent.variant", "effort is installed as the agent's default model variant; OpenCode applies it only while the agent uses its configured model and that model defines a variant of this name, otherwise the model default is used; the main-config model written with --default carries no effort", 0, 0)
	}
	patch.Files = []FilePatch{file}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.agent_limits", "agent", "custom prompt replaces the stock agent prompt and native trimming applies; a primary is selectable but not made default, a subagent is eligible but delegation is unverified; higher-precedence config and runtime enforcement remain unverified", 0, 0)
	return patch, nil
}

// EffortSupport lets planning list an effort no built-in model variant carries as not applied.
func (openCodeAdapter) EffortSupport(effort string) (bool, string) {
	return opencode.EffortSupport(effort)
}

// planOpenCodeMainConfig patches the top-level model field and, independently, installs
// exactly one validated portable skill resource into the explicit main config file.
func planOpenCodeMainConfig(input AdapterInput) (Patch, error) {
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
	configFile := FilePatch{Content: configPatch.Content, Fields: []string{"config.model"}, LiveFields: true}
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
		patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.model_only", "target.config.model", "only the exact top-level model field is applied; authentication, effort, provider options, permissions, tools, instructions, skills, plugins, MCP, and runtime enforcement remain unmanaged", 0, 0)
	} else {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.skills_narrow", "target.config.skills.paths", "one skill is copied beside the explicit config and exposed through a directory-wide skills.paths entry; other skills in that directory may also be discovered, so this is not an exclusive allowlist; instructions, additional skills, permissions, tools, authentication, effort, plugins, MCP, and runtime enforcement remain unmanaged", 0, 0)
	}
	return patch, nil
}

// NamedProfileFile is the Mango-owned OpenCode 1.18.31 primary agent definition that
// `opencode --agent <name>` selects.
func (openCodeAdapter) NamedProfileFile(name string) (string, error) {
	if !validOpenCodeAgentName(name) {
		return "", fmt.Errorf("OpenCode agent name must be simple lowercase letters, digits, or hyphens")
	}
	for _, reserved := range openCodeReservedAgentNames {
		if name == reserved {
			return "", fmt.Errorf("agent name %q is a reserved OpenCode built-in", name)
		}
	}
	return filepath.Join("agents", name+".md"), nil
}

func (openCodeAdapter) NamedProfileUse(name string) string { return "opencode --agent " + name }

func validOpenCodeAgentName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, character := range name {
		letter := character >= 'a' && character <= 'z'
		digit := character >= '0' && character <= '9'
		if !letter && !digit && character != '-' {
			return false
		}
	}
	return true
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
	return nil
}

func validateOpenCodeSkillResource(input AdapterInput) (*render.Resource, error) {
	if len(input.Profile.Skills) != openCodeMaxSkills {
		return nil, fmt.Errorf("OpenCode install supports exactly one skill resource")
	}
	var resource *render.Resource
	for index := range input.Resources {
		if input.Resources[index].Digest.Kind != "skill" {
			continue
		}
		if resource != nil {
			return nil, fmt.Errorf("OpenCode install requires exactly one matching skill resource")
		}
		resource = &input.Resources[index]
	}
	if resource == nil {
		return nil, fmt.Errorf("OpenCode install requires exactly one matching skill resource")
	}
	if resource.Digest.Path != input.Profile.Skills[0] || filepath.Base(resource.Digest.Path) != "SKILL.md" {
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
	return resource, nil
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

// SupportedRequirements reports that a named OpenCode agent always installs instructions;
// an install without --agent also installs one skill, but only alongside --default, since
// OpenCode skills are a global skills.paths entry rather than a per-agent one.
func (openCodeAdapter) SupportedRequirements(agent AgentDestination, setsDefault bool) []string {
	if agent.Empty() && setsDefault {
		return []string{RequirementInstructions, RequirementSkills}
	}
	return []string{RequirementInstructions}
}

// openCodeMaxSkills is how many skill resources a default OpenCode install copies.
const openCodeMaxSkills = 1

// MaxSkills reports that a default OpenCode install copies at most one skill;
// a profile with more lists every skill as skipped instead of blocking.
func (openCodeAdapter) MaxSkills() int { return openCodeMaxSkills }
