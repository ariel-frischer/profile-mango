package install

import (
	"fmt"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ohMyPiRolePriorPrefix marks, on the config's manifest entry, the selector a
// non-default role had before profile-mango first wrote it ("" = absent), so a
// later route without that role gives it back.
const ohMyPiRolePriorPrefix = "ohmypi-role-prior:"

// ohMyPiSettingPriorPrefix marks the value a dotted setting (task.maxEffort) had
// before profile-mango first wrote it, with the same release rules as roles.
const ohMyPiSettingPriorPrefix = "ohmypi-setting-prior:"

type ohMyPiAdapter struct{}

func (ohMyPiAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Target:         ohmypi.TargetName,
		Version:        ohmypi.TargetVersion,
		AdapterVersion: ohmypi.AdapterVersion,
		EvidenceSHA256: ohmypi.EvidenceSHA256,
		Installable:    true,
		Status:         StatusReady,
		Reason:         "exact Oh My Pi 18.6.0 source review qualifies modelRoles selectors with :effort suffixes and task.maxEffort; isolated 18.6.0 config list reads and historical 18.2.6 Settings.loadReadOnly getters qualify storage only; install writes a Mango-owned --config overlay, and patches config.yml with --default; authentication, full precedence and runtime enforcement remain unmanaged",
	}
}

// SupportedRequirements reports that Oh My Pi installs per-role routes as modelRoles
// slots and the subagent effort cap as task.maxEffort.
func (ohMyPiAdapter) SupportedRequirements(AgentDestination, bool) []string {
	return []string{RequirementRoles, RequirementSubagentMaxEffort}
}

// Plan writes the route to a Mango-owned whole-file overlay used with
// `omp --config <path>` (native --profile relocates auth and sessions, so it is not
// used), and additionally patches config.yml when --default is set. A named agent
// destination, which carries no install mode, patches config.yml directly.
func (ohMyPiAdapter) Plan(input AdapterInput) (Patch, error) {
	if err := validateOhMyPiProfile(input); err != nil {
		return Patch{}, err
	}
	if input.Install.Mode != InstallModeNamedProfile {
		return planOhMyPiConfig(input)
	}
	patch, err := planOhMyPiOverlay(input.Install.ProfileName, input.Route, input.NamedFile.Content)
	if err != nil || !input.Install.SetsDefault {
		return patch, err
	}
	configPatch, err := planOhMyPiConfig(input)
	if err != nil {
		return Patch{}, err
	}
	patch.Files = append(patch.Files, configPatch.Files...)
	patch.Fields = append(patch.Fields, configPatch.Fields...)
	patch.Diagnostics = append(patch.Diagnostics, configPatch.Diagnostics...)
	return patch, nil
}

