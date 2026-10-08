package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const (
	APIVersion = "profilemango.dev/render/v1alpha1"
	Kind       = "RenderReport"
)

const (
	StatusSupported  = "supported"
	StatusPartial    = "partial"
	StatusUnverified = "unverified"
	StatusBlocking   = "blocking"
)

// TargetBuild identifies the exact target observation supplied to a renderer.
type TargetBuild struct {
	Name           string
	Version        string
	EvidenceSHA256 string
}

// Input is pure renderer input with no filesystem or target-home handle.
type Input struct {
	Profile   profilemango.ResolvedProfile
	Route     profilemango.RouteBinding
	Resources []Resource
	Target    TargetBuild
}

// RenderInput is the descriptive name for the adapter boundary.
type RenderInput = Input

// Resource carries validated canonical metadata and package bytes.
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

// NewResult creates an empty report with adapter-owned evidence metadata.
func NewResult(profileName string, target TargetBuild, adapterVersion string, evidence Evidence) Result {
	return Result{
		APIVersion:     APIVersion,
		Kind:           Kind,
		Profile:        profileName,
		Target:         target.Name,
		TargetVersion:  target.Version,
		AdapterVersion: adapterVersion,
		Applicable:     false,
		Evidence:       evidence,
		Capabilities:   make([]Capability, 0),
		Diagnostics:    make(profilemango.Diagnostics, 0),
		Artifacts:      make([]Artifact, 0),
	}
}

// UnknownTarget returns a target-neutral fail-closed report with no artifacts.
func UnknownTarget(profileName string, target TargetBuild) Result {
	version := target.Version
	if version == "" {
		version = "unknown"
	}
	result := NewResult(profileName, target, "profilemango.dev/render/unknown/v1alpha1", Evidence{
		Target:  target.Name,
		Version: version,
		SHA256:  strings.Repeat("0", 64),
		Source:  "unknown target",
		Level:   "none",
	})
	result.Diagnostics.Add(profilemango.SeverityError, "render.target.unsupported", "target", "no renderer is registered for the requested target", 0, 0)
	return result
}

// AddCapability records one capability and keeps capability order stable.
func (result *Result) AddCapability(field, status, reason string) {
	for _, capability := range result.Capabilities {
		if capability.Field == field {
			return
		}
	}
	result.Capabilities = append(result.Capabilities, Capability{Field: field, Status: status, Reason: reason})
	sort.Slice(result.Capabilities, func(i, j int) bool {
		return result.Capabilities[i].Field < result.Capabilities[j].Field
	})
}

