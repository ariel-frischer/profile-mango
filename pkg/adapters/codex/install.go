package codex

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const maxConfigBytes = 8 << 20

// ConfigFieldChange describes one native root configuration field change.
type ConfigFieldChange struct {
	Key    string
	Before string
	After  string
}

// ConfigPatch contains a byte-preserving Codex config update.
type ConfigPatch struct {
	Content []byte
	Fields  []ConfigFieldChange
}

// PatchConfig changes only the supported Codex root route fields.
func PatchConfig(source []byte, route profilemango.RouteBinding) (ConfigPatch, error) {
	values, err := validateInstallRoute(route)
	if err != nil {
		return ConfigPatch{}, err
	}
	if len(source) > maxConfigBytes {
		return ConfigPatch{}, fmt.Errorf("codex config exceeds %d-byte limit", maxConfigBytes)
	}
	if err := validateConfigBytes(source); err != nil {
		return ConfigPatch{}, err
	}
	if err := validateTOML(source); err != nil {
		return ConfigPatch{}, err
	}
	document, err := scanConfig(source)
	if err != nil {
		return ConfigPatch{}, err
	}
	patch := applyConfigPatch(source, document, values)
	if err := validateTOML(patch.Content); err != nil {
		return ConfigPatch{}, fmt.Errorf("validate patched config: %w", err)
	}
	return patch, nil
}

func validateTOML(data []byte) error {
	var decoded map[string]any
	if err := toml.Unmarshal(data, &decoded); err != nil {
		var parseError toml.ParseError
		if errors.As(err, &parseError) && parseError.Position.Line > 0 {
			return fmt.Errorf("invalid codex TOML at line %d; correct syntax before installing", parseError.Position.Line)
		}
		return fmt.Errorf("invalid codex TOML; correct syntax before installing")
	}
	return nil
}

func validateInstallRoute(route profilemango.RouteBinding) (map[string]string, error) {
	if route.Transport != "native" {
		return nil, fmt.Errorf("codex install requires native transport")
	}
	if route.Authentication != "oauth" {
		return nil, fmt.Errorf("codex install requires oauth authentication; credentials remain target-owned")
	}
	if route.Provider != "openai" {
		return nil, fmt.Errorf("codex install supports only the built-in openai provider")
	}
	if !validCodexValue(route.Model) {
		return nil, fmt.Errorf("codex model must be a non-empty safe string")
	}
	if !slices.Contains(installEfforts, route.Effort) {
		return nil, fmt.Errorf("codex install supports only source-qualified reasoning efforts %s, not %q", strings.Join(installEfforts, ", "), route.Effort)
	}
	return map[string]string{
		"model_provider":         route.Provider,
		"model":                  route.Model,
		"model_reasoning_effort": route.Effort,
	}, nil
}

// installEfforts are the model_reasoning_effort values rust-v0.154.0
// (codex-rs/protocol/src/openai_models.rs ReasoningEffort::from_str) parses as named
// variants and that the install qualifies. Whether a given model accepts a level
// remains model-dependent and unverified.
var installEfforts = []string{"none", "minimal", "low", "medium", "high", "xhigh"}

