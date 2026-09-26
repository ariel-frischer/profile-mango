package jcodefork

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

const (
	TargetName            = "jcode-fork"
	TargetVersion         = "0.83.909-dev (ca8017a3a)"
	BuildIdentity         = "jcode v0.83.909-dev (ca8017a3a)"
	BuildCommit           = "ca8017a3a"
	DocumentationSnapshot = "ed9b93b894454619f73ccddd24c2ff7a3c98ddc3"
	EvidenceSHA256        = "392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992"
	EvidenceSource        = "docs/dev/target-evidence.md"
	EvidenceLevel         = "isolated-synthetic-profile-resolution"
	AdapterVersion        = "profilemango.dev/jcode-fork/v1alpha1"
	ExperimentalLabel     = "Jcode fork, experimental-only"
	RenderAPIVersion      = render.APIVersion
	RenderKind            = render.Kind
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

// DefaultTarget returns the exact tested custom-fork build identity.
func DefaultTarget() TargetBuild {
	return TargetBuild{Name: TargetName, Version: TargetVersion, EvidenceSHA256: EvidenceSHA256}
}

// ResourceFromContent creates deterministic canonical resource metadata.
func ResourceFromContent(resourcePath, kind string, content []byte) Resource {
	return render.ResourceFromContent(resourcePath, kind, content)
}

// NewResult creates a blocked report with exact custom-fork evidence metadata.
func NewResult(profileName string, target TargetBuild) Result {
	result := render.NewResult(profileName, target, AdapterVersion, Evidence{
		Target: TargetName, Version: TargetVersion, SHA256: EvidenceSHA256,
		Source: EvidenceSource, Level: EvidenceLevel,
	})
	result.Diagnostics.Add(profilemango.SeverityWarning, "jcodefork.experimental_only", "target", experimentalMessage(), 0, 0)
	result.AddCapability("target.status", StatusBlocking, experimentalMessage())
	return result
}

// Render emits an inert TOML candidate and never claims target applicability.
func Render(input Input) Result {
	result := NewResult(input.Profile.Metadata.Name, input.Target)
	result.Diagnostics = append(result.Diagnostics, targetDiagnostics(input.Target)...)
	addTargetCapabilities(&result, input.Target)
	result.Diagnostics = append(result.Diagnostics, routeDiagnostics(input.Route)...)
	addRouteCapabilities(&result)
	addEvidenceBlockers(&result)
	profileDiagnostics(&result, input.Profile)

	resourceArtifacts, resourceDiagnostics := render.ResourceArtifacts(input.Profile, input.Resources, render.ResourceArtifactOptions{
		CodePrefix:         "jcodefork",
		InstructionMessage: "Custom Jcode fork instruction delivery and precedence are unverified",
		SkillMessage:       "Custom Jcode fork skill discovery and delivery precedence are unverified",
	})
	result.Diagnostics = append(result.Diagnostics, resourceDiagnostics...)
	if !render.HasCodePrefix(resourceDiagnostics, "jcodefork.resource.") {
		candidate, candidateDiagnostics := candidateArtifact(input.Profile, input.Route, input.Resources)
		result.Diagnostics = append(result.Diagnostics, candidateDiagnostics...)
		if !candidateDiagnostics.HasErrors() {
			result.Artifacts = append(result.Artifacts, candidate)
			result.Artifacts = append(result.Artifacts, resourceArtifacts...)
		}
	}
	result.Artifacts = render.SortedArtifacts(result.Artifacts)
	result.Diagnostics = result.Diagnostics.Sorted()
	result.Applicable = false
	return result
}

func experimentalMessage() string {
	return ExperimentalLabel + " renderer only; not upstream Jcode or a supported public target"
}

func addTargetCapabilities(result *Result, target TargetBuild) {
	if target.Name != TargetName || target.Version != TargetVersion || target.EvidenceSHA256 != EvidenceSHA256 {
		result.AddCapability("target.version", StatusBlocking, "the exact tested custom-fork build, commit, and SHA-256 are not qualified")
		return
	}
	result.AddCapability("target.version", StatusSupported, "exact custom-fork build, commit, and SHA-256 are pinned")
}

func addRouteCapabilities(result *Result) {
	for _, field := range []string{"provider", "model", "effort"} {
		result.AddCapability("route."+field, StatusPartial, "exact-build synthetic profile resolution observed candidate fields only")
	}
	result.AddCapability("route.authentication", StatusBlocking, "authentication identity and credential handling are unverified")
	result.AddCapability("route.transport", StatusBlocking, "custom-fork transport mapping is not qualified")
}

func addEvidenceBlockers(result *Result) {
	blockers := []struct {
		code, field, message string
	}{
		{"jcodefork.config.acceptance_unverified", "config.acceptance", "the retained exact-build probe observed synthetic profile fields, not this renderer's generated profile combination"},
		{"jcodefork.config.discovery_unverified", "config.discovery", "target discovery and repository context resolution are unverified"},
		{"jcodefork.config.precedence_unverified", "config.precedence", "full profile, project, environment, flag, and session precedence is unverified"},
		{"jcodefork.child.overrides_unverified", "child.overrides", "child and swarm override boundaries are unverified"},
		{"jcodefork.hooks.extensions.mcp_unverified", "hooks.extensions.mcp", "hooks, extensions, and MCP effects are unverified"},
		{"jcodefork.runtime.enforcement_unverified", "runtime.enforcement", "runtime route, tool, skill, instruction, and policy enforcement is unverified"},
	}
	for _, blocker := range blockers {
		result.Diagnostics.Add(profilemango.SeverityError, blocker.code, blocker.field, ExperimentalLabel+": "+blocker.message, 0, 0)
		result.AddCapability(blocker.field, StatusBlocking, blocker.message)
	}
}

func profileDiagnostics(result *Result, profile profilemango.ResolvedProfile) {
	if !render.ValidName(profile.Metadata.Name) {
		result.Diagnostics.Add(profilemango.SeverityError, "jcodefork.profile.name_invalid", "metadata.name", "profile name must be a simple canonical name", 0, 0)
	}
	if profile.Permissions != nil {
		permissionDiagnostics(result, profile.Permissions)
	}
	if profile.Tools != nil {
		if !profile.Tools.Managed || !profile.Tools.Closed {
			result.Diagnostics.Add(profilemango.SeverityError, "jcodefork.tools.policy_unsupported", "spec.tools", "only a canonical closed tool allowlist can be projected to observed custom-fork selectors", 0, 0)
		}
		validateSelectors(result, "spec.tools.allow", profile.Tools.Allow)
		validateSelectors(result, "spec.tools.deny", profile.Tools.Deny)
		result.Diagnostics.Add(profilemango.SeverityError, "jcodefork.tools.delivery_unverified", "tools", "custom-fork tool selectors are candidate syntax only; discovery, enforcement, and child overrides are unverified", 0, 0)
		result.AddCapability("tools", StatusPartial, "observed allow and disabled selectors are candidate syntax only; enforcement and child overrides are unverified")
	}
	if len(profile.Skills) > 0 {
		result.AddCapability("skills.selectors", StatusPartial, "canonical resource paths are emitted as candidate selectors; discovery and enforcement are unverified")
	}
	for _, item := range append(append([]string{}, profile.Instructions...), profile.Skills...) {
		if !render.SafePath(item) {
			result.Diagnostics.Add(profilemango.SeverityError, "jcodefork.resource.path_unsupported", item, "canonical resource references must be relative and cannot escape the resource root", 0, 0)
		}
	}
	if len(profile.Instructions) > 0 {
		result.AddCapability("instructions.metadata", StatusPartial, "only presence and character-count metadata is projected; content delivery and precedence are unverified")
	}
}

func permissionDiagnostics(result *Result, permissions *profilemango.PermissionPolicy) {
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "mode", value: permissions.Mode},
		{name: "network", value: permissions.Network},
		{name: "shell", value: permissions.Shell},
	} {
		if field.value == nil || *field.value == "unmanaged" {
			continue
		}
		result.Diagnostics.Add(profilemango.SeverityError, "jcodefork.permissions."+field.name+"_unverified", "spec.permissions."+field.name, "custom-fork permission enforcement is unverified", 0, 0)
		result.AddCapability("permissions."+field.name, StatusBlocking, "custom-fork permission enforcement is not verified")
	}
}

