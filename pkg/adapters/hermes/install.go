package hermes

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gopkg.in/yaml.v3"
)

// InstallAdapterVersion identifies the bounded, config-only Hermes installer.
const InstallAdapterVersion = "profilemango.dev/hermes-install/v1alpha1"

// ConfigPatch is a lossless patch of the Hermes config.yaml document.
type ConfigPatch struct {
	Content []byte
	Before  map[string]string
}

type configValues struct {
	provider string
	model    string
	effort   string
}

type configEdit struct {
	start       int
	end         int
	replacement []byte
}

type sourceLayout struct {
	data    []byte
	starts  []int
	newline string
}

type mapField struct {
	key   string
	path  string
	value string
}

// PatchConfig changes only model.provider, model.default, and
// agent.reasoning_effort. Authentication is validated but never emitted.
func PatchConfig(source []byte, route profilemango.RouteBinding) (ConfigPatch, error) {
	values, err := validateInstallRoute(route)
	if err != nil {
		return ConfigPatch{}, err
	}
	if len(source) == 0 {
		return generatedPatch(values), nil
	}
	root, err := parseConfig(source)
	if err != nil {
		return ConfigPatch{}, err
	}
	if root == nil {
		return appendGeneratedPatch(source, values), nil
	}
	layout := newSourceLayout(source)
	edits, before, err := planConfigEdits(source, layout, root, values)
	if err != nil {
		return ConfigPatch{}, err
	}
	return ConfigPatch{Content: applyEdits(source, edits), Before: before}, nil
}

func validateInstallRoute(route profilemango.RouteBinding) (configValues, error) {
	if route.Transport != "native" {
		return configValues{}, fmt.Errorf("hermes config install requires native transport")
	}
	if route.Authentication == "" {
		return configValues{}, fmt.Errorf("hermes config install requires an authentication mode")
	}
	for name, value := range map[string]string{
		"provider": route.Provider,
		"model":    route.Model,
		"effort":   route.Effort,
	} {
		if err := validateRouteValue(name, value); err != nil {
			return configValues{}, err
		}
	}
	if !validEffort(route.Effort) {
		return configValues{}, fmt.Errorf("hermes config install rejects unsupported effort %q", route.Effort)
	}
	return configValues{provider: route.Provider, model: route.Model, effort: route.Effort}, nil
}

func validateRouteValue(name, value string) error {
	if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) {
		return fmt.Errorf("hermes %s contains an unsafe value", name)
	}
	if strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return fmt.Errorf("hermes %s contains an unsafe value", name)
	}
	return nil
}

func generatedPatch(values configValues) ConfigPatch {
	return ConfigPatch{
		Content: generatedConfig(values),
		Before:  map[string]string{},
	}
}

func appendGeneratedPatch(source []byte, values configValues) ConfigPatch {
	layout := newSourceLayout(source)
	content := append([]byte(nil), source...)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, []byte(layout.newline)...)
	}
	content = append(content, generatedConfig(values)...)
	return ConfigPatch{Content: content, Before: map[string]string{}}
}

func generatedConfig(values configValues) []byte {
	return []byte("model:\n  provider: " + yamlString(values.provider) + "\n  default: " + yamlString(values.model) + "\nagent:\n  reasoning_effort: " + yamlString(values.effort) + "\n")
}

func parseConfig(source []byte) (*yaml.Node, error) {
	if bytes.HasPrefix(source, []byte{0xef, 0xbb, 0xbf}) {
		return nil, fmt.Errorf("hermes config must be UTF-8 without a byte-order mark")
	}
	if !utf8.Valid(source) {
		return nil, fmt.Errorf("hermes config is not valid UTF-8")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(source))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("parse Hermes config YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("hermes config contains multiple YAML documents")
		}
		return nil, fmt.Errorf("parse Hermes config documents: %w", err)
	}
	if len(document.Content) == 0 {
		return nil, nil
	}
	if err := validateYAMLNode(&document, "document"); err != nil {
		return nil, err
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("hermes config must have a top-level mapping")
	}
	if root.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("hermes top-level flow mapping is ambiguous")
	}
	return root, nil
}

