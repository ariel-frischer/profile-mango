package ohmypi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gopkg.in/yaml.v3"
)

// ConfigPatch is the lossless change set for the Oh My Pi model-role
// selectors that have exact source-native YAML evidence. Each selector carries
// its own `:<effort>` suffix, so no global thinking default is changed. It
// deliberately does not represent credentials, provider options, or policy
// settings.
type ConfigPatch struct {
	Content []byte
	// Roles lists the default role first, then every other role by name.
	Roles []RoleChange
}

// RoleChange is one modelRoles.<Role> selector before and after the patch.
// Before is empty when the role was absent.
type RoleChange struct {
	Role   string
	Before string
	After  string
}

// BuiltInRoles are the non-default model roles defined by the pinned Oh My Pi
// 18.2.6 source (packages/coding-agent/src/config/model-roles.ts MODEL_ROLES).
var BuiltInRoles = []string{"advisor", "commit", "plan", "slow", "smol", "task", "tiny", "vision"}

type roleAssignment struct {
	role     string
	selector string
}

// PatchConfig sets modelRoles.default and every route role to a
// `provider/model[:effort]` selector while preserving unrelated YAML bytes and keys.
func PatchConfig(source []byte, route profilemango.RouteBinding) (ConfigPatch, error) {
	assignments, err := installAssignments(route)
	if err != nil {
		return ConfigPatch{}, err
	}
	document, err := parseConfig(source)
	if err != nil {
		return ConfigPatch{}, err
	}
	edits, changes, err := patchModelRoles(document, assignments)
	if err != nil {
		return ConfigPatch{}, err
	}
	return ConfigPatch{Content: applyEdits(source, edits), Roles: changes}, nil
}

// RoleSelector formats an Oh My Pi model-role selector. The pinned source
// splits a trailing `:<thinking level>` off role values (model-selector.ts
// splitThinkingSuffix), and an explicit default-role suffix takes precedence over
// defaultThinkingLevel (sdk.ts pickInitialThinkingLevel).
func RoleSelector(provider, model, effort string) string {
	if effort == "" {
		return provider + "/" + model
	}
	return provider + "/" + model + ":" + effort
}

// SplitRoleSelector reverses RoleSelector for display: it separates a
// trailing supported thinking level from the model selector.
func SplitRoleSelector(selector string) (string, string) {
	colon := strings.LastIndex(selector, ":")
	if colon < 0 || !validThinkingLevel(selector[colon+1:]) {
		return selector, ""
	}
	return selector[:colon], selector[colon+1:]
}

func installAssignments(route profilemango.RouteBinding) ([]roleAssignment, error) {
	selector, err := validateInstallRoute(route)
	if err != nil {
		return nil, err
	}
	assignments := []roleAssignment{{role: profilemango.ReservedRoleDefault, selector: selector}}
	for _, name := range route.SortedRoleNames() {
		selector, err := roleSelector(name, route.Roles[name])
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, roleAssignment{role: name, selector: selector})
	}
	return assignments, nil
}

func validateInstallRoute(route profilemango.RouteBinding) (string, error) {
	if route.Transport != "native" {
		return "", fmt.Errorf("oh my pi install requires native transport")
	}
	if route.Provider == "" || route.Model == "" {
		return "", fmt.Errorf("oh my pi install requires provider and model")
	}
	if route.Authentication == "" {
		return "", fmt.Errorf("oh my pi install requires an authentication mode")
	}
	if err := validateSelectorParts(route.Provider, route.Model); err != nil {
		return "", err
	}
	if !validThinkingLevel(route.Effort) {
		return "", fmt.Errorf("oh my pi install rejects unsupported thinking level %q", route.Effort)
	}
	return RoleSelector(route.Provider, route.Model, route.Effort), nil
}

func roleSelector(name string, role profilemango.RoleRoute) (string, error) {
	if !slices.Contains(BuiltInRoles, name) {
		return "", fmt.Errorf("oh my pi install rejects role %q; Oh My Pi %s built-in roles are %s", name, TargetVersion, strings.Join(BuiltInRoles, ", "))
	}
	if role.Provider == "" || role.Model == "" {
		return "", fmt.Errorf("oh my pi role %q requires provider and model", name)
	}
	if err := validateSelectorParts(role.Provider, role.Model); err != nil {
		return "", fmt.Errorf("oh my pi role %q: %w", name, err)
	}
	if role.Effort != "" && !validThinkingLevel(role.Effort) {
		return "", fmt.Errorf("oh my pi role %q rejects unsupported thinking level %q", name, role.Effort)
	}
	if role.Effort == "" && ambiguousThinkingSuffix(role.Model) {
		return "", fmt.Errorf("oh my pi role %q model %q ends in a thinking-level suffix; set effort explicitly", name, role.Model)
	}
	return RoleSelector(role.Provider, role.Model, role.Effort), nil
}

