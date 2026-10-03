package install

import (
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ModelPresetStatus reports a complete owned preset match, not arbitrary role edits.
// It is file-state evidence: Mango does not inspect the live omp session.
type ModelPresetStatus struct {
	Name                   string `json:"name"`
	Route                  string `json:"route"`
	TaskMaxEffort          string `json:"taskMaxEffort,omitempty"`
	RouteSubagentMaxEffort string `json:"routeSubagentMaxEffort,omitempty"`
	MaxEffortDiffers       bool   `json:"maxEffortDiffers"`
}

func matchingOhMyPiPreset(content []byte, entry ManifestFile) *ModelPresetStatus {
	names, hashes := ownedOhMyPiPresetHashes(entry)
	fields := []string{"config.modelRoles", "config.defaultThinkingLevel", "config.task.maxEffort"}
	for _, name := range names {
		fields = append(fields, "config.modelPresets."+name)
	}
	values, err := ohmypi.ConfigValues(content, fields)
	if err != nil {
		return nil
	}
	for _, name := range names {
		path := "config.modelPresets." + name
		if installfs.Hash([]byte(values[path])) != hashes[path] {
			continue
		}
		var preset ohmypi.ModelPreset
		if json.Unmarshal([]byte(values[path]), &preset) != nil {
			continue
		}
		roles, _ := json.Marshal(preset.ModelRoles)
		if string(roles) != values["config.modelRoles"] {
			continue
		}
		if preset.DefaultThinkingLevel != "" && preset.DefaultThinkingLevel != values["config.defaultThinkingLevel"] {
			continue
		}
		return &ModelPresetStatus{Name: name, Route: ohMyPiPresetRouteName(name), TaskMaxEffort: values["config.task.maxEffort"]}
	}
	return nil
}

func ownedOhMyPiPresetHashes(entry ManifestFile) ([]string, map[string]string) {
	hashes := map[string]string{}
	var names []string
	for _, field := range entry.Fields {
		marker, ok := strings.CutPrefix(field, writtenSHA256Prefix)
		if !ok {
			continue
		}
		path, hash, _ := strings.Cut(marker, "=")
		if name, ok := strings.CutPrefix(path, "config.modelPresets."); ok {
			names = append(names, name)
			hashes[path] = hash
		}
	}
	slices.Sort(names)
	return names, hashes
}

func ohMyPiPresetRouteName(name string) string {
	route, _ := strings.CutPrefix(name, "mango-")
	var decoded strings.Builder
	for index := 0; index < len(route); index++ {
		if route[index] == '_' && index+2 < len(route) {
			if value, err := hex.DecodeString(route[index+1 : index+3]); err == nil {
				decoded.Write(value)
				index += 2
				continue
			}
		}
		decoded.WriteByte(route[index])
	}
	return decoded.String()
}

func ohMyPiSwitchedPreset(input AdapterInput) *ModelPresetStatus {
	if input.Target.Name != ohmypi.TargetName {
		return nil
	}
	entry, owned := manifestEntry(input.Ownership, input.ConfigPath)
	if !owned {
		return nil
	}
	return matchingOhMyPiPreset(input.Config.Content, entry)
}

// A recognized complete switch may change managed roles but no other owned
// field. In particular edited presets and task.maxEffort still need override.
func ohMyPiPresetFieldsIntact(file FilePatch, entry ManifestFile, fields map[string]FieldChange) bool {
	if file.PresetSwitch == "" {
		return false
	}
	filtered := entry
	filtered.Fields = nil
	for _, field := range entry.Fields {
		if strings.HasPrefix(field, writtenSHA256Prefix+"config.modelRoles.") {
			continue
		}
		filtered.Fields = append(filtered.Fields, field)
	}
	return ownedFieldsIntact(file, filtered, fields)
}

func inspectOhMyPiPreset(status *TargetStatus, manifest Manifest) {
	if status.Target.Name != ohmypi.TargetName {
		return
	}
	entry, owned := manifestEntry(manifest, status.ConfigPath)
	if !owned {
		return
	}
	snapshot, err := installfs.SnapshotFile(status.ConfigPath)
	if err != nil || !snapshot.Exists {
		return
	}
	status.ModelPreset = matchingOhMyPiPreset(snapshot.Content, entry)
}

func reconcileOhMyPiPreset(request StatusRequest, status *TargetStatus) {
	preset := status.ModelPreset
	if preset == nil {
		return
	}
	readOhMyPiPresetCap(request, preset)
	drift := status.Drift[:0]
	for _, field := range status.Drift {
		if !strings.HasPrefix(field.Path, "config.modelRoles.") {
			drift = append(drift, field)
		}
	}
	status.Drift = drift
	if len(drift) == 0 && status.Source != SourceUnknown {
		status.Source, status.SourceReason = SourceCurrent, "roles match mango preset "+preset.Route+" (switched via omp); profile resources and task.maxEffort stay applied"
	}
	configEdited := false
	for _, field := range drift {
		if strings.HasPrefix(field.Path, "config.") {
			configEdited = true
		}
	}
	for index, file := range status.Files {
		if !configEdited && file.Kind == "config" && (file.State == FileEdited || file.State == FileOtherEdits) {
			status.Files[index].State = FilePresetSwitched
		}
	}
}

func readOhMyPiPresetCap(request StatusRequest, preset *ModelPresetStatus) {
	if snapshot, err := installfs.SnapshotFile(request.BindingsPath); err == nil && snapshot.Exists {
		bindings, diagnostics := profilemango.ParseBindings(snapshot.Content)
		if !diagnostics.HasErrors() {
			if route, found := bindings.RouteFor(preset.Route, ohmypi.TargetName); found {
				preset.RouteSubagentMaxEffort = route.SubagentMaxEffort
				expected := route.SubagentMaxEffort
				if expected == "" {
					expected = "max"
				}
				actual := preset.TaskMaxEffort
				if actual == "" {
					actual = "max"
				}
				preset.MaxEffortDiffers = expected != actual
			}
		}
	}
}
