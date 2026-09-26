package codex

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName          = "codex"
	TargetVersion       = "0.154.0"
	CodexVersion        = TargetVersion
	AdapterVersion      = "profilemango.dev/codex/v1alpha1"
	RenderAPIVersion    = render.APIVersion
	RenderKind          = render.Kind
	EvidenceSHA256      = "3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022"
	CodexEvidenceSHA256 = EvidenceSHA256
	EvidenceSource      = "docs/dev/target-evidence.md"
	EvidenceLevel       = "native-config-parsing"
)

const (
	StatusSupported  = render.StatusSupported
	StatusPartial    = render.StatusPartial
	StatusUnverified = render.StatusUnverified
	StatusBlocking   = render.StatusBlocking
)

type TargetBuild = render.TargetBuild

// DefaultTarget returns the only version-qualified target observation.
func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

// Input is the pure renderer input. It contains no filesystem or target-home handle.
type Input = render.Input

// RenderInput is retained as the descriptive name used by the adapter boundary.
type RenderInput = Input

// Resource carries already validated canonical metadata and its package bytes.
type Resource = render.Resource

// ResourceFromContent creates deterministic canonical resource metadata.
func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	return render.ResourceFromContent(resourcePath, kind, content)
}

type Evidence = render.Evidence

type Capability = render.Capability

type Artifact = render.Artifact

type Result = render.Result

// NewResult creates an empty report with pinned evidence metadata.
func NewResult(profileName string, target TargetBuild) Result {
	return render.NewResult(profileName, target, AdapterVersion, Evidence{
		Target:  TargetName,
		Version: TargetVersion,
		SHA256:  EvidenceSHA256,
		Source:  EvidenceSource,
		Level:   EvidenceLevel,
	})
}

// Render deterministically renders documented candidate syntax and evaluates it fail-closed.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	targetBlocked := targetDiagnostics(input.Target)
	result.Diagnostics = append(result.Diagnostics, targetBlocked...)
	addTargetCapabilities(&result, input.Target)
	result.Diagnostics.Add(profilemango.SeverityError, "codex.route.authentication_unverified", "route.authentication", "Codex authentication route is not verified by the pinned evidence; no safe automated remedy is known. Establish exact-build, credential-free route-identity evidence before applying output; credentials and route identity are never inferred", 0, 0)
	result.AddCapability("route.authentication", StatusBlocking, "no safe automated remedy is known; exact-build route identity evidence is required")
	permissionDiagnostics(&result, input.Profile.Permissions)
	if input.Profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "codex.security.permissions_unverified", "spec.permissions", "Codex permission equivalence and runtime enforcement are not verified", 0, 0)
	}
	if input.Profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "codex.tools.allowlist_unverified", "tools", "Codex closed tool enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "codex.security.tools_unverified", "spec.tools", "Codex tool allowlist and deny-wins enforcement are not verified", 0, 0)
		result.AddCapability("tools", StatusBlocking, "closed tool enforcement is not verified")
	}
	if len(input.Profile.Instructions) > 0 || len(input.Profile.Skills) > 0 {
		result.Diagnostics.Add(profilemango.SeverityError, "codex.delivery.unverified", "resources", "Codex profile, instruction, and skill delivery precedence is not verified for the pinned build", 0, 0)
	}
	routeBlocked := routeDiagnostics(input.Route)
	result.Diagnostics = append(result.Diagnostics, routeBlocked...)
	addRouteCapabilities(&result, input.Route)
	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix:         "codex",
		InstructionMessage: "Codex instruction delivery and precedence are unverified",
		SkillMessage:       "Codex skill delivery and precedence are unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "codex.resource.") {
		result.Artifacts = append(result.Artifacts, candidateArtifact(input.Profile, input.Route))
		result.Artifacts = append(result.Artifacts, resourceArtifacts...)
	}
	result.Artifacts = render.SortedArtifacts(result.Artifacts)
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Applicable = false
	return result
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "exact target build evidence is not qualified")
		return
	}
	result.AddCapability("target.version", StatusSupported, "exact Codex CLI build observation is pinned")
}

func addRouteCapabilities(result *Result, route profilemango.RouteBinding) {
	for _, field := range []string{"provider", "model", "effort"} {
		result.AddCapability("route."+field, StatusPartial, "documented candidate syntax only")
	}
	if route.Transport != "native" {
		result.AddCapability("route.transport", StatusBlocking, "only the native transport has candidate syntax")
	} else {
		result.AddCapability("route.transport", StatusPartial, "documented candidate syntax only")
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
		result.AddCapability("permissions."+field.name, StatusBlocking, "target enforcement is not verified")
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
	if !render.ValidName(profile.Metadata.Name) || route.Provider == "" || route.Model == "" || route.Effort == "" {
		return Artifact{}
	}
	content := []byte(fmt.Sprintf("# profile-mango: INERT PREVIEW ONLY\n# NON-APPLICABLE: candidate syntax for Codex CLI %s.\n# This is not an active configuration. Authentication, delivery, and enforcement are unverified.\n\nmodel_provider = %s\nmodel = %s\nmodel_reasoning_effort = %s\n", TargetVersion, tomlString(route.Provider), tomlString(route.Model), tomlString(route.Effort)))
	return render.NewArtifact("preview/"+profile.Metadata.Name+".config.toml.preview", "candidate-config", content)
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
