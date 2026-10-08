package ohmypi

import (
	"fmt"
	"slices"
	"strings"
)

// ReleaseRoles gives back model roles a new route no longer sets. prior maps each
// role to the selector it had before profile-mango first wrote it; an empty prior
// means the role was absent, so its line is removed. Roles already missing from
// source are left alone. Unrelated YAML bytes are preserved.
func ReleaseRoles(source []byte, prior map[string]string) ([]byte, []RoleChange, error) {
	return releaseBlock(source, "modelRoles", prior)
}

// ReleaseSettings gives back dotted settings (e.g. task.maxEffort) a new route no
// longer sets, with the same prior-value rules as ReleaseRoles. A block left with
// no keys after its last owned key is removed is deleted too.
func ReleaseSettings(source []byte, prior map[string]string) ([]byte, []SettingChange, error) {
	var changes []SettingChange
	for _, path := range sortedKeys(prior) {
		block, key, found := strings.Cut(path, ".")
		if !found || strings.Contains(key, ".") {
			return nil, nil, fmt.Errorf("oh my pi setting %q must be <block>.<key>", path)
		}
		released, roleChanges, err := releaseBlock(source, block, map[string]string{key: prior[path]})
		if err != nil {
			return nil, nil, err
		}
		source = released
		for _, change := range roleChanges {
			changes = append(changes, SettingChange{Path: path, Before: change.Before, After: change.After})
		}
	}
	return source, changes, nil
}

func releaseBlock(source []byte, block string, prior map[string]string) ([]byte, []RoleChange, error) {
	if len(prior) == 0 {
		return source, nil, nil
	}
	document, err := parseConfig(source)
	if err != nil {
		return nil, nil, err
	}
	entry, found, err := findEntry(document.root, block)
	if err != nil || !found {
		return source, nil, err
	}
	var edits []textEdit
	var changes []RoleChange
	for _, key := range sortedKeys(prior) {
		child, present, err := findEntry(entry.value, key)
		if err != nil {
			return nil, nil, err
		}
		if !present {
			continue
		}
		edit, before, err := releaseRole(document, child, prior[key])
		if err != nil {
			return nil, nil, fmt.Errorf("release %s.%s: %w", block, key, err)
		}
		edits = append(edits, edit)
		changes = append(changes, RoleChange{Role: key, Before: before, After: prior[key]})
	}
	if removesEveryKey(entry, prior, len(changes)) {
		edits = append(edits, lineDeletion(document, entry.key.Line))
	}
	return applyEdits(source, edits), changes, nil
}

// removesEveryKey reports whether the release deletes every key of the block,
// which would leave a bare `block:` (null) header behind.
func removesEveryKey(entry mappingEntry, prior map[string]string, released int) bool {
	if released != len(entry.value.Content)/2 || entry.value.Line == entry.key.Line {
		return false
	}
	for index := 0; index+1 < len(entry.value.Content); index += 2 {
		if prior[entry.value.Content[index].Value] != "" {
			return false
		}
	}
	return true
}

// releaseRole restores a key's prior value in place, or deletes its whole
// single-line entry when the key did not exist before.
func releaseRole(document yamlDocument, entry mappingEntry, prior string) (textEdit, string, error) {
	edit, before, err := replaceScalar(document, entry.value, prior)
	if err != nil || prior != "" {
		return edit, before, err
	}
	if entry.value.Line != entry.key.Line {
		return textEdit{}, "", fmt.Errorf("value must be on the key's line")
	}
	return lineDeletion(document, entry.key.Line), before, nil
}

// lineDeletion removes one whole source line, including its line ending.
func lineDeletion(document yamlDocument, line int) textEdit {
	index := line - 1
	end := len(document.source)
	if index+1 < len(document.lines) {
		end = document.lines[index+1].start
	}
	return textEdit{start: document.lines[index].start, end: end}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
