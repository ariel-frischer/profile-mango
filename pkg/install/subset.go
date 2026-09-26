package install

import (
	"fmt"
	"slices"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
)

// Known profile requirements an adapter may be unable to install.
const (
	RequirementPermissions  = "permissions"
	RequirementTools        = "tools"
	RequirementInstructions = "instructions"
	RequirementSkills       = "skills"
	// RequirementRoles is the bound route's per-role routes (routes.<name>.roles).
	RequirementRoles = "roles"
	// RequirementRoleDefinitions is the profile's portable role definitions (roles).
	RequirementRoleDefinitions = "role-definitions"
	// RequirementSubagentMaxEffort is the bound route's subagentMaxEffort cap.
	RequirementSubagentMaxEffort = "subagentMaxEffort"
	// RequirementEffort is the bound route's effort when the target cannot write it.
	RequirementEffort = "effort"
)

// Skip reasons for route and role requirements a target cannot install.
const (
	rolesUnsupportedReason             = "per-role routes are only installed for Oh My Pi modelRoles; this target installs the default route only"
	roleDefinitionsUnsupportedReason   = "no qualified target surface installs role descriptions or instructions; Oh My Pi model slots take only a model selector"
	subagentMaxEffortUnsupportedReason = "the subagent effort cap is only installed for Oh My Pi task.maxEffort"
)

