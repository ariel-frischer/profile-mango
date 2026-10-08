package openclaw

import (
	"fmt"
	"strconv"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName             = "openclaw"
	TargetVersion          = "2026.9.5"
	OpenClawVersion        = TargetVersion
	AdapterVersion         = "profilemango.dev/openclaw/v1alpha1"
	RenderAPIVersion       = render.APIVersion
	RenderKind             = render.Kind
	EvidenceSHA256         = "0e15e679795134cf7d488302f2bdaf0682ad4413e19a7f5c6cc22584f03d02a4"
	OpenClawEvidenceSHA256 = EvidenceSHA256
	EvidenceSource         = "docs/dev/target-evidence.md"
	EvidenceLevel          = "source-entrypoint-review"
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

// DefaultTarget returns the exact source-qualified OpenClaw observation.
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
		Target:  TargetName,
		Version: TargetVersion,
		SHA256:  EvidenceSHA256,
		Source:  EvidenceSource,
		Level:   EvidenceLevel,
	})
}

// Render emits deterministic inert JSON5 candidate syntax and never claims applicability.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	result.Diagnostics = append(result.Diagnostics, targetDiagnostics(input.Target)...)
	addTargetCapabilities(&result, input.Target)
	addEvidenceBlockers(&result)
	result.Diagnostics = append(result.Diagnostics, routeDiagnostics(input.Route)...)
	addRouteCapabilities(&result, input.Route)
	profileDiagnostics(&result, input.Profile)
	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix:         "openclaw",
		InstructionMessage: "OpenClaw instruction delivery and precedence are unverified",
		SkillMessage:       "OpenClaw skill delivery and precedence are unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "openclaw.resource.") {
		result.Artifacts = append(result.Artifacts, candidateArtifact(input.Profile, input.Route))
		result.Artifacts = append(result.Artifacts, resourceArtifacts...)
	}
	result.Artifacts = render.SortedArtifacts(result.Artifacts)
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Applicable = false
	return result
}

func addEvidenceBlockers(result *Result) {
	blockers := []struct {
		code, path, field, message string
	}{
		{"openclaw.target.artifact_unavailable", "target.artifact", "target.artifact", "no exact OpenClaw runtime artifact was built or observed; no safe automated remedy is known without an exact pinned runtime artifact"},
		{"openclaw.config.acceptance_unverified", "target.config.acceptance", "config.acceptance", "JSON5 candidate syntax is source-grounded but native parser acceptance is unverified"},
		{"openclaw.config.inspector_unsafe", "target.config-inspection", "config.inspection", "config validation reads plugin, state, dotenv, and include paths; no safe isolated execution was established. No safe automated remedy is known without first bounding those effects"},
		{"openclaw.config.effective_state_unverified", "target.config.effective-state", "config.effective-state", "no safe command emits merged effective state with per-field provenance"},
		{"openclaw.config.precedence_unverified", "target.config.precedence", "config.precedence", "configuration precedence is documented but not runtime-observed for this exact build"},
		{"openclaw.runtime.enforcement_unverified", "target.runtime.enforcement", "runtime.enforcement", "runtime route, permission, tool, and policy enforcement was not observed"},
		{"openclaw.route.authentication_unverified", "route.authentication", "route.authentication", "authentication identity and credential handling are unverified; no safe automated remedy is known. Establish exact-source, credential-free route-identity evidence before applying output; credentials are never inferred"},
	}
	for _, blocker := range blockers {
		result.Diagnostics.Add(profilemango.SeverityError, blocker.code, blocker.path, blocker.message, 0, 0)
		result.AddCapability(blocker.field, StatusBlocking, blocker.message)
	}
	result.AddCapability("config.fidelity", StatusPartial, "source-grounded candidate fields only")
	result.AddCapability("delivery", StatusBlocking, "target-owned config and resource delivery is unverified")
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "exact OpenClaw source evidence is not qualified")
		return
	}
	result.AddCapability("target.version", StatusSupported, "exact OpenClaw v2026.9.5 source and entrypoint review is pinned")
}

