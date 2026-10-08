package install

import (
	"bytes"
	"fmt"
	"maps"
	"strings"
	"unicode/utf8"

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
				for _, fragment := range file.Fragments {
					check(kind.prefix+target+"."+file.Name, fragment.Path, fragment.Content)
				}
			}
		}
	}
	for _, role := range sortedNames(loaded.RoleInstructions) {
		check("roles."+role+".instructions", "", loaded.RoleInstructions[role])
	}
	for _, bundle := range loaded.Skills {
		for _, file := range bundle.Files {
			if skillFileRenders(file.Content) {
				check(bundle.Dir+"/"+file.Path, "", file.Content)
			}
		}
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
	if loaded.Skills, err = renderSkills(loaded.Skills, loaded.Bindings, target); err != nil {
		return loadedInput{}, err
	}
	return loaded, nil
}

// renderFiles renders placeholders in each fragment of per-target files, then
// recomposes each file, without changing the input map.
func renderFiles(files map[string][]globalFile, bindings profilemango.Bindings, target string) (map[string][]globalFile, error) {
	rendered := maps.Clone(files)
	for key, list := range files {
		rendered[key] = make([]globalFile, len(list))
		for index, file := range list {
			fragments := make([]fileFragment, len(file.Fragments))
			for part, fragment := range file.Fragments {
				content, err := profilemango.RenderRouteRefs(fragment.Content, bindings, target)
				if err != nil {
					return nil, fmt.Errorf("render route placeholders in %s: %w", fragment.Path, err)
				}
				fragments[part] = fileFragment{Path: fragment.Path, Content: content}
			}
			rendered[key][index] = composedFile(file.Name, fragments...)
		}
	}
	return rendered, nil
}

// skillFileRenders reports whether a skill file is UTF-8 text naming a placeholder;
// every other file, binary ones included, is copied verbatim.
func skillFileRenders(content []byte) bool {
	return utf8.Valid(content) && bytes.Contains(content, []byte("{{route."))
}

// renderSkills renders placeholders in each skill's text files for target, without
// changing the input bundles. Provenance was already checked over the source bytes.
func renderSkills(bundles []skillBundle, bindings profilemango.Bindings, target string) ([]skillBundle, error) {
	rendered := make([]skillBundle, len(bundles))
	for index, bundle := range bundles {
		files := make([]skillBundleFile, len(bundle.Files))
		for part, file := range bundle.Files {
			files[part] = file
			if !skillFileRenders(file.Content) {
				continue
			}
			content, err := profilemango.RenderRouteRefs(file.Content, bindings, target)
			if err != nil {
				return nil, fmt.Errorf("render route placeholders in %s/%s: %w", bundle.Dir, file.Path, err)
			}
			files[part].Content = content
		}
		bundle.Files = files
		rendered[index] = bundle
	}
	return rendered, nil
}
