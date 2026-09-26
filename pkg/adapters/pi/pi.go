package pi

import (
	"encoding/json"
	"fmt"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName           = "pi"
	TargetVersion        = "0.87.1"
	PiVersion            = TargetVersion
	ReleaseTag           = "v0.87.1"
	SourceCommit         = "f07218c4d4bbc12bef056a7058c3dd49dfe41abe"
	SourceArchiveSHA256  = "c3902f45689af9ed9c8ee225554d31a649a993b06f04ab2022bb91ed75e808dc"
	CommitArchiveSHA256  = "f6ba24ed7e1e6dbda1844ca55c61e20e3ef21e5cc66f9eb2e0611318f6c46c15"
	PackageName          = "@earendil-works/pi-coding-agent"
	PackageTarballSHA256 = "1423ee3c61e7c96464e1cbf3c8dc24d3056cb3410995c3671a98c3ecc527540f"
	PackageIntegrity     = "sha512-m8ArJUtVcQMSe1lLE/Ei7vX/JV7O39sWmWBsXV2NOU70F0qCp8GubA24pT3LnwTmM6LL2xV80/h6sQg85n69ew=="
	Entrypoint           = "dist/bundle/cli.js"
	EntrypointSHA256     = "e79626f2dd6f94aa45d30f3fa63cd84319a6eefcd150b353cfaf274366926774"
	EvidenceSHA256       = PackageTarballSHA256
	PiEvidenceSHA256     = EvidenceSHA256
	EvidenceSource       = "docs/dev/target-evidence.md"
	EvidenceLevel        = "immutable-source-and-package-review"
	AdapterVersion       = "profilemango.dev/pi/v1alpha1"
	RenderAPIVersion     = render.APIVersion
	RenderKind           = render.Kind
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

// DefaultTarget returns the exact package-qualified Pi observation.
func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

// ResourceFromContent creates deterministic canonical resource metadata.
func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	return render.ResourceFromContent(resourcePath, kind, content)
}

// NewResult creates a blocked report with immutable source and package evidence.
func NewResult(profileName string, target TargetBuild) Result {
	return render.NewResult(profileName, target, AdapterVersion, Evidence{
		Target: TargetName, Version: TargetVersion, SHA256: EvidenceSHA256,
		Source: EvidenceSource, Level: EvidenceLevel,
	})
}

// Render emits deterministic inert Pi settings JSON and never claims applicability.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	result.Diagnostics = append(result.Diagnostics, targetDiagnostics(input.Target)...)
	addTargetCapabilities(&result, input.Target)
	addEvidenceBlockers(&result)
	result.Diagnostics = append(result.Diagnostics, routeDiagnostics(input.Route)...)
	addRouteCapabilities(&result)
	profileDiagnostics(&result, input.Profile)

	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix:         "pi",
		InstructionMessage: "Pi context and instruction delivery precedence are unverified",
		SkillMessage:       "Pi skill discovery and delivery precedence are unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "pi.resource.") {
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
		{"pi.config.acceptance_unverified", "target.config.acceptance", "config.acceptance", "JSON settings candidate syntax is source-grounded but native parser acceptance is unverified"},
		{"pi.config.inspector_unsafe", "target.config-inspection", "config.inspection", "Pi startup and config/model inspection load target state, project resources, and extensions; no safe isolated execution was accepted. No safe automated remedy is known without first bounding those effects"},
		{"pi.config.effective_state_unverified", "target.config.effective-state", "config.effective-state", "no safe command emits merged effective settings with per-field provenance"},
		{"pi.config.precedence_unverified", "target.config.precedence", "config.precedence", "documented settings precedence was not runtime-observed for this exact release"},
		{"pi.extensions.discovery_unverified", "target.extensions", "extensions", "extension and package discovery and execution were not observed in an isolated exact-release run"},
		{"pi.route.authentication_unverified", "route.authentication", "route.authentication", "authentication identity and credential handling are unverified; no safe automated remedy is known. Establish exact-release, credential-free route-identity evidence before applying output; credentials are never inferred"},
		{"pi.runtime.enforcement_unverified", "target.runtime.enforcement", "runtime.enforcement", "route, permission, tool, instruction, skill, extension, and policy enforcement was not observed"},
	}
	for _, blocker := range blockers {
		result.Diagnostics.Add(profilemango.SeverityError, blocker.code, blocker.path, blocker.message, 0, 0)
		result.AddCapability(blocker.field, StatusBlocking, blocker.message)
	}
	result.AddCapability("config.syntax", StatusPartial, "exact release settings keys are source-grounded candidate syntax only")
	result.AddCapability("config.fidelity", StatusPartial, "defaultProvider, defaultModel, and defaultThinkingLevel candidate fields only")
	result.AddCapability("delivery", StatusBlocking, "target-owned settings, context, and resource delivery is unverified")
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "exact Pi v0.87.1 source and package evidence is not qualified")
		return
	}
	result.AddCapability("target.artifact", StatusSupported, "immutable Pi npm package and bundled entrypoint hashes are pinned")
	result.AddCapability("target.version", StatusSupported, "exact Pi v0.87.1 source commit and package version are pinned")
}

