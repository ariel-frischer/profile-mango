package ohmypi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gopkg.in/yaml.v3"
)

// ConfigPatch is the lossless change set for the two Oh My Pi settings that
// have exact source-native YAML evidence: the default model role and thinking
// level. It deliberately does not represent credentials, provider options, or
// policy settings.
type ConfigPatch struct {
	Content             []byte
	BeforeModel         string
	AfterModel          string
	BeforeThinkingLevel string
	AfterThinkingLevel  string
}

// PatchConfig changes only modelRoles.default and defaultThinkingLevel while
// preserving unrelated YAML bytes and keys.
func PatchConfig(source []byte, route profilemango.RouteBinding) (ConfigPatch, error) {
	model, effort, err := validateInstallRoute(route)
	if err != nil {
		return ConfigPatch{}, err
	}
	document, err := parseConfig(source)
	if err != nil {
		return ConfigPatch{}, err
	}
	patch := ConfigPatch{Content: append([]byte(nil), source...), AfterModel: model, AfterThinkingLevel: effort}
	modelEdit, beforeModel, err := patchModelRole(document, model)
	if err != nil {
		return ConfigPatch{}, err
	}
	patch.BeforeModel = beforeModel
	thinkingEdit, beforeThinking, err := patchThinkingLevel(document, effort)
	if err != nil {
		return ConfigPatch{}, err
	}
	patch.BeforeThinkingLevel = beforeThinking
	edits := appendEdit([]textEdit{modelEdit}, thinkingEdit)
	patch.Content = applyEdits(source, edits)
	return patch, nil
}

func patchModelRole(document yamlDocument, model string) (textEdit, string, error) {
	roles, found, err := findEntry(document.root, "modelRoles")
	if err != nil {
		return textEdit{}, "", err
	}
	if !found {
		return rootInsertion(document, "modelRoles:\n  default: "+yamlString(model)+"\n"), "", nil
	}
	if roles.value.Kind != yaml.MappingNode || roles.value.Style&yaml.FlowStyle != 0 {
		return textEdit{}, "", fmt.Errorf("modelRoles must be a block mapping")
	}
	defaultRole, found, err := findEntry(roles.value, "default")
	if err != nil {
		return textEdit{}, "", err
	}
	if !found {
		edit, err := insertRole(document, roles, "default", model)
		return edit, "", err
	}
	edit, before, err := replaceScalar(document, defaultRole.value, model)
	if err != nil {
		return textEdit{}, "", fmt.Errorf("patch modelRoles.default: %w", err)
	}
	return edit, before, nil
}

func patchThinkingLevel(document yamlDocument, effort string) (textEdit, string, error) {
	thinking, found, err := findEntry(document.root, "defaultThinkingLevel")
	if err != nil {
		return textEdit{}, "", err
	}
	if !found {
		return rootInsertion(document, "defaultThinkingLevel: "+yamlString(effort)+"\n"), "", nil
	}
	edit, before, err := replaceScalar(document, thinking.value, effort)
	if err != nil {
		return textEdit{}, "", fmt.Errorf("patch defaultThinkingLevel: %w", err)
	}
	return edit, before, nil
}

func validateInstallRoute(route profilemango.RouteBinding) (string, string, error) {
	if route.Transport != "native" {
		return "", "", fmt.Errorf("Oh My Pi install requires native transport")
	}
	if route.Provider == "" || route.Model == "" {
		return "", "", fmt.Errorf("Oh My Pi install requires provider and model")
	}
	if route.Authentication == "" {
		return "", "", fmt.Errorf("Oh My Pi install requires an authentication mode")
	}
	if unsafeRoutePart(route.Provider) || unsafeRoutePart(route.Model) || strings.Contains(route.Provider, "/") {
		return "", "", fmt.Errorf("Oh My Pi provider/model contains unsupported characters")
	}
	if !validThinkingLevel(route.Effort) {
		return "", "", fmt.Errorf("Oh My Pi install rejects unsupported thinking level %q", route.Effort)
	}
	return route.Provider + "/" + route.Model, route.Effort, nil
}

