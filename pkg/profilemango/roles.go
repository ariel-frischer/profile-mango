package profilemango

import (
	"fmt"
	"slices"
	"strings"
)

// Portable role names. A profile describes them under roles.<role>; bindings
// route them under routes.<name>.roles.<role>; adapters map them to native slots.
const (
	// RoleWorker is an implementation subagent.
	RoleWorker = "worker"
	// RolePlanner does planning and architecture work.
	RolePlanner = "planner"
	// RoleResearch does read-only exploration and scouting.
	RoleResearch = "research"
	// RoleTiny does small mechanical tasks such as commit messages.
	RoleTiny = "tiny"
)

// PortableRoles is the fixed role vocabulary, in documentation order.
var PortableRoles = []string{RoleWorker, RolePlanner, RoleResearch, RoleTiny}

// roleNameHints explains well-known names that are not portable roles: the base
// route's default role and the Oh My Pi native slot names bindings used before.
var roleNameHints = map[string]string{
	ReservedRoleDefault: "the route's own provider, model, and effort are the default role; remove roles.default",
	"task":              `"task" is an Oh My Pi slot name; use the portable role "worker"`,
	"plan":              `"plan" is an Oh My Pi slot name; use the portable role "planner"`,
	"slow":              `"slow" is an Oh My Pi slot name; use the portable role "planner"`,
	"smol":              `"smol" is an Oh My Pi slot name; use the portable role "research"`,
	"commit":            `"commit" is an Oh My Pi slot name; use the portable role "tiny"`,
	"advisor":           `"advisor" is an Oh My Pi slot name with no portable role`,
	"vision":            `"vision" is an Oh My Pi slot name with no portable role`,
}

// PortableRole reports whether name is in the fixed role vocabulary.
func PortableRole(name string) bool {
	return slices.Contains(PortableRoles, name)
}

// unknownRoleMessage explains why name is not a portable role, naming the
// portable replacement for a known native slot name.
func unknownRoleMessage(name string) string {
	expected := "expected one of " + strings.Join(PortableRoles, ", ")
	if hint, found := roleNameHints[name]; found {
		return fmt.Sprintf("unknown role %q: %s; %s", name, hint, expected)
	}
	return fmt.Sprintf("unknown role %q; %s", name, expected)
}

// validateRoleDefinitions checks profile roles: portable names, a non-empty
// description, and an optional instructions resource beneath the package root.
func validateRoleDefinitions(roles map[string]RoleDefinition, diagnostics *Diagnostics) {
	if roles != nil && len(roles) == 0 {
		diagnostics.Add(SeverityError, "profile.roles_empty", "roles", "roles must define at least one role", 0, 0)
	}
	for _, name := range sortedKeys(roles) {
		path := "roles." + name
		if !PortableRole(name) {
			diagnostics.Add(SeverityError, "profile.role_unknown", path, unknownRoleMessage(name), 0, 0)
		}
		role := roles[name]
		if strings.TrimSpace(role.Description) == "" {
			diagnostics.Add(SeverityError, "profile.role_description_required", path+".description", "description is required and must not be blank", 0, 0)
		}
		if role.Instructions != nil && !safeGlobalResource(*role.Instructions) {
			diagnostics.Add(SeverityError, "profile.role_instructions_path_invalid", path+".instructions", "resource path must be non-empty, relative, and remain beneath the package root", 0, 0)
		}
	}
}

// mergeRoleDefinitions applies a child's role definitions over the parent's. A
// child role replaces the parent's definition of that role; other roles inherit.
func mergeRoleDefinitions(parent, child map[string]RoleDefinition) map[string]RoleDefinition {
	if len(parent) == 0 && len(child) == 0 {
		return nil
	}
	result := make(map[string]RoleDefinition, len(parent)+len(child))
	for _, source := range []map[string]RoleDefinition{parent, child} {
		for name, role := range source {
			result[name] = role
		}
	}
	return result
}
