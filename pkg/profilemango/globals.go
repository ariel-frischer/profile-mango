package profilemango

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// globalInstructionFilePattern is a plain Markdown file name such as AGENTS.md or RULES.md.
var globalInstructionFilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`)

// HomeInstructions is the target-neutral globalInstructions key for files in the
// user's home directory, such as ~/AGENTS.md, which agents read as an ancestor of
// the working directory rather than from their own config folder.
const HomeInstructions = "home"

// validateGlobalInstructions checks globalInstructions: known target names or the
// home key, plain Markdown file names, and resource paths that stay beneath the
// package root. Whether an agent reads a given file is decided by the installer's
// evidence table.
func validateGlobalInstructions(globals map[string]map[string]string, diagnostics *Diagnostics) {
	for _, target := range sortedKeys(globals) {
		path := "globalInstructions." + target
		if target != HomeInstructions && !knownRouteTarget(target) {
			message := fmt.Sprintf("unknown target %q; expected %s or one of %s", target, HomeInstructions, strings.Join(RouteTargets, ", "))
			diagnostics.Add(SeverityError, "profile.global_instructions_target_unknown", path, message, 0, 0)
		}
		for _, file := range sortedKeys(globals[target]) {
			filePath := path + "." + file
			if !globalInstructionFilePattern.MatchString(file) {
				diagnostics.Add(SeverityError, "profile.global_instructions_file_invalid", filePath, "file name must be a plain Markdown file name such as AGENTS.md", 0, 0)
			}
			if !safeGlobalResource(globals[target][file]) {
				diagnostics.Add(SeverityError, "profile.global_instructions_path_invalid", filePath, "resource path must be relative and remain beneath the package root", 0, 0)
			}
		}
	}
}

func safeGlobalResource(value string) bool {
	clean := filepath.Clean(value)
	return value != "" && !filepath.IsAbs(value) && clean != "." && clean != ".." &&
		!strings.HasPrefix(clean, ".."+string(filepath.Separator)) && !strings.ContainsRune(value, '\x00')
}

// mergeTargetFiles applies a child's per-target file maps (globalInstructions,
// agentFiles) over the parent's.
// A child target entry replaces the parent's entry for that target; an empty map clears it.
func mergeTargetFiles(parent, child map[string]map[string]string) map[string]map[string]string {
	if parent == nil && child == nil {
		return nil
	}
	result := make(map[string]map[string]string, len(parent)+len(child))
	for _, source := range []map[string]map[string]string{parent, child} {
		for target, files := range source {
			if len(files) == 0 {
				delete(result, target)
				continue
			}
			copied := make(map[string]string, len(files))
			for file, resource := range files {
				copied[file] = resource
			}
			result[target] = copied
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