// planOhMyPiOverlay renders the named overlay from scratch: the same modelRoles
// selectors and task.maxEffort the default install writes, and nothing else. Each
// field's Before is its value in the current overlay, when that file parses.
func planOhMyPiOverlay(name string, route profilemango.RouteBinding, current []byte) (Patch, error) {
	overlay, err := ohmypi.PatchConfig(nil, route)
	if err != nil {
		return Patch{}, err
	}
	befores := make(map[string]string)
	if prior, err := ohmypi.PatchConfig(current, route); err == nil {
		for _, role := range prior.Roles {
			befores["modelRoles."+role.Role] = role.Before
		}
		for _, setting := range prior.Settings {
			befores[setting.Path] = setting.Before
		}
	}
	file := FilePatch{Path: ohmypi.ProfileFileName(name), Content: overlay.Content}
	patch := Patch{OverrideAllowed: true}
	for _, role := range overlay.Roles {
		file.Fields = append(file.Fields, "profile.modelRoles."+role.Role)
		patch.Fields = append(patch.Fields, FieldChange{Path: "profile.modelRoles." + role.Role, Before: befores["modelRoles."+role.Role], After: role.After})
	}
	for _, setting := range overlay.Settings {
		file.Fields = append(file.Fields, "profile."+setting.Path)
		patch.Fields = append(patch.Fields, FieldChange{Path: "profile." + setting.Path, Before: befores[setting.Path], After: setting.After})
	}
	patch.Files = []FilePatch{file}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.named_profile_overlay_only", "target.profile", "only modelRoles selectors and task.maxEffort are written to the emulated profile overlay, used with omp --config <path>; role slots it does not set still come from config.yml; authentication, provider options, permissions, tools, instructions, skills, role definitions, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// planOhMyPiConfig patches modelRoles selectors and task.maxEffort in config.yml,
// giving back role slots and settings a previous route owned but this one drops.
func planOhMyPiConfig(input AdapterInput) (Patch, error) {
	configPatch, err := ohmypi.PatchConfig(input.Config.Content, input.Route)
	if err != nil {
		return Patch{}, err
	}
	rolePriors := fieldPriors(input.Ownership, input.ConfigPath, ohMyPiRolePriorPrefix)
	content, released, err := ohmypi.ReleaseRoles(configPatch.Content, withoutKeys(rolePriors, roleKeys(configPatch.Roles)))
	if err != nil {
		return Patch{}, err
	}
	settingPriors := fieldPriors(input.Ownership, input.ConfigPath, ohMyPiSettingPriorPrefix)
	content, releasedSettings, err := ohmypi.ReleaseSettings(content, withoutKeys(settingPriors, settingKeys(configPatch.Settings)))
	if err != nil {
		return Patch{}, err
	}
	file := FilePatch{Content: content, LiveFields: true}
	fields := ohMyPiRoleFields(&file, rolePriors, configPatch.Roles, released)
	fields = append(fields, ohMyPiSettingFields(&file, settingPriors, configPatch.Settings, releasedSettings)...)
	patch := Patch{Files: []FilePatch{file}, Fields: fields, OverrideAllowed: true}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.route_fields_only", "target.config", "only modelRoles selectors and task.maxEffort are applied; their :effort suffixes, non-default role slots, and task.maxEffort are source-reviewed, not natively observed; defaultThinkingLevel, authentication, provider options, permissions, tools, instructions, skills, role definitions, precedence, and runtime enforcement remain unmanaged", 0, 0)
	return patch, nil
}

// NamedProfileFile is the Mango-owned emulated profile overlay. Oh My Pi loads any
// --config file, so the location is a profile-mango convention, not target behavior.
func (ohMyPiAdapter) NamedProfileFile(name string) (string, error) {
	if !ohmypi.ValidProfileName(name) {
		return "", fmt.Errorf("oh my pi profile names may use only letters, digits, '_' or '-'")
	}
	return ohmypi.ProfileFileName(name), nil
}

// NamedProfileUse shows the default overlay location; it receives only the name.
func (ohMyPiAdapter) NamedProfileUse(name string) string {
	return "omp --config ~/.omp/agent/" + ohmypi.ProfileFileName(name)
}

// NamedProfileUseFile starts Oh My Pi with the resolved overlay path, already
// shell-ready.
func (ohMyPiAdapter) NamedProfileUseFile(path string) string { return "omp --config " + path }

// ohMyPiRoleFields records owned modelRoles fields and prior markers on file and
// returns every planned and released role change.
func ohMyPiRoleFields(file *FilePatch, priors map[string]string, planned, released []ohmypi.RoleChange) []FieldChange {
	var fields []FieldChange
	for _, role := range planned {
		path := "config.modelRoles." + role.Role
		file.Fields = append(file.Fields, path)
		fields = append(fields, FieldChange{Path: path, Before: role.Before, After: role.After})
		if role.Role != profilemango.ReservedRoleDefault {
			file.Ownership = append(file.Ownership, fieldPriorMarker(ohMyPiRolePriorPrefix, priors, role.Role, role.Before))
		}
	}
	for _, role := range released {
		fields = append(fields, FieldChange{Path: "config.modelRoles." + role.Role, Before: role.Before, After: role.After})
	}
	return fields
}

// ohMyPiSettingFields records owned setting fields and prior markers on file and
// returns every planned and released setting change.
func ohMyPiSettingFields(file *FilePatch, priors map[string]string, planned, released []ohmypi.SettingChange) []FieldChange {
	var fields []FieldChange
	for _, setting := range planned {
		path := "config." + setting.Path
		file.Fields = append(file.Fields, path)
		fields = append(fields, FieldChange{Path: path, Before: setting.Before, After: setting.After})
		file.Ownership = append(file.Ownership, fieldPriorMarker(ohMyPiSettingPriorPrefix, priors, setting.Path, setting.Before))
	}
	for _, setting := range released {
		fields = append(fields, FieldChange{Path: "config." + setting.Path, Before: setting.Before, After: setting.After})
	}
	return fields
}

// withoutKeys keeps the priors of recorded keys the new route no longer sets.
func withoutKeys(priors map[string]string, planned []string) map[string]string {
	released := make(map[string]string, len(priors))
	for key, prior := range priors {
		released[key] = prior
	}
	for _, key := range planned {
		delete(released, key)
	}
	return released
}

func roleKeys(changes []ohmypi.RoleChange) []string {
	keys := make([]string, 0, len(changes))
	for _, change := range changes {
		keys = append(keys, change.Role)
	}
	return keys
}

func settingKeys(changes []ohmypi.SettingChange) []string {
	keys := make([]string, 0, len(changes))
	for _, change := range changes {
		keys = append(keys, change.Path)
	}
	return keys
}

func validateOhMyPiProfile(input AdapterInput) error {
	if input.Target.Name != ohmypi.TargetName || input.Target.Version != ohmypi.TargetVersion {
		return fmt.Errorf("oh my pi install adapter requires exact target %s@%s", ohmypi.TargetName, ohmypi.TargetVersion)
	}
	if input.Profile.Permissions != nil {
		return fmt.Errorf("oh my pi permission requirements remain install-blocking")
	}
	if input.Profile.Tools != nil {
		return fmt.Errorf("oh my pi tool requirements remain install-blocking")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 || len(input.Resources) > 0 {
		return fmt.Errorf("oh my pi instruction and skill delivery remains install-blocking")
	}
	return nil
}

// SkillSkipReason gates Oh My Pi skill folders like its subagent files.
func (adapter ohMyPiAdapter) SkillSkipReason(agent AgentDestination, setsDefault bool) string {
	return defaultSkillSkipReason(adapter, agent, setsDefault)
}

// SkillRoot is <config dir>/skills, the user skills directory Oh My Pi reads.
func (ohMyPiAdapter) SkillRoot(configPath string, _ PathEnv) (string, error) {
	return configSkillRoot(configPath)
}
