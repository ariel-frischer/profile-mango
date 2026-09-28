package install

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type openClawAdapter struct{}

// openClawSkillsPriorPrefix marks, on a config's manifest entry, the allowlist an agent
// had before profile-mango first wrote agents.entries.<id>.skills ("" = absent), so a
// profile without skills, or a different --agent, gives it back.
const openClawSkillsPriorPrefix = "openclaw-skills-prior:"

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
	if err := addOpenClawAllowlists(&patch, files, input); err != nil {
		return Patch{}, err
	}
	if len(files) > 0 && files[0].path != "" {
		patch.Diagnostics.Add(profilemango.SeverityWarning, "openclaw.install.profile_state_separate", "target.config", "the profile config is "+filepath.Join(filepath.Base(filepath.Dir(files[0].path)), "openclaw.json")+" beside the .openclaw state directory; openclaw --profile "+input.Install.ProfileName+" uses that separate state directory, so authentication, sessions, and other state from the default profile are not copied and remain target-owned, and an OPENCLAW_CONFIG_PATH or OPENCLAW_STATE_DIR already set to another location keeps taking precedence", 0, 0)
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "openclaw.install.model_thinking_only", "target.config", "only agents.defaults.model.primary, agents.defaults.thinkingDefault, skill folders under each written state directory's skills folder, and with --agent that agent's agents.entries.<id>.skills allowlist are applied; authentication, fallbacks, provider options, permissions, tools, instructions, plugins, MCP, delivery, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// addOpenClawAllowlists sets agents.entries.<id>.skills in every written config that
// defines a managed agent: the --agent id, else the agents the manifest already records.
// A profile without skills gives each recorded agent its prior allowlist back.
func addOpenClawAllowlists(patch *Patch, files []openClawFile, input AdapterInput) error {
	value := ""
	if len(input.Skills) > 0 {
		value = openclaw.SkillsValue(input.Skills)
	}
	written := false
	for index, file := range files {
		path := file.path
		if path == "" {
			path = input.ConfigPath
		}
		fields, requested, err := planOpenClawAllowlist(&patch.Files[index], file.prefix, fieldPriors(input.Ownership, path, openClawSkillsPriorPrefix), value, input.SkillAgent)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		patch.Fields = append(patch.Fields, fields...)
		written = written || requested
	}
	switch {
	case input.SkillAgent != "" && value == "":
		patch.Diagnostics.Add(profilemango.SeverityWarning, "openclaw.install.allowlist_no_skills", "agents.entries."+input.SkillAgent+".skills", "the profile has no skills, so the "+input.SkillAgent+" allowlist is not written", 0, 0)
	case input.SkillAgent != "" && !written:
		return fmt.Errorf("openclaw agent %q is not defined (agents.entries.%s) in any config this install writes; define the agent first, or pass --default when it is defined in the main config", input.SkillAgent, input.SkillAgent)
	}
	return nil
}

// planOpenClawAllowlist patches one config: each managed agent's allowlist becomes value
// and records its prior; each recorded agent no longer managed gets its prior back. An
// agent whose entry is gone is skipped. requested reports whether --agent was written.
func planOpenClawAllowlist(file *FilePatch, prefix string, priors map[string]string, value, agent string) ([]FieldChange, bool, error) {
	managed := openClawManagedAgents(priors, value, agent)
	var fields []FieldChange
	requested := false
	for _, id := range sortedNames(withAgents(priors, managed)) {
		before, found, err := openclaw.AgentSkills(file.Content, id)
		if err != nil || !found {
			if err != nil {
				return nil, false, err
			}
			continue
		}
		_, keep := managed[id]
		after := priors[id]
		if keep {
			after = value
		}
		if file.Content, err = openclaw.SetAgentSkills(file.Content, id, after); err != nil {
			return nil, false, err
		}
		name := prefix + "agents.entries." + id + ".skills"
		if keep {
			file.Fields = append(file.Fields, name)
			file.Ownership = append(file.Ownership, fieldPriorMarker(openClawSkillsPriorPrefix, priors, id, before))
			requested = requested || id == agent
		}
		if keep || before != after {
			fields = append(fields, FieldChange{Path: name, Before: before, After: after})
		}
	}
	return fields, requested, nil
}

// openClawManagedAgents is the set whose allowlist this install writes: none without
// skills, the --agent id when given, else every agent the manifest records.
func openClawManagedAgents(priors map[string]string, value, agent string) map[string]struct{} {
	managed := map[string]struct{}{}
	switch {
	case value == "":
	case agent != "":
		managed[agent] = struct{}{}
	default:
		for id := range priors {
			managed[id] = struct{}{}
		}
	}
	return managed
}

func withAgents(priors map[string]string, managed map[string]struct{}) map[string]struct{} {
	all := make(map[string]struct{}, len(priors)+len(managed))
	for id := range priors {
		all[id] = struct{}{}
	}
	for id := range managed {
		all[id] = struct{}{}
	}
	return all
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

// ValidSkillAgent accepts an agents.entries key for --agent openclaw=<id>.
func (openClawAdapter) ValidSkillAgent(agent string) error {
	if !openclaw.ValidAgentID(agent) {
		return fmt.Errorf("openclaw agent id %q must be lowercase letters, digits, '_' or '-' (at most 64)", agent)
	}
	return nil
}

// SkillSkipReason is always empty: every OpenClaw install writes a config, and skill
// folders go beside each config it writes (see SkillRoot).
func (openClawAdapter) SkillSkipReason(AgentDestination, bool) string { return "" }

// SkillRoot is <state dir>/skills for the config at configPath: OpenClaw 2026.9.5 loads
// managed skills from CONFIG_DIR/skills, and CONFIG_DIR is the config file's directory
// (~/.openclaw, or ~/.openclaw-<name> for openclaw --profile <name>).
func (openClawAdapter) SkillRoot(configPath string, _ PathEnv) (string, error) {
	return configSkillRoot(configPath)
}

func (openClawAdapter) skillRootPerConfig() {}

// SkillRootWarning notes that an exported OPENCLAW_STATE_DIR outranks the config
// directory, so OpenClaw would read skills from that state directory instead.
func (openClawAdapter) SkillRootWarning(_ string, root string, env PathEnv) string {
	if env.Getenv == nil {
		return ""
	}
	state := strings.TrimSpace(env.Getenv("OPENCLAW_STATE_DIR"))
	if state == "" || filepath.Clean(state) == filepath.Dir(root) {
		return ""
	}
	return "OPENCLAW_STATE_DIR is set to " + state + ", so OpenClaw reads managed skills from its skills folder, not " + root
}
