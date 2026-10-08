package install

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// Status report states.
const (
	StatusManaged   = "managed"
	StatusUnmanaged = "unmanaged"

	FileInSync  = "in-sync"
	FileEdited  = "edited"
	FileMissing = "missing"
	// FileOtherEdits is a field-owned file whose owned values match the profile
	// while other bytes changed, e.g. the agent re-serialized its config.
	FileOtherEdits     = "other-edits"
	FilePresetSwitched = "preset-switched"

	SourceCurrent = "current"
	SourceChanged = "changed"
	SourceUnknown = "unknown"

	DriftDiffers = "differs"
)

// StatusRequest selects the targets and profile inputs status inspects. It never writes.
type StatusRequest struct {
	ProfilesRoot string
	ResourceRoot string
	BindingsPath string
	Registry     *Registry
	Env          PathEnv
	// Targets limits the report; empty inspects every registered target's default path.
	Targets []TargetRequest
	// DetectVersion, when set, reports each qualified target's installed agent version.
	DetectVersion VersionDetector
	// GlobalSkillsRoot is the global skills directory compared with profile-owned
	// skills; empty means $HOME/.agents/skills.
	GlobalSkillsRoot string
	// StateDir is the Mango state directory the source comparison searches for the
	// backups a release would restore; it is only read.
	StateDir string
}

// StatusReport is the deterministic state of every inspected target.
type StatusReport struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Targets    []TargetStatus `json:"targets"`
	// Skills lists profile-owned skills whose global copy differs from the profile.
	Skills []SkillDrift `json:"skills,omitempty"`
}

// TargetStatus is one target's ownership state. Paths are relative to its config directory.
type TargetStatus struct {
	Target Target `json:"target"`
	// RecordedVersion is the target version the ownership manifest was written at, when
	// it differs from the qualified adapter version; the next apply rewrites it.
	RecordedVersion string       `json:"recordedVersion,omitempty"`
	State           string       `json:"state"`
	Reason          string       `json:"reason,omitempty"`
	Profile         string       `json:"profile,omitempty"`
	Generation      uint64       `json:"generation,omitempty"`
	Source          string       `json:"source,omitempty"`
	SourceReason    string       `json:"sourceReason,omitempty"`
	Files           []FileStatus `json:"files,omitempty"`
	// Drift lists each owned setting, or whole owned file, whose live state differs
	// from what the current profile sources would install.
	Drift []FieldDrift `json:"drift,omitempty"`
	// VersionCheck compares the installed agent version with the adapter's tested
	// version, detected the same way install plans detect it.
	VersionCheck *VersionCheck `json:"versionCheck,omitempty"`
	// ModelPreset is set only for a complete match of a manifest-owned snapshot.
	ModelPreset *ModelPresetStatus `json:"modelPreset,omitempty"`
	ConfigPath  string             `json:"-"`
	ownsConfig  bool
}

// FieldDrift is one owned setting whose live value differs from the profile's value
// (absent values are empty), or, with State set, a whole owned file that differs.
type FieldDrift struct {
	Path    string `json:"path"`
	Live    string `json:"live,omitempty"`
	Profile string `json:"profile,omitempty"`
	State   string `json:"state,omitempty"`
}

// FileStatus is one owned file: its kind, current state, and the owned field names.
type FileStatus struct {
	Path   string   `json:"path"`
	Kind   string   `json:"kind"`
	State  string   `json:"state"`
	Fields []string `json:"fields,omitempty"`
}

// JSON encodes the report with sorted targets.
func (report StatusReport) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode status: %w", err)
	}
	return append(data, '\n'), nil
}

// Managed returns a default-path request for every managed target, for mango use.
func (report StatusReport) Managed() []TargetRequest {
	var result []TargetRequest
	for _, target := range report.Targets {
		if target.State == StatusManaged {
			result = append(result, TargetRequest{Target: target.Target})
		}
	}
	return result
}

// InspectStatus reads each target's ownership manifest and owned files, then plans the
// recorded profile read-only to say whether its sources changed since it was applied.
func InspectStatus(request StatusRequest) (StatusReport, error) {
	registry := request.Registry
	if registry == nil {
		registry = DefaultRegistry()
	}
	targets := request.Targets
	if len(targets) == 0 {
		for _, target := range registry.Targets() {
			targets = append(targets, TargetRequest{Target: target})
		}
	}
	report := StatusReport{APIVersion: "profilemango.dev/status/v1alpha1", Kind: "Status"}
	for _, target := range targets {
		status, err := inspectTarget(registry, request.Env, target)
		if err != nil {
			return StatusReport{}, err
		}
		if status.VersionCheck, err = statusVersionCheck(registry, request.DetectVersion, target.Target); err != nil {
			return StatusReport{}, err
		}
		if status.State == StatusManaged {
			sourceState(request, registry, &status)
		}
		report.Targets = append(report.Targets, status)
	}
	sort.Slice(report.Targets, func(i, j int) bool { return report.Targets[i].Target.String() < report.Targets[j].Target.String() })
	report.Skills = globalSkillDrift(request, registry, report.Targets)
	return report, nil
}

