package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	PlanAPIVersion     = "profilemango.dev/install-plan/v1alpha1"
	ManifestAPIVersion = "profilemango.dev/install-manifest/v1alpha1"
	PlanKind           = "InstallPlan"
	ManifestKind       = "InstallManifest"
)

const (
	StatusReady        = "ready"
	StatusNoop         = "noop"
	StatusBlocked      = "blocked"
	StatusConflict     = "conflict"
	StatusUnavailable  = "unavailable"
	StatusNotAttempted = "not-attempted"
	// StatusSkipped marks a target left out because its agent is not installed (install --all).
	StatusSkipped = "skipped"
)

const (
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionOverride = "override"
	// ActionAdopt patches managed fields into an existing unowned file after a required backup.
	ActionAdopt  = "adopt"
	ActionNoop   = "noop"
	ActionDelete = "delete"
)

type Target struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (target Target) String() string { return target.Name + "@" + target.Version }

func ParseTarget(value string) (Target, error) {
	target, err := ParseTargetSelector(value)
	if err != nil {
		return Target{}, err
	}
	if target.Version == "" {
		return Target{}, fmt.Errorf("target must use exact target@version syntax")
	}
	return target, nil
}

// ParseTargetSelector parses "name" or "name@version"; Version is empty for a bare name.
func ParseTargetSelector(value string) (Target, error) {
	name, version, found := strings.Cut(strings.TrimSpace(value), "@")
	if name == "" || (found && version == "") || strings.ContainsAny(name+version, "\x00\r\n") {
		return Target{}, fmt.Errorf("target must use target or target@version syntax")
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
		return Target{}, fmt.Errorf("target name must be a simple name")
	}
	return Target{Name: name, Version: version}, nil
}

// Matches reports whether selector names target, treating an empty selector version as any version.
func (selector Target) Matches(target Target) bool {
	return selector.Name == target.Name && (selector.Version == "" || selector.Version == target.Version)
}

type AdapterMetadata struct {
	Target         string `json:"target"`
	Version        string `json:"version"`
	AdapterVersion string `json:"adapterVersion"`
	EvidenceSHA256 string `json:"evidenceSHA256,omitempty"`
	// CompatibleRange overrides the default tilde range (">=A <B") of tested
	// agent versions; widen it only with recorded evidence.
	CompatibleRange string `json:"compatibleRange,omitempty"`
	Installable     bool   `json:"installable"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
}

type Snapshot struct {
	Exists  bool
	SHA256  string
	Size    int64
	Mode    uint32
	Device  uint64
	Inode   uint64
	ModTime int64
	Content []byte `json:"-"`
}

func snapshotFromFS(value installfs.Snapshot) Snapshot {
	return Snapshot{Exists: value.Exists, SHA256: value.SHA256, Size: value.Size, Mode: uint32(value.Mode), Device: value.Identity.Device, Inode: value.Identity.Inode, ModTime: value.Identity.ModTime, Content: append([]byte(nil), value.Content...)}
}

type AdapterInput struct {
	Target       Target
	Agent        AgentDestination
	ConfigPath   string
	ManifestPath string
	Profile      profilemango.ResolvedProfile
	Route        profilemango.RouteBinding
	Resources    []render.Resource
	Config       Snapshot
	Manifest     Snapshot
	Ownership    Manifest
	HasManifest  bool
	Override     bool
	// Install says where the profile goes; for a named profile, NamedFile is its current file.
	Install   InstallMode
	NamedFile Snapshot
}

type Adapter interface {
	Metadata() AdapterMetadata
	Plan(AdapterInput) (Patch, error)
}

type FilePatch struct {
	// NoOverride preserves an edited or unowned file even when the patch permits overrides.
	NoOverride bool
	Delete     bool
	Path       string
	Content    []byte
	Fields     []string
	// Ownership records target-specific provenance needed for safe future cleanup.
	Ownership []string `json:"-"`
}

type FieldChange struct {
	Path      string `json:"path"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

func (field FieldChange) public() FieldChange {
	if !field.Sensitive {
		return field
	}
	field.Before = "<redacted>"
	field.After = "<redacted>"
	return field
}

type Patch struct {
	Files           []FilePatch
	Fields          []FieldChange
	Diagnostics     profilemango.Diagnostics
	OverrideAllowed bool
}

type TargetRequest struct {
	Target       Target
	Agent        AgentDestination
	ConfigPath   string
	ManifestPath string
}

// AgentDestination selects a native named OpenCode definition, not the default config.
type AgentDestination struct {
	Mode string `json:"mode"`
	Name string `json:"name"`
}

func (agent AgentDestination) Empty() bool { return agent.Mode == "" && agent.Name == "" }

type Request struct {
	ProfileName  string
	ProfilesRoot string
	ResourceRoot string
	BindingsPath string
	Targets      []TargetRequest
	All          bool
	Backup       bool
	Override     bool
	Registry     *Registry
	// Env resolves documented default config paths for targets without an explicit path.
	Env PathEnv
	// DetectVersion, when set, reports each ready target's installed agent version.
	DetectVersion VersionDetector
	// SkipNotInstalled (install --all) skips a default-path target whose config folder is
	// missing and whose command DetectVersion does not find, instead of blocking the plan.
	SkipNotInstalled bool
	// Strict blocks a target on any known requirement it cannot install instead
	// of installing the supported subset and listing the rest as skipped.
	Strict bool
	// Default also makes an installed named profile the agent's default.
	Default bool
}

type Manifest struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Owner      string         `json:"owner"`
	Generation uint64         `json:"generation"`
	Profile    string         `json:"profile"`
	Target     Target         `json:"target"`
	PlanID     string         `json:"planID"`
	Files      []ManifestFile `json:"files"`
	Fields     []string       `json:"fields,omitempty"`
}

