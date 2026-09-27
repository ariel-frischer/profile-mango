package install

import (
	"fmt"
	"slices"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
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
	rolesUnsupportedReason             = "per-role routes install only as Oh My Pi modelRoles or as the model of a declared role's subagent file"
	roleDefinitionsUnsupportedReason   = "no pinned evidence qualifies user-global subagent files for this target"
	roleFilesNamedOnlyReason           = "subagent files are global; install with --default or mango use"
	roleFilesAgentReason               = "named agent destinations do not install subagent files"
	subagentMaxEffortUnsupportedReason = "the subagent effort cap is only installed for Oh My Pi task.maxEffort"
)

// SkippedRequirement is a known profile requirement a target cannot honor. A
// non-strict plan lists it and installs the rest instead of blocking. Value is
// the route value that was not applied (effort); Role names the portable role
// whose subagent file lacks it; Reason is set when the requirement name alone
// does not explain the skip.
type SkippedRequirement struct {
	Requirement string `json:"requirement"`
	Count       int    `json:"count,omitempty"`
	Value       string `json:"value,omitempty"`
	Role        string `json:"role,omitempty"`
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
	if count := len(profile.Skills); count > 0 {
		if skip, reason := skillSkip(adapter, agent, setsDefault); skip {
			skipped = append(skipped, SkippedRequirement{Requirement: RequirementSkills, Count: count, Reason: reason})
		}
	}
	return profile, resources, skipped
}

// routeSubset strips the per-role routes and subagent effort cap a target cannot
// install. A target without modelRoles still installs a route role as the model of a
// declared role's subagent file (definitions). It reports the rest as skipped, or as a
// strict blocking reason when strict is set.
func routeSubset(adapter Adapter, agent AgentDestination, setsDefault, strict bool, route profilemango.RouteBinding, definitions map[string]profilemango.RoleDefinition) (profilemango.RouteBinding, []SkippedRequirement, string) {
	var skipped []SkippedRequirement
	if !supportsRequirement(adapter, agent, setsDefault, RequirementRoles) {
		if count := uncoveredRoles(route.Roles, definitions); count > 0 {
			if strict {
				return route, nil, fmt.Sprintf("--strict: %d route roles cannot be installed: %s", count, rolesUnsupportedReason)
			}
			skipped = append(skipped, SkippedRequirement{Requirement: RequirementRoles, Count: count, Reason: rolesUnsupportedReason})
		}
		route.Roles = nil
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

// uncoveredRoles counts the route roles no installed role definition carries.
func uncoveredRoles(roles map[string]profilemango.RoleRoute, definitions map[string]profilemango.RoleDefinition) int {
	count := 0
	for name := range roles {
		if _, covered := definitions[name]; !covered {
			count++
		}
	}
	return count
}

// roleDefinitionSubset strips the profile's role definitions when the target writes no
// subagent files for this install, reporting them as skipped or, with strict, as a
// blocking reason.
func roleDefinitionSubset(adapter Adapter, agent AgentDestination, setsDefault, strict bool, profile profilemango.ResolvedProfile) (profilemango.ResolvedProfile, *SkippedRequirement, string) {
	count := len(profile.Roles)
	reason := roleFileSkipReason(adapter, agent, setsDefault)
	if count == 0 || reason == "" {
		return profile, nil, ""
	}
	if strict {
		return profile, nil, fmt.Sprintf("--strict: %d role definitions cannot be installed: %s", count, reason)
	}
	profile.Roles = nil
	return profile, &SkippedRequirement{Requirement: RequirementRoleDefinitions, Count: count, Reason: reason}, ""
}

// roleFileSkipReason explains why an install writes no subagent files: no qualified
// surface, a named agent destination, or a named profile that is not made the default,
// since the agent reads subagent files for every profile.
func roleFileSkipReason(adapter Adapter, agent AgentDestination, setsDefault bool) string {
	if _, qualified := adapter.(roleFileWriter); !qualified {
		return roleDefinitionsUnsupportedReason
	}
	if !agent.Empty() {
		return roleFilesAgentReason
	}
	if _, named := adapter.(namedProfileUser); named && !setsDefault {
		return roleFilesNamedOnlyReason
	}
	return ""
}

// targetSubset applies every requirement subset for one target: the profile and
// resources it installs, its effective route, and what it skips in a fixed order.
// A non-empty strict reason blocks the target.
func targetSubset(adapter Adapter, request Request, target TargetRequest, loaded loadedInput) (profilemango.ResolvedProfile, []render.Resource, profilemango.RouteBinding, []SkippedRequirement, string) {
	profile, resources := loaded.Profile, loaded.Resources
	var skipped []SkippedRequirement
	if !request.Strict {
		profile, resources, skipped = supportedSubset(adapter, target.Agent, request.Default, loaded)
	} else if skip, reason := skillSkip(adapter, target.Agent, request.Default); skip && len(profile.Skills) > 0 {
		return profile, resources, profilemango.RouteBinding{}, nil, fmt.Sprintf("--strict: %d skills cannot be installed: %s", len(profile.Skills), orNoEvidence(reason))
	}
	// Skill folders are written by addSkillPatches, never by an adapter's own config patch.
	profile.Skills = nil
	profile, skippedDefinitions, strictReason := roleDefinitionSubset(adapter, target.Agent, request.Default, request.Strict, profile)
	if strictReason != "" {
		return profile, resources, profilemango.RouteBinding{}, nil, strictReason
	}
	route, skippedRoute, strictReason := routeSubset(adapter, target.Agent, request.Default, request.Strict, loaded.Route.For(target.Target.Name), profile.Roles)
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

// orNoEvidence names the missing evidence when a skip carries no reason of its own.
func orNoEvidence(reason string) string {
	if reason == "" {
		return "no pinned evidence qualifies a user skill folder for this target"
	}
	return reason
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
