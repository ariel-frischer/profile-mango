package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const (
	TargetName          = "codex"
	TargetVersion       = "0.154.0"
	CodexVersion        = TargetVersion
	AdapterVersion      = "profilemango.dev/codex/v1alpha1"
	RenderAPIVersion    = "profilemango.dev/render/v1alpha1"
	RenderKind          = "RenderReport"
	EvidenceSHA256      = "3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022"
	CodexEvidenceSHA256 = EvidenceSHA256
	EvidenceSource      = "docs/dev/target-evidence.md"
	EvidenceLevel       = "native-config-parsing"
)

const (
	StatusSupported  = "supported"
	StatusPartial    = "partial"
	StatusUnverified = "unverified"
	StatusBlocking   = "blocking"
)

// TargetBuild identifies the exact target observation supplied to the renderer.
type TargetBuild struct {
	Name           string
	Version        string
	EvidenceSHA256 string
}

// DefaultTarget returns the only version-qualified target observation.
func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

// Input is the pure renderer input. It contains no filesystem or target-home handle.
type Input struct {
	Profile   profilemango.ResolvedProfile
	Route     profilemango.RouteBinding
	Resources []Resource
	Target    TargetBuild
}

// RenderInput is retained as the descriptive name used by the adapter boundary.
type RenderInput = Input

// Resource carries already validated canonical metadata and its package bytes.
type Resource struct {
	Digest  profilemango.ResourceDigest
	Content []byte
}

// ResourceFromContent creates deterministic canonical resource metadata.
func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	sum := sha256.Sum256(content)
	return Resource{
		Digest: profilemango.ResourceDigest{
			Path:   resourcePath,
			Kind:   kind,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(content)),
		},
		Content: append([]byte(nil), content...),
	}
}

type Evidence struct {
	Target  string `json:"target"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Source  string `json:"source"`
	Level   string `json:"level"`
}

type Capability struct {
	Field  string `json:"field"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// Artifact is an inert file candidate. Content is intentionally excluded from JSON.
type Artifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	Content []byte `json:"-"`
}

// Result is the deterministic render and applicability report.
type Result struct {
	APIVersion     string                   `json:"apiVersion"`
	Kind           string                   `json:"kind"`
	Profile        string                   `json:"profile"`
	Target         string                   `json:"target"`
	TargetVersion  string                   `json:"targetVersion"`
	AdapterVersion string                   `json:"adapterVersion"`
	Applicable     bool                     `json:"applicable"`
	Preview        bool                     `json:"preview"`
	Evidence       Evidence                 `json:"evidence"`
	Capabilities   []Capability             `json:"capabilities"`
	Diagnostics    profilemango.Diagnostics `json:"diagnostics"`
	Artifacts      []Artifact               `json:"artifacts"`
}

// NewResult creates an empty report with pinned evidence metadata.
func NewResult(profileName string, target TargetBuild) Result {
	return Result{
		APIVersion:     RenderAPIVersion,
		Kind:           RenderKind,
		Profile:        profileName,
		Target:         target.Name,
		TargetVersion:  target.Version,
		AdapterVersion: AdapterVersion,
		Applicable:     false,
		Evidence: Evidence{
			Target:  TargetName,
			Version: TargetVersion,
			SHA256:  EvidenceSHA256,
			Source:  EvidenceSource,
			Level:   EvidenceLevel,
		},
		Capabilities: make([]Capability, 0),
		Diagnostics:  make(profilemango.Diagnostics, 0),
		Artifacts:    make([]Artifact, 0),
	}
}

// Render deterministically renders documented candidate syntax and evaluates it fail-closed.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	targetBlocked := targetDiagnostics(input.Target)
	result.Diagnostics = append(result.Diagnostics, targetBlocked...)
	addTargetCapabilities(&result, input.Target)
	result.Diagnostics.Add(profilemango.SeverityError, "codex.route.authentication_unverified", "route.authentication", "Codex authentication route is not verified by the pinned evidence; credentials and route identity are never inferred", 0, 0)
	result.addCapability("route.authentication", StatusBlocking, "authentication identity is not verified")
	permissionDiagnostics(&result, input.Profile.Permissions)
	if input.Profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "codex.security.permissions_unverified", "spec.permissions", "Codex permission equivalence and runtime enforcement are not verified", 0, 0)
	}
	if input.Profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "codex.tools.allowlist_unverified", "tools", "Codex closed tool enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "codex.security.tools_unverified", "spec.tools", "Codex tool allowlist and deny-wins enforcement are not verified", 0, 0)
		result.addCapability("tools", StatusBlocking, "closed tool enforcement is not verified")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 {
		result.Diagnostics.Add(profilemango.SeverityError, "codex.delivery.unverified", "resources", "Codex profile, instruction, and skill delivery precedence is not verified for the pinned build", 0, 0)
	}
	routeBlocked := routeDiagnostics(input.Route)
	result.Diagnostics = append(result.Diagnostics, routeBlocked...)
	addRouteCapabilities(&result, input.Route)
	resourceArtifacts, resourceDiagnostics := resourceArtifacts(input.Profile, input.Resources)
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !hasCodePrefix(resourceDiagnostics, "codex.resource.") {
		result.Artifacts = append(result.Artifacts, candidateArtifact(input.Profile, input.Route))
		result.Artifacts = append(result.Artifacts, resourceArtifacts...)
		result.Artifacts = append(result.Artifacts, metadataArtifacts(input)...)
	}
	result.Artifacts = sortedArtifacts(result.Artifacts)
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Applicable = false
	return result
}