// statusVersionCheck detects a qualified target's installed version; it is nil
// without a detector or a qualified adapter.
func statusVersionCheck(registry *Registry, detect VersionDetector, target Target) (*VersionCheck, error) {
	adapter, found := registry.Lookup(target)
	if detect == nil || !found || !adapter.Metadata().Installable {
		return nil, nil
	}
	check, err := CheckVersion(adapter.Metadata(), detect(target))
	if err != nil {
		return nil, fmt.Errorf("check %s version: %w", target, err)
	}
	return &check, nil
}

func inspectTarget(registry *Registry, env PathEnv, target TargetRequest) (TargetStatus, error) {
	status := TargetStatus{Target: target.Target, State: StatusUnmanaged}
	adapter, found := registry.Lookup(target.Target)
	if !found || !adapter.Metadata().Installable {
		status.Reason = "no qualified adapter"
		return status, nil
	}
	if target.ConfigPath == "" {
		path, reason := resolveDefaultConfigPath(adapter, target, env)
		if reason != "" {
			status.Reason = reason
			return status, nil
		}
		target.ConfigPath = path
	}
	status.ConfigPath = target.ConfigPath
	manifestPath := target.ManifestPath
	if manifestPath == "" {
		manifestPath = target.ConfigPath + ".profile-mango.manifest.json"
	}
	snapshot, err := installfs.SnapshotFile(manifestPath)
	if err != nil {
		return TargetStatus{}, fmt.Errorf("inspect %s manifest: %w", target.Target, err)
	}
	if !snapshot.Exists {
		return status, nil
	}
	manifest, err := decodeManifest(snapshot.Content, target.Target)
	if err != nil {
		return TargetStatus{}, fmt.Errorf("inspect %s manifest: %w", target.Target, err)
	}
	status.State, status.Profile, status.Generation = StatusManaged, manifest.Profile, manifest.Generation
	if manifest.Target.Version != "" && manifest.Target.Version != target.Target.Version {
		status.RecordedVersion = manifest.Target.Version
	}
	err = addOwnedFiles(&status, manifest, env, adapter)
	if err == nil {
		inspectOhMyPiPreset(&status, manifest)
	}
	return status, err
}

func addOwnedFiles(status *TargetStatus, manifest Manifest, env PathEnv, adapter Adapter) error {
	for _, file := range manifest.Files {
		fileStatus, err := inspectOwnedFile(status.ConfigPath, file, env, adapter)
		if err != nil {
			return err
		}
		status.ownsConfig = status.ownsConfig || defaultOnlyKinds[fileStatus.Kind]
		status.Files = append(status.Files, fileStatus)
	}
	sort.Slice(status.Files, func(i, j int) bool { return status.Files[i].Path < status.Files[j].Path })
	return nil
}

// defaultOnlyKinds are owned file kinds a named profile writes only when it is also made
// the default, so owning one records a default install.
var defaultOnlyKinds = map[string]bool{"config": true, "global-instruction": true, "role-definition": true, "skill": true}

// ownedFieldReader reads normalized live values independently of the recorded profile.
// This also covers named files belonging to a different profile from the default.
type ownedFieldReader interface {
	OwnedFieldValues(content []byte, fields []string) (map[string]FieldChange, error)
}

func inspectOwnedFile(configPath string, file ManifestFile, env PathEnv, adapter Adapter) (FileStatus, error) {
	snapshot, err := installfs.SnapshotFile(file.Path)
	if err != nil {
		return FileStatus{}, fmt.Errorf("inspect owned file %s: %w", filepath.Base(file.Path), err)
	}
	result := FileStatus{Path: relativeOwnedPath(configPath, file.Path, env), Kind: ownedFileKind(configPath, file), State: FileInSync}
	switch {
	case !snapshot.Exists:
		result.State = FileMissing
	case snapshot.SHA256 != file.SHA256:
		result.State = FileEdited
		if reader, ok := adapter.(ownedFieldReader); ok && wholeFileKind(file.Fields) == "" {
			fields, err := reader.OwnedFieldValues(snapshot.Content, file.Fields)
			if err == nil && ownedFieldsIntact(FilePatch{LiveFields: true}, file, fields) {
				result.State = FileOtherEdits
			}
		}
	}
	for _, field := range file.Fields {
		if !isOwnershipMarker(field) {
			result.Fields = append(result.Fields, field)
		}
	}
	return result, nil
}