func addRouteCapabilities(result *Result, route profilemango.RouteBinding) {
	for _, field := range []string{"provider", "model"} {
		result.AddCapability("route."+field, StatusPartial, "documented provider/model and thinking candidate syntax only")
	}
	if !validThinkingLevel(route.Effort) && route.Effort != "" {
		result.AddCapability("route.effort", StatusBlocking, "effort is not an OpenClaw thinking level")
	} else {
		result.AddCapability("route.effort", StatusPartial, "documented thinking candidate syntax only")
	}
	result.AddCapability("route.transport", StatusBlocking, "OpenClaw transport mapping is not qualified")
}

func profileDiagnostics(result *Result, profile profilemango.ResolvedProfile) {
	if profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "openclaw.security.permissions_unverified", "spec.permissions", "OpenClaw permission equivalence and runtime enforcement are unverified", 0, 0)
		permissionCapabilities(result, profile.Permissions)
	}
	if profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "openclaw.tools.allowlist_unverified", "tools", "OpenClaw closed tool enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "openclaw.security.tools_unverified", "spec.tools", "OpenClaw tool allowlist, plugins, and deny-wins enforcement are unverified", 0, 0)
		result.AddCapability("tools", StatusBlocking, "closed tool and plugin enforcement is unverified")
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
		result.Diagnostics.Add(profilemango.SeverityError, "openclaw.permissions."+field.name+"_unverified", "permissions."+field.name, "OpenClaw enforcement is unverified", 0, 0)
		result.AddCapability("permissions."+field.name, StatusBlocking, "target enforcement is unverified")
	}
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "openclaw.target.unsupported", "target", fmt.Sprintf("renderer only accepts target %q", TargetName), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "openclaw.target.version_required", "targetVersion", "an exact target version is required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "openclaw.target.version_unsupported", "targetVersion", fmt.Sprintf("only OpenClaw %s is evidenced", TargetVersion), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "openclaw.version.unsupported", "targetVersion", fmt.Sprintf("only OpenClaw %s is evidenced", TargetVersion), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "openclaw.version.evidence_required", "targetVersion", "the exact target evidence hash is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "openclaw.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned OpenClaw source archive", 0, 0)
	}
	return diagnostics
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "openclaw.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	if route.Transport != "" && route.Transport != "native" {
		diagnostics.Add(profilemango.SeverityError, "openclaw.route.transport_unsupported", "route.transport", "OpenClaw transport mapping is not qualified", 0, 0)
	}
	if route.Effort != "" && !validThinkingLevel(route.Effort) {
		diagnostics.Add(profilemango.SeverityError, "openclaw.route.effort_unsupported", "route.effort", "effort is not an OpenClaw thinking level", 0, 0)
	}
	return diagnostics
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding) Artifact {
	if !render.ValidName(profile.Metadata.Name) || route.Provider == "" || route.Model == "" || !validThinkingLevel(route.Effort) {
		return Artifact{}
	}
	content := []byte(fmt.Sprintf("// profile-mango: INERT PREVIEW ONLY\n// NON-APPLICABLE: candidate syntax for OpenClaw %s.\n// This is not an active ~/.openclaw/openclaw.json. Authentication, state, delivery, precedence, and enforcement are unverified.\n\n{\n  agents: {\n    defaults: {\n      model: {\n        primary: %s,\n        fallbacks: [],\n      },\n      thinkingDefault: %s,\n    },\n  },\n}\n", TargetVersion, strconv.Quote(route.Provider+"/"+route.Model), strconv.Quote(route.Effort)))
	return render.NewArtifact("preview/"+profile.Metadata.Name+".config.json5.preview", "candidate-config", content)
}

func validThinkingLevel(value string) bool {
	switch value {
	case "off", "minimal", "low", "medium", "high", "xhigh", "adaptive", "max", "ultra":
		return true
	default:
		return false
	}
}