func validThinkingLevel(value string) bool {
	switch value {
	case "minimal", "low", "medium", "high", "xhigh", "max", "auto":
		return true
	default:
		return false
	}
}

func unsafeRoutePart(value string) bool {
	return strings.IndexFunc(value, func(character rune) bool {
		return character == 0 || unicode.IsSpace(character) || unicode.IsControl(character)
	}) >= 0
}

type yamlDocument struct {
	source []byte
	lines  []sourceLine
	root   *yaml.Node
}

type sourceLine struct {
	start   int
	end     int
	content []byte
}

type mappingEntry struct {
	key   *yaml.Node
	value *yaml.Node
}

type textEdit struct {
	start   int
	end     int
	content []byte
}

func parseConfig(source []byte) (yamlDocument, error) {
	document := yamlDocument{source: source, lines: splitLines(source)}
	decoder := yaml.NewDecoder(bytes.NewReader(source))
	var parsed yaml.Node
	if err := decoder.Decode(&parsed); err != nil && !errors.Is(err, io.EOF) {
		return yamlDocument{}, fmt.Errorf("parse Oh My Pi YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return yamlDocument{}, fmt.Errorf("Oh My Pi config must contain one YAML document")
		}
		return yamlDocument{}, fmt.Errorf("parse Oh My Pi YAML: %w", err)
	}
	if len(parsed.Content) == 0 {
		return document, nil
	}
	root := parsed.Content[0]
	if root.Kind != yaml.MappingNode || root.Style&yaml.FlowStyle != 0 {
		return yamlDocument{}, fmt.Errorf("Oh My Pi config root must be a block mapping")
	}
	if err := rejectDuplicateKeys(root, ""); err != nil {
		return yamlDocument{}, err
	}
	if err := rejectUnsupportedFeatures(root, ""); err != nil {
		return yamlDocument{}, err
	}
	document.root = root
	return document, nil
}

func rejectUnsupportedFeatures(node *yaml.Node, path string) error {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("Oh My Pi config aliases are unsupported at %s", path)
	}
	if node.Anchor != "" {
		return fmt.Errorf("Oh My Pi config anchors are unsupported at %s", path)
	}
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			if key.Value == "<<" {
				return fmt.Errorf("Oh My Pi config merge keys are unsupported at %s", joinPath(path, key.Value))
			}
			if err := rejectUnsupportedFeatures(key, joinPath(path, key.Value)); err != nil {
				return err
			}
			if err := rejectUnsupportedFeatures(value, joinPath(path, key.Value)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range node.Content {
		if err := rejectUnsupportedFeatures(child, path); err != nil {
			return err
		}
	}
	return nil
}

func rejectDuplicateKeys(node *yaml.Node, path string) error {
	if node == nil || node.Kind == yaml.AliasNode {
		return nil
	}
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]struct{}, len(node.Content)/2)
		for index := 0; index+1 < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			if key.Kind != yaml.ScalarNode {
				return fmt.Errorf("Oh My Pi config has a non-scalar mapping key at %s", path)
			}
			if _, found := seen[key.Value]; found {
				return fmt.Errorf("Oh My Pi config has duplicate key %q", joinPath(path, key.Value))
			}
			seen[key.Value] = struct{}{}
			if err := rejectDuplicateKeys(value, joinPath(path, key.Value)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range node.Content {
		if err := rejectDuplicateKeys(child, path); err != nil {
			return err
		}
	}
	return nil
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func findEntry(mapping *yaml.Node, wanted string) (mappingEntry, bool, error) {
	if mapping == nil {
		return mappingEntry{}, false, nil
	}
	if mapping.Kind != yaml.MappingNode {
		return mappingEntry{}, false, fmt.Errorf("Oh My Pi config expected a mapping for %q", wanted)
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		key, value := mapping.Content[index], mapping.Content[index+1]
		if key.Value == wanted {
			return mappingEntry{key: key, value: value}, true, nil
		}
	}
	return mappingEntry{}, false, nil
}

func replaceScalar(document yamlDocument, node *yaml.Node, value string) (textEdit, string, error) {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return textEdit{}, "", fmt.Errorf("target field must be a string scalar")
	}
	if node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		return textEdit{}, "", fmt.Errorf("target field cannot use a block scalar")
	}
	start, end, raw, err := scalarSpan(document, node)
	if err != nil {
		return textEdit{}, "", err
	}
	if node.Style == 0 && strings.TrimSpace(string(raw)) != node.Value {
		return textEdit{}, "", fmt.Errorf("target field cannot use a multiline scalar")
	}
	before, err := decodeScalar(raw, node.Style)
	if err != nil {
		return textEdit{}, "", err
	}
	return textEdit{start: start, end: end, content: []byte(yamlString(value))}, before, nil
}

