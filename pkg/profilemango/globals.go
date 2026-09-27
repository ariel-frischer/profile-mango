package profilemango

import (
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// globalInstructionFilePattern is a plain Markdown file name such as AGENTS.md or RULES.md.
var globalInstructionFilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`)

// HomeInstructions is the target-neutral globalInstructions key for files in the
// user's home directory, such as ~/AGENTS.md, which agents read as an ancestor of
// the working directory rather than from their own config folder.
const HomeInstructions = "home"

// ResourceList is a globalInstructions file value: one resource path (YAML scalar)
// or a list of fragment paths composed in order into one installed file.
type ResourceList []string

// UnmarshalYAML accepts a scalar resource path or a sequence of them. Shape,
// null, and emptiness checks run on the node tree and in validation, so this
// decodes whatever the strict parser lets through.
func (list *ResourceList) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		items := []string{}
		if err := node.Decode(&items); err != nil {
			return fmt.Errorf("decode resource list: %w", err)
		}
		*list = items
		return nil
	}
	var single string
	if err := node.Decode(&single); err != nil {
		return fmt.Errorf("decode resource path: %w", err)
	}
	*list = ResourceList{single}
	return nil
}

// MarshalJSON writes a one-item list as its string, so the scalar form keeps its
// resolved-profile JSON, and a longer list as an array.
func (list ResourceList) MarshalJSON() ([]byte, error) {
	if len(list) == 1 {
		return json.Marshal(list[0])
	}
	return json.Marshal([]string(list))
}

// validateGlobalInstructions checks globalInstructions: known target names or the
// home key, plain Markdown file names, and non-empty resource lists whose paths
// stay beneath the package root. Whether an agent reads a given file is decided by
// the installer's evidence table.
func validateGlobalInstructions(globals map[string]map[string]ResourceList, diagnostics *Diagnostics) {
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
			validateResourceList(globals[target][file], filePath, diagnostics)
		}
	}
}

// validateResourceList requires at least one resource and a safe path for each. A
// list item's diagnostic path carries its index; a single resource's does not.
func validateResourceList(list ResourceList, path string, diagnostics *Diagnostics) {
	if len(list) == 0 {
		diagnostics.Add(SeverityError, "profile.global_instructions_empty", path, "must name a resource path or a non-empty list of them", 0, 0)
		return
	}
	for index, resource := range list {
		itemPath := path
		if len(list) > 1 {
			itemPath = fmt.Sprintf("%s[%d]", path, index)
		}
		switch {
		case strings.TrimSpace(resource) == "":
			diagnostics.Add(SeverityError, "profile.global_instructions_path_invalid", itemPath, "resource path must not be empty", 0, 0)
		case !safeGlobalResource(resource):
			diagnostics.Add(SeverityError, "profile.global_instructions_path_invalid", itemPath, "resource path must be relative and remain beneath the package root", 0, 0)
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
func mergeTargetFiles[V any](parent, child map[string]map[string]V) map[string]map[string]V {
	if parent == nil && child == nil {
		return nil
	}
	result := make(map[string]map[string]V, len(parent)+len(child))
	for _, source := range []map[string]map[string]V{parent, child} {
		for target, files := range source {
			if len(files) == 0 {
				delete(result, target)
				continue
			}
			result[target] = maps.Clone(files)
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