func validateSelectorParts(provider, model string) error {
	if unsafeRoutePart(provider) || unsafeRoutePart(model) || strings.Contains(provider, "/") {
		return fmt.Errorf("oh my pi provider/model contains unsupported characters")
	}
	return nil
}

// thinkingSelectors are every value the pinned selector parser accepts as a
// thinking suffix, including by unambiguous prefix (tui/src/thinking.ts).
var thinkingSelectors = []string{"inherit", "off", "minimal", "low", "medium", "high", "xhigh", "max", "auto"}

// ambiguousThinkingSuffix reports a bare model whose last `:` segment Oh My
// Pi would read as a thinking level instead of part of the model id.
func ambiguousThinkingSuffix(model string) bool {
	colon := strings.LastIndex(model, ":")
	if colon < 0 {
		return false
	}
	suffix := model[colon+1:]
	for _, selector := range thinkingSelectors {
		if suffix == selector || len(suffix) >= 2 && strings.HasPrefix(selector, suffix) {
			return true
		}
	}
	return false
}

func patchModelRoles(document yamlDocument, assignments []roleAssignment) ([]textEdit, []RoleChange, error) {
	changes := make([]RoleChange, 0, len(assignments))
	for _, assignment := range assignments {
		changes = append(changes, RoleChange{Role: assignment.role, After: assignment.selector})
	}
	roles, found, err := findEntry(document.root, "modelRoles")
	if err != nil {
		return nil, nil, err
	}
	ending := lineEnding(document.source)
	if !found {
		content := "modelRoles:" + ending + roleLines(assignments, 2, ending)
		return []textEdit{rootInsertion(document, content)}, changes, nil
	}
	if roles.value.Kind != yaml.MappingNode || roles.value.Style&yaml.FlowStyle != 0 {
		return nil, nil, fmt.Errorf("modelRoles must be a block mapping")
	}
	edits, err := patchExistingRoles(document, roles, assignments, changes)
	return edits, changes, err
}

// patchExistingRoles replaces present role scalars in place, recording their
// prior values in changes, and inserts absent roles as one block.
func patchExistingRoles(document yamlDocument, roles mappingEntry, assignments []roleAssignment, changes []RoleChange) ([]textEdit, error) {
	var edits []textEdit
	var missing []roleAssignment
	for index, assignment := range assignments {
		entry, found, err := findEntry(roles.value, assignment.role)
		if err != nil {
			return nil, err
		}
		if !found {
			missing = append(missing, assignment)
			continue
		}
		edit, before, err := replaceScalar(document, entry.value, assignment.selector)
		if err != nil {
			return nil, fmt.Errorf("patch modelRoles.%s: %w", assignment.role, err)
		}
		edits, changes[index].Before = append(edits, edit), before
	}
	if len(missing) == 0 {
		return edits, nil
	}
	edit, err := insertRoles(document, roles, missing)
	return append(edits, edit), err
}

func roleLines(assignments []roleAssignment, indent int, ending string) string {
	var builder strings.Builder
	for _, assignment := range assignments {
		builder.WriteString(strings.Repeat(" ", indent) + assignment.role + ": " + yamlString(assignment.selector) + ending)
	}
	return builder.String()
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
			return yamlDocument{}, fmt.Errorf("oh my pi config must contain one YAML document")
		}
		return yamlDocument{}, fmt.Errorf("parse Oh My Pi YAML: %w", err)
	}
	if len(parsed.Content) == 0 {
		return document, nil
	}
	root := parsed.Content[0]
	if root.Kind != yaml.MappingNode || root.Style&yaml.FlowStyle != 0 {
		return yamlDocument{}, fmt.Errorf("oh my pi config root must be a block mapping")
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
		return fmt.Errorf("oh my pi config aliases are unsupported at %s", path)
	}
	if node.Anchor != "" {
		return fmt.Errorf("oh my pi config anchors are unsupported at %s", path)
	}
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			if key.Value == "<<" {
				return fmt.Errorf("oh my pi config merge keys are unsupported at %s", joinPath(path, key.Value))
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
				return fmt.Errorf("oh my pi config has a non-scalar mapping key at %s", path)
			}
			if _, found := seen[key.Value]; found {
				return fmt.Errorf("oh my pi config has duplicate key %q", joinPath(path, key.Value))
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
		return mappingEntry{}, false, fmt.Errorf("oh my pi config expected a mapping for %q", wanted)
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

func insertRoles(document yamlDocument, roles mappingEntry, assignments []roleAssignment) (textEdit, error) {
	indent := roles.key.Column - 1 + 2
	if len(roles.value.Content) > 0 {
		childKey := roles.value.Content[0]
		indent = childKey.Column - 1
		if indent <= roles.key.Column-1 {
			return textEdit{}, fmt.Errorf("modelRoles child indentation is ambiguous")
		}
	}
	offset := mappingInsertionOffset(document, roles.key.Line, roles.key.Column-1)
	ending := lineEnding(document.source)
	content := linePrefix(document.source, offset, ending) + roleLines(assignments, indent, ending)
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