func addRouteCapabilities(result *Result) {
	result.AddCapability("route.provider", StatusPartial, "defaultProvider candidate key is source-grounded only")
	result.AddCapability("route.model", StatusPartial, "defaultModel candidate key is source-grounded only")
	result.AddCapability("route.effort", StatusPartial, "defaultThinkingLevel candidate key is source-grounded only")
	result.AddCapability("route.transport", StatusBlocking, "Pi transport mapping is not release-qualified")
}

func profileDiagnostics(result *Result, profile profilemango.ResolvedProfile) {
	if profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "pi.security.permissions_unverified", "spec.permissions", "Pi has no built-in sandbox and permission equivalence is unverified", 0, 0)
		permissionCapabilities(result, profile.Permissions)
	}
	if profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "pi.tools.allowlist_unverified", "tools", "Pi closed tool and approval enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "pi.security.tools_unverified", "spec.tools", "Pi tool allowlist and deny-wins enforcement are unverified", 0, 0)
		result.AddCapability("tools", StatusBlocking, "closed tool enforcement is unverified")
	}
	if len(profile.Instructions) > 0 || len(profile.Skills) > 0 {
		result.Diagnostics.Add(profilemango.SeverityError, "pi.delivery.unverified", "resources", "Pi context, instruction, and skill delivery precedence is unverified", 0, 0)
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
		result.Diagnostics.Add(profilemango.SeverityError, "pi.permissions."+field.name+"_unverified", "permissions."+field.name, "Pi permission enforcement is unverified", 0, 0)
		result.AddCapability("permissions."+field.name, StatusBlocking, "target enforcement is unverified")
	}
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "pi.target.unsupported", "target", fmt.Sprintf("renderer only accepts target %q", TargetName), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "pi.target.version_required", "targetVersion", "an exact target version is required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "pi.target.version_unsupported", "targetVersion", fmt.Sprintf("only Pi %s is evidenced", TargetVersion), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "pi.version.unsupported", "targetVersion", fmt.Sprintf("only Pi %s is evidenced", TargetVersion), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "pi.version.evidence_required", "targetVersion", "the exact target evidence hash is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "pi.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned Pi package", 0, 0)
	}
	return diagnostics
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "pi.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	if route.Transport != "" && route.Transport != "native" {
		diagnostics.Add(profilemango.SeverityError, "pi.route.transport_unsupported", "route.transport", "Pi transport mapping is not qualified", 0, 0)
	}
	if route.Effort != "" && !validThinkingLevel(route.Effort) {
		diagnostics.Add(profilemango.SeverityError, "pi.route.effort_unsupported", "route.effort", "effort is not one of Pi's documented thinking levels", 0, 0)
	}
	return diagnostics
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding) Artifact {
	if !render.ValidName(profile.Metadata.Name) || route.Transport != "native" || route.Provider == "" || route.Authentication == "" || route.Model == "" || !validThinkingLevel(route.Effort) {
		return Artifact{}
	}
	content, err := json.MarshalIndent(struct {
		DefaultProvider      string `json:"defaultProvider"`
		DefaultModel         string `json:"defaultModel"`
		DefaultThinkingLevel string `json:"defaultThinkingLevel"`
	}{DefaultProvider: route.Provider, DefaultModel: route.Model, DefaultThinkingLevel: route.Effort}, "", "  ")
	if err != nil {
		return Artifact{}
	}
	content = append(content, '\n')
	return render.NewArtifact("preview/"+profile.Metadata.Name+".settings.json.preview", "candidate-config", content)
}

func validThinkingLevel(value string) bool {
	switch value {
	case "off", "minimal", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}
