package claudecode

import (
	"encoding/json"
	"fmt"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName            = "claude-code"
	TargetVersion         = "2.1.278"
	ClaudeCodeVersion     = TargetVersion
	ReleaseTag            = "v2.1.278"
	AdapterVersion        = "profilemango.dev/claudecode/v1alpha1"
	RenderAPIVersion      = render.APIVersion
	RenderKind            = render.Kind
	NativePackageSHA256   = "d1fb51ab0a0234d1bd7f418ee9d6b6b124c2412b2ddaf3dfc3256bad8063f1c7"
	EvidenceSHA256        = NativePackageSHA256
	ClaudeCodeEvidenceSHA = EvidenceSHA256
	EvidenceSource        = "docs/dev/target-evidence.md"
	EvidenceLevel         = "immutable-package-review"
	ReleaseCommit         = "bf7d404e26a5fb6167d21b46c93a2bf6c22ab274"
	WrapperPackageSHA256  = "08c6dfcf3dafcfd30e09b2926c596e274f0fa20844a5801ada7f1c8e6227157e"
	NativeBinarySHA256    = "5c4735937844e84f8a93306e841a5b0e12252909b07870f789b190468da147ab"
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

func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	return render.ResourceFromContent(resourcePath, kind, content)
}

func NewResult(profileName string, target TargetBuild) Result {
	return render.NewResult(profileName, target, AdapterVersion, Evidence{
		Target:  TargetName,
		Version: TargetVersion,
		SHA256:  EvidenceSHA256,
		Source:  EvidenceSource,
		Level:   EvidenceLevel,
	})
}

func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	result.Diagnostics = append(result.Diagnostics, targetDiagnostics(input.Target)...)
	addTargetCapabilities(&result, input.Target)
	addEvidenceBlockers(&result)
	result.Diagnostics = append(result.Diagnostics, routeDiagnostics(input.Route)...)
	addRouteCapabilities(&result)
	profileDiagnostics(&result, input.Profile)
	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix:         "claudecode",
		InstructionMessage: "CLAUDE.md delivery and hierarchy are documentation-context only and unverified",
		SkillMessage:       "Claude Code skill discovery and precedence are documentation-context only and unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "claudecode.resource.") {
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
		{"claudecode.config.acceptance_unverified", "target.config.acceptance", "config.acceptance", "settings JSON candidate syntax is documentation-context only; native parser acceptance is unverified"},
		{"claudecode.config.inspector_unsafe", "target.config-inspection", "config.inspection", "no release-qualified safe inspector was established; doctor and status effects remain unverified"},
		{"claudecode.config.effective_state_unverified", "target.config.effective-state", "config.effective-state", "no safe command emits merged effective state with per-field provenance"},
		{"claudecode.config.precedence_unverified", "target.config.precedence", "config.precedence", "settings precedence is documentation-context only and was not observed for this release"},
		{"claudecode.extensions.plugins_hooks_mcp_unverified", "target.extensions.plugins-hooks-mcp", "extensions.plugins-hooks-mcp", "plugin, hook, and MCP discovery and enforcement were not observed"},
		{"claudecode.runtime.enforcement_unverified", "target.runtime.enforcement", "runtime.enforcement", "route, permission, tool, instruction, skill, plugin, hook, and MCP enforcement was not observed"},
	}
	for _, blocker := range blockers {
		result.Diagnostics.Add(profilemango.SeverityError, blocker.code, blocker.path, blocker.message, 0, 0)
		result.AddCapability(blocker.field, StatusBlocking, blocker.message)
	}
	result.AddCapability("config.fidelity", StatusPartial, "the model field is emitted as a documentation-context inert candidate only")
	result.AddCapability("delivery", StatusBlocking, "target-owned settings, CLAUDE.md, and skill delivery is unverified")
	result.AddCapability("permissions", StatusBlocking, "permission modes and their runtime enforcement are unverified")
	result.AddCapability("tools", StatusBlocking, "tool, plugin, hook, and MCP enforcement is unverified")
	result.AddCapability("instructions.claude-md", StatusBlocking, "CLAUDE.md hierarchy and delivery are unverified")
	result.AddCapability("skills", StatusBlocking, "skill discovery, precedence, and execution are unverified")
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "exact Claude Code package and release evidence is not qualified")
		return
	}
	result.AddCapability("target.artifact", StatusSupported, "immutable npm wrapper and linux-x64 native package hashes are pinned")
	result.AddCapability("target.version", StatusSupported, "exact Claude Code v2.1.278 release context is pinned")
}

