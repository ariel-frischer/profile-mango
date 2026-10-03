package ohmypi

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gopkg.in/yaml.v3"
)

// ModelPreset is the native omp 18.6.0 roles-and-thinking snapshot.
type ModelPreset struct {
	ModelRoles           map[string]string `json:"modelRoles" yaml:"modelRoles"`
	DefaultThinkingLevel string            `json:"defaultThinkingLevel,omitempty" yaml:"defaultThinkingLevel,omitempty"`
}

type PresetChange struct {
	Name   string
	Before string
	After  string
	// Prior preserves the complete entry bytes for release after an explicit override.
	Prior string
}

type PresetPatch struct {
	Content []byte
	Presets []PresetChange
}

// PresetName escapes non-alphanumeric bytes injectively into omp's name grammar.
func PresetName(route string) string {
	var name strings.Builder
	name.WriteString("mango-")
	for _, b := range []byte(route) {
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-' {
			name.WriteByte(b)
		} else {
			fmt.Fprintf(&name, "_%02x", b)
		}
	}
	return name.String()
}

// RoutePreset uses PatchConfig's mapping, filling omitted portable slots with
// the default selector so omp cannot clear a Mango-managed slot on a switch.
func RoutePreset(route profilemango.RouteBinding) (ModelPreset, error) {
	assignments, err := installAssignments(route)
	if err != nil {
		return ModelPreset{}, err
	}
	preset := ModelPreset{ModelRoles: map[string]string{}, DefaultThinkingLevel: route.Effort}
	for _, slots := range RoleSlots {
		for _, slot := range slots {
			preset.ModelRoles[slot] = assignments[0].selector
		}
	}
	for _, assignment := range assignments {
		preset.ModelRoles[assignment.role] = assignment.selector
	}
	return preset, nil
}

// PatchPresets changes only named Mango entries, leaving other preset bytes alone.
func PatchPresets(source []byte, routes map[string]profilemango.RouteBinding) (PresetPatch, error) {
	patch := PresetPatch{Content: source}
	names := make([]string, 0, len(routes))
	for name := range routes {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		preset, err := RoutePreset(routes[name])
		if err != nil {
			return PresetPatch{}, fmt.Errorf("preset route %s: %w", name, err)
		}
		content, change, err := patchPreset(patch.Content, PresetName(name), preset)
		if err != nil {
			return PresetPatch{}, err
		}
		patch.Content = content
		patch.Presets = append(patch.Presets, change)
	}
	return patch, nil
}

func patchPreset(source []byte, name string, preset ModelPreset) ([]byte, PresetChange, error) {
	document, err := parseConfig(source)
	if err != nil {
		return nil, PresetChange{}, err
	}
	after := presetValue(preset)
	change := PresetChange{Name: name, After: after}
	block, found, err := findEntry(document.root, "modelPresets")
	if err != nil {
		return nil, change, err
	}
	ending := lineEnding(source)
	if !found {
		edit := rootInsertion(document, "modelPresets:"+ending+presetLines(name, preset, 2, ending))
		return applyEdits(source, []textEdit{edit}), change, nil
	}
	if block.value.Kind != yaml.MappingNode || block.value.Style&yaml.FlowStyle != 0 {
		return nil, change, fmt.Errorf("modelPresets must be a block mapping")
	}
	entry, exists, err := findEntry(block.value, name)
	if err != nil {
		return nil, change, err
	}
	indent := block.key.Column + 1
	if len(block.value.Content) > 0 {
		indent = block.value.Content[0].Column - 1
	}
	if !exists {
		offset := mappingInsertionOffset(document, block.key.Line, block.key.Column-1)
		edit := textEdit{start: offset, end: offset, content: []byte(linePrefix(source, offset, ending) + presetLines(name, preset, indent, ending))}
		return applyEdits(source, []textEdit{edit}), change, nil
	}
	change.Before, err = nodeValue(entry.value)
	if err != nil {
		return nil, change, err
	}
	start, end, err := presetSpan(document, entry)
	if err != nil {
		return nil, change, err
	}
	change.Prior = string(source[start:end])
	if change.Before == after {
		return source, change, nil
	}
	edit := textEdit{start: start, end: end, content: []byte(presetLines(name, preset, indent, ending))}
	return applyEdits(source, []textEdit{edit}), change, nil
}

