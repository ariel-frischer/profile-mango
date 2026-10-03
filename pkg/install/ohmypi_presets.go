package install

import (
	"fmt"
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
	presets, err := ohmypi.PatchPresets(file.Content, routes, input.Config.Content)
	if err != nil {
		return Patch{}, err
	}
	priors := map[string]string{}
	prefix := "profile.modelPresets."
	if global {
		priors = fieldPriors(input.Ownership, input.ConfigPath, ohMyPiPresetPriorPrefix)
		prefix = "config.modelPresets."
	}
	names := make([]string, 0, len(presets.Presets))
	for _, preset := range presets.Presets {
		if _, owned := priors[preset.Name]; global && !owned && preset.Before != "" && !input.Override {
			return Patch{}, ohMyPiPresetConflict(fmt.Sprintf("model preset %s already exists and is not Mango-owned; use --override to adopt it with a backup", preset.Name))
		}
		path := prefix + preset.Name
		file.Fields = append(file.Fields, path)
		patch.Fields = append(patch.Fields, FieldChange{Path: path, Before: preset.Before, After: preset.After})
		if global {
			file.Ownership = append(file.Ownership, fieldPriorMarker(ohMyPiPresetPriorPrefix, priors, preset.Name, preset.Prior))
		}
		names = append(names, preset.Name)
	}
	file.Content = presets.Content
	if global {
		content, released, err := ohmypi.ReleasePresets(file.Content, withoutKeys(priors, names))
		if err != nil {
			return Patch{}, err
		}
		file.Content = content
		for _, preset := range released {
			patch.Fields = append(patch.Fields, FieldChange{Path: prefix + preset.Name, Before: preset.Before, After: preset.After})
		}
	}
	patch.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.model_presets", "config.modelPresets", "model presets: "+strings.Join(names, ", ")+"; switch in omp with /modelpreset switch <name> or /models Roles Ctrl+Left/Right; only roles and default thinking switch, task.maxEffort and profile resources stay applied", 0, 0)
	return patch, nil
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