func validCodexValue(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

type configDocument struct {
	assignments map[string]configAssignment
	rootKeys    map[string]struct{}
	tableNames  map[string]struct{}
	arrayCounts map[string]int
	firstTable  int
	newline     string
}

// tableScope records a table header and returns the key prefix for its
// assignments. Each [[array]] element gets its own name[index] scope, so the
// same key in two elements is not a duplicate; "[" is never bare, so scopes
// cannot alias a bare path.
func (document configDocument) tableScope(name string, array bool) string {
	document.tableNames[name] = struct{}{}
	if !array {
		return name
	}
	index := document.arrayCounts[name]
	document.arrayCounts[name] = index + 1
	return fmt.Sprintf("%s[%d]", name, index)
}

type configAssignment struct {
	key       string
	value     string
	valueFrom int
	valueTo   int
}

func scanConfig(data []byte) (configDocument, error) {
	if err := validateConfigBytes(data); err != nil {
		return configDocument{}, err
	}
	document := configDocument{
		assignments: make(map[string]configAssignment),
		rootKeys:    make(map[string]struct{}),
		tableNames:  make(map[string]struct{}),
		arrayCounts: make(map[string]int),
		firstTable:  -1,
		newline:     lineEnding(data),
	}
	table := ""
	for offset := 0; offset < len(data); {
		lineEnd, next := nextLine(data, offset)
		start := skipSpace(data[offset:lineEnd]) + offset
		if start == lineEnd || data[start] == '#' {
			offset = next
			continue
		}
		if data[start] == '[' {
			parsed, array, err := parseTableHeader(data[start:lineEnd])
			if err != nil {
				return configDocument{}, err
			}
			table = document.tableScope(parsed, array)
			if document.firstTable < 0 {
				document.firstTable = offset
			}
			offset = next
			continue
		}
		assignment, statementEnd, err := parseAssignment(data, offset, lineEnd, table)
		if err != nil {
			return configDocument{}, err
		}
		if table == "" && isInstallKey(assignment.key) && isMultilineString(data, assignment.valueFrom) {
			return configDocument{}, fmt.Errorf("codex target field %q does not support multiline strings", assignment.key)
		}
		if table == "" && isInstallKey(assignment.key) && data[assignment.valueFrom] != '"' && data[assignment.valueFrom] != '\'' {
			return configDocument{}, fmt.Errorf("codex target field %q must be a string", assignment.key)
		}
		if _, found := document.assignments[assignment.key]; found {
			return configDocument{}, fmt.Errorf("duplicate codex TOML key")
		}
		document.assignments[assignment.key] = assignment
		if table == "" {
			document.rootKeys[assignment.key] = struct{}{}
		}
		offset = statementEnd
	}
	if err := validateInstallPrecedence(document); err != nil {
		return configDocument{}, err
	}
	return document, nil
}

func validateInstallPrecedence(document configDocument) error {
	if hasConfigPath(document.rootKeys, document.tableNames, "profile") {
		return errLegacyProfileSelector
	}
	if hasProviderShadow(document) ||
		hasRootKey(document.rootKeys, "openai_base_url") {
		return fmt.Errorf("codex config contains provider override state that may shadow the built-in openai provider")
	}
	return nil
}

func hasProviderShadow(document configDocument) bool {
	if hasRootKey(document.rootKeys, "model_providers") {
		return true
	}
	for key := range document.assignments {
		if isOpenAIProviderPath(key) {
			return true
		}
	}
	for key := range document.rootKeys {
		if isOpenAIProviderPath(key) {
			return true
		}
	}
	for table := range document.tableNames {
		if table == "model_providers" || isOpenAIProviderPath(table) {
			return true
		}
	}
	return false
}

// isOpenAIProviderPath reports whether a canonical path (see canonicalDottedKey)
// addresses the built-in openai provider under model_providers.
func isOpenAIProviderPath(path string) bool {
	rest, found := strings.CutPrefix(path, "model_providers.")
	if !found {
		return false
	}
	provider, _, _ := strings.Cut(rest, ".")
	return provider == "openai"
}

func hasConfigPath(rootKeys, tableNames map[string]struct{}, path string) bool {
	for key := range rootKeys {
		if configPathMatches(key, path) {
			return true
		}
	}
	for table := range tableNames {
		if configPathMatches(table, path) {
			return true
		}
	}
	return false
}

func configPathMatches(candidate, expected string) bool {
	return candidate == expected || strings.HasPrefix(candidate, expected+".")
}

func hasRootKey(rootKeys map[string]struct{}, key string) bool {
	_, found := rootKeys[key]
	return found
}

func isInstallKey(key string) bool {
	switch key {
	case "model_provider", "model", "model_reasoning_effort":
		return true
	default:
		return false
	}
}

func isMultilineString(data []byte, offset int) bool {
	return offset+2 < len(data) && ((data[offset] == '"' && data[offset+1] == '"' && data[offset+2] == '"') || (data[offset] == '\'' && data[offset+1] == '\'' && data[offset+2] == '\''))
}

func validateConfigBytes(data []byte) error {
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return fmt.Errorf("codex config must be UTF-8 without a byte-order mark")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("codex config is not valid UTF-8")
	}
	return nil
}