func presetLines(name string, preset ModelPreset, indent int, ending string) string {
	pad := strings.Repeat(" ", indent)
	text := pad + name + ":" + ending + pad + "  modelRoles:" + ending
	keys := make([]string, 0, len(preset.ModelRoles))
	for role := range preset.ModelRoles {
		keys = append(keys, role)
	}
	slices.Sort(keys)
	for _, role := range keys {
		text += pad + "    " + role + ": " + yamlString(preset.ModelRoles[role]) + ending
	}
	if preset.DefaultThinkingLevel != "" {
		text += pad + "  defaultThinkingLevel: " + yamlString(preset.DefaultThinkingLevel) + ending
	}
	return text
}

func presetValue(preset ModelPreset) string {
	value := map[string]any{"modelRoles": preset.ModelRoles}
	if preset.DefaultThinkingLevel != "" {
		value["defaultThinkingLevel"] = preset.DefaultThinkingLevel
	}
	bytes, _ := json.Marshal(value)
	return string(bytes)
}

func nodeValue(node *yaml.Node) (string, error) {
	var value any
	if err := node.Decode(&value); err != nil {
		return "", err
	}
	bytes, err := json.Marshal(value)
	return string(bytes), err
}

func presetSpan(document yamlDocument, entry mappingEntry) (int, int, error) {
	last, err := presetLastLine(entry.value)
	if err != nil {
		return 0, 0, err
	}
	if last > len(document.lines) {
		return 0, 0, fmt.Errorf("preset line out of bounds")
	}
	end := len(document.source)
	if last < len(document.lines) {
		end = document.lines[last].start
	}
	return document.lines[entry.key.Line-1].start, end, nil
}

func presetLastLine(node *yaml.Node) (int, error) {
	if node.Kind == yaml.ScalarNode && strings.Contains(node.Value, "\n") {
		return 0, fmt.Errorf("managed preset cannot contain multiline scalars")
	}
	last := node.Line
	for _, child := range node.Content {
		line, err := presetLastLine(child)
		if err != nil {
			return 0, err
		}
		last = max(last, line)
	}
	return last, nil
}

// ReleasePresets restores adopted entries, or removes only recorded owned names.
func ReleasePresets(source []byte, priors map[string]string) ([]byte, []PresetChange, error) {
	var changes []PresetChange
	for _, name := range sortedKeys(priors) {
		document, err := parseConfig(source)
		if err != nil {
			return nil, nil, err
		}
		block, found, err := findEntry(document.root, "modelPresets")
		if err != nil {
			return nil, nil, err
		}
		if !found {
			continue
		}
		entry, found, err := findEntry(block.value, name)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			continue
		}
		before, err := nodeValue(entry.value)
		if err != nil {
			return nil, nil, err
		}
		start, end, err := presetSpan(document, entry)
		if err != nil {
			return nil, nil, err
		}
		edits := []textEdit{{start: start, end: end, content: []byte(priors[name])}}
		if priors[name] == "" && len(block.value.Content) == 2 {
			edits = append(edits, lineDeletion(document, block.key.Line))
		}
		source = applyEdits(source, edits)
		changes = append(changes, PresetChange{Name: name, Before: before})
	}
	return source, changes, nil
}

// ConfigValues reads semantic field values, independent of omp's YAML quoting.
func ConfigValues(source []byte, fields []string) (map[string]string, error) {
	document, err := parseConfig(source)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, field := range fields {
		path, ok := strings.CutPrefix(field, "config.")
		if !ok {
			continue
		}
		node := document.root
		for _, key := range strings.Split(path, ".") {
			entry, found, err := findEntry(node, key)
			if err != nil {
				return nil, err
			}
			if !found {
				node = nil
				break
			}
			node = entry.value
		}
		value := ""
		if node != nil {
			if path == "modelRoles" || strings.HasPrefix(path, "modelPresets.") {
				value, err = nodeValue(node)
			} else if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
				value = node.Value
			} else {
				return nil, fmt.Errorf("%s must be a string", path)
			}
		}
		if err != nil {
			return nil, err
		}
		values[field] = value
	}
	return values, nil
}