type ManifestFile struct {
	Path   string   `json:"path"`
	SHA256 string   `json:"sha256"`
	Fields []string `json:"fields,omitempty"`
}

type FilePlan struct {
	Path         string        `json:"path"`
	Action       string        `json:"action"`
	BeforeSHA256 string        `json:"beforeSHA256,omitempty"`
	AfterSHA256  string        `json:"afterSHA256,omitempty"`
	Owned        bool          `json:"owned"`
	Fields       []FieldChange `json:"fields,omitempty"`
	Delete       bool          `json:"delete,omitempty"`
	targetPath   string        `json:"-"`
	ownership    []string      `json:"-"`
}

type TargetPlan struct {
	Target            Target                   `json:"target"`
	Agent             *AgentDestination        `json:"agent,omitempty"`
	Install           *InstallMode             `json:"install,omitempty"`
	Config            *ConfigDestination       `json:"config,omitempty"`
	Metadata          AdapterMetadata          `json:"metadata"`
	Status            string                   `json:"status"`
	Reason            string                   `json:"reason,omitempty"`
	DestinationSHA256 string                   `json:"destinationSHA256,omitempty"`
	Files             []FilePlan               `json:"files,omitempty"`
	Fields            []FieldChange            `json:"fields,omitempty"`
	Diagnostics       profilemango.Diagnostics `json:"diagnostics,omitempty"`
	VersionCheck      *VersionCheck            `json:"versionCheck,omitempty"`
	// SkippedRequirements lists profile requirements this target does not install.
	SkippedRequirements []SkippedRequirement `json:"skippedRequirements,omitempty"`
	ConfigPath          string               `json:"-"`
	ManifestPath        string               `json:"-"`
	changes             []installfs.Change   `json:"-"`
	checks              []installfs.Change   `json:"-"`
}

type Plan struct {
	APIVersion  string                   `json:"apiVersion"`
	Kind        string                   `json:"kind"`
	PlanID      string                   `json:"planID"`
	Profile     string                   `json:"profile"`
	Status      string                   `json:"status"`
	Backup      bool                     `json:"backup"`
	Override    bool                     `json:"override"`
	Strict      bool                     `json:"strict"`
	InputSHA256 string                   `json:"inputSHA256"`
	Targets     []TargetPlan             `json:"targets"`
	Diagnostics profilemango.Diagnostics `json:"diagnostics,omitempty"`
	sources     []sourceCheck            `json:"-"`
}

func (plan Plan) JSON() ([]byte, error) {
	plan.normalize()
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode install plan: %w", err)
	}
	return append(data, '\n'), nil
}

func (plan *Plan) normalize() {
	sort.Slice(plan.Targets, func(i, j int) bool { return plan.Targets[i].Target.String() < plan.Targets[j].Target.String() })
	plan.Diagnostics = plan.Diagnostics.Sorted()
	for index := range plan.Targets {
		plan.Targets[index].Diagnostics = plan.Targets[index].Diagnostics.Sorted()
		for fieldIndex := range plan.Targets[index].Fields {
			plan.Targets[index].Fields[fieldIndex] = plan.Targets[index].Fields[fieldIndex].public()
		}
		for fileIndex := range plan.Targets[index].Files {
			for fieldIndex := range plan.Targets[index].Files[fileIndex].Fields {
				plan.Targets[index].Files[fileIndex].Fields[fieldIndex] = plan.Targets[index].Files[fileIndex].Fields[fieldIndex].public()
			}
		}
	}
}

type ApplyOptions struct {
	ExpectedPlanID string
}

type ApplyTargetResult struct {
	Target string `json:"target"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type ApplyReport struct {
	Status  string              `json:"status"`
	Targets []ApplyTargetResult `json:"targets"`
}

func hashValue(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