func addRouteCapabilities(result *Result) {
	result.AddCapability("route.authentication", StatusBlocking, "authentication identity and credential handling are unverified")
	result.AddCapability("route.effort", StatusBlocking, "effort mapping is not release-qualified")
	result.AddCapability("route.provider", StatusBlocking, "provider and cloud route mapping is not release-qualified")
	result.AddCapability("route.transport", StatusBlocking, "transport mapping is not release-qualified")
	result.AddCapability("route.model", StatusPartial, "model is echoed in a documentation-context candidate only")
}

func profileDiagnostics(result *Result, profile profilemango.ResolvedProfile) {
	if profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "claudecode.security.permissions_unverified", "spec.permissions", "Claude Code permission equivalence and runtime enforcement are unverified", 0, 0)
		permissionCapabilities(result, profile.Permissions)
	}
	if profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "claudecode.tools.allowlist_unverified", "tools", "Claude Code tool and plugin allowlist enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "claudecode.security.tools_unverified", "spec.tools", "Claude Code tool, hook, MCP, and deny-wins enforcement is unverified", 0, 0)
		result.AddCapability("tools", StatusBlocking, "tool, plugin, hook, and MCP enforcement is unverified")
	}
	if len(profile.Instructions) > 0 {
		result.AddCapability("instructions.claude-md", StatusBlocking, "CLAUDE.md hierarchy and delivery are unverified")
	}
	if len(profile.Skills) > 0 {
		result.AddCapability("skills", StatusBlocking, "skill discovery, precedence, and execution are unverified")
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
		result.Diagnostics.Add(profilemango.SeverityError, "claudecode.permissions."+field.name+"_unverified", "permissions."+field.name, "Claude Code enforcement is unverified", 0, 0)
		result.AddCapability("permissions."+field.name, StatusBlocking, "target enforcement is unverified")
	}
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "claudecode.target.unsupported", "target", fmt.Sprintf("renderer only accepts target %q", TargetName), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "claudecode.target.version_required", "targetVersion", "an exact target version is required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "claudecode.target.version_unsupported", "targetVersion", fmt.Sprintf("only Claude Code %s is evidenced", TargetVersion), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "claudecode.version.unsupported", "targetVersion", fmt.Sprintf("only Claude Code %s is evidenced", TargetVersion), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "claudecode.version.evidence_required", "targetVersion", "the exact target evidence hash is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "claudecode.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned linux-x64 native package", 0, 0)
	}
	return diagnostics
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	diagnostics.Add(profilemango.SeverityError, "claudecode.route.authentication_unverified", "route.authentication", "authentication identity and credential handling are unverified; no safe automated remedy is known. Establish exact-release, credential-free route-identity evidence before applying output; credentials are never inferred", 0, 0)
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "claudecode.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	if route.Transport != "" && route.Transport != "native" {
		diagnostics.Add(profilemango.SeverityError, "claudecode.route.transport_unsupported", "route.transport", "Claude Code transport mapping is not qualified", 0, 0)
	}
	if route.Effort != "" && !validEffort(route.Effort) {
		diagnostics.Add(profilemango.SeverityError, "claudecode.route.effort_unsupported", "route.effort", "effort is not a supported canonical reasoning level", 0, 0)
	}
	return diagnostics
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding) Artifact {
	if !render.ValidName(profile.Metadata.Name) || route.Model == "" || !validEffort(route.Effort) {
		return Artifact{}
	}
	content, err := json.MarshalIndent(struct {
		Model string `json:"model"`
	}{Model: route.Model}, "", "  ")
	if err != nil {
		return Artifact{}
	}
	content = append(content, '\n')
	return render.NewArtifact("preview/"+profile.Metadata.Name+".settings.json.preview", "candidate-config", content)
}

func validEffort(value string) bool {
	switch value {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra":
		return true
	default:
		return false
	}
}