func validateYAMLNode(node *yaml.Node, path string) error {
	if node == nil {
		return nil
	}
	if node.Tag == "!!null" {
		return fmt.Errorf("hermes config contains a null value at %s", path)
	}
	if node.Kind == yaml.AliasNode || node.Alias != nil {
		return fmt.Errorf("hermes config contains an ambiguous YAML alias at %s", path)
	}
	if node.Anchor != "" {
		return fmt.Errorf("hermes config contains an ambiguous YAML anchor/alias at %s", path)
	}
	if node.Kind != yaml.MappingNode {
		for _, child := range node.Content {
			if err := validateYAMLNode(child, path); err != nil {
				return err
			}
		}
		return nil
	}
	seen := make(map[string]struct{}, len(node.Content)/2)
	for index := 0; index+1 < len(node.Content); index += 2 {
		key, value := node.Content[index], node.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return fmt.Errorf("hermes config contains a non-string mapping key at %s", path)
		}
		if key.Value == "<<" {
			return fmt.Errorf("hermes config contains an ambiguous YAML merge alias at %s", path)
		}
		if _, found := seen[key.Value]; found {
			return fmt.Errorf("hermes config contains a duplicate mapping key %q at %s", key.Value, path)
		}
		seen[key.Value] = struct{}{}
		childPath := key.Value
		if path != "document" {
			childPath = path + "." + key.Value
		}
		if err := validateYAMLNode(value, childPath); err != nil {
			return err
		}
	}
	return nil
}

func planConfigEdits(source []byte, layout sourceLayout, root *yaml.Node, values configValues) ([]configEdit, map[string]string, error) {
	modelKey, modelNode := mappingEntry(root, "model")
	agentKey, agentNode := mappingEntry(root, "agent")
	before := map[string]string{}
	var edits []configEdit
	if modelKey == nil {
		edits = append(edits, configEdit{start: len(source), end: len(source), replacement: generatedSection(layout, "model", []mapField{{key: "provider", value: values.provider}, {key: "default", value: values.model}})})
	} else {
		modelEdits, err := planModelEdits(source, layout, modelKey, modelNode, values, before)
		if err != nil {
			return nil, nil, err
		}
		edits = append(edits, modelEdits...)
	}
	if agentKey == nil {
		edits = append(edits, configEdit{start: len(source), end: len(source), replacement: generatedSection(layout, "agent", []mapField{{key: "reasoning_effort", value: values.effort}})})
	} else {
		agentEdits, err := planMapEdits(source, layout, agentKey, agentNode, []mapField{{key: "reasoning_effort", path: "agent.reasoning_effort", value: values.effort}}, before)
		if err != nil {
			return nil, nil, err
		}
		edits = append(edits, agentEdits...)
	}
	if len(edits) > 1 && edits[len(edits)-1].start == len(source) && edits[len(edits)-2].start == len(source) {
		edits[len(edits)-2].replacement = append(edits[len(edits)-2].replacement, edits[len(edits)-1].replacement...)
		edits = edits[:len(edits)-1]
	}
	return edits, before, nil
}

func planModelEdits(source []byte, layout sourceLayout, key, node *yaml.Node, values configValues, before map[string]string) ([]configEdit, error) {
	if node.Kind == yaml.ScalarNode {
		start, end, err := scalarSpan(source, node)
		if err != nil {
			return nil, err
		}
		before["model.default"] = node.Value
		replacement := []byte("{provider: " + yamlString(values.provider) + ", default: " + yamlString(values.model) + "}")
		return []configEdit{{start: start, end: end, replacement: replacement}}, nil
	}
	if node.Kind != yaml.MappingNode || node.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("hermes model mapping uses ambiguous flow or non-mapping YAML")
	}
	return planMapEdits(source, layout, key, node, []mapField{
		{key: "provider", path: "model.provider", value: values.provider},
		{key: "default", path: "model.default", value: values.model},
	}, before)
}

