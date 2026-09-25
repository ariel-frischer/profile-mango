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
	// RequirementEffort is the bound route's effort when the target cannot write it.
	RequirementEffort = "effort"
)

// rolesUnsupportedReason explains why a target without role support skips roles.
const rolesUnsupportedReason = "per-role routes are only installed for Oh My Pi modelRoles; this target installs the default route only"

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

// routeSubset strips per-role routes a target cannot install. It reports them
// as skipped, or as a strict blocking reason when strict is set.
func routeSubset(adapter Adapter, agent AgentDestination, setsDefault, strict bool, route profilemango.RouteBinding) (profilemango.RouteBinding, *SkippedRequirement, string) {
	count := len(route.Roles)
	if count == 0 || supportsRequirement(adapter, agent, setsDefault, RequirementRoles) {
		return route, nil, ""
	}
	if strict {
		return route, nil, fmt.Sprintf("--strict: %d route roles cannot be installed: %s", count, rolesUnsupportedReason)
	}
	route.Roles = nil
	return route, &SkippedRequirement{Requirement: RequirementRoles, Count: count, Reason: rolesUnsupportedReason}, ""
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