func validateSelectors(result *Result, field string, selectors []string) {
	result.Diagnostics = append(result.Diagnostics, selectorDiagnostics(field, selectors)...)
}

func selectorDiagnostics(field string, selectors []string) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	seen := make(map[string]struct{}, len(selectors))
	for _, selector := range selectors {
		if selector == "" || !utf8.ValidString(selector) || strings.ContainsAny(selector, "\r\n\x00") {
			diagnostics.Add(profilemango.SeverityError, "jcodefork.selector.invalid", field, "target selector must be non-empty, valid UTF-8, and single-line", 0, 0)
			continue
		}
		if _, duplicate := seen[selector]; duplicate {
			diagnostics.Add(profilemango.SeverityError, "jcodefork.selector.duplicate", field, "target selector is duplicated", 0, 0)
			continue
		}
		seen[selector] = struct{}{}
	}
	return diagnostics
}

func targetDiagnostics(target TargetBuild) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if target.Name != TargetName {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.target.unsupported", "target", fmt.Sprintf("renderer only accepts the %q target for the %s", TargetName, ExperimentalLabel), 0, 0)
	}
	if target.Version == "" {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.target.version_required", "targetVersion", "an exact tested custom-fork version and commit are required", 0, 0)
	} else if target.Version != TargetVersion {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.target.version_unsupported", "targetVersion", fmt.Sprintf("only %s is evidenced", BuildIdentity), 0, 0)
		diagnostics.Add(profilemango.SeverityError, "jcodefork.version.unsupported", "targetVersion", fmt.Sprintf("only %s is evidenced", BuildIdentity), 0, 0)
	}
	if target.EvidenceSHA256 == "" {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.version.evidence_required", "targetVersion", "the exact tested custom-fork SHA-256 is required", 0, 0)
	} else if target.EvidenceSHA256 != EvidenceSHA256 {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.version.evidence_mismatch", "targetVersion", "target evidence hash does not match the pinned custom-fork build", 0, 0)
	}
	return diagnostics
}