func lineEnding(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func nextLine(data []byte, offset int) (int, int) {
	lineEnd := bytes.IndexByte(data[offset:], '\n')
	if lineEnd < 0 {
		return len(data), len(data)
	}
	lineEnd += offset
	return lineEnd, lineEnd + 1
}

func skipSpace(data []byte) int {
	index := 0
	for index < len(data) && (data[index] == ' ' || data[index] == '\t' || data[index] == '\r') {
		index++
	}
	return index
}

func parseTableHeader(line []byte) (string, bool, error) {
	trimmed := strings.TrimSpace(string(stripComment(line)))
	if strings.HasPrefix(trimmed, "[[") && strings.HasSuffix(trimmed, "]]") {
		name, err := nonEmptyTableName(trimmed[2 : len(trimmed)-2])
		return name, true, err
	}
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		name, err := nonEmptyTableName(trimmed[1 : len(trimmed)-1])
		return name, false, err
	}
	return "", false, fmt.Errorf("invalid Codex TOML table header")
}

func nonEmptyTableName(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("invalid Codex TOML table header")
	}
	name, err := canonicalDottedKey(raw)
	if err != nil {
		return "", fmt.Errorf("invalid Codex TOML table header: %w", err)
	}
	return name, nil
}

// canonicalDottedKey parses a TOML key (bare, quoted, or dotted with optional
// whitespace around dots) into one canonical path. Bare-valid components are
// written as-is and every other component is Go-quoted, so a quoted key such
// as projects."/home/u" can never collide with a bare path Mango inspects.
// Errors never echo target-owned key text.
func canonicalDottedKey(raw string) (string, error) {
	components, err := splitDottedKey(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	for index, component := range components {
		if !validBareConfigComponent(component) {
			components[index] = strconv.Quote(component)
		}
	}
	return strings.Join(components, "."), nil
}

func splitDottedKey(rest string) ([]string, error) {
	var components []string
	for {
		component, remainder, err := readKeyComponent(rest)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
		remainder = strings.TrimLeft(remainder, " \t")
		if remainder == "" {
			return components, nil
		}
		if remainder[0] != '.' {
			return nil, errInvalidConfigKey
		}
		rest = strings.TrimLeft(remainder[1:], " \t")
	}
}

var errInvalidConfigKey = errors.New("invalid codex TOML key")

func readKeyComponent(text string) (string, string, error) {
	switch {
	case strings.HasPrefix(text, `"`):
		return readBasicKeyString(text)
	case strings.HasPrefix(text, "'"):
		end := strings.IndexByte(text[1:], '\'')
		if end < 0 {
			return "", "", errInvalidConfigKey
		}
		return text[1 : end+1], text[end+2:], nil
	}
	end := strings.IndexFunc(text, func(character rune) bool { return !validBareConfigComponent(string(character)) })
	if end < 0 {
		end = len(text)
	}
	if end == 0 {
		return "", "", errInvalidConfigKey
	}
	return text[:end], text[end:], nil
}

func readBasicKeyString(text string) (string, string, error) {
	for index := 1; index < len(text); index++ {
		switch text[index] {
		case '\\':
			index++
		case '"':
			value, err := strconv.Unquote(text[:index+1])
			if err != nil {
				return "", "", errInvalidConfigKey
			}
			return value, text[index+1:], nil
		}
	}
	return "", "", errInvalidConfigKey
}

func validBareConfigComponent(component string) bool {
	if component == "" {
		return false
	}
	for _, character := range component {
		switch {
		case character >= 'a' && character <= 'z', character >= 'A' && character <= 'Z', character >= '0' && character <= '9', character == '_', character == '-':
		default:
			return false
		}
	}
	return true
}

func stripComment(line []byte) []byte {
	quote := byte(0)
	escaped := false
	for index, character := range line {
		if quote == '"' && escaped {
			escaped = false
			continue
		}
		if quote == '"' && character == '\\' {
			escaped = true
			continue
		}
		if (character == '"' || character == '\'') && quote == 0 {
			quote = character
			continue
		}
		if character == quote && quote != 0 {
			quote = 0
			continue
		}
		if character == '#' && quote == 0 {
			return line[:index]
		}
	}
	return line
}

func parseAssignment(data []byte, offset, lineEnd int, table string) (configAssignment, int, error) {
	keyEnd, err := findEquals(data[offset:lineEnd])
	if err != nil {
		return configAssignment{}, 0, err
	}
	key, err := normalizeKey(data[offset+skipSpace(data[offset:offset+keyEnd]) : offset+keyEnd])
	if err != nil {
		return configAssignment{}, 0, err
	}
	valueFrom := offset + keyEnd + 1
	valueFrom = skipValueSpace(data, valueFrom, lineEnd)
	if valueFrom >= lineEnd || data[valueFrom] == '#' {
		return configAssignment{}, 0, fmt.Errorf("codex TOML key has no value")
	}
	value, valueTo, statementEnd, err := parseValue(data, valueFrom, lineEnd)
	if err != nil {
		return configAssignment{}, 0, err
	}
	if table != "" {
		key = table + "." + key
	}
	return configAssignment{key: key, value: value, valueFrom: valueFrom, valueTo: valueTo}, statementEnd, nil
}

func findEquals(line []byte) (int, error) {
	quote := byte(0)
	escaped := false
	for index, character := range line {
		if quote == '"' && escaped {
			escaped = false
			continue
		}
		if quote == '"' && character == '\\' {
			escaped = true
			continue
		}
		if (character == '"' || character == '\'') && quote == 0 {
			quote = character
			continue
		}
		if character == quote && quote != 0 {
			quote = 0
			continue
		}
		if character == '=' && quote == 0 {
			return index, nil
		}
	}
	return 0, fmt.Errorf("codex TOML line is missing an equals sign")
}

func normalizeKey(raw []byte) (string, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return "", fmt.Errorf("codex TOML key is empty")
	}
	return canonicalDottedKey(string(raw))
}