func hasCodePrefix(diagnostics profilemango.Diagnostics, prefix string) bool {
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Code, prefix) {
			return true
		}
	}
	return false
}

func metadataArtifacts(input Input) []Artifact {
	resources := make([]profilemango.ResourceDigest, 0, len(input.Resources))
	for _, resource := range input.Resources {
		resources = append(resources, resource.Digest)
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Path < resources[j].Path })
	plan, err := profilemango.BuildPlan(input.Profile, TargetName, profilemango.Bindings{Routes: map[string]profilemango.RouteBinding{input.Profile.RouteRef: input.Route}}, resources)
	if err != nil {
		return nil
	}
	manifest := profilemango.Manifest{
		APIVersion: profilemango.ManifestVersion,
		Kind:       "Manifest",
		Owner:      "profile-mango",
		Generation: 1,
		Profile:    input.Profile.Metadata.Name,
		Target:     TargetName,
		Resources:  resources,
	}
	planData, err := profilemango.CanonicalJSON(plan)
	if err != nil {
		return nil
	}
	manifestData, err := profilemango.CanonicalJSON(manifest)
	if err != nil {
		return nil
	}
	return []Artifact{
		newArtifact("plan.json", "plan", planData),
		newArtifact("manifest.json", "manifest", manifestData),
	}
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.addCapability("target.version", StatusBlocking, "exact target build evidence is not qualified")
		return
	}
	result.addCapability("target.version", StatusSupported, "exact Codex CLI build observation is pinned")
}

func addRouteCapabilities(result *Result, route profilemango.RouteBinding) {
	for _, field := range []string{"provider", "model", "effort"} {
		result.addCapability("route."+field, StatusPartial, "documented candidate syntax only")
	}
	if route.Transport != "native" {
		result.addCapability("route.transport", StatusBlocking, "only the native transport has candidate syntax")
	} else {
		result.addCapability("route.transport", StatusPartial, "documented candidate syntax only")
	}
}

func permissionDiagnostics(result *Result, permissions *profilemango.PermissionPolicy) {
	if permissions == nil {
		return
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "mode", value: permissions.Mode},
		{name: "network", value: permissions.Network},
		{name: "shell", value: permissions.Shell},
	} {
		if field.value == nil || (*field.value == "unmanaged" && field.name != "mode") {
			continue
		}
		result.Diagnostics.Add(profilemango.SeverityError, "codex.permissions."+field.name+"_unverified", "permissions."+field.name, "Codex enforcement is unverified", 0, 0)
		result.addCapability("permissions."+field.name, StatusBlocking, "target enforcement is not verified")
	}
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "codex.target.unsupported", "target", fmt.Sprintf("renderer only accepts target %q", TargetName), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "codex.target.version_required", "targetVersion", "an exact target version is required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "codex.target.version_unsupported", "targetVersion", fmt.Sprintf("only Codex CLI %s is evidenced", TargetVersion), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "codex.version.unsupported", "targetVersion", fmt.Sprintf("only Codex CLI %s is evidenced", TargetVersion), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "codex.version.evidence_required", "targetVersion", "the exact target evidence hash is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "codex.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned Codex build", 0, 0)
	}
	return diagnostics
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "codex.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	if route.Transport != "" && route.Transport != "native" {
		diagnostics.Add(profilemango.SeverityError, "codex.route.transport_unsupported", "route.transport", "only the native transport has candidate syntax", 0, 0)
	}
	return diagnostics
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding) Artifact {
	if !validName(profile.Metadata.Name) || route.Provider == "" || route.Model == "" || route.Effort == "" {
		return Artifact{}
	}
	content := []byte(fmt.Sprintf("# profile-mango: INERT PREVIEW ONLY\n# NON-APPLICABLE: candidate syntax for Codex CLI %s.\n# This is not an active configuration. Authentication, delivery, and enforcement are unverified.\n\nmodel_provider = %s\nmodel = %s\nmodel_reasoning_effort = %s\n", TargetVersion, tomlString(route.Provider), tomlString(route.Model), tomlString(route.Effort)))
	return newArtifact("preview/"+profile.Metadata.Name+".config.toml.preview", "candidate-config", content)
}

