package install

import (
	"fmt"
	"maps"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

// routeRefDiagnostics checks every {{route.…}} placeholder once. Route, role, and
// field names do not depend on the target, so a placeholder that resolves for the
// base route resolves for every target.
func (loaded loadedInput) routeRefDiagnostics() profilemango.Diagnostics {
	_, diagnostics := render.RenderRouteRefs(loaded.Resources, loaded.Bindings, "")
	check := func(path, resource string, content []byte) {
		if _, err := profilemango.RenderRouteRefs(content, loaded.Bindings, ""); err != nil {
			diagnostics.Add(profilemango.SeverityError, render.RouteRefInvalidCode, path, strings.TrimPrefix(resource+": ", ": ")+err.Error(), 0, 0)
		}
	}
	for _, kind := range []struct {
		prefix string
		files  map[string][]globalFile
	}{{"globalInstructions.", loaded.Globals}, {"agentFiles.", loaded.AgentFiles}} {
		for _, target := range sortedNames(kind.files) {
			for _, file := range kind.files[target] {
				check(kind.prefix+target+"."+file.Name, file.Digest.Path, file.Content)
			}
		}
	}
	for _, role := range sortedNames(loaded.RoleInstructions) {
		check("roles."+role+".instructions", "", loaded.RoleInstructions[role])
	}
	return diagnostics
}

// forTarget returns the input with every rendered resource's placeholders resolved
// from target's effective routes, and each digest recomputed on the rendered bytes.
func (loaded loadedInput) forTarget(target string) (loadedInput, error) {
	resources, diagnostics := render.RenderRouteRefs(loaded.Resources, loaded.Bindings, target)
	if err := diagnostics.Err(); err != nil {
		return loadedInput{}, fmt.Errorf("render route placeholders: %w", err)
	}
	loaded.Resources = resources
	var err error
	if loaded.Globals, err = renderFiles(loaded.Globals, loaded.Bindings, target); err != nil {
		return loadedInput{}, err
	}
	if loaded.AgentFiles, err = renderFiles(loaded.AgentFiles, loaded.Bindings, target); err != nil {
		return loadedInput{}, err
	}
	roles := make(map[string][]byte, len(loaded.RoleInstructions))
	for name, content := range loaded.RoleInstructions {
		if roles[name], err = profilemango.RenderRouteRefs(content, loaded.Bindings, target); err != nil {
			return loadedInput{}, fmt.Errorf("render route placeholders in role %s instructions: %w", name, err)
		}
	}
	loaded.RoleInstructions = roles
	return loaded, nil
}

// renderFiles renders placeholders in per-target files without changing the input map.
func renderFiles(files map[string][]globalFile, bindings profilemango.Bindings, target string) (map[string][]globalFile, error) {
	rendered := maps.Clone(files)
	for key, list := range files {
		rendered[key] = make([]globalFile, len(list))
		for index, file := range list {
			content, err := profilemango.RenderRouteRefs(file.Content, bindings, target)
			if err != nil {
				return nil, fmt.Errorf("render route placeholders in %s: %w", file.Digest.Path, err)
			}
			file.Content = content
			file.Digest.SHA256, file.Digest.Size = installfs.Hash(content), int64(len(content))
			rendered[key][index] = file
		}
	}
	return rendered, nil
}