func routeDiagnostics(route profilemango.RouteBinding) profilemango.Diagnostics {
	var diagnostics profilemango.Diagnostics
	if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.route.incomplete", "route", "provider, transport, authentication, model, and effort are required", 0, 0)
	}
	if route.Transport != "" && route.Transport != "native" {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.route.transport_unsupported", "route.transport", "only the native transport has a candidate route mapping", 0, 0)
	}
	if route.Authentication != "" {
		diagnostics.Add(profilemango.SeverityError, "jcodefork.route.authentication_unverified", "route.authentication", "authentication identity is not verified; no safe automated remedy is known. Establish exact-build, credential-free route-identity evidence before applying output; route authentication is never emitted", 0, 0)
	}
	return diagnostics
}

type candidateFields struct {
	toolProfile string
	tools       []string
	disabled    []string
	skills      []string
	skillsNone  bool
}

func candidateArtifact(profile profilemango.ResolvedProfile, route profilemango.RouteBinding, resources []Resource) (Artifact, profilemango.Diagnostics) {
	var diagnostics profilemango.Diagnostics
	if !render.ValidName(profile.Metadata.Name) || route.Provider == "" || route.Model == "" || route.Effort == "" {
		return Artifact{}, diagnostics
	}
	fields, fieldDiagnostics := candidateFieldsFor(profile)
	diagnostics = append(diagnostics, fieldDiagnostics...)
	if diagnostics.HasErrors() {
		return Artifact{}, diagnostics
	}
	var builder strings.Builder
	builder.WriteString("# profile-mango: INERT PREVIEW ONLY\n")
	builder.WriteString("# TARGET: Custom Jcode fork (experimental-only)\n")
	builder.WriteString("# This is a candidate profile, not an active target configuration.\n")
	builder.WriteString("# Route identity, discovery, delivery, precedence, child overrides, hooks, extensions, MCP, and enforcement remain unverified.\n\n")
	fmt.Fprintf(&builder, "[profiles.%s]\n", profile.Metadata.Name)
	fmt.Fprintf(&builder, "provider = %s\n", tomlString(route.Provider))
	fmt.Fprintf(&builder, "model = %s\n", tomlString(route.Model))
	fmt.Fprintf(&builder, "reasoning_effort = %s\n", tomlString(route.Effort))
	if fields.toolProfile != "" {
		fmt.Fprintf(&builder, "tool_profile = %s\n", tomlString(fields.toolProfile))
	}
	if len(fields.tools) > 0 {
		fmt.Fprintf(&builder, "tools = %s\n", tomlArray(fields.tools))
	}
	if len(fields.disabled) > 0 {
		fmt.Fprintf(&builder, "disabled_tools = %s\n", tomlArray(fields.disabled))
	}
	if fields.skillsNone {
		builder.WriteString("skills_mode = \"none\"\n")
	}
	if len(fields.skills) > 0 {
		fmt.Fprintf(&builder, "skills = %s\n", tomlArray(fields.skills))
		builder.WriteString("# Non-empty canonical skills are emitted without target mode or exclusion projections.\n")
	}
	if len(profile.Instructions) > 0 {
		present, chars := instructionMetadata(profile, resources)
		fmt.Fprintf(&builder, "# observed instruction metadata: instructions_present = %t, instructions_chars = %d\n", present, chars)
		builder.WriteString("# instruction content is delivered only as inert resource bytes; target precedence is unverified.\n")
	}
	return render.NewArtifact("preview/"+profile.Metadata.Name+".config.toml.preview", "candidate-config", []byte(builder.String())), diagnostics
}