func skipValueSpace(data []byte, offset, lineEnd int) int {
	for offset < lineEnd && (data[offset] == ' ' || data[offset] == '\t') {
		offset++
	}
	return offset
}

func parseValue(data []byte, start, lineEnd int) (string, int, int, error) {
	switch data[start] {
	case '"':
		return parseQuotedValue(data, start, lineEnd, '"')
	case '\'':
		return parseQuotedValue(data, start, lineEnd, '\'')
	case '[', '{':
		return parseDelimitedValue(data, start, lineEnd)
	default:
		return parseBareValue(data, start, lineEnd)
	}
}

func parseQuotedValue(data []byte, start, lineEnd int, quote byte) (string, int, int, error) {
	triple := start+2 < len(data) && data[start : start+3][0] == quote && data[start : start+3][1] == quote && data[start : start+3][2] == quote
	if triple {
		end, statementEnd, err := findTripleQuote(data, start, quote)
		if err != nil {
			return "", 0, 0, err
		}
		return "", end, statementEnd, nil
	}
	end, err := findQuote(data, start, lineEnd, quote)
	if err != nil {
		return "", 0, 0, err
	}
	value, err := decodeQuotedValue(data[start:end], quote)
	if err != nil {
		return "", 0, 0, err
	}
	if hasTrailingContent(data, end, lineEnd) {
		return "", 0, 0, fmt.Errorf("unexpected Codex TOML content after string")
	}
	return value, end, nextStatement(data, end, lineEnd), nil
}