func resourceArtifacts(profile profilemango.ResolvedProfile, resources []Resource) ([]Artifact, profilemango.Diagnostics) {
	var diagnostics profilemango.Diagnostics
	if len(profile.Instructions) > 0 {
		diagnostics.Add(profilemango.SeverityError, "codex.instructions.delivery_unverified", "instructions", "Codex instruction delivery and precedence are unverified", 0, 0)
	}
	if len(profile.Skills) > 0 {
		diagnostics.Add(profilemango.SeverityError, "codex.skills.delivery_unverified", "skills", "Codex skill delivery and precedence are unverified", 0, 0)
	}
	requested := make(map[string]struct{}, len(profile.Instructions)+len(profile.Skills))
	for _, item := range profile.Instructions {
		requested[path.Clean(item)] = struct{}{}
	}
	for _, item := range profile.Skills {
		requested[path.Clean(item)] = struct{}{}
	}
	seen := make(map[string]struct{}, len(resources))
	artifacts := make([]Artifact, 0, len(resources))
	for _, resource := range resources {
		name := path.Clean(resource.Digest.Path)
		if !safePath(resource.Digest.Path) || name != resource.Digest.Path {
			diagnostics.Add(profilemango.SeverityError, "codex.resource.path_invalid", resource.Digest.Path, "resource path must be canonical and relative", 0, 0)
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			diagnostics.Add(profilemango.SeverityError, "codex.resource.duplicate", name, "resource path is duplicated", 0, 0)
			continue
		}
		seen[name] = struct{}{}
		if _, requested := requested[name]; !requested {
			diagnostics.Add(profilemango.SeverityError, "codex.resource.unrequested", name, "resource is not referenced by the resolved profile", 0, 0)
			continue
		}
		sum := sha256.Sum256(resource.Content)
		actual := hex.EncodeToString(sum[:])
		if actual != resource.Digest.SHA256 || int64(len(resource.Content)) != resource.Digest.Size {
			diagnostics.Add(profilemango.SeverityError, "codex.resource.digest_mismatch", name, "resource bytes do not match the canonical digest", 0, 0)
			continue
		}
		artifacts = append(artifacts, newArtifact("resources/"+name, "resource", resource.Content))
		delete(requested, name)
	}
	missing := make([]string, 0, len(requested))
	for name := range requested {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	for _, name := range missing {
		diagnostics.Add(profilemango.SeverityError, "codex.resource.missing", name, "resolved resource was not supplied to the renderer", 0, 0)
	}
	return artifacts, diagnostics
}

func newArtifact(name, kind string, content []byte) Artifact {
	sum := sha256.Sum256(content)
	return Artifact{Path: name, Kind: kind, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(content)), Content: append([]byte(nil), content...)}
}

func sortedArtifacts(artifacts []Artifact) []Artifact {
	filtered := artifacts[:0]
	for _, artifact := range artifacts {
		if artifact.Path != "" {
			filtered = append(filtered, artifact)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Path < filtered[j].Path })
	return filtered
}

func safePath(value string) bool {
	return value != "" && !path.IsAbs(value) && value != "." && value != ".." && !strings.HasPrefix(value, "../") && !strings.Contains(value, "\\")
}

func validName(value string) bool {
	if value == "" || len(value) > 63 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, char := range value[1:] {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return utf8.ValidString(value)
}

func tomlString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, char := range value {
		switch char {
		case '\\':
			builder.WriteString("\\\\")
		case '"':
			builder.WriteString("\\\"")
		case '\b':
			builder.WriteString("\\b")
		case '\t':
			builder.WriteString("\\t")
		case '\n':
			builder.WriteString("\\n")
		case '\f':
			builder.WriteString("\\f")
		case '\r':
			builder.WriteString("\\r")
		default:
			if char < 0x20 || char == 0x7f {
				fmt.Fprintf(&builder, "\\u%04x", char)
			} else {
				builder.WriteRune(char)
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func (result *Result) addCapability(field, status, reason string) {
	for _, capability := range result.Capabilities {
		if capability.Field == field {
			return
		}
	}
	result.Capabilities = append(result.Capabilities, Capability{Field: field, Status: status, Reason: reason})
	sort.Slice(result.Capabilities, func(i, j int) bool { return result.Capabilities[i].Field < result.Capabilities[j].Field })
}

// Report returns the metadata-only report form for the requested output mode.
func (result Result) Report(preview bool) Result {
	result.Preview = preview
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Artifacts = sortedArtifacts(result.Artifacts)
	return result
}

// ReportJSON returns deterministic report bytes.
func (result Result) ReportJSON(preview bool) ([]byte, error) {
	report := result.Report(preview)
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode render report: %w", err)
	}
	return append(data, '\n'), nil
}