func scalarSpan(document yamlDocument, node *yaml.Node) (int, int, []byte, error) {
	if node.Line < 1 || node.Line > len(document.lines) {
		return 0, 0, nil, fmt.Errorf("target field has no source line")
	}
	line := document.lines[node.Line-1]
	offset, err := byteOffsetForColumn(line.content, node.Column)
	if err != nil {
		return 0, 0, nil, err
	}
	start := line.start + offset
	if start < line.start || start >= line.end {
		return 0, 0, nil, fmt.Errorf("target field has no inline scalar")
	}
	end, err := scalarEnd(line.content, offset, node.Style)
	if err != nil {
		return 0, 0, nil, err
	}
	return start, line.start + end, document.source[start : line.start+end], nil
}

func byteOffsetForColumn(line []byte, column int) (int, error) {
	if column < 1 {
		return 0, fmt.Errorf("target field has an invalid source column")
	}
	for offset, runeColumn := 0, 1; offset < len(line); runeColumn++ {
		if runeColumn == column {
			return offset, nil
		}
		_, size := utf8.DecodeRune(line[offset:])
		offset += size
	}
	if column == utf8.RuneCount(line)+1 {
		return len(line), nil
	}
	return 0, fmt.Errorf("target field has no source column")
}

func scalarEnd(line []byte, start int, style yaml.Style) (int, error) {
	if style&yaml.DoubleQuotedStyle != 0 {
		return quotedEnd(line, start, '"')
	}
	if style&yaml.SingleQuotedStyle != 0 {
		return quotedEnd(line, start, '\'')
	}
	for index := start; index < len(line); index++ {
		if line[index] == '#' && (index == start || spaceBefore(line, start, index)) {
			return trimRightSpace(line, start, index), nil
		}
	}
	return trimRightSpace(line, start, len(line)), nil
}

func spaceBefore(line []byte, start, index int) bool {
	if index <= start {
		return true
	}
	runeValue, _ := utf8.DecodeLastRune(line[start:index])
	return unicode.IsSpace(runeValue)
}

