package hermes

import (
	"fmt"
	"strconv"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName                = "hermes"
	TargetVersion             = "0.21.3"
	HermesVersion             = TargetVersion
	SourceRelease             = "v2026.9.14"
	SourceCommit              = "345cd2b057a452236de401d3534b8502a7465e8d"
	SourceTree                = "6e14b9791cdc5a47068685e9429dd5d6bdc5ef5f"
	SourceArchiveSHA256       = "71f2db39a64fbba282e3bd3be4b0f7b935585948a59a368d61deeec0f0827c47"
	HermesSourceRelease       = SourceRelease
	HermesSourceCommit        = SourceCommit
	HermesSourceTree          = SourceTree
	HermesSourceArchiveSHA256 = SourceArchiveSHA256
	EvidenceSHA256            = SourceArchiveSHA256
	HermesEvidenceSHA256      = EvidenceSHA256
	EvidenceSource            = "docs/dev/target-evidence.md"
	EvidenceLevel             = "source-entrypoint-review"
	AdapterVersion            = "profilemango.dev/hermes/v1alpha1"
	RenderAPIVersion          = render.APIVersion
	RenderKind                = render.Kind
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

// DefaultTarget returns the exact Hermes source-qualified observation.
func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

// ResourceFromContent creates deterministic canonical resource metadata.
func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	return render.ResourceFromContent(resourcePath, kind, content)
}

// NewResult creates a blocked report with source-review evidence metadata.
func NewResult(profileName string, target TargetBuild) Result {
	return render.NewResult(profileName, target, AdapterVersion, Evidence{
		Target: TargetName, Version: TargetVersion, SHA256: EvidenceSHA256,
		Source: EvidenceSource, Level: EvidenceLevel,
	})
}

// Render emits deterministic inert YAML candidate syntax and never claims applicability.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	result.Diagnostics = append(result.Diagnostics, targetDiagnostics(input.Target)...)
	addTargetCapabilities(&result, input.Target)
	addEvidenceBlockers(&result)
	result.Diagnostics = append(result.Diagnostics, routeDiagnostics(input.Route)...)
	addRouteCapabilities(&result)
	profileDiagnostics(&result, input.Profile)
	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix: "hermes", InstructionMessage: "Hermes project-context and instruction delivery precedence are unverified",
		SkillMessage: "Hermes skill discovery and delivery precedence are unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "hermes.resource.") {
		result.Artifacts = append(result.Artifacts, candidateArtifact(input.Profile, input.Route))
		result.Artifacts = append(result.Artifacts, resourceArtifacts...)
	}
	result.Artifacts = render.SortedArtifacts(result.Artifacts)
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Applicable = false
	return result
}

func addEvidenceBlockers(result *Result) {
	blockers := []struct{ code, field, message string }{
		{"hermes.target.artifact_unavailable", "target.artifact", "no exact Hermes runtime artifact was built or observed"},
		{"hermes.config.acceptance_unverified", "config.acceptance", "YAML candidate fields are source-grounded, but native parser acceptance is unverified"},
		{"hermes.config.inspector_unsafe", "config.inspection", "candidate inspectors load dotenv, config, credential, profile, plugin, and state paths; no safe isolated execution was accepted"},
		{"hermes.config.effective_state_unverified", "config.effective-state", "no safe merged effective state report with per-field provenance was established"},
		{"hermes.config.precedence_unverified", "config.precedence", "documented precedence was not runtime-observed for this exact release"},
		{"hermes.context.discovery_unverified", "context", "SOUL.md and project context discovery and precedence are target-owned and unverified"},
		{"hermes.runtime.enforcement_unverified", "runtime.enforcement", "route, permission, tool, context, and policy enforcement was not observed"},
		{"hermes.route.authentication_unverified", "route.authentication", "authentication identity and credential handling are unverified; credentials are never inferred"},
	}
	for _, blocker := range blockers {
		result.Diagnostics.Add(profilemango.SeverityError, blocker.code, blocker.field, blocker.message, 0, 0)
		result.AddCapability(blocker.field, StatusBlocking, blocker.message)
	}
	result.AddCapability("config.fidelity", StatusPartial, "source-grounded model and reasoning fields only")
	result.AddCapability("delivery", StatusBlocking, "target-owned config, context, and resource delivery is unverified")
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "exact Hermes source version and evidence hash are not qualified")
		return
	}
	result.AddCapability("target.version", StatusSupported, "exact Hermes Agent v0.21.3 source release and entrypoint review are pinned")
}

