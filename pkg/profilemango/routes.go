package profilemango

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// RouteTargets lists the target names a route's targets map may override.
// It mirrors the adapter TargetName constants; pkg/install tests keep them aligned.
var RouteTargets = []string{"claude-code", "codex", "hermes", "jcode-fork", "oh-my-pi", "openclaw", "opencode", "pi"}

// RouteFor returns the effective route for one target: the named base route
// with any non-empty fields from routes.<name>.targets.<target> applied,
// including its per-role fields.
func (bindings Bindings) RouteFor(name, target string) (RouteBinding, bool) {
	route, found := bindings.Routes[name]
	if !found {
		return RouteBinding{}, false
	}
	return route.For(target), true
}

// For merges the target override, if any, over the base route fields.
func (route RouteBinding) For(target string) RouteBinding {
	override := route.Targets[target]
	route.Targets = nil
	route.Provider = overrideField(route.Provider, override.Provider)
	route.Transport = overrideField(route.Transport, override.Transport)
	route.Authentication = overrideField(route.Authentication, override.Authentication)
	route.Model = overrideField(route.Model, override.Model)
	route.Effort = overrideField(route.Effort, override.Effort)
	route.Roles = overrideRoles(route.Roles, override.Roles)
	return route
}

// overrideRoles returns a copy of base with each role override's non-empty
// fields applied. Overrides for roles base does not bind are ignored; parsing
// rejects them.
func overrideRoles(base map[string]RoleRoute, overrides map[string]RoleOverride) map[string]RoleRoute {
	if len(overrides) == 0 {
		return base
	}
	result := make(map[string]RoleRoute, len(base))
	for name, role := range base {
		override := overrides[name]
		role.Provider = overrideField(role.Provider, override.Provider)
		role.Model = overrideField(role.Model, override.Model)
		role.Effort = overrideField(role.Effort, override.Effort)
		result[name] = role
	}
	return result
}

func overrideField(base, override string) string {
	if override != "" {
		return override
	}
	return base
}

func validateRouteTargets(path string, route RouteBinding, diagnostics *Diagnostics) {
	if route.Targets != nil && len(route.Targets) == 0 {
		diagnostics.Add(SeverityError, "binding.targets_empty", path+".targets", "targets must override at least one target", 0, 0)
	}
	names := make([]string, 0, len(route.Targets))
	for name := range route.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		targetPath := path + ".targets." + name
		if !knownRouteTarget(name) {
			message := fmt.Sprintf("unknown target %q; expected one of %s", name, strings.Join(RouteTargets, ", "))
			diagnostics.Add(SeverityError, "binding.target_unknown", targetPath, message, 0, 0)
		}
		override := route.Targets[name]
		if override.empty() {
			diagnostics.Add(SeverityError, "binding.target_override_empty", targetPath, "target override must set at least one route field", 0, 0)
		}
		validateTargetRoles(targetPath, route.Roles, override.Roles, diagnostics)
	}
}

// validateTargetRoles checks targets.<agent>.roles: each entry must name a
// portable role the base route binds and change at least one field.
func validateTargetRoles(path string, base map[string]RoleRoute, overrides map[string]RoleOverride, diagnostics *Diagnostics) {
	if overrides != nil && len(overrides) == 0 {
		diagnostics.Add(SeverityError, "binding.roles_empty", path+".roles", "roles must define at least one role", 0, 0)
	}
	for _, name := range sortedKeys(overrides) {
		rolePath := path + ".roles." + name
		switch _, bound := base[name]; {
		case !ValidRoleName(name):
			diagnostics.Add(SeverityError, "binding.role_unknown", rolePath, unknownRoleMessage(name), 0, 0)
		case !bound:
			diagnostics.Add(SeverityError, "binding.target_role_unbound", rolePath, fmt.Sprintf("role %q is not in the route's base roles; add roles.%s first", name, name), 0, 0)
		}
		if overrides[name] == (RoleOverride{}) {
			diagnostics.Add(SeverityError, "binding.target_role_override_empty", rolePath, "role override must set at least one of provider, model, or effort", 0, 0)
		}
	}
}

// empty reports whether the override sets no field; an empty roles map counts
// as set so its own diagnostic reports it.
func (override RouteOverride) empty() bool {
	return override.Provider == "" && override.Transport == "" && override.Authentication == "" &&
		override.Model == "" && override.Effort == "" && override.Roles == nil
}

func knownRouteTarget(name string) bool {
	for _, candidate := range RouteTargets {
		if candidate == name {
			return true
		}
	}
	return false
}

// SortedRoleNames returns the route's role names in a stable order.
func (route RouteBinding) SortedRoleNames() []string {
	names := make([]string, 0, len(route.Roles))
	for name := range route.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func validateRouteRoles(path string, route RouteBinding, diagnostics *Diagnostics) {
	if route.Roles != nil && len(route.Roles) == 0 {
		diagnostics.Add(SeverityError, "binding.roles_empty", path+".roles", "roles must define at least one role", 0, 0)
	}
	for _, name := range route.SortedRoleNames() {
		rolePath := path + ".roles." + name
		if !ValidRoleName(name) {
			diagnostics.Add(SeverityError, "binding.role_unknown", rolePath, unknownRoleMessage(name), 0, 0)
		}
		role := route.Roles[name]
		if role.Provider == "" || role.Model == "" {
			diagnostics.Add(SeverityError, "binding.role_incomplete", rolePath, "provider and model are required", 0, 0)
		}
	}
}

// SubagentEfforts are the accepted subagentMaxEffort values, matching the
// thinking-effort enum of Oh My Pi's task.maxEffort (the only target that installs it).
var SubagentEfforts = []string{"minimal", "low", "medium", "high", "xhigh", "max"}

func validateSubagentMaxEffort(path string, route RouteBinding, diagnostics *Diagnostics) {
	if route.SubagentMaxEffort == "" || slices.Contains(SubagentEfforts, route.SubagentMaxEffort) {
		return
	}
	diagnostics.Add(SeverityError, "binding.subagent_max_effort_invalid", path+".subagentMaxEffort", fmt.Sprintf("unknown effort %q; expected one of %s", route.SubagentMaxEffort, strings.Join(SubagentEfforts, ", ")), 0, 0)
}
