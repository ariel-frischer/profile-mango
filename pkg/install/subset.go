package install

import (
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
)

// SkippedRequirement is a known profile requirement a target cannot honor. A
// non-strict plan lists it and installs the rest instead of blocking.
type SkippedRequirement struct {
	Requirement string `json:"requirement"`
	Count       int    `json:"count,omitempty"`
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

func withoutResourceKind(resources []render.Resource, kind string) []render.Resource {
	result := make([]render.Resource, 0, len(resources))
	for _, resource := range resources {
		if resource.Digest.Kind != kind {
			result = append(result, resource)
		}
	}
	return result
}
