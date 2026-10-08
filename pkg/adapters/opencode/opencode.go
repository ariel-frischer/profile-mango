package opencode

import (
	"fmt"
	"strconv"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName                = "opencode"
	TargetVersion             = "1.18.31"
	OpenCodeVersion           = TargetVersion
	ReleaseTag                = "v1.18.31"
	SourceCommit              = "a97622c801f4ca571530ddc51076af659a9c32cd"
	SourceArchiveSHA256       = "76f69fe27ec2b44e23fa1749029e7c012eb7e975a0f0c7819e9458198dfd3896"
	LinuxX64AssetSHA256       = "e9312be75ed803b7415fc2aeabda1f4fe938912a39673762dc0c38c0e11ebde4"
	LinuxX64BinarySHA256      = "f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11"
	ReleaseAssetSHA256        = LinuxX64AssetSHA256
	ConfigSourceSHA256        = "87a9071af1ddb04d65947dba49be3fb4c94ecf63e120f17ce46ee81ff25dd45a"
	CoreConfigSourceSHA256    = "b99bcbd98df6da79e59cda482363f759cea9d4b9792c6c8e83b6a8d686138d30"
	ProviderSourceSHA256      = "c496dea619e8e9d2d09b7a2353f2e62bff6a2276471c161c3988059a1b6ac303"
	MutableConfigSchemaSHA256 = "e8cb6e287a3852ee3403f4803be5ad6b19db94948037eaa9672125c333427922"
	EvidenceSHA256            = SourceArchiveSHA256
	OpenCodeEvidenceSHA256    = EvidenceSHA256
	EvidenceSource            = "docs/dev/target-evidence.md"
	EvidenceLevel             = "isolated-native-config-acceptance"
	AdapterVersion            = "profilemango.dev/opencode/v1alpha1"
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

// DefaultTarget returns the exact source and release-artifact-qualified observation.
func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

// ResourceFromContent creates deterministic canonical resource metadata.
func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	return render.ResourceFromContent(resourcePath, kind, content)
}

// NewResult creates a blocked report with immutable OpenCode evidence metadata.
func NewResult(profileName string, target TargetBuild) Result {
	return render.NewResult(profileName, target, AdapterVersion, Evidence{
		Target:  TargetName,
		Version: TargetVersion,
		SHA256:  EvidenceSHA256,
		Source:  EvidenceSource,
		Level:   EvidenceLevel,
	})
}

// Render emits a deterministic inert JSONC candidate and never claims applicability.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	result.Diagnostics = append(result.Diagnostics, targetDiagnostics(input.Target)...)
	addTargetCapabilities(&result, input.Target)
	addEvidenceBlockers(&result)
	result.Diagnostics = append(result.Diagnostics, routeDiagnostics(input.Route)...)
	addRouteCapabilities(&result)
	profileDiagnostics(&result, input.Profile)

	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix:         "opencode",
		InstructionMessage: "OpenCode instruction delivery and precedence are unverified",
		SkillMessage:       "OpenCode skill delivery and precedence are unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "opencode.resource.") {
		result.Artifacts = append(result.Artifacts, candidateArtifact(input.Profile, input.Route))
		result.Artifacts = append(result.Artifacts, resourceArtifacts...)
	}
	result.Artifacts = render.SortedArtifacts(result.Artifacts)
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Applicable = false
	return result
}

func addEvidenceBlockers(result *Result) {
	addNativeValidationCapabilities(result)
	blockers := []struct {
		code, path, field, message string
	}{
		{"opencode.extensions.plugins_mcp_unverified", "target.extensions.plugins-mcp", "extensions.plugins-mcp", "plugin and MCP discovery, configuration, and enforcement were not observed"},
		{"opencode.runtime.enforcement_unverified", "target.runtime.enforcement", "runtime.enforcement", "route, permission, tool, instruction, skill, plugin, MCP, and policy enforcement was not observed; no safe automated remedy is known"},
	}
	for _, blocker := range blockers {
		result.Diagnostics.Add(profilemango.SeverityError, blocker.code, blocker.path, blocker.message, 0, 0)
		result.AddCapability(blocker.field, StatusBlocking, blocker.message)
	}
	result.AddCapability("config.fidelity", StatusPartial, "the exact model candidate and one local skill path are natively accepted; other portable fields remain unverified")
	result.AddCapability("install", StatusPartial, "only the exact top-level model field and one validated local skill resource have narrow adapters; full target installation remains blocked")
	result.AddCapability("install.model", StatusSupported, "the exact top-level model field can be losslessly patched at one explicit path with transactional safeguards")
	result.AddCapability("install.skills", StatusSupported, "one validated local SKILL.md resource can be installed beside the explicit config and exposed through skills.paths")
	result.Diagnostics.Add(profilemango.SeverityWarning, "opencode.install.narrow", "target.install", "only the exact top-level model field and one validated local skill resource are installable; other profile and target fields remain blocked or unmanaged", 0, 0)
	result.AddCapability("delivery", StatusBlocking, "target-owned instruction or skill delivery is unverified")
	result.AddCapability("permissions", StatusBlocking, "permission mapping and runtime enforcement are unverified")
	result.AddCapability("tools", StatusBlocking, "tool, plugin, and MCP enforcement is unverified")
}

