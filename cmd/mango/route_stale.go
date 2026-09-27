package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// staleRouteFields are the route fields whose old literal value route set looks for.
var staleRouteFields = []string{"provider", "model"}

// writeRouteStaleReferences lists where profiles using the route still spell a
// provider or model value the edit replaces, so they can switch to a placeholder.
func writeRouteStaleReferences(output io.Writer, options *routeOptions, before, after []byte, edit profilemango.RouteEdit, profilesRoot string, users []string) error {
	oldValues := changedRouteValues(before, after, edit)
	if len(oldValues) == 0 || len(users) == 0 {
		return nil
	}
	resourceRoot := options.resourceRoot
	if resourceRoot == "" {
		home, err := selectedHome()
		if err != nil {
			return err
		}
		resourceRoot = home
	}
	return writeStaleReferences(output, oldValues, routeReferenceFiles(profilesRoot, resourceRoot, users), resourceRoot)
}

// changedRouteValues returns the old provider and model values the edit changes at its
// scope (the base route, one agent's effective route, or one role), in field order.
func changedRouteValues(before, after []byte, edit profilemango.RouteEdit) []string {
	oldBindings, _ := profilemango.ParseBindings(before)
	newBindings, _ := profilemango.ParseBindings(after)
	oldValues, newValues := routeScopeValues(oldBindings, edit), routeScopeValues(newBindings, edit)
	var changed []string
	for _, field := range staleRouteFields {
		if old := oldValues[field]; old != "" && old != newValues[field] && !slices.Contains(changed, old) {
			changed = append(changed, old)
		}
	}
	return changed
}

func routeScopeValues(bindings profilemango.Bindings, edit profilemango.RouteEdit) map[string]string {
	route := bindings.Routes[edit.Route]
	switch {
	case edit.Role != "":
		role := route.Roles[edit.Role]
		return map[string]string{"provider": role.Provider, "model": role.Model}
	case edit.Target != "":
		route = route.For(edit.Target)
	}
	return map[string]string{"provider": route.Provider, "model": route.Model}
}

// routeReferenceFiles lists each profile's profile.yaml and every package resource
// the profile renders (instructions, skills, globalInstructions, agentFiles, and role
// instructions), deduplicated and sorted. Unreadable profiles are skipped.
func routeReferenceFiles(profilesRoot, resourceRoot string, names []string) []string {
	profiles, _ := profilemango.LoadProfiles(profilesRoot)
	var files []string
	for _, name := range names {
		files = append(files, filepath.Join(profilesRoot, name, "profile.yaml"))
		resolved, diagnostics := profilemango.Resolve(profiles, name)
		if diagnostics.HasErrors() {
			continue
		}
		for _, resource := range profileResources(resolved) {
			files = append(files, filepath.Join(resourceRoot, filepath.FromSlash(resource)))
		}
	}
	slices.Sort(files)
	return slices.Compact(files)
}

func profileResources(profile profilemango.ResolvedProfile) []string {
	resources := append(slices.Clone(profile.Instructions), profile.Skills...)
	for _, files := range profile.GlobalInstructions {
		for _, fragments := range files {
			resources = append(resources, fragments...)
		}
	}
	for _, files := range profile.AgentFiles {
		for _, resource := range files {
			resources = append(resources, resource)
		}
	}
	for _, role := range profile.Roles {
		if role.Instructions != nil {
			resources = append(resources, *role.Instructions)
		}
	}
	return resources
}

// writeStaleReferences prints, per old value, the file:line of each listed file that
// still names it outside {{route.…}} placeholders. Missing files are skipped.
func writeStaleReferences(output io.Writer, oldValues, files []string, displayRoot string) error {
	var text strings.Builder
	for _, old := range oldValues {
		var hits []string
		for _, file := range files {
			for _, line := range literalLines(file, old) {
				hits = append(hits, fmt.Sprintf("  %s:%d", displayPath(displayRoot, file), line))
			}
		}
		if len(hits) > 0 {
			fmt.Fprintf(&text, "Still names %s:\n%s\n", old, strings.Join(hits, "\n"))
		}
	}
	if _, err := io.WriteString(output, text.String()); err != nil {
		return fmt.Errorf("writing stale references: %w", err)
	}
	return nil
}

// literalLines returns the 1-based lines of path naming value as a whole token.
func literalLines(path, value string) []int {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []int
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(nil, len(data)+1)
	for number := 1; scanner.Scan(); number++ {
		if containsToken(profilemango.WithoutRouteRefs(scanner.Text()), value) {
			lines = append(lines, number)
		}
	}
	return lines
}

// containsToken reports value in line not directly extended by a name character, so
// gpt-6 does not match gpt-6-sol or gpt-6.1 but does match "gpt-6." at a sentence end.
func containsToken(line, value string) bool {
	for offset := 0; ; {
		index := strings.Index(line[offset:], value)
		if index < 0 {
			return false
		}
		start, end := offset+index, offset+index+len(value)
		if !nameByteAt(line, start-1) && !continuesName(line, end) {
			return true
		}
		offset = start + 1
	}
}

func continuesName(line string, index int) bool {
	if index < len(line) && line[index] == '.' {
		return nameByteAt(line, index+1)
	}
	return nameByteAt(line, index)
}

func nameByteAt(line string, index int) bool {
	if index < 0 || index >= len(line) {
		return false
	}
	c := line[index]
	return c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// displayPath names path relative to root when it lies beneath it.
func displayPath(root, path string) string {
	if relative, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	return path
}
