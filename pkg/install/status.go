package install

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

// Status report states.
const (
	StatusManaged   = "managed"
	StatusUnmanaged = "unmanaged"

	FileInSync  = "in-sync"
	FileEdited  = "edited"
	FileMissing = "missing"

	SourceCurrent = "current"
	SourceChanged = "changed"
	SourceUnknown = "unknown"
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
}

// StatusReport is the deterministic state of every inspected target.
type StatusReport struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Targets    []TargetStatus `json:"targets"`
}

// TargetStatus is one target's ownership state. Paths are relative to its config directory.
type TargetStatus struct {
	Target       Target       `json:"target"`
	State        string       `json:"state"`
	Reason       string       `json:"reason,omitempty"`
	Profile      string       `json:"profile,omitempty"`
	Generation   uint64       `json:"generation,omitempty"`
	Source       string       `json:"source,omitempty"`
	SourceReason string       `json:"sourceReason,omitempty"`
	Files        []FileStatus `json:"files,omitempty"`
	ConfigPath   string       `json:"-"`
	ownsConfig   bool
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
		if status.State == StatusManaged {
			status.Source, status.SourceReason = sourceState(request, registry, status)
		}
		report.Targets = append(report.Targets, status)
	}
	sort.Slice(report.Targets, func(i, j int) bool { return report.Targets[i].Target.String() < report.Targets[j].Target.String() })
	return report, nil
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
	err = addOwnedFiles(&status, manifest, env)
	return status, err
}

func addOwnedFiles(status *TargetStatus, manifest Manifest, env PathEnv) error {
	for _, file := range manifest.Files {
		fileStatus, err := inspectOwnedFile(status.ConfigPath, file, env)
		if err != nil {
			return err
		}
		status.ownsConfig = status.ownsConfig || fileStatus.Kind == "config"
		status.Files = append(status.Files, fileStatus)
	}
	sort.Slice(status.Files, func(i, j int) bool { return status.Files[i].Path < status.Files[j].Path })
	return nil
}

func inspectOwnedFile(configPath string, file ManifestFile, env PathEnv) (FileStatus, error) {
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
	return wholeFileKind([]string{field}) != "" || field == priorAbsent || strings.HasPrefix(field, priorSHA256Prefix) || strings.HasPrefix(field, ohMyPiRolePriorPrefix)
}

// sourceState plans the recorded profile again without writing: a no-op plan means the
// installed files match the current profile sources and bindings.
func sourceState(request StatusRequest, registry *Registry, status TargetStatus) (string, string) {
	if request.ProfilesRoot == "" || status.Profile == "" {
		return SourceUnknown, "no profile home to compare against"
	}
	plan, err := BuildPlan(Request{
		ProfileName: status.Profile, ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot,
		BindingsPath: request.BindingsPath, Registry: registry, Env: request.Env, Backup: true,
		Default: status.ownsConfig, Release: true,
		Targets: []TargetRequest{{Target: status.Target, ConfigPath: status.ConfigPath}},
	})
	if err != nil {
		return SourceUnknown, err.Error()
	}
	switch plan.Status {
	case StatusNoop:
		return SourceCurrent, ""
	case StatusReady:
		return SourceChanged, "run mango use " + status.Profile + " to apply the changed sources"
	}
	if len(plan.Targets) > 0 && plan.Targets[0].Reason != "" {
		return SourceUnknown, plan.Targets[0].Reason
	}
	return SourceUnknown, "the recorded profile cannot be planned"
}
