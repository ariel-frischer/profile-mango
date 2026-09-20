package profilemango

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LoadProfiles reads immediate <root>/<name>/profile.yaml profile folders.
func LoadProfiles(root string) (map[string]PolicyProfile, Diagnostics) {
	entries, err := os.ReadDir(root)
	if err != nil {
		var diagnostics Diagnostics
		diagnostics.Add(SeverityError, "repository.read", "profiles", err.Error(), 0, 0)
		return nil, diagnostics
	}
	profiles := make(map[string]PolicyProfile)
	var diagnostics Diagnostics
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "profile.yaml")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			diagnostics.Add(SeverityError, "repository.read", entry.Name(), err.Error(), 0, 0)
			continue
		}
		profile, found := ParseProfile(data)
		for _, item := range found {
			item.Path = entry.Name() + "." + item.Path
			diagnostics = append(diagnostics, item)
		}
		if profile.Metadata.Name != "" && profile.Metadata.Name != entry.Name() {
			diagnostics.Add(SeverityError, "repository.name_mismatch", entry.Name(), "folder name must match metadata.name", 0, 0)
		}
		if profile.Metadata.Name != "" {
			profiles[profile.Metadata.Name] = profile
		}
	}
	return profiles, diagnostics.Sorted()
}

// Resolve deterministically resolves one optional parent chain.
func Resolve(profiles map[string]PolicyProfile, name string) (ResolvedProfile, Diagnostics) {
	return resolve(profiles, name, nil)
}

func resolve(profiles map[string]PolicyProfile, name string, stack []string) (ResolvedProfile, Diagnostics) {
	profile, found := profiles[name]
	if !found {
		var diagnostics Diagnostics
		diagnostics.Add(SeverityError, "resolution.missing_profile", "spec.extends", fmt.Sprintf("profile %q not found", name), 0, 0)
		return ResolvedProfile{}, diagnostics
	}
	for _, current := range stack {
		if current == name {
			var diagnostics Diagnostics
			diagnostics.Add(SeverityError, "resolution.cycle", "spec.extends", fmt.Sprintf("inheritance cycle includes %q", name), 0, 0)
			return ResolvedProfile{}, diagnostics
		}
	}

	result := emptyResolved(profile)
	var diagnostics Diagnostics
	if profile.Spec.Extends != "" {
		parent, parentDiagnostics := resolve(profiles, profile.Spec.Extends, append(stack, name))
		diagnostics = append(diagnostics, parentDiagnostics...)
		if !parentDiagnostics.HasErrors() {
			result = mergeProfile(parent, profile)
		}
	}
	if result.RouteRef == "" {
		diagnostics.Add(SeverityError, "profile.route_required", "spec.routeRef", "routeRef is required after inheritance", 0, 0)
	}
	if duplicates := duplicateStrings(result.Instructions); len(duplicates) > 0 {
		diagnostics.Add(SeverityError, "resolution.duplicate_instruction", "spec.instructions.append", fmt.Sprintf("duplicate resolved paths: %v", duplicates), 0, 0)
	}
	return result, diagnostics.Sorted()
}

func emptyResolved(profile PolicyProfile) ResolvedProfile {
	resolved := ResolvedProfile{
		APIVersion: APIVersion,
		Kind:       KindPolicyProfile,
		Metadata:   cloneMetadata(profile.Metadata),
		RouteRef:   profile.Spec.RouteRef,
		Parent:     profile.Spec.Extends,
	}
	resolved.Permissions = mergePermissions(nil, profile.Spec.Permissions)
	resolved.Tools = resolveRules(nil, profile.Spec.Tools)
	if profile.Spec.Instructions.Append != nil {
		resolved.Instructions = append([]string(nil), (*profile.Spec.Instructions.Append)...)
	}
	if profile.Spec.Skills != nil {
		resolved.Skills = append([]string(nil), (*profile.Spec.Skills)...)
	}
	return resolved
}

func mergeProfile(parent ResolvedProfile, child PolicyProfile) ResolvedProfile {
	result := parent
	result.Metadata = mergeMetadata(parent.Metadata, child.Metadata)
	result.Parent = child.Spec.Extends
	if child.Spec.RouteRef != "" {
		result.RouteRef = child.Spec.RouteRef
	}
	result.Permissions = mergePermissions(parent.Permissions, child.Spec.Permissions)
	result.Tools = resolveRules(parent.Tools, child.Spec.Tools)
	if child.Spec.Instructions.Append != nil {
		result.Instructions = append(append([]string(nil), parent.Instructions...), (*child.Spec.Instructions.Append)...)
	}
	if child.Spec.Skills != nil {
		result.Skills = append([]string(nil), (*child.Spec.Skills)...)
	}
	return result
}

func mergeMetadata(parent, child Metadata) Metadata {
	result := cloneMetadata(parent)
	result.Name = child.Name
	if child.Description != "" {
		result.Description = child.Description
	}
	if result.Labels == nil {
		result.Labels = map[string]string{}
	}
	for key, value := range child.Labels {
		result.Labels[key] = value
	}
	return result
}

func cloneMetadata(metadata Metadata) Metadata {
	result := metadata
	if metadata.Labels != nil {
		result.Labels = make(map[string]string, len(metadata.Labels))
		for key, value := range metadata.Labels {
			result.Labels[key] = value
		}
	}
	return result
}

func mergePermissions(parent, child *PermissionPolicy) *PermissionPolicy {
	if parent == nil && child == nil {
		return nil
	}
	result := &PermissionPolicy{}
	if parent != nil {
		*result = *parent
	}
	if child != nil {
		if child.Mode != nil {
			result.Mode = child.Mode
		}
		if child.Network != nil {
			result.Network = child.Network
		}
		if child.Shell != nil {
			result.Shell = child.Shell
		}
	}
	return result
}

func resolveRules(parent *ResolvedRules, child *AccessRules) *ResolvedRules {
	if parent == nil && child == nil {
		return nil
	}
	result := &ResolvedRules{Managed: true}
	if parent != nil {
		result.Allow = append([]string(nil), parent.Allow...)
		result.Deny = append([]string(nil), parent.Deny...)
		result.Closed = parent.Closed
	}
	if child != nil {
		if child.Allow != nil {
			result.Allow = uniqueSorted(*child.Allow)
			result.Closed = true
		}
		if child.Deny != nil {
			result.Deny = uniqueSorted(*child.Deny)
		}
	}
	denied := make(map[string]struct{}, len(result.Deny))
	for _, item := range result.Deny {
		denied[item] = struct{}{}
	}
	filtered := result.Allow[:0]
	for _, item := range result.Allow {
		if _, blocked := denied[item]; !blocked {
			filtered = append(filtered, item)
		}
	}
	result.Allow = filtered
	return result
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func duplicateStrings(values []string) []string {
	seen := map[string]bool{}
	duplicates := map[string]struct{}{}
	for _, value := range values {
		if seen[value] {
			duplicates[value] = struct{}{}
		}
		seen[value] = true
	}
	result := make([]string, 0, len(duplicates))
	for value := range duplicates {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// CanonicalJSON returns deterministic bytes for golden files and hashing.
func CanonicalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal canonical JSON: %w", err)
	}
	return append(data, '\n'), nil
}