func findQuote(data []byte, start, lineEnd int, quote byte) (int, error) {
	escaped := false
	for index := start + 1; index < lineEnd; index++ {
		character := data[index]
		if quote == '"' && escaped {
			escaped = false
			continue
		}
		if quote == '"' && character == '\\' {
			escaped = true
			continue
		}
		if character == quote {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("unterminated Codex TOML string")
}

func findTripleQuote(data []byte, start int, quote byte) (int, int, error) {
	for index := start + 3; index+2 < len(data); index++ {
		if data[index] == quote && data[index+1] == quote && data[index+2] == quote {
			end := index + 3
			return end, nextStatement(data, end, len(data)), nil
		}
	}
	return 0, 0, fmt.Errorf("unterminated Codex TOML multiline string")
}

func decodeQuotedValue(raw []byte, quote byte) (string, error) {
	if quote == '\'' {
		return string(raw[1 : len(raw)-1]), nil
	}
	value, err := strconv.Unquote(string(raw))
	if err != nil {
		return "", fmt.Errorf("invalid Codex TOML string: %w", err)
	}
	return value, nil
}

func parseDelimitedValue(data []byte, start, lineEnd int) (string, int, int, error) {
	open := data[start]
	close := byte(']')
	if open == '{' {
		close = '}'
	}
	depth := 0
	for index := start; index < len(data); index++ {
		character := data[index]
		if character == '"' || character == '\'' {
			end, err := findQuote(data, index, lineLimit(data, index), character)
			if err != nil {
				return "", 0, 0, err
			}
			index = end - 1
			continue
		}
		if character == '#' {
			for index < len(data) && data[index] != '\n' {
				index++
			}
			continue
		}
		if character == open {
			depth++
		}
		if character == close {
			depth--
			if depth == 0 {
				closeLineEnd, _ := nextLine(data, index)
				if hasTrailingContent(data, index+1, closeLineEnd) {
					return "", 0, 0, fmt.Errorf("unexpected Codex TOML content after value")
				}
				return "", index + 1, nextStatement(data, index+1, closeLineEnd), nil
			}
		}
	}
	return "", 0, 0, fmt.Errorf("unterminated Codex TOML value")
}

func parseBareValue(data []byte, start, lineEnd int) (string, int, int, error) {
	end := start
	for end < lineEnd && data[end] != '#' && data[end] != ' ' && data[end] != '\t' && data[end] != '\r' {
		end++
	}
	if end == start {
		return "", 0, 0, fmt.Errorf("codex TOML value is empty")
	}
	return "", end, nextStatement(data, end, lineEnd), nil
}

func lineLimit(data []byte, offset int) int {
	lineEnd, _ := nextLine(data, offset)
	return lineEnd
}

func hasTrailingContent(data []byte, offset, lineEnd int) bool {
	for offset < lineEnd && (data[offset] == ' ' || data[offset] == '\t' || data[offset] == '\r') {
		offset++
	}
	return offset < lineEnd && data[offset] != '#'
}

func nextStatement(data []byte, end, lineEnd int) int {
	for end < lineEnd && (data[end] == ' ' || data[end] == '\t' || data[end] == '\r') {
		end++
	}
	if end < lineEnd && data[end] != '#' {
		return end
	}
	_, next := nextLine(data, lineEnd)
	return next
}

func applyConfigPatch(source []byte, document configDocument, values map[string]string) ConfigPatch {
	fields := make([]ConfigFieldChange, 0, len(values))
	replacements := make([]configReplacement, 0, len(values))
	missing := make([]string, 0, len(values))
	for _, key := range []string{"model_provider", "model", "model_reasoning_effort"} {
		value := values[key]
		assignment, found := document.assignments[key]
		if !found {
			missing = append(missing, key)
			fields = append(fields, ConfigFieldChange{Key: key, After: value})
			continue
		}
		fields = append(fields, ConfigFieldChange{Key: key, Before: assignment.value, After: value})
		replacements = append(replacements, configReplacement{start: assignment.valueFrom, end: assignment.valueTo, content: tomlString(value)})
	}
	content := replaceConfigSpans(source, replacements)
	if len(missing) > 0 {
		content = insertMissingFields(content, missing, values, document, replacements)
	}
	return ConfigPatch{Content: content, Fields: fields}
}

type configReplacement struct {
	start   int
	end     int
	content string
}

func replaceConfigSpans(source []byte, replacements []configReplacement) []byte {
	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start > replacements[j].start })
	content := append([]byte(nil), source...)
	for _, replacement := range replacements {
		content = append(append(append([]byte(nil), content[:replacement.start]...), replacement.content...), content[replacement.end:]...)
	}
	return content
}

func insertMissingFields(source []byte, missing []string, values map[string]string, document configDocument, replacements []configReplacement) []byte {
	var builder strings.Builder
	for _, key := range missing {
		fmt.Fprintf(&builder, "%s = %s%s", key, tomlString(values[key]), document.newline)
	}
	insert := []byte(builder.String())
	offset := len(source)
	if document.firstTable >= 0 {
		offset = document.firstTable
		for _, replacement := range replacements {
			if replacement.start < offset {
				offset += len(replacement.content) - (replacement.end - replacement.start)
			}
		}
	}
	if offset > 0 && source[offset-1] != '\n' {
		insert = append([]byte(document.newline), insert...)
	}
	return append(append(append([]byte(nil), source[:offset]...), insert...), source[offset:]...)
}
