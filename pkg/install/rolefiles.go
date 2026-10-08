package install

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// roleFile is one portable role rendered as a target's native subagent file. Model and
// Effort are empty when the role has no route of its own and inherits.
type roleFile struct {
	Name         string
	Description  string
	Instructions string
	Model        string
	Effort       string
}

// roleFileWriter is implemented by adapters whose pinned target discovers user-global
// subagent files. Evidence lives in docs/dev/agents/<target>.md.
type roleFileWriter interface {
	// roleFilePath is the role's file relative to the main config directory.
	roleFilePath(role string) string
	// roleFileModel is the model selector for a bound role, or why none is written.
	roleFileModel(role string, route profilemango.RoleRoute) (model, reason string)
	// roleEffortSupport reports whether a role effort is written, and why not otherwise.
	roleEffortSupport(effort string) (supported bool, reason string)
	renderRoleFile(file roleFile) ([]byte, error)
}

// roleInput is what one target installs as subagent files: the role definitions kept
// by roleDefinitionSubset, their instructions, the bound route's per-role routes, and
// the profile's agentFiles with why this install cannot write them, if it cannot.
type roleInput struct {
	writer       roleFileWriter
	definitions  map[string]profilemango.RoleDefinition
	instructions map[string][]byte
	routes       map[string]profilemango.RoleRoute
	agentFiles   []globalFile
	agentSkip    string
}

func newRoleInput(adapter Adapter, request Request, target TargetRequest, profile profilemango.ResolvedProfile, loaded loadedInput) roleInput {
	input := roleInput{agentFiles: loaded.AgentFiles[target.Target.Name]}
	if len(input.agentFiles) > 0 {
		input.agentSkip = roleFileSkipReason(adapter, target.Agent, request.Default)
	}
	writer, qualified := adapter.(roleFileWriter)
	if !qualified {
		return input
	}
	input.writer = writer
	if len(profile.Roles) > 0 {
		input.definitions, input.instructions, input.routes = profile.Roles, loaded.RoleInstructions, loaded.Route.For(target.Target.Name).Roles
	}
	return input
}

// loadRoleInstructions reads every role's instructions resource beneath root.
func loadRoleInstructions(root string, profile profilemango.ResolvedProfile) (map[string][]byte, []profilemango.ResourceDigest, profilemango.Diagnostics, []sourceCheck) {
	result := make(map[string][]byte)
	var digests []profilemango.ResourceDigest
	var diagnostics profilemango.Diagnostics
	var sources []sourceCheck
	for _, name := range sortedNames(profile.Roles) {
		resource := profile.Roles[name].Instructions
		if resource == nil {
			continue
		}
		fragment, digest, source, err := loadResource(root, *resource, "role-instruction")
		if err != nil {
			diagnostics.Add(profilemango.SeverityError, "resource.read", "roles."+name+".instructions", err.Error(), 0, 0)
			continue
		}
		result[name] = fragment.Content
		digests = append(digests, digest)
		sources = append(sources, source)
	}
	return result, digests, diagnostics.Sorted(), sources
}

// addRolePatches appends one owned subagent file per installed role, recording route
// parts the target cannot write as skipped (or, with --strict, blocking).
func addRolePatches(request Request, roles roleInput, patch *Patch, targetPlan *TargetPlan) (string, string) {
	for _, name := range sortedNames(roles.definitions) {
		file, skipped := roles.file(name)
		if skipped != nil && request.Strict {
			return fmt.Sprintf("--strict: role %s %s cannot be installed: %s", name, strings.TrimSpace(skipped.Requirement+" "+skipped.Value), skipped.Reason), "install.strict_requirement_unsupported"
		}
		if skipped != nil {
			targetPlan.SkippedRequirements = append(targetPlan.SkippedRequirements, *skipped)
		}
		content, err := roles.writer.renderRoleFile(file)
		if err != nil {
			return fmt.Sprintf("render role %s subagent file: %v", name, err), "install.role_file_invalid"
		}
		path := roles.writer.roleFilePath(name)
		if plannedPath(patch.Files, path) {
			return fmt.Sprintf("role %s subagent file %s is also written as another profile file; rename the profile or role", name, path), "install.role_file_conflict"
		}
		patch.Files = append(patch.Files, FilePatch{Path: path, Label: path, Content: content, Adoptable: true, Ownership: []string{ownershipRoleDefinition}})
	}
	return "", ""
}

// file renders role name, dropping the route model or effort the target cannot write.
func (roles roleInput) file(name string) (roleFile, *SkippedRequirement) {
	definition := roles.definitions[name]
	file := roleFile{Name: name, Description: strings.TrimSpace(definition.Description), Instructions: strings.TrimSpace(string(roles.instructions[name]))}
	if file.Instructions == "" {
		file.Instructions = file.Description
	}
	route, bound := roles.routes[name]
	if !bound {
		return file, nil
	}
	model, reason := roles.writer.roleFileModel(name, route)
	if reason != "" {
		return file, &SkippedRequirement{Requirement: RequirementRoles, Role: name, Reason: reason}
	}
	file.Model = model
	if route.Effort == "" {
		return file, nil
	}
	if supported, reason := roles.writer.roleEffortSupport(route.Effort); !supported {
		return file, &SkippedRequirement{Requirement: RequirementEffort, Value: route.Effort, Role: name, Reason: reason}
	}
	file.Effort = route.Effort
	return file, nil
}

func plannedPath(files []FilePatch, path string) bool {
	for _, file := range files {
		if filepath.Clean(file.Path) == filepath.Clean(path) {
			return true
		}
	}
	return false
}

// markdownAgent renders a YAML frontmatter subagent file: each non-empty field as a
// double-quoted scalar, then the body.
func markdownAgent(fields [][2]string, body string) []byte {
	var text strings.Builder
	text.WriteString("---\n")
	for _, field := range fields {
		if field[1] != "" {
			text.WriteString(field[0] + ": " + strconv.Quote(field[1]) + "\n")
		}
	}
	text.WriteString("---\n\n" + body + "\n")
	return []byte(text.String())
}

// requireRoleText rejects a role whose description or prompt is blank.
func requireRoleText(target string, file roleFile) error {
	if file.Description == "" || file.Instructions == "" {
		return fmt.Errorf("%s role %q requires a description and instructions", target, file.Name)
	}
	return nil
}