// Report returns metadata-only report form for the requested output mode.
func (result Result) Report(preview bool) Result {
	result.Preview = preview
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Artifacts = SortedArtifacts(result.Artifacts)
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

// ResourceArtifactOptions supplies target-specific diagnostic codes and messages.
type ResourceArtifactOptions struct {
	CodePrefix         string
	InstructionMessage string
	SkillMessage       string
}

// ResourceArtifacts validates resource bytes and returns only inert resource candidates.
func ResourceArtifacts(profile profilemango.ResolvedProfile, resources []Resource, options ResourceArtifactOptions) ([]Artifact, profilemango.Diagnostics) {
	var diagnostics profilemango.Diagnostics
	addDeliveryDiagnostics(&diagnostics, profile, options)
	requested := requestedResources(profile)
	seen := make(map[string]struct{}, len(resources))
	artifacts := make([]Artifact, 0, len(resources))
	for _, resource := range resources {
		name := path.Clean(resource.Digest.Path)
		if !SafePath(resource.Digest.Path) || name != resource.Digest.Path {
			addResourceDiagnostic(&diagnostics, options.CodePrefix, "path_invalid", resource.Digest.Path, "resource path must be canonical and relative")
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			addResourceDiagnostic(&diagnostics, options.CodePrefix, "duplicate", name, "resource path is duplicated")
			continue
		}
		seen[name] = struct{}{}
		if _, requested := requested[name]; !requested {
			addResourceDiagnostic(&diagnostics, options.CodePrefix, "unrequested", name, "resource is not referenced by the resolved profile")
			continue
		}
		if !resourceMatchesDigest(resource) {
			addResourceDiagnostic(&diagnostics, options.CodePrefix, "digest_mismatch", name, "resource bytes do not match the canonical digest")
			continue
		}
		artifacts = append(artifacts, NewArtifact("resources/"+name, "resource", resource.Content))
		delete(requested, name)
	}
	addMissingDiagnostics(&diagnostics, requested, options.CodePrefix)
	return artifacts, diagnostics
}

func addDeliveryDiagnostics(diagnostics *profilemango.Diagnostics, profile profilemango.ResolvedProfile, options ResourceArtifactOptions) {
	if len(profile.Instructions) > 0 && options.InstructionMessage != "" {
		diagnostics.Add(profilemango.SeverityError, options.CodePrefix+".instructions.delivery_unverified", "instructions", options.InstructionMessage, 0, 0)
	}
	if len(profile.Skills) > 0 && options.SkillMessage != "" {
		diagnostics.Add(profilemango.SeverityError, options.CodePrefix+".skills.delivery_unverified", "skills", options.SkillMessage, 0, 0)
	}
}

func requestedResources(profile profilemango.ResolvedProfile) map[string]struct{} {
	requested := make(map[string]struct{}, len(profile.Instructions)+len(profile.Skills))
	for _, item := range profile.Instructions {
		requested[path.Clean(item)] = struct{}{}
	}
	for _, item := range profilemango.SkillPaths(profile.Skills) {
		requested[path.Clean(item)] = struct{}{}
	}
	return requested
}

func resourceMatchesDigest(resource Resource) bool {
	sum := sha256.Sum256(resource.Content)
	return hex.EncodeToString(sum[:]) == resource.Digest.SHA256 && int64(len(resource.Content)) == resource.Digest.Size
}

func addResourceDiagnostic(diagnostics *profilemango.Diagnostics, prefix, suffix, name, message string) {
	diagnostics.Add(profilemango.SeverityError, prefix+".resource."+suffix, name, message, 0, 0)
}

func addMissingDiagnostics(diagnostics *profilemango.Diagnostics, requested map[string]struct{}, prefix string) {
	missing := make([]string, 0, len(requested))
	for name := range requested {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	for _, name := range missing {
		addResourceDiagnostic(diagnostics, prefix, "missing", name, "resolved resource was not supplied to the renderer")
	}
}

// NewArtifact hashes and copies inert artifact content.
func NewArtifact(name, kind string, content []byte) Artifact {
	sum := sha256.Sum256(content)
	return Artifact{Path: name, Kind: kind, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(content)), Content: append([]byte(nil), content...)}
}

// SortedArtifacts filters invalid placeholders and returns path-sorted artifacts.
func SortedArtifacts(artifacts []Artifact) []Artifact {
	filtered := make([]Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Path != "" {
			filtered = append(filtered, artifact)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Path < filtered[j].Path })
	return filtered
}

// HasCodePrefix reports whether diagnostics include a target-specific code prefix.
func HasCodePrefix(diagnostics profilemango.Diagnostics, prefix string) bool {
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Code, prefix) {
			return true
		}
	}
	return false
}

// SafePath accepts canonical relative portable paths only.
func SafePath(value string) bool {
	return value != "" && !path.IsAbs(value) && value != "." && value != ".." && !strings.HasPrefix(value, "../") && !strings.Contains(value, "\\")
}

// ValidName validates the simple profile name used in candidate artifact paths.
func ValidName(value string) bool {
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

// RenderRouteRefs resolves {{route.…}} placeholders in each resource for target and
// rehashes the rendered bytes, so digests, plans, and drift describe what is written.
// A placeholder that does not resolve is an error diagnostic on its resource path.
func RenderRouteRefs(resources []Resource, bindings profilemango.Bindings, target string) ([]Resource, profilemango.Diagnostics) {
	var diagnostics profilemango.Diagnostics
	rendered := make([]Resource, 0, len(resources))
	for _, resource := range resources {
		content, err := profilemango.RenderRouteRefs(resource.Content, bindings, target)
		if err != nil {
			diagnostics.Add(profilemango.SeverityError, RouteRefInvalidCode, resource.Digest.Path, err.Error(), 0, 0)
			continue
		}
		if !bytes.Equal(content, resource.Content) {
			resource = ResourceFromContent(resource.Digest.Path, resource.Digest.Kind, content)
		}
		rendered = append(rendered, resource)
	}
	return rendered, diagnostics
}

// RouteRefInvalidCode marks a {{route.…}} placeholder that names an unknown route,
// role, or field, or is malformed.
const RouteRefInvalidCode = "resource.route_placeholder_invalid"
