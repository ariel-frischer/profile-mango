package profilemango

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// agentFilePattern is a plain subagent file name such as scout.md or reviewer.toml.
// Which extension a target reads is decided by the installer's role-file writer.
var agentFilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.(md|toml)$`)

// validateAgentFiles checks agentFiles: known target names, plain subagent file names,
// and resource paths that stay beneath the package root. Whether a target installs
// subagent files, and with which extension, is decided by the installer.
func validateAgentFiles(files map[string]map[string]string, diagnostics *Diagnostics) {
	for _, target := range sortedKeys(files) {
		path := "agentFiles." + target
		if !knownRouteTarget(target) {
			message := fmt.Sprintf("unknown target %q; expected one of %s", target, strings.Join(RouteTargets, ", "))
			diagnostics.Add(SeverityError, "profile.agent_files_target_unknown", path, message, 0, 0)
		}
		for _, file := range sortedKeys(files[target]) {
			filePath := path + "." + file
			if !agentFilePattern.MatchString(file) {
				diagnostics.Add(SeverityError, "profile.agent_files_file_invalid", filePath, "file name must be a plain .md or .toml file name such as scout.md", 0, 0)
			}
			if !safeGlobalResource(files[target][file]) {
				diagnostics.Add(SeverityError, "profile.agent_files_path_invalid", filePath, "resource path must be relative and remain beneath the package root", 0, 0)
			}
		}
	}
}

// validateAgentFileRoles rejects an agent file named like a resolved role, since the
// role's generated subagent file would be written to the same path.
func validateAgentFileRoles(profile ResolvedProfile, diagnostics *Diagnostics) {
	for _, target := range sortedKeys(profile.AgentFiles) {
		for _, file := range sortedKeys(profile.AgentFiles[target]) {
			role := strings.TrimSuffix(file, filepath.Ext(file))
			if _, declared := profile.Roles[role]; !declared {
				continue
			}
			message := fmt.Sprintf("%s would overwrite the subagent file generated for role %s; rename the file or remove the role", file, role)
			diagnostics.Add(SeverityError, "profile.agent_files_role_conflict", "agentFiles."+target+"."+file, message, 0, 0)
		}
	}
}