func planMapEdits(source []byte, layout sourceLayout, parentKey, node *yaml.Node, fields []mapField, before map[string]string) ([]configEdit, error) {
	if node.Kind != yaml.MappingNode || node.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("hermes mapping at %s uses ambiguous flow or non-mapping YAML", parentKey.Value)
	}
	var edits []configEdit
	var missing []mapField
	for _, field := range fields {
		_, value := mappingEntry(node, field.key)
		if value == nil {
			missing = append(missing, field)
			continue
		}
		start, end, err := scalarSpan(source, value)
		if err != nil {
			return nil, fmt.Errorf("patch Hermes %s: %w", field.path, err)
		}
		before[field.path] = value.Value
		edits = append(edits, configEdit{start: start, end: end, replacement: []byte(yamlString(field.value))})
	}
	if len(missing) > 0 {
		insert, err := missingFieldsEdit(layout, parentKey, node, fields, missing)
		if err != nil {
			return nil, err
		}
		edits = append(edits, insert)
	}
	return edits, nil
}

func mappingEntry(node *yaml.Node, wanted string) (*yaml.Node, *yaml.Node) {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == wanted {
			return node.Content[index], node.Content[index+1]
		}
	}
	return nil, nil
}

func missingFieldsEdit(layout sourceLayout, parentKey *yaml.Node, node *yaml.Node, allFields, missing []mapField) (configEdit, error) {
	indent := parentKey.Column + 1
	if len(node.Content) >= 2 {
		indent = node.Content[0].Column
	}
	if parentKey.Line <= 0 || len(node.Content) == 0 || node.Content[0].Line <= parentKey.Line {
		return configEdit{}, fmt.Errorf("hermes mapping %q is not a block mapping", parentKey.Value)
	}
	insertLine := parentKey.Line + 1
	firstExistingIndex, firstExistingLine, lastExistingLine := -1, 0, 0
	for index, field := range allFields {
		key, value := mappingEntry(node, field.key)
		if key == nil || value == nil {
			continue
		}
		if firstExistingIndex < 0 {
			firstExistingIndex, firstExistingLine = index, key.Line
		}
		lastExistingLine = key.Line
	}
	if firstExistingIndex >= 0 {
		firstMissingIndex := len(allFields)
		for index, field := range allFields {
			if _, value := mappingEntry(node, field.key); value == nil && index < firstMissingIndex {
				firstMissingIndex = index
			}
		}
		if firstMissingIndex < firstExistingIndex {
			insertLine = firstExistingLine
		} else {
			insertLine = lastExistingLine + 1
		}
	}
	var builder strings.Builder
	for _, field := range missing {
		builder.WriteString(strings.Repeat(" ", indent-1))
		builder.WriteString(field.key)
		builder.WriteString(": ")
		builder.WriteString(yamlString(field.value))
		builder.WriteString(layout.newline)
	}
	start := layout.lineStart(insertLine)
	return configEdit{start: start, end: start, replacement: []byte(builder.String())}, nil
}

func generatedSection(layout sourceLayout, name string, fields []mapField) []byte {
	var builder strings.Builder
	if len(layout.data) > 0 && layout.data[len(layout.data)-1] != '\n' {
		builder.WriteString(layout.newline)
	}
	builder.WriteString(name)
	builder.WriteString(":")
	builder.WriteString(layout.newline)
	for _, field := range fields {
		builder.WriteString("  ")
		builder.WriteString(field.key)
		builder.WriteString(": ")
		builder.WriteString(yamlString(field.value))
		builder.WriteString(layout.newline)
	}
	return []byte(builder.String())
}