func addNativeValidationCapabilities(result *Result) {
	result.AddCapability("config.acceptance", StatusSupported, "OpenCode 1.18.31 natively accepted the deterministic JSONC model candidate in an isolated exact-binary probe")
	result.AddCapability("config.inspection", StatusPartial, "debug config emitted merged configuration but created target-owned scratch state and is not a zero-write inspector")
	result.AddCapability("config.effective-state", StatusPartial, "merged model output was observed without general per-field provenance")
	result.AddCapability("config.precedence", StatusPartial, "inline config precedence over global state and explicit custom-file consumption were observed; project and managed precedence remain incomplete")
	result.Diagnostics.Add(profilemango.SeverityWarning, "opencode.config.inspector_writeful", "target.config-inspection", "debug config is usable only in disposable isolated state because it creates directories, logs, locks, metadata, and gitignore files", 0, 0)
	result.Diagnostics.Add(profilemango.SeverityWarning, "opencode.config.effective_state_partial", "target.config.effective-state", "merged configuration was observed without general per-field provenance", 0, 0)
	result.Diagnostics.Add(profilemango.SeverityWarning, "opencode.config.precedence_partial", "target.config.precedence", "inline-over-global precedence and explicit custom-file consumption were observed; other layers remain unverified", 0, 0)
	result.Diagnostics.Add(profilemango.SeverityWarning, "opencode.config.unknown_keys_ignored", "target.config.acceptance", "unknown configuration keys were accepted and omitted from resolved output", 0, 0)
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "exact OpenCode v1.18.31 source and release evidence is not qualified")
		return
	}
	result.AddCapability("target.artifact", StatusSupported, "immutable OpenCode source archive, Linux x64 release archive, and extracted binary hashes are pinned")
	result.AddCapability("target.version", StatusSupported, "exact OpenCode v1.18.31 source commit, release tag, and isolated binary identity are pinned")
}

func addRouteCapabilities(result *Result) {
	result.AddCapability("route.authentication", StatusBlocking, "authentication identity and credential handling are unverified")
	result.AddCapability("route.effort", StatusBlocking, "effort is not emitted and target effort mapping is unverified")
	result.AddCapability("route.model", StatusPartial, "model is emitted only as a provider/model inert candidate string")
	result.AddCapability("route.provider", StatusPartial, "provider is represented only within the provider/model inert candidate string")
	result.AddCapability("route.transport", StatusBlocking, "transport mapping is not release-qualified")
}

func profileDiagnostics(result *Result, profile profilemango.ResolvedProfile) {
	if profile.Permissions != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "opencode.security.permissions_unverified", "spec.permissions", "OpenCode permission equivalence and runtime enforcement are unverified", 0, 0)
		permissionCapabilities(result, profile.Permissions)
	}
	if profile.Tools != nil {
		result.Diagnostics.Add(profilemango.SeverityError, "opencode.tools.allowlist_unverified", "tools", "OpenCode closed tool and permission enforcement is unverified", 0, 0)
		result.Diagnostics.Add(profilemango.SeverityError, "opencode.security.tools_unverified", "spec.tools", "OpenCode tool, plugin, MCP, and deny-wins enforcement is unverified", 0, 0)
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
		result.Diagnostics.Add(profilemango.SeverityError, "opencode.permissions."+field.name+"_unverified", "permissions."+field.name, "OpenCode permission enforcement is unverified", 0, 0)
		result.AddCapability("permissions."+field.name, StatusBlocking, "target enforcement is unverified")
	}
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "opencode.target.unsupported", "target", fmt.Sprintf("renderer only accepts target %q", TargetName), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "opencode.target.version_required", "targetVersion", "an exact target version is required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "opencode.target.version_unsupported", "targetVersion", fmt.Sprintf("only OpenCode %s is evidenced", TargetVersion), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "opencode.version.unsupported", "targetVersion", fmt.Sprintf("only OpenCode %s is evidenced", TargetVersion), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "opencode.version.evidence_required", "targetVersion", "the exact target evidence hash is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "opencode.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned OpenCode source archive", 0, 0)
	}
	return diagnostics
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	diagnostics.Add(profilemango.SeverityError, "opencode.route.authentication_unverified", "route.authentication", "authentication identity and credential handling are unverified; credentials are never inferred or emitted", 0, 0)
	diagnostics.Add(profilemango.SeverityError, "opencode.route.effort_unverified", "route.effort", "effort is not emitted and target effort mapping is unverified", 0, 0)
	diagnostics.Add(profilemango.SeverityError, "opencode.route.transport_unverified", "route.transport", "transport mapping is not release-qualified", 0, 0)
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "opencode.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	if route.Transport != "" && route.Transport != "native" {
		diagnostics.Add(profilemango.SeverityError, "opencode.route.transport_unsupported", "route.transport", "only the native transport has a candidate route mapping", 0, 0)
	}
	if route.Effort != "" && !validEffort(route.Effort) {
		diagnostics.Add(profilemango.SeverityError, "opencode.route.effort_unsupported", "route.effort", "effort is not a supported canonical reasoning level", 0, 0)
	}
	return diagnostics
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding) Artifact {
	if !render.ValidName(profile.Metadata.Name) || route.Transport != "native" || route.Provider == "" || route.Authentication == "" || route.Model == "" || !validEffort(route.Effort) {
		return Artifact{}
	}
	model := strconv.Quote(route.Provider + "/" + route.Model)
	content := []byte(fmt.Sprintf("// profile-mango: INERT PREVIEW ONLY\n// NON-APPLICABLE: candidate syntax for OpenCode %s.\n// This is not an active OpenCode config.json or config.jsonc. Authentication, provider options, effort, delivery, precedence, plugins, MCP, and enforcement are unverified.\n\n{\n  \"model\": %s,\n}\n", TargetVersion, model))
	return render.NewArtifact("preview/"+profile.Metadata.Name+".opencode.jsonc.preview", "candidate-config", content)
}

func validEffort(value string) bool {
	switch value {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra":
		return true
	default:
		return false
	}
}