func candidateFieldsFor(profile profilemango.ResolvedProfile) (candidateFields, profilemango.Diagnostics) {
	var diagnostics profilemango.Diagnostics
	fields := candidateFields{}
	if profile.Tools != nil {
		if !profile.Tools.Managed || !profile.Tools.Closed {
			diagnostics.Add(profilemango.SeverityError, "jcodefork.tools.policy_unsupported", "spec.tools", "only a canonical closed tool allowlist can be projected to observed custom-fork selectors", 0, 0)
		} else {
			fields.toolProfile = "none"
			fields.tools = sortedCopy(profile.Tools.Allow)
			fields.disabled = sortedCopy(profile.Tools.Deny)
		}
		diagnostics = append(diagnostics, selectorDiagnostics("spec.tools.allow", profile.Tools.Allow)...)
		diagnostics = append(diagnostics, selectorDiagnostics("spec.tools.deny", profile.Tools.Deny)...)
	}
	fields.skills = sortedCopy(profile.Skills)
	fields.skillsNone = len(fields.skills) == 0 && profile.Skills != nil
	return fields, diagnostics
}

func instructionMetadata(profile profilemango.ResolvedProfile, resources []Resource) (bool, int) {
	requested := make(map[string]struct{}, len(profile.Instructions))
	for _, item := range profile.Instructions {
		requested[item] = struct{}{}
	}
	chars := 0
	for _, resource := range resources {
		if resource.Digest.Kind != "instruction" {
			continue
		}
		if _, ok := requested[resource.Digest.Path]; !ok {
			continue
		}
		chars += utf8.RuneCount(resource.Content)
	}
	return true, chars
}

func sortedCopy(values []string) []string {
	copyOfValues := append([]string(nil), values...)
	sort.Strings(copyOfValues)
	return copyOfValues
}

func tomlArray(values []string) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = tomlString(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
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