func quotedEnd(line []byte, start int, quote byte) (int, error) {
	if start >= len(line) || line[start] != quote {
		return 0, fmt.Errorf("target field has an invalid quoted scalar")
	}
	for index := start + 1; index < len(line); index++ {
		if quote == '\'' && line[index] == quote {
			if index+1 < len(line) && line[index+1] == quote {
				index++
				continue
			}
			return index + 1, nil
		}
		if quote == '"' && line[index] == '\\' {
			index++
			continue
		}
		if line[index] == quote {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("target field has an unterminated quoted scalar")
}

func trimRightSpace(line []byte, start, end int) int {
	for end > start {
		runeValue, size := utf8.DecodeLastRune(line[start:end])
		if !unicode.IsSpace(runeValue) {
			break
		}
		end -= size
	}
	return end
}

func decodeScalar(raw []byte, style yaml.Style) (string, error) {
	if style&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle) != 0 {
		var value string
		if err := yaml.Unmarshal(raw, &value); err != nil {
			return "", fmt.Errorf("decode target scalar: %w", err)
		}
		return value, nil
	}
	return strings.TrimSpace(string(raw)), nil
}

func insertRole(document yamlDocument, roles mappingEntry, key, value string) (textEdit, error) {
	indent := roles.key.Column - 1 + 2
	if len(roles.value.Content) > 0 {
		childKey := roles.value.Content[0]
		indent = childKey.Column - 1
		if indent <= roles.key.Column-1 {
			return textEdit{}, fmt.Errorf("modelRoles child indentation is ambiguous")
		}
	}
	offset := mappingInsertionOffset(document, roles.key.Line, roles.key.Column-1)
	content := linePrefix(document.source, offset, lineEnding(document.source))
	content += strings.Repeat(" ", indent) + key + ": " + yamlString(value) + lineEnding(document.source)
	return textEdit{start: offset, end: offset, content: []byte(content)}, nil
}

func mappingInsertionOffset(document yamlDocument, keyLine, baseIndent int) int {
	for index := keyLine; index < len(document.lines); index++ {
		line := document.lines[index]
		trimmed := bytes.TrimSpace(line.content)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}
		if leadingSpaces(line.content) <= baseIndent {
			return line.start
		}
	}
	return len(document.source)
}

func rootInsertion(document yamlDocument, content string) textEdit {
	offset := len(document.source)
	for _, line := range document.lines {
		if leadingSpaces(line.content) == 0 && isDocumentEnd(line.content) {
			offset = line.start
			break
		}
	}
	content = linePrefix(document.source, offset, lineEnding(document.source)) + content
	return textEdit{start: offset, end: offset, content: []byte(content)}
}

func isDocumentEnd(line []byte) bool {
	trimmed := strings.TrimSpace(string(line))
	if !strings.HasPrefix(trimmed, "...") {
		return false
	}
	rest := strings.TrimSpace(trimmed[3:])
	return rest == "" || strings.HasPrefix(rest, "#")
}

func appendEdit(edits []textEdit, edit textEdit) []textEdit {
	for index := range edits {
		if edits[index].start == edit.start && edits[index].end == edit.end {
			edits[index].content = append(edits[index].content, edit.content...)
			return edits
		}
	}
	return append(edits, edit)
}

func applyEdits(source []byte, edits []textEdit) []byte {
	sort.SliceStable(edits, func(left, right int) bool { return edits[left].start > edits[right].start })
	result := append([]byte(nil), source...)
	for _, edit := range edits {
		updated := make([]byte, 0, len(result)+len(edit.content)-(edit.end-edit.start))
		updated = append(updated, result[:edit.start]...)
		updated = append(updated, edit.content...)
		updated = append(updated, result[edit.end:]...)
		result = updated
	}
	return result
}

func splitLines(source []byte) []sourceLine {
	if len(source) == 0 {
		return nil
	}
	lines := make([]sourceLine, 0, bytes.Count(source, []byte{'\n'})+1)
	start := 0
	for start < len(source) {
		newline := bytes.IndexByte(source[start:], '\n')
		if newline < 0 {
			lines = append(lines, sourceLine{start: start, end: len(source), content: source[start:]})
			break
		}
		end := start + newline
		contentEnd := end
		if contentEnd > start && source[contentEnd-1] == '\r' {
			contentEnd--
		}
		lines = append(lines, sourceLine{start: start, end: contentEnd, content: source[start:contentEnd]})
		start = end + 1
	}
	return lines
}

func leadingSpaces(line []byte) int {
	count := 0
	for count < len(line) && line[count] == ' ' {
		count++
	}
	return count
}

func lineEnding(source []byte) string {
	if bytes.Contains(source, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func linePrefix(source []byte, offset int, ending string) string {
	if offset == 0 || source[offset-1] == '\n' {
		return ""
	}
	return ending
}
