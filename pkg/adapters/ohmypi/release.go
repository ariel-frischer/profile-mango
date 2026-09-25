package ohmypi

import (
	"fmt"
	"slices"
)

// ReleaseRoles gives back model roles a new route no longer sets. prior maps each
// role to the selector it had before profile-mango first wrote it; an empty prior
// means the role was absent, so its line is removed. Roles already missing from
// source are left alone. Unrelated YAML bytes are preserved.
func ReleaseRoles(source []byte, prior map[string]string) ([]byte, []RoleChange, error) {
	if len(prior) == 0 {
		return source, nil, nil
	}
	document, err := parseConfig(source)
	if err != nil {
		return nil, nil, err
	}
	roles, found, err := findEntry(document.root, "modelRoles")
	if err != nil || !found {
		return source, nil, err
	}
	var edits []textEdit
	var changes []RoleChange
	for _, role := range sortedRoles(prior) {
		entry, present, err := findEntry(roles.value, role)
		if err != nil {
			return nil, nil, err
		}
		if !present {
			continue
		}
		edit, before, err := releaseRole(document, entry, prior[role])
		if err != nil {
			return nil, nil, fmt.Errorf("release modelRoles.%s: %w", role, err)
		}
		edits = append(edits, edit)
		changes = append(changes, RoleChange{Role: role, Before: before, After: prior[role]})
	}
	return applyEdits(source, edits), changes, nil
}

// releaseRole restores a role's prior selector in place, or deletes its whole
// single-line entry when the role did not exist before.
func releaseRole(document yamlDocument, entry mappingEntry, prior string) (textEdit, string, error) {
	edit, before, err := replaceScalar(document, entry.value, prior)
	if err != nil || prior != "" {
		return edit, before, err
	}
	if entry.value.Line != entry.key.Line {
		return textEdit{}, "", fmt.Errorf("role value must be on the key's line")
	}
	index := entry.key.Line - 1
	end := len(document.source)
	if index+1 < len(document.lines) {
		end = document.lines[index+1].start
	}
	return textEdit{start: document.lines[index].start, end: end}, before, nil
}

func sortedRoles(prior map[string]string) []string {
	roles := make([]string, 0, len(prior))
	for role := range prior {
		roles = append(roles, role)
	}
	slices.Sort(roles)
	return roles
}
