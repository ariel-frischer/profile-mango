package install

import (
	"fmt"
	"path"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// RequirementAgentFiles is the profile's agentFiles for a target.
const RequirementAgentFiles = "agentFiles"

// loadAgentFiles reads every agentFiles resource beneath root.
func loadAgentFiles(root string, profile profilemango.ResolvedProfile) (map[string][]globalFile, []profilemango.ResourceDigest, profilemango.Diagnostics, []sourceCheck) {
	result := make(map[string][]globalFile, len(profile.AgentFiles))
	var digests []profilemango.ResourceDigest
	var diagnostics profilemango.Diagnostics
	var sources []sourceCheck
	for _, target := range sortedNames(profile.AgentFiles) {
		files := profile.AgentFiles[target]
		for _, name := range sortedNames(files) {
			file, source, err := loadGlobalFile(root, name, files[name])
			if err != nil {
				diagnostics.Add(profilemango.SeverityError, "resource.read", "agentFiles."+target+"."+name, err.Error(), 0, 0)
				continue
			}
			file.Digest.Kind = ownershipAgentFile
			result[target] = append(result[target], file)
			digests = append(digests, file.Digest)
			sources = append(sources, source)
		}
	}
	return result, digests, diagnostics.Sorted(), sources
}

// addAgentFilePatches appends the target's agentFiles verbatim as owned whole files in
// its subagent directory, or reports them skipped (blocking with --strict) when this
// install writes no subagent files.
func addAgentFilePatches(request Request, roles roleInput, patch *Patch, targetPlan *TargetPlan) (string, string) {
	count := len(roles.agentFiles)
	if count == 0 {
		return "", ""
	}
	if roles.agentSkip != "" {
		if request.Strict {
			return fmt.Sprintf("--strict: %d agent files cannot be installed: %s", count, roles.agentSkip), "install.strict_requirement_unsupported"
		}
		targetPlan.SkippedRequirements = append(targetPlan.SkippedRequirements, SkippedRequirement{Requirement: RequirementAgentFiles, Count: count, Reason: roles.agentSkip})
		return "", ""
	}
	for _, file := range roles.agentFiles {
		relative, reason, code := roles.agentFilePath(file.Name)
		if reason != "" {
			return reason, code
		}
		if plannedPath(patch.Files, relative) {
			return fmt.Sprintf("agent file %s is also written as another profile file", relative), "install.agent_file_path_conflict"
		}
		patch.Files = append(patch.Files, FilePatch{Path: relative, Label: relative, Content: file.Content, Adoptable: true, Ownership: []string{ownershipAgentFile}})
	}
	return "", ""
}

// agentFilePath places name in the directory holding the target's role files, requiring
// the role-file extension. Profile resolution already rejects a name a role generates.
func (roles roleInput) agentFilePath(name string) (string, string, string) {
	sample := roles.writer.roleFilePath("agent")
	if path.Ext(name) != path.Ext(sample) {
		return "", fmt.Sprintf("agent file %s must be a %s file for this target, like its role files", name, path.Ext(sample)), "install.agent_file_extension_invalid"
	}
	return path.Join(path.Dir(sample), name), "", ""
}