// relativeOwnedPath names an owned file relative to the config directory, as
// ~/<path> when it lives elsewhere under the home directory, else by base name.
func relativeOwnedPath(configPath, path string, env PathEnv) string {
	if label := homeLabel(env, configPath, path); label != "" {
		return label
	}
	relative, err := filepath.Rel(filepath.Dir(configPath), path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return filepath.Base(path)
	}
	return filepath.ToSlash(relative)
}

func ownedFileKind(configPath string, file ManifestFile) string {
	switch {
	case filepath.Clean(file.Path) == filepath.Clean(configPath):
		return "config"
	case wholeFileKind(file.Fields) != "":
		return wholeFileKind(file.Fields)
	}
	return "file"
}

func isOwnershipMarker(field string) bool {
	return wholeFileKind([]string{field}) != "" || field == priorAbsent || strings.HasPrefix(field, priorSHA256Prefix) || strings.HasPrefix(field, ohMyPiRolePriorPrefix) ||
		strings.HasPrefix(field, ohMyPiSettingPriorPrefix) || strings.HasPrefix(field, ohMyPiPresetPriorPrefix) || strings.HasPrefix(field, openClawSkillsPriorPrefix) || strings.HasPrefix(field, writtenSHA256Prefix)
}

// sourceState plans the recorded profile again without writing. Override lets the plan
// see past live edits of owned files, so a no-op plan means the installed files match
// the current profile sources and bindings, and any difference is named as drift.
func sourceState(request StatusRequest, registry *Registry, status *TargetStatus) {
	if request.ProfilesRoot == "" || status.Profile == "" {
		status.Source, status.SourceReason = SourceUnknown, "no profile home to compare against"
		return
	}
	plan, err := BuildPlan(Request{
		ProfileName: status.Profile, ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot,
		BindingsPath: request.BindingsPath, Registry: registry, Env: request.Env, Backup: true, Override: true,
		Default: status.ownsConfig, Release: true, StateDir: request.StateDir,
		Targets: []TargetRequest{{Target: status.Target, ConfigPath: status.ConfigPath}},
	})
	if err != nil {
		status.Source, status.SourceReason = SourceUnknown, err.Error()
		return
	}
	if len(plan.Targets) > 0 {
		status.Drift = targetDrift(plan.Targets[0], *status, request.Env)
		markOtherEdits(status, plan.Targets[0], request.Env)
	}
	status.Source, status.SourceReason = sourceVerdict(plan, *status)
	reconcileOhMyPiPreset(request, status)
}

// markOtherEdits reconciles live content with the planned profile. Byte-identical
// whole files are in sync even when their recorded hash is stale; field-owned files
// retain their distinct label for changes outside the owned values.
func markOtherEdits(status *TargetStatus, target TargetPlan, env PathEnv) {
	unchanged := map[string]string{}
	for _, file := range target.Files {
		intact := file.Action == ActionNoop || (file.Action == ActionUpdate && target.Status == StatusReady)
		if intact && file.targetPath != "" {
			state := FileOtherEdits
			if wholeFileKind(file.ownership) != "" && file.Action == ActionNoop {
				state = FileInSync
			}
			if wholeFileKind(file.ownership) == "" || file.Action == ActionNoop {
				unchanged[relativeOwnedPath(status.ConfigPath, file.targetPath, env)] = state
			}
		}
	}
	for index, file := range status.Files {
		if state := unchanged[file.Path]; file.State == FileEdited && state != "" {
			status.Files[index].State = state
		}
	}
}

// sourceVerdict says whether the plan would change anything and how to apply it; live
// edits of owned files need --override, since install otherwise refuses them.
func sourceVerdict(plan Plan, status TargetStatus) (string, string) {
	switch plan.Status {
	case StatusNoop:
		return SourceCurrent, ""
	case StatusReady:
		apply := statusApplyCommand(plan.Targets[0], status)
		if liveEdits(plan.Targets[0]) {
			return SourceChanged, "owned files were edited; run " + apply + " --override to replace the edits with the profile"
		}
		if len(status.Drift) == 0 {
			return SourceChanged, "only the ownership manifest would change; run " + apply + " to rewrite it" + recordedSuffix(status)
		}
		return SourceChanged, "run " + apply + " to apply the changed sources"
	}
	if len(plan.Targets) > 0 && plan.Targets[0].Reason != "" {
		return SourceUnknown, plan.Targets[0].Reason
	}
	return SourceUnknown, "the recorded profile cannot be planned"
}

