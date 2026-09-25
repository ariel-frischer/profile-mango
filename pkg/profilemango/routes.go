package profilemango

import (
	"fmt"
	"sort"
	"strings"
)

// RouteTargets lists the target names a route's targets map may override.
// It mirrors the adapter TargetName constants; pkg/install tests keep them aligned.
var RouteTargets = []string{"claude-code", "codex", "hermes", "jcode-fork", "oh-my-pi", "openclaw", "opencode", "pi"}

// RouteFor returns the effective route for one target: the named base route
// with any non-empty fields from routes.<name>.targets.<target> applied.
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
	return route
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
		if route.Targets[name] == (RouteOverride{}) {
			diagnostics.Add(SeverityError, "binding.target_override_empty", targetPath, "target override must set at least one route field", 0, 0)
		}
	}
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
		if !profileNamePattern.MatchString(name) {
			diagnostics.Add(SeverityError, "binding.role_name_invalid", rolePath, "role name must be lowercase kebab-case", 0, 0)
		}
		if name == ReservedRoleDefault {
			diagnostics.Add(SeverityError, "binding.role_name_reserved", rolePath, "the route's own provider, model, and effort are the default role; remove roles.default", 0, 0)
		}
		role := route.Roles[name]
		if role.Provider == "" || role.Model == "" {
			diagnostics.Add(SeverityError, "binding.role_incomplete", rolePath, "provider and model are required", 0, 0)
		}
	}
}
