package install

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
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
		Reason:         "an install without --agent writes a Mango-owned primary agent definition (model and instructions) to agents/<profile>.md beside the config, prints its use-it command, and leaves the main config untouched unless --default, which also writes the exact top-level model field into that main config and the profile's skill folders to skills/<name>/ beside it; an explicit --agent opencode@1.18.31=primary:name or subagent:name writes the same kind of definition at an explicit --config path instead",
	}
}

func (openCodeAdapter) Plan(input AdapterInput) (Patch, error) {
	if !input.Agent.Empty() {
		if input.Profile.Permissions != nil || input.Profile.Tools != nil {
			return Patch{}, fmt.Errorf("OpenCode named agent permissions and tools remain unqualified; remove these requirements or use a supported destination")
		}
		return planOpenCodeAgent(input, input.Agent.Mode, "")
	}
	return planOpenCodeDefaultAgent(input)
}

// planOpenCodeDefaultAgent renders the Mango-owned primary agent for an install without
// --agent, and, with --default, also patches the main config exactly as before.
func planOpenCodeDefaultAgent(input AdapterInput) (Patch, error) {
	filePath, err := (openCodeAdapter{}).NamedProfileFile(input.Install.ProfileName)
	if err != nil {
		return Patch{}, err
	}
	patch, err := planOpenCodeAgent(input, "primary", filePath)
	if err != nil {
		return Patch{}, err
	}
	if !input.Install.SetsDefault {
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

// planOpenCodeMainConfig patches the top-level model field of the explicit main config.
// Skill folders are written to <config dir>/skills by the shared skill helper, so it
// releases the skills.paths entry and the SKILL.md beside the config that earlier
// versions installed.
func planOpenCodeMainConfig(input AdapterInput) (Patch, error) {
	if err := validateOpenCodeBase(input); err != nil {
		return Patch{}, err
	}
	configPatch, err := opencode.PatchConfig(input.Config.Content, input.Route, nil)
	if err != nil {
		return Patch{}, err
	}
	fields := []FieldChange{{Path: "config.model", Before: configPatch.ModelBefore, After: configPatch.ModelAfter}}
	configFile := FilePatch{Content: configPatch.Content, Fields: []string{"config.model"}, LiveFields: true}
	patch := Patch{Files: []FilePatch{configFile}, Fields: fields, OverrideAllowed: true}
	if err := releaseOpenCodeLegacySkill(input, &patch); err != nil {
		return Patch{}, err
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.model_only", "target.config.model", "only the exact top-level model field is applied to the main config; authentication, effort, provider options, permissions, tools, instructions, plugins, MCP, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// releaseOpenCodeLegacySkill removes the Mango-marked skills.paths entry and the owned
// SKILL.md beside the config that the one-skill install of earlier versions wrote. An
// unmarked skills.paths entry is preserved with a warning.
func releaseOpenCodeLegacySkill(input AdapterInput, patch *Patch) error {
	legacyPath := openCodeLegacySkillPath(input.ConfigPath)
	if manifestOwnsField(input.Ownership, input.ConfigPath, openCodeSkillsPathOwnership) {
		configPatch, err := opencode.PatchConfigRemoveSkillPath(patch.Files[0].Content, input.Route, filepath.Dir(legacyPath))
		if err != nil {
			return err
		}
		patch.Files[0].Content = configPatch.Content
		if configPatch.SkillsRemoved {
			patch.Files[0].Fields = append(patch.Files[0].Fields, "config.skills.paths")
			patch.Fields = append(patch.Fields, FieldChange{Path: "config.skills.paths", Before: strings.Join(configPatch.SkillsBefore, ","), After: strings.Join(configPatch.SkillsAfter, ","), Sensitive: true})
		}
	} else if manifestHasFile(input.Ownership, legacyPath) {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.skills_path_preserved", "target.config.skills.paths", "existing skills.paths entry has no profile-mango provenance marker and is preserved", 0, 0)
	}
	if manifestHasFile(input.Ownership, legacyPath) {
		patch.Files = append(patch.Files, FilePatch{Path: "SKILL.md", Delete: true, Fields: []string{"resources.skills"}, NoOverride: true})
	}
	return nil
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

// SupportedRequirements reports that OpenCode installs instructions for every
// destination; skill folders are gated by SkillSkipReason.
func (openCodeAdapter) SupportedRequirements(AgentDestination, bool) []string {
	return []string{RequirementInstructions}
}

// SkillSkipReason reports that skill folders are installed for every profile made the
// default, never for a named agent destination.
func (adapter openCodeAdapter) SkillSkipReason(agent AgentDestination, setsDefault bool) string {
	return defaultSkillSkipReason(adapter, agent, setsDefault)
}

// SkillRoot is <config dir>/skills: OpenCode 1.18.31 scans {skill,skills}/**/SKILL.md in
// every config directory, including its global config directory.
func (openCodeAdapter) SkillRoot(configPath string, _ PathEnv) (string, error) {
	return configSkillRoot(configPath)
}

// CheckSkill rejects a skill whose SKILL.md sets no frontmatter name: OpenCode 1.18.31
// skips such a skill.
func (openCodeAdapter) CheckSkill(bundle skillBundle) error {
	if !bundle.HasName {
		return fmt.Errorf("OpenCode 1.18.31 loads only skills whose SKILL.md frontmatter sets name; add \"name: %s\" to %s/SKILL.md", bundle.Name, bundle.Dir)
	}
	return nil
}