func scalarSpan(source []byte, node *yaml.Node) (int, int, error) {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag == "!!null" {
		return 0, 0, fmt.Errorf("hermes target field must be a non-null scalar")
	}
	if node.Style&(yaml.LiteralStyle|yaml.FoldedStyle|yaml.FlowStyle) != 0 {
		return 0, 0, fmt.Errorf("hermes target field uses unsupported multiline or flow scalar syntax")
	}
	layout := newSourceLayout(source)
	line := layout.lineText(node.Line)
	if line == "" {
		return 0, 0, fmt.Errorf("hermes target field has no source value")
	}
	column := columnOffset(line, node.Column)
	if column >= len(line) {
		return 0, 0, fmt.Errorf("hermes target field has no source value")
	}
	end, err := scalarEnd(line, column)
	if err != nil {
		return 0, 0, err
	}
	return layout.lineStart(node.Line) + column, layout.lineStart(node.Line) + end, nil
}

func scalarEnd(line string, start int) (int, error) {
	switch line[start] {
	case '"':
		return doubleQuotedEnd(line, start)
	case '\'':
		return singleQuotedEnd(line, start)
	}
	end := len(line)
	for index := start; index < len(line); index++ {
		if line[index] == '#' && index > start && unicode.IsSpace(rune(line[index-1])) {
			end = index
			break
		}
	}
	for end > start && (line[end-1] == ' ' || line[end-1] == '\t' || line[end-1] == '\r') {
		end--
	}
	if end == start || strings.ContainsAny(line[start:end], " \t") {
		return 0, fmt.Errorf("hermes target field has ambiguous plain scalar syntax")
	}
	return end, nil
}

func doubleQuotedEnd(line string, start int) (int, error) {
	escaped := false
	for index := start + 1; index < len(line); index++ {
		if line[index] == '"' && !escaped {
			return index + 1, trailingScalarCheck(line, index+1)
		}
		if line[index] == '\\' && !escaped {
			escaped = true
		} else {
			escaped = false
		}
	}
	return 0, fmt.Errorf("hermes target field has an unterminated quoted scalar")
}

func singleQuotedEnd(line string, start int) (int, error) {
	for index := start + 1; index < len(line); index++ {
		if line[index] != '\'' {
			continue
		}
		if index+1 < len(line) && line[index+1] == '\'' {
			index++
			continue
		}
		return index + 1, trailingScalarCheck(line, index+1)
	}
	return 0, fmt.Errorf("hermes target field has an unterminated quoted scalar")
}

func trailingScalarCheck(line string, start int) error {
	trailing := strings.TrimSpace(line[start:])
	if trailing != "" && !strings.HasPrefix(trailing, "#") {
		return fmt.Errorf("hermes target field has trailing ambiguous content")
	}
	return nil
}

func newSourceLayout(data []byte) sourceLayout {
	layout := sourceLayout{data: data, starts: []int{0}, newline: "\n"}
	if bytes.Contains(data, []byte("\r\n")) {
		layout.newline = "\r\n"
	}
	for index, value := range data {
		if value == '\n' && index+1 < len(data) {
			layout.starts = append(layout.starts, index+1)
		}
	}
	return layout
}

func (layout sourceLayout) lineStart(line int) int {
	if line <= 0 || line > len(layout.starts) {
		return len(layout.data)
	}
	return layout.starts[line-1]
}

func (layout sourceLayout) lineText(line int) string {
	start := layout.lineStart(line)
	if start >= len(layout.data) {
		return ""
	}
	end := len(layout.data)
	if next := layout.lineStart(line + 1); next > start && next < end {
		end = next
	}
	lineText := string(layout.data[start:end])
	return strings.TrimSuffix(lineText, "\n")
}

func columnOffset(line string, column int) int {
	if column <= 1 {
		return 0
	}
	position, current := 0, 1
	for position < len(line) && current < column {
		_, size := utf8.DecodeRuneInString(line[position:])
		position += size
		current++
	}
	return position
}

func applyEdits(source []byte, edits []configEdit) []byte {
	result := append([]byte(nil), source...)
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	for _, edit := range edits {
		if edit.start < 0 || edit.end < edit.start || edit.end > len(result) {
			continue
		}
		next := make([]byte, 0, len(result)-edit.end+edit.start+len(edit.replacement))
		next = append(next, result[:edit.start]...)
		next = append(next, edit.replacement...)
		next = append(next, result[edit.end:]...)
		result = next
	}
	return result
}
