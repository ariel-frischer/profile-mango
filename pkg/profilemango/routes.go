package profilemango

import (
	"fmt"
	"sort"
	"strings"
)

// RouteTargets lists the target names a route's targets map may override.
// It mirrors the adapter TargetName constants; pkg/install tests keep them aligned.
var RouteTargets = []string{"ariel-jcode", "claude-code", "codex", "hermes", "oh-my-pi", "openclaw", "opencode", "pi"}

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