// statusApplyCommand re-applies the recorded profile to this target only, so no other
// agent is switched. Releasing files the profile dropped needs mango use; otherwise the
// install form reproduces what status planned.
func statusApplyCommand(target TargetPlan, status TargetStatus) string {
	for _, file := range target.Files {
		if file.release && file.Action != ActionNoop {
			return "mango use " + status.Profile + " --target " + status.Target.Name
		}
	}
	return ReapplyCommand(status.Profile, []string{status.Target.Name}, status.ownsConfig)
}

// ReapplyCommand is the mango install command that re-applies a profile to exactly
// these targets in one plan, with --default when the profile is their default.
func ReapplyCommand(profile string, targets []string, makeDefault bool) string {
	var command strings.Builder
	command.WriteString("mango install " + profile)
	for _, target := range targets {
		command.WriteString(" --target " + target)
	}
	if makeDefault {
		command.WriteString(" --default")
	}
	return command.String()
}

// OwnsConfig reports that the recorded profile is the agent's default: the manifest owns
// the main config file or a file only a default install writes. A default install whose
// main config already held the profile's values leaves it unowned.
func (status TargetStatus) OwnsConfig() bool { return status.ownsConfig }

func recordedSuffix(status TargetStatus) string {
	if status.RecordedVersion == "" {
		return ""
	}
	return " at " + status.Target.String()
}

func liveEdits(target TargetPlan) bool {
	for _, file := range target.Files {
		if file.Owned && file.Action == ActionOverride {
			return true
		}
	}
	return false
}

// targetDrift names every planned file the apply would change: by its owned fields
// when their live values were read, else as a whole file with its state.
func targetDrift(target TargetPlan, status TargetStatus, env PathEnv) []FieldDrift {
	fields := make(map[string]FieldChange, len(target.Fields))
	for _, field := range target.Fields {
		fields[field.Path] = field
	}
	if len(target.Files) == 0 {
		return blockedDrift(target, status)
	}
	var drift []FieldDrift
	for _, file := range target.Files {
		if file.Action == ActionNoop || file.targetPath == "" || file.targetPath == target.ManifestPath {
			continue
		}
		changed, liveRead := fileFieldDrift(file, fields)
		if len(changed) > 0 && (liveRead || file.targetPath == status.ConfigPath) {
			drift = append(drift, changed...)
			continue
		}
		path := relativeOwnedPath(status.ConfigPath, file.targetPath, env)
		drift = append(drift, FieldDrift{Path: path, State: fileDriftState(status, path, file)})
	}
	return drift
}

// blockedDrift names drift when planning stopped before it compared files: owned
// fields whose live value was read and differs, and whole owned files not in sync.
func blockedDrift(target TargetPlan, status TargetStatus) []FieldDrift {
	configFields := map[string]bool{}
	var drift []FieldDrift
	for _, file := range status.Files {
		for _, name := range file.Fields {
			configFields[name] = configFields[name] || file.Kind == "config"
		}
		if file.State != FileInSync && wholeFileKind([]string{file.Kind}) != "" {
			drift = append(drift, FieldDrift{Path: file.Path, State: file.State})
		}
	}
	for _, field := range target.Fields {
		if field.Before != field.After && (field.Before != "" || configFields[field.Path]) {
			drift = append(drift, FieldDrift{Path: field.Path, Live: field.Before, Profile: field.After})
		}
	}
	return drift
}

// fileFieldDrift returns the file's owned fields whose live and profile values differ,
// and whether any live value was read (a whole-file render whose current file its
// adapter could not parse carries none).
func fileFieldDrift(file FilePlan, fields map[string]FieldChange) ([]FieldDrift, bool) {
	var changed []FieldDrift
	liveRead := false
	for _, name := range file.Fields {
		field, found := fields[name.Path]
		if !found || field.Before == field.After {
			continue
		}
		liveRead = liveRead || field.Before != ""
		changed = append(changed, FieldDrift{Path: field.Path, Live: field.Before, Profile: field.After})
	}
	return changed, liveRead
}

func fileDriftState(status TargetStatus, path string, file FilePlan) string {
	if file.Action == ActionCreate {
		return FileMissing
	}
	for _, owned := range status.Files {
		if owned.Path == path && owned.State != FileInSync {
			return owned.State
		}
	}
	return DriftDiffers
}