// SkippedRequirement is a known profile requirement a target cannot honor. A
// non-strict plan lists it and installs the rest instead of blocking. Value is
// the route value that was not applied (effort); Reason is set when the
// requirement name alone does not explain the skip.
type SkippedRequirement struct {
	Requirement string `json:"requirement"`
	Count       int    `json:"count,omitempty"`
	Value       string `json:"value,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// effortSupporter lets an adapter report which route efforts it writes. An adapter
// without it installs every effort it accepts or rejects the route itself.
type effortSupporter interface {
	EffortSupport(effort string) (supported bool, reason string)
}

// requirementSupporter lets an adapter declare the profile requirements it installs.
// setsDefault reports the request's --default flag, since some adapters (OpenCode) only
// support an additional requirement alongside the main-config write --default triggers.
type requirementSupporter interface {
	SupportedRequirements(agent AgentDestination, setsDefault bool) []string
}

func supportsRequirement(adapter Adapter, agent AgentDestination, setsDefault bool, name string) bool {
	supporter, ok := adapter.(requirementSupporter)
	return ok && slices.Contains(supporter.SupportedRequirements(agent, setsDefault), name)
}

// supportedSubset strips the requirements adapter cannot install from the
// profile and its resources for the given destination, and returns what it stripped in a fixed order.
func supportedSubset(adapter Adapter, agent AgentDestination, setsDefault bool, loaded loadedInput) (profilemango.ResolvedProfile, []render.Resource, []SkippedRequirement) {
	profile, resources := loaded.Profile, loaded.Resources
	var skipped []SkippedRequirement
	if profile.Permissions != nil && !supportsRequirement(adapter, agent, setsDefault, RequirementPermissions) {
		profile.Permissions = nil
		skipped = append(skipped, SkippedRequirement{Requirement: RequirementPermissions})
	}
	if profile.Tools != nil && !supportsRequirement(adapter, agent, setsDefault, RequirementTools) {
		profile.Tools = nil
		skipped = append(skipped, SkippedRequirement{Requirement: RequirementTools})
	}
	if count := len(profile.Instructions); count > 0 && !supportsRequirement(adapter, agent, setsDefault, RequirementInstructions) {
		profile.Instructions = nil
		resources = withoutResourceKind(resources, "instruction")
		skipped = append(skipped, SkippedRequirement{Requirement: RequirementInstructions, Count: count})
	}
	if count := len(profile.Skills); count > 0 && !supportsRequirement(adapter, agent, setsDefault, RequirementSkills) {
		profile.Skills = nil
		resources = withoutResourceKind(resources, "skill")
		skipped = append(skipped, SkippedRequirement{Requirement: RequirementSkills, Count: count})
	}
	return profile, resources, skipped
}

// routeSubset strips the per-role routes and subagent effort cap a target cannot
// install. It reports them as skipped, or as a strict blocking reason when strict is set.
func routeSubset(adapter Adapter, agent AgentDestination, setsDefault, strict bool, route profilemango.RouteBinding) (profilemango.RouteBinding, []SkippedRequirement, string) {
	var skipped []SkippedRequirement
	if count := len(route.Roles); count > 0 && !supportsRequirement(adapter, agent, setsDefault, RequirementRoles) {
		if strict {
			return route, nil, fmt.Sprintf("--strict: %d route roles cannot be installed: %s", count, rolesUnsupportedReason)
		}
		route.Roles = nil
		skipped = append(skipped, SkippedRequirement{Requirement: RequirementRoles, Count: count, Reason: rolesUnsupportedReason})
	}
	if effort := route.SubagentMaxEffort; effort != "" && !supportsRequirement(adapter, agent, setsDefault, RequirementSubagentMaxEffort) {
		if strict {
			return route, nil, fmt.Sprintf("--strict: subagentMaxEffort %s cannot be installed: %s", effort, subagentMaxEffortUnsupportedReason)
		}
		route.SubagentMaxEffort = ""
		skipped = append(skipped, SkippedRequirement{Requirement: RequirementSubagentMaxEffort, Value: effort, Reason: subagentMaxEffortUnsupportedReason})
	}
	return route, skipped, ""
}

// roleDefinitionSubset strips the profile's role definitions when the target cannot
// install them, reporting them as skipped or, with strict, as a blocking reason.
func roleDefinitionSubset(adapter Adapter, agent AgentDestination, setsDefault, strict bool, profile profilemango.ResolvedProfile) (profilemango.ResolvedProfile, *SkippedRequirement, string) {
	count := len(profile.Roles)
	if count == 0 || supportsRequirement(adapter, agent, setsDefault, RequirementRoleDefinitions) {
		return profile, nil, ""
	}
	if strict {
		return profile, nil, fmt.Sprintf("--strict: %d role definitions cannot be installed: %s", count, roleDefinitionsUnsupportedReason)
	}
	profile.Roles = nil
	return profile, &SkippedRequirement{Requirement: RequirementRoleDefinitions, Count: count, Reason: roleDefinitionsUnsupportedReason}, ""
}

// targetSubset applies every requirement subset for one target: the profile and
// resources it installs, its effective route, and what it skips in a fixed order.
// A non-empty strict reason blocks the target.
func targetSubset(adapter Adapter, request Request, target TargetRequest, loaded loadedInput) (profilemango.ResolvedProfile, []render.Resource, profilemango.RouteBinding, []SkippedRequirement, string) {
	profile, resources := loaded.Profile, loaded.Resources
	var skipped []SkippedRequirement
	if !request.Strict {
		profile, resources, skipped = supportedSubset(adapter, target.Agent, request.Default, loaded)
	}
	profile, skippedDefinitions, strictReason := roleDefinitionSubset(adapter, target.Agent, request.Default, request.Strict, profile)
	if strictReason != "" {
		return profile, resources, profilemango.RouteBinding{}, nil, strictReason
	}
	route, skippedRoute, strictReason := routeSubset(adapter, target.Agent, request.Default, request.Strict, loaded.Route.For(target.Target.Name))
	if strictReason != "" {
		return profile, resources, route, nil, strictReason
	}
	skippedEffort, strictReason := effortSubset(adapter, request.Strict, route.Effort)
	if strictReason != "" {
		return profile, resources, route, nil, strictReason
	}
	if skippedDefinitions != nil {
		skipped = append(skipped, *skippedDefinitions)
	}
	skipped = append(skipped, skippedRoute...)
	if skippedEffort != nil {
		skipped = append(skipped, *skippedEffort)
	}
	return profile, resources, route, skipped, ""
}

// effortSubset reports the route effort an adapter will not write, as skipped or,
// with strict, as a blocking reason, instead of silently dropping it.
func effortSubset(adapter Adapter, strict bool, effort string) (*SkippedRequirement, string) {
	supporter, ok := adapter.(effortSupporter)
	if !ok || effort == "" {
		return nil, ""
	}
	supported, reason := supporter.EffortSupport(effort)
	if supported {
		return nil, ""
	}
	if strict {
		return nil, fmt.Sprintf("--strict: effort %s cannot be applied: %s", effort, reason)
	}
	return &SkippedRequirement{Requirement: RequirementEffort, Value: effort, Reason: reason}, ""
}

func withoutResourceKind(resources []render.Resource, kind string) []render.Resource {
	result := make([]render.Resource, 0, len(resources))
	for _, resource := range resources {
		if resource.Digest.Kind != kind {
			result = append(result, resource)
		}
	}
	return result
}
