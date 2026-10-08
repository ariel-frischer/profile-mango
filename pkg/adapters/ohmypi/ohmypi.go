package ohmypi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName     = "oh-my-pi"
	TargetVersion  = "18.6.0"
	OhMyPiVersion  = TargetVersion
	AdapterVersion = "profilemango.dev/oh-my-pi/v1alpha1"
	EvidenceSHA256 = "2fcbf1a46b27ad7c631bb210abaa217dc092a6e629a74f977a8bde07b59c6d24"
	EvidenceSource = "docs/dev/target-evidence.md"
	EvidenceLevel  = "source-entrypoint-version"
)

const (
	StatusSupported  = render.StatusSupported
	StatusPartial    = render.StatusPartial
	StatusUnverified = render.StatusUnverified
	StatusBlocking   = render.StatusBlocking
)

type TargetBuild = render.TargetBuild
type Input = render.Input
type RenderInput = render.RenderInput
type Resource = render.Resource
type Evidence = render.Evidence
type Capability = render.Capability
type Artifact = render.Artifact
type Result = render.Result

// DefaultTarget returns the exact source-qualified Oh My Pi observation.
func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

// ResourceFromContent creates deterministic canonical resource metadata.
func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	return render.ResourceFromContent(resourcePath, kind, content)
}

// NewResult creates a blocked report with source-level evidence metadata.
func NewResult(profileName string, target TargetBuild) Result {
	return render.NewResult(profileName, target, AdapterVersion, Evidence{
		Target:  TargetName,
		Version: TargetVersion,
		SHA256:  EvidenceSHA256,
		Source:  EvidenceSource,
		Level:   EvidenceLevel,
	})
}

// Render emits deterministic inert candidate syntax and never claims applicability.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	result.Diagnostics = append(result.Diagnostics, targetDiagnostics(input.Target)...)
	addTargetCapabilities(&result, input.Target)
	addSupportBlockers(&result, input.Profile)
	result.Diagnostics = append(result.Diagnostics, routeDiagnostics(input.Route)...)
	addRouteCapabilities(&result, input.Route)
	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix:         "ohmypi",
		InstructionMessage: "Oh My Pi instruction delivery and precedence are unverified",
		SkillMessage:       "Oh My Pi skill delivery and precedence are unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "ohmypi.resource.") {
		result.Artifacts = append(result.Artifacts, candidateArtifact(input.Profile, input.Route))
		result.Artifacts = append(result.Artifacts, resourceArtifacts...)
	}
	result.Artifacts = render.SortedArtifacts(result.Artifacts)
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Applicable = false
	return result
}

func addSupportBlockers(result *Result, profile profilemango.ResolvedProfile) {
	result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.route.authentication_unverified", "route.authentication", "Oh My Pi authentication identity and credential handling are not verified; no safe automated remedy is known. Establish exact-source, credential-free route-identity evidence before applying output; credentials are never inferred", 0, 0)
	result.AddCapability("route.authentication", StatusBlocking, "authentication identity and credential handling are not verified")
	result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.config.inspector_unsafe", "target.config-inspection", "the pinned config inspector initializes settings, project discovery, and migration paths; no safe isolated inspection was accepted. No safe automated remedy is known without first bounding those effects", 0, 0)
	result.AddCapability("target.config-inspection", StatusBlocking, "config path/list effects are not safe to claim")
	result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.target.artifact_unavailable", "target.artifact", "the exact standalone build is blocked by the missing pinned native addon; no safe automated remedy is known without the exact addon", 0, 0)
	result.AddCapability("target.artifact", StatusBlocking, "compiled Oh My Pi artifact is unavailable")
	if profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.security.permissions_unverified", "spec.permissions", "Oh My Pi permission equivalence and runtime enforcement are not verified", 0, 0)
		permissionCapabilities(result, profile.Permissions)
	}
	if profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.tools.allowlist_unverified", "tools", "Oh My Pi closed tool enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.security.tools_unverified", "spec.tools", "Oh My Pi tool allowlist and approval enforcement are not verified", 0, 0)
		result.AddCapability("tools", StatusBlocking, "closed tool and approval enforcement is not verified")
	}
	if len(profile.Instructions) > 0 || len(profile.Skills) > 0 {
		result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.delivery.unverified", "resources", "Oh My Pi profile, instruction, and skill delivery precedence is not verified for the pinned source", 0, 0)
	}
}