func addRouteCapabilities(result *Result) {
	for _, field := range []string{"provider", "model", "effort"} {
		result.AddCapability("route."+field, StatusPartial, "source-grounded Hermes YAML candidate syntax only")
	}
	result.AddCapability("route.authentication", StatusBlocking, "authentication identity and credential handling are not verified")
	result.AddCapability("route.transport", StatusBlocking, "Hermes transport mapping is not qualified")
}

func profileDiagnostics(result *Result, profile profilemango.ResolvedProfile) {
	if profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "hermes.security.permissions_unverified", "spec.permissions", "Hermes permission equivalence and runtime enforcement are unverified", 0, 0)
		permissionCapabilities(result, profile.Permissions)
	}
	if profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "hermes.tools.allowlist_unverified", "tools", "Hermes toolset and closed allowlist enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "hermes.security.tools_unverified", "spec.tools", "Hermes tool, plugin, and deny-wins enforcement are unverified", 0, 0)
		result.AddCapability("tools", StatusBlocking, "toolset, plugin, and closed allowlist enforcement is unverified")
	}
}

func permissionCapabilities(result *Result, permissions *profilemango.PermissionPolicy) {
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "mode", value: permissions.Mode}, {name: "network", value: permissions.Network}, {name: "shell", value: permissions.Shell},
	} {
		if field.value == nil || (*field.value == "unmanaged" && field.name != "mode") {
			continue
		}
		result.Diagnostics.Add(profilemango.SeverityError, "hermes.permissions."+field.name+"_unverified", "permissions."+field.name, "Hermes enforcement is unverified", 0, 0)
		result.AddCapability("permissions."+field.name, StatusBlocking, "target enforcement is unverified")
	}
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "hermes.target.unsupported", "target", fmt.Sprintf("renderer only accepts target %q", TargetName), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "hermes.target.version_required", "targetVersion", "an exact target version is required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "hermes.target.version_unsupported", "targetVersion", fmt.Sprintf("only Hermes Agent %s is evidenced", TargetVersion), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "hermes.version.unsupported", "targetVersion", fmt.Sprintf("only Hermes Agent %s is evidenced", TargetVersion), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "hermes.version.evidence_required", "targetVersion", "the exact target evidence hash is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "hermes.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned Hermes source archive", 0, 0)
	}
	return diagnostics
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "hermes.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	if route.Transport != "" && route.Transport != "native" {
		diagnostics.Add(profilemango.SeverityError, "hermes.route.transport_unsupported", "route.transport", "Hermes transport mapping is not qualified", 0, 0)
	}
	if route.Effort != "" && !validEffort(route.Effort) {
		diagnostics.Add(profilemango.SeverityError, "hermes.route.effort_unsupported", "route.effort", "effort is not a documented Hermes reasoning effort", 0, 0)
	}
	return diagnostics
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding) Artifact {
	if !render.ValidName(profile.Metadata.Name) || route.Provider == "" || route.Model == "" || !validEffort(route.Effort) {
		return Artifact{}
	}
	content := []byte(fmt.Sprintf("# profile-mango: INERT PREVIEW ONLY\n# NON-APPLICABLE: candidate syntax for Hermes Agent %s (source release %s).\n# This is not an active ~/.hermes/config.yaml. Authentication, state, delivery, precedence, and enforcement are unverified.\n\nmodel:\n  provider: %s\n  default: %s\nagent:\n  reasoning_effort: %s\n", TargetVersion, SourceRelease, yamlString(route.Provider), yamlString(route.Model), yamlString(route.Effort)))
	return render.NewArtifact("preview/"+profile.Metadata.Name+".config.yaml.preview", "candidate-config", content)
}

func yamlString(value string) string {
	return strconv.Quote(value)
}

func validEffort(value string) bool {
	switch value {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra":
		return true
	default:
		return false
	}
}
