package install

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const ohMyPiPresetPriorPrefix = "ohmypi-preset-prior:"

type ohMyPiPresetConflict string

func (conflict ohMyPiPresetConflict) Error() string { return string(conflict) }

func ohMyPiPresetRoutes(bindings profilemango.Bindings) map[string]profilemango.RouteBinding {
	routes := map[string]profilemango.RouteBinding{}
	for name := range bindings.Routes {
		route, _ := bindings.RouteFor(name, ohmypi.TargetName)
		routes[name] = route
	}
	return routes
}

// Default installs put presets in config.yml, like /modelpreset save. Named-only
// installs put them in the selected overlay and leave global config untouched.
func addOhMyPiPresets(input AdapterInput, patch Patch, global bool) (Patch, error) {
	if len(input.Bindings.Routes) == 0 {
		return patch, nil
	}
	file := &patch.Files[0]
	routes := ohMyPiPresetRoutes(input.Bindings)
	roleSource, err := ohMyPiPresetRoleSource(input, patch, global)
	if err != nil {
		return Patch{}, err
	}
	if global {
		markers, err := ohMyPiPresetRoleMarkers(input, routes)
		if err != nil {
			return Patch{}, err
		}
		file.Ownership = append(file.Ownership, markers...)
	}
	presets, err := ohmypi.PatchPresets(file.Content, routes, roleSource)
	if err != nil {
		return Patch{}, err
	}
	priors := fieldPriors(input.Ownership, input.ConfigPath, ohMyPiPresetPriorPrefix)
	names, err := recordOhMyPiPresets(input, &patch, presets, priors, global)
	if err != nil {
		return Patch{}, err
	}
	file.Content = presets.Content
	if global {
		if err := releaseOhMyPiPresets(&patch, withoutKeys(priors, names)); err != nil {
			return Patch{}, err
		}
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.model_presets", "config.modelPresets", "model presets: "+strings.Join(names, ", ")+"; switch in omp with /modelpreset switch <name> or /models Roles Ctrl+Left/Right; only roles and default thinking switch, task.maxEffort and profile resources stay applied", 0, 0)
	return patch, nil
}

func recordOhMyPiPresets(input AdapterInput, patch *Patch, presets ohmypi.PresetPatch, priors map[string]string, global bool) ([]string, error) {
	prefix := "profile.modelPresets."
	if global {
		prefix = "config.modelPresets."
	}
	file := &patch.Files[0]
	names := make([]string, 0, len(presets.Presets))
	for _, preset := range presets.Presets {
		if _, owned := priors[preset.Name]; global && !owned && preset.Before != "" && !input.Override {
			return nil, ohMyPiPresetConflict(fmt.Sprintf("model preset %s already exists and is not Mango-owned; use --override to adopt it with a backup", preset.Name))
		}
		path := prefix + preset.Name
		file.Fields = append(file.Fields, path)
		patch.Fields = append(patch.Fields, FieldChange{Path: path, Before: preset.Before, After: preset.After})
		if global {
			file.Ownership = append(file.Ownership, fieldPriorMarker(ohMyPiPresetPriorPrefix, priors, preset.Name, preset.Prior))
		}
		names = append(names, preset.Name)
	}
	return names, nil
}

func releaseOhMyPiPresets(patch *Patch, priors map[string]string) error {
	file := &patch.Files[0]
	content, released, err := ohmypi.ReleasePresets(file.Content, priors)
	if err != nil {
		return err
	}
	file.Content = content
	for _, preset := range released {
		patch.Fields = append(patch.Fields, FieldChange{Path: "config.modelPresets." + preset.Name, Before: preset.Before, After: preset.After})
	}
	return nil
}

func (ohMyPiAdapter) OwnedFieldValues(content []byte, fields []string) (map[string]FieldChange, error) {
	values, err := ohmypi.ConfigValues(content, fields)
	if err != nil {
		return nil, err
	}
	result := map[string]FieldChange{}
	for name, value := range values {
		result[name] = FieldChange{Path: name, Before: value}
	}
	return result, nil
}

// Preset-only custom slots need their own priors: a native preset switch may
// assign them even when the applied route never owned them.
func ohMyPiPresetRolePrefix(name string) string { return "ohmypi-preset-role-prior:" + name + ":" }

func ohMyPiPresetRoleMarkers(input AdapterInput, routes map[string]profilemango.RouteBinding) ([]string, error) {
	owned := fieldPriors(input.Ownership, input.ConfigPath, ohMyPiRolePriorPrefix)
	beforeRoles, err := ohMyPiConfigRoles(input.Config.Content)
	if err != nil {
		return nil, err
	}
	var markers []string
	for name, route := range routes {
		prefix := ohMyPiPresetRolePrefix(ohmypi.PresetName(name))
		priors := fieldPriors(input.Ownership, input.ConfigPath, prefix)
		for _, role := range route.SortedRoleNames() {
			for _, slot := range ohmypi.SlotsForRole(role) {
				before := beforeRoles[slot]
				if prior, found := owned[slot]; found {
					before = prior
				}
				markers = append(markers, fieldPriorMarker(prefix, priors, slot, before))
			}
		}
	}
	slices.Sort(markers)
	return markers, nil
}

func addOhMyPiPresetRolePriors(input AdapterInput, name string, priors map[string]string) {
	presetPriors := fieldPriors(input.Ownership, input.ConfigPath, ohMyPiPresetRolePrefix(name))
	// Older manifests did not record preset-only priors for semantic fallback slots.
	for _, role := range profilemango.SemanticRoles {
		for _, slot := range ohmypi.SlotsForRole(role) {
			if _, recorded := presetPriors[slot]; !recorded {
				presetPriors[slot] = ""
			}
		}
	}
	for slot, prior := range presetPriors {
		if _, recorded := priors[slot]; !recorded {
			priors[slot] = prior
		}
	}
}

func ohMyPiConfigRoles(content []byte) (map[string]string, error) {
	values, err := ohmypi.ConfigValues(content, []string{"config.modelRoles"})
	if err != nil {
		return nil, fmt.Errorf("reading preset role priors: %w", err)
	}
	roles := map[string]string{}
	if value := values["config.modelRoles"]; value != "" {
		if err := json.Unmarshal([]byte(value), &roles); err != nil {
			return nil, fmt.Errorf("decoding preset role priors: %w", err)
		}
	}
	return roles, nil
}

// Do not carry selectors Mango just released into the next preset snapshot.
// Keep the pre-assignment source otherwise, so another route cannot inherit
// this route's managed custom selectors as if they were unmanaged.
func ohMyPiPresetRoleSource(input AdapterInput, patch Patch, global bool) ([]byte, error) {
	if !global {
		return input.Config.Content, nil
	}
	released := map[string]string{}
	for _, field := range patch.Fields {
		slot, role := strings.CutPrefix(field.Path, "config.modelRoles.")
		if role && !slices.Contains(patch.Files[0].Fields, field.Path) {
			released[slot] = field.After
		}
	}
	if len(released) == 0 {
		return input.Config.Content, nil
	}
	content, _, err := ohmypi.ReleaseRoles(input.Config.Content, released)
	if err != nil {
		return nil, fmt.Errorf("releasing preset carry roles: %w", err)
	}
	return content, nil
}