func permissionCapabilities(result *Result, permissions *profilemango.PermissionPolicy) {
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
		result.Diagnostics.Add(profilemango.SeverityError, "ohmypi.permissions."+field.name+"_unverified", "permissions."+field.name, "Oh My Pi enforcement is unverified", 0, 0)
		result.AddCapability("permissions."+field.name, StatusBlocking, "target enforcement is not verified")
	}
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "exact Oh My Pi source version and evidence hash are not qualified")
		return
	}
	result.AddCapability("target.version", StatusSupported, "exact Oh My Pi v18.6.0 source entrypoint is pinned")
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "ohmypi.target.unsupported", "target", fmt.Sprintf("renderer only accepts target %q", TargetName), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "ohmypi.target.version_required", "targetVersion", "an exact target version is required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "ohmypi.target.version_unsupported", "targetVersion", fmt.Sprintf("only Oh My Pi %s is evidenced", TargetVersion), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "ohmypi.version.unsupported", "targetVersion", fmt.Sprintf("only Oh My Pi %s is evidenced", TargetVersion), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "ohmypi.version.evidence_required", "targetVersion", "the exact target evidence hash is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "ohmypi.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned Oh My Pi source manifest", 0, 0)
	}
	return diagnostics
}

func addRouteCapabilities(result *Result, route profilemango.RouteBinding) {
	for _, field := range []string{"provider", "model", "effort"} {
		result.AddCapability("route."+field, StatusPartial, "documented candidate syntax only")
	}
	if len(route.Roles) > 0 {
		result.AddCapability("route.roles", StatusPartial, "portable roles map to documented modelRoles slots (worker: task; planner: plan, slow; research: smol; tiny: commit, tiny)")
	}
	if route.SubagentMaxEffort != "" {
		result.AddCapability("route.subagentMaxEffort", StatusPartial, "documented task.maxEffort setting syntax only")
	}
	result.AddCapability("route.transport", StatusBlocking, "Oh My Pi transport mapping is not qualified")
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "ohmypi.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	return diagnostics
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding) Artifact {
	if !render.ValidName(profile.Metadata.Name) || route.Provider == "" || route.Model == "" || route.Effort == "" {
		return Artifact{}
	}
	slots, err := roleSlots(route)
	if err != nil {
		return Artifact{}
	}
	var roles strings.Builder
	fmt.Fprintf(&roles, "  default: %s\n", yamlString(RoleSelector(route.Provider, route.Model, route.Effort)))
	for _, slot := range slots {
		fmt.Fprintf(&roles, "  %s: %s\n", slot.slot, yamlString(RoleSelector(slot.route.Provider, slot.route.Model, slot.route.Effort)))
	}
	if route.SubagentMaxEffort != "" {
		fmt.Fprintf(&roles, "task:\n  maxEffort: %s\n", yamlString(route.SubagentMaxEffort))
	}
	content := []byte(fmt.Sprintf("# profile-mango: INERT PREVIEW ONLY\n# NON-APPLICABLE: candidate syntax for Oh My Pi %s.\n# This is not an active configuration. Authentication, delivery, and enforcement are unverified.\n\nmodelRoles:\n%s", TargetVersion, roles.String()))
	return render.NewArtifact("preview/"+profile.Metadata.Name+".config.yml.preview", "candidate-config", content)
}

func yamlString(value string) string {
	return strconv.Quote(value)
}
