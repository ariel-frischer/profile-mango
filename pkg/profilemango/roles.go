package profilemango

import (
	"fmt"
	"strings"
)

// Semantic role names have target-specific mappings. Other valid identifiers
// are custom roles; profiles define them and bindings route them by name.
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

// SemanticRoles lists the roles with target-specific semantic mappings.
var SemanticRoles = []string{RoleWorker, RolePlanner, RoleResearch, RoleTiny}

// ValidRoleName reports whether name is a canonical identifier other than default.
func ValidRoleName(name string) bool {
	return name != ReservedRoleDefault && profileNamePattern.MatchString(name)
}

// unknownRoleMessage explains the identifier contract and the reserved base role.
func unknownRoleMessage(name string) string {
	if name == ReservedRoleDefault {
		return fmt.Sprintf("unknown role %q: the route's own provider, model, and effort are the default role; remove roles.default", name)
	}
	return fmt.Sprintf("unknown role %q; expected an identifier matching ^[a-z][a-z0-9-]{0,62}$ (default is reserved)", name)
}

// validateRoleDefinitions checks role identifiers, a non-empty description,
// and an optional instructions resource beneath the package root.
func validateRoleDefinitions(roles map[string]RoleDefinition, diagnostics *Diagnostics) {
	if roles != nil && len(roles) == 0 {
		diagnostics.Add(SeverityError, "profile.roles_empty", "roles", "roles must define at least one role", 0, 0)
	}
	for _, name := range sortedKeys(roles) {
		path := "roles." + name
		if !ValidRoleName(name) {
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
