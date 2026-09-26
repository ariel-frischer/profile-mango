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
		profile, found := ParseProfileAt(data, entry.Name())
		for _, item := range found {
			item.Path = entry.Name() + "." + item.Path
			diagnostics = append(diagnostics, item)
		}
		profiles[entry.Name()] = profile
	}
	return profiles, diagnostics.Sorted()
}

// Resolve deterministically resolves one optional parent chain. A profile
// without a name takes its map key as its name.
func Resolve(profiles map[string]PolicyProfile, name string) (ResolvedProfile, Diagnostics) {
	return resolve(profiles, name, nil)
}

func resolve(profiles map[string]PolicyProfile, name string, stack []string) (ResolvedProfile, Diagnostics) {
	profile, found := profiles[name]
	if !found {
		var diagnostics Diagnostics
		diagnostics.Add(SeverityError, "resolution.missing_profile", "extends", fmt.Sprintf("profile %q not found", name), 0, 0)
		return ResolvedProfile{}, diagnostics
	}
	for _, current := range stack {
		if current == name {
			var diagnostics Diagnostics
			diagnostics.Add(SeverityError, "resolution.cycle", "extends", fmt.Sprintf("inheritance cycle includes %q", name), 0, 0)
			return ResolvedProfile{}, diagnostics
		}
	}

	if profile.Name == "" {
		profile.Name = name
	}
	result := emptyResolved(profile)
	var diagnostics Diagnostics
	if profile.Extends != "" {
		parent, parentDiagnostics := resolve(profiles, profile.Extends, append(stack, name))
		diagnostics = append(diagnostics, parentDiagnostics...)
		if !parentDiagnostics.HasErrors() {
			result = mergeProfile(parent, profile)
		}
	}
	if result.RouteRef == "" {
		diagnostics.Add(SeverityError, "profile.route_required", "route", "route is required after inheritance", 0, 0)
	}
	if duplicates := duplicateStrings(result.Instructions); len(duplicates) > 0 {
		diagnostics.Add(SeverityError, "resolution.duplicate_instruction", "instructions.append", fmt.Sprintf("duplicate resolved paths: %v", duplicates), 0, 0)
	}
	return result, diagnostics.Sorted()
}

func emptyResolved(profile PolicyProfile) ResolvedProfile {
	resolved := ResolvedProfile{
		APIVersion: APIVersion,
		Kind:       KindPolicyProfile,
		Metadata:   cloneMetadata(profileMetadata(profile)),
		RouteRef:   profile.Route,
		Parent:     profile.Extends,
	}
	resolved.Permissions = mergePermissions(nil, profile.Permissions)
	resolved.Tools = resolveRules(nil, profile.Tools)
	if profile.Instructions.Append != nil {
		resolved.Instructions = append([]string(nil), (*profile.Instructions.Append)...)
	}
	if profile.Skills != nil {
		resolved.Skills = append([]string(nil), (*profile.Skills)...)
	}
	resolved.GlobalInstructions = mergeGlobalInstructions(nil, profile.GlobalInstructions)
	resolved.Roles = mergeRoleDefinitions(nil, profile.Roles)
	return resolved
}

func mergeProfile(parent ResolvedProfile, child PolicyProfile) ResolvedProfile {
	result := parent
	result.Metadata = mergeMetadata(parent.Metadata, profileMetadata(child))
	result.Parent = child.Extends
	if child.Route != "" {
		result.RouteRef = child.Route
	}
	result.Permissions = mergePermissions(parent.Permissions, child.Permissions)
	result.Tools = resolveRules(parent.Tools, child.Tools)
	if child.Instructions.Append != nil {
		result.Instructions = append(append([]string(nil), parent.Instructions...), (*child.Instructions.Append)...)
	}
	if child.Skills != nil {
		result.Skills = append([]string(nil), (*child.Skills)...)
	}
	result.GlobalInstructions = mergeGlobalInstructions(parent.GlobalInstructions, child.GlobalInstructions)
	result.Roles = mergeRoleDefinitions(parent.Roles, child.Roles)
	return result
}

func profileMetadata(profile PolicyProfile) Metadata {
	return Metadata{Name: profile.Name, Description: profile.Description, Labels: profile.Labels}
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
