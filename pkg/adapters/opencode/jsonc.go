package opencode

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ModelPatch is the result of changing only the top-level JSONC model field.
type JSONCModelPatch struct {
	Content  []byte
	Before   string
	Found    bool
	Inserted bool
}

// JSONCSkillsPatch is the result of adding one or more absolute skill roots.
type JSONCSkillsPatch struct {
	Content  []byte
	Before   []string
	After    []string
	Found    bool
	Inserted bool
}

// PatchModelJSONC validates a top-level JSONC object and changes only its model
// string literal, or inserts a deterministic model property when it is absent.
func PatchModelJSONC(data []byte, model string) (JSONCModelPatch, error) {
	if err := validateModelValue(model); err != nil {
		return JSONCModelPatch{}, err
	}
	scanner, err := newJSONCScanner(data)
	if err != nil {
		return JSONCModelPatch{}, err
	}
	if scanner.model != nil {
		if scanner.model.value == model {
			return JSONCModelPatch{Content: append([]byte(nil), data...), Before: model, Found: true}, nil
		}
		content := replaceJSONCSpan(data, scanner.model.start, scanner.model.end, strconv.Quote(model))
		return JSONCModelPatch{Content: content, Before: scanner.model.value, Found: true}, nil
	}
	return JSONCModelPatch{Content: insertModel(data, scanner, strconv.Quote(model)), Inserted: true}, nil
}

// PatchSkillsJSONC adds absolute skill roots to the top-level skills.paths array.
// It changes only the required array/object span and preserves all other bytes.
func PatchSkillsJSONC(data []byte, paths []string) (JSONCSkillsPatch, error) {
	desired, err := normalizeSkillPaths(paths)
	if err != nil {
		return JSONCSkillsPatch{}, err
	}
	scanner, err := newJSONCScanner(data)
	if err != nil {
		return JSONCSkillsPatch{}, err
	}
	if scanner.skills == nil {
		content := insertAt(data, scanner.rootOpen+1, topLevelSkills(desired)+topLevelSeparator(scanner))
		return JSONCSkillsPatch{Content: content, After: desired, Inserted: true}, nil
	}
	before := []string(nil)
	if scanner.skills.paths != nil {
		before = append(before, scanner.skills.paths.values...)
	}
	after, missing := mergeSkillPaths(before, desired)
	if len(missing) == 0 {
		return JSONCSkillsPatch{Content: append([]byte(nil), data...), Before: before, After: after, Found: true}, nil
	}
	content := insertSkillsPaths(data, scanner.skills, missing)
	return JSONCSkillsPatch{Content: content, Before: before, After: after, Found: scanner.skills.paths != nil, Inserted: scanner.skills.paths == nil}, nil
}

func normalizeSkillPaths(paths []string) ([]string, error) {
	result := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, value := range paths {
		if err := validateSkillPath(value); err != nil {
			return nil, err
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("OpenCode skills.paths requires at least one path")
	}
	return result, nil
}

func validateSkillPath(value string) error {
	if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) {
		return fmt.Errorf("OpenCode skills.paths contains an invalid path")
	}
	for _, character := range value {
		if character < 0x20 {
			return fmt.Errorf("OpenCode skills.paths contains an unsafe control character")
		}
	}
	return nil
}

func mergeSkillPaths(before, desired []string) ([]string, []string) {
	after := append([]string(nil), before...)
	seen := make(map[string]struct{}, len(before)+len(desired))
	for _, value := range before {
		seen[value] = struct{}{}
	}
	missing := make([]string, 0, len(desired))
	for _, value := range desired {
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		after = append(after, value)
		missing = append(missing, value)
	}
	return after, missing
}

func topLevelSkills(paths []string) string {
	return `"skills":{"paths":` + quoteStringArray(paths) + `}`
}

func topLevelSeparator(scanner *jsoncScanner) string {
	if scanner.firstKey >= 0 {
		return ","
	}
	return ""
}

func insertSkillsPaths(data []byte, skills *skillsObjectSpan, missing []string) []byte {
	if skills.paths == nil {
		field := `"paths":` + quoteStringArray(missing)
		separator := ""
		if skills.hasEntries && !skills.trailingComma {
			separator = ","
		}
		return insertAt(data, skills.close, separator+field)
	}
	separator := ""
	if skills.paths.hasEntries && !skills.paths.trailingComma {
		separator = ","
	}
	return insertAt(data, skills.paths.close, separator+quoteStringArrayItems(missing))
}

func quoteStringArray(values []string) string {
	return "[" + quoteStringArrayItems(values) + "]"
}

func quoteStringArrayItems(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = strconv.Quote(value)
	}
	return strings.Join(quoted, ",")
}

// PatchJSONCModel is an alias with the operation words in the opposite order.
func PatchJSONCModel(data []byte, model string) (JSONCModelPatch, error) {
	return PatchModelJSONC(data, model)
}

type modelSpan struct {
	start int
	end   int
	value string
}

type arraySpan struct {
	open          int
	close         int
	values        []string
	hasEntries    bool
	trailingComma bool
}

type skillsObjectSpan struct {
	open          int
	close         int
	paths         *arraySpan
	hasEntries    bool
	trailingComma bool
}

type jsoncScanner struct {
	data      []byte
	pos       int
	rootOpen  int
	firstKey  int
	model     *modelSpan
	skills    *skillsObjectSpan
	triviaErr error
	depth     int
}

func newJSONCScanner(data []byte) (*jsoncScanner, error) {
	if err := validateJSONCBytes(data); err != nil {
		return nil, err
	}
	scanner := &jsoncScanner{data: data, firstKey: -1}
	scanner.skipTrivia()
	if scanner.triviaErr != nil {
		return nil, scanner.triviaErr
	}
	if scanner.pos >= len(data) || data[scanner.pos] != '{' {
		return nil, fmt.Errorf("OpenCode config must be a top-level JSONC object")
	}
	scanner.rootOpen = scanner.pos
	if err := scanner.parseObject(true); err != nil {
		return nil, err
	}
	scanner.skipTrivia()
	if scanner.triviaErr != nil {
		return nil, scanner.triviaErr
	}
	if scanner.pos != len(data) {
		return nil, fmt.Errorf("trailing content after top-level JSONC object")
	}
	return scanner, nil
}

func validateJSONCBytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("OpenCode config is empty")
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return fmt.Errorf("OpenCode config must be UTF-8 without a byte-order mark")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("OpenCode config is not valid UTF-8")
	}
	return nil
}

func validateModelValue(value string) error {
	if value == "" || strings.TrimSpace(value) != value {
		return fmt.Errorf("OpenCode model must be a non-empty string without surrounding whitespace")
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("OpenCode model is not valid UTF-8")
	}
	for _, character := range value {
		if character < 0x20 {
			return fmt.Errorf("OpenCode model contains an unsafe control character")
		}
	}
	return nil
}

func (scanner *jsoncScanner) parseObject(top bool) error {
	if err := scanner.enterContainer(); err != nil {
		return err
	}
	defer scanner.leaveContainer()
	if !scanner.consume('{') {
		return scanner.errorf("expected object")
	}
	keys := map[string]struct{}{}
	scanner.skipTrivia()
	if scanner.consume('}') {
		return nil
	}
	for {
		if err := scanner.parseObjectEntry(top, keys); err != nil {
			return err
		}
		done, err := scanner.finishObjectEntry()
		if done || err != nil {
			return err
		}
	}
}

func (scanner *jsoncScanner) parseObjectEntry(top bool, keys map[string]struct{}) error {
	keyStart := scanner.pos
	key, _, _, err := scanner.parseString()
	if err != nil {
		return err
	}
	if _, found := keys[key]; found {
		return scanner.errorf("duplicate JSONC object key %q", key)
	}
	keys[key] = struct{}{}
	scanner.skipTrivia()
	if !scanner.consume(':') {
		return scanner.errorf("expected colon after object key")
	}
	scanner.skipTrivia()
	if err := scanner.parseObjectValue(top, key); err != nil {
		return err
	}
	if top && scanner.firstKey < 0 {
		scanner.firstKey = keyStart
	}
	return nil
}

func (scanner *jsoncScanner) parseObjectValue(top bool, key string) error {
	if top && key == "skills" {
		return scanner.parseSkillsObject()
	}
	if !top || key != "model" {
		return scanner.parseValue()
	}
	if scanner.pos >= len(scanner.data) || scanner.data[scanner.pos] != '"' {
		return scanner.errorf("top-level model must be a string")
	}
	start := scanner.pos
	value, _, end, err := scanner.parseString()
	if err != nil {
		return err
	}
	scanner.model = &modelSpan{start: start, end: end, value: value}
	return nil
}

func (scanner *jsoncScanner) parseSkillsObject() error {
	if !scanner.consume('{') {
		return scanner.errorf("top-level skills must be an object")
	}
	span := &skillsObjectSpan{open: scanner.pos - 1}
	scanner.skills = span
	keys := map[string]struct{}{}
	scanner.skipTrivia()
	if scanner.consume('}') {
		span.close = scanner.pos - 1
		return nil
	}
	for {
		if err := scanner.parseSkillsEntry(span, keys); err != nil {
			return err
		}
		done, err := scanner.finishSkillsEntry(span)
		if done || err != nil {
			return err
		}
	}
}

func (scanner *jsoncScanner) parseSkillsEntry(span *skillsObjectSpan, keys map[string]struct{}) error {
	key, _, _, err := scanner.parseString()
	if err != nil {
		return err
	}
	if _, found := keys[key]; found {
		return scanner.errorf("duplicate JSONC object key %q", key)
	}
	keys[key] = struct{}{}
	span.hasEntries = true
	scanner.skipTrivia()
	if !scanner.consume(':') {
		return scanner.errorf("expected colon after skills object key")
	}
	scanner.skipTrivia()
	if key == "paths" {
		paths, err := scanner.parseStringArray()
		if err != nil {
			return err
		}
		span.paths = paths
		return nil
	}
	return scanner.parseValue()
}

func (scanner *jsoncScanner) finishSkillsEntry(span *skillsObjectSpan) (bool, error) {
	scanner.skipTrivia()
	if scanner.consume('}') {
		span.close = scanner.pos - 1
		return true, nil
	}
	if !scanner.consume(',') {
		return false, scanner.errorf("expected comma or skills object close")
	}
	span.trailingComma = true
	scanner.skipTrivia()
	if scanner.consume('}') {
		span.close = scanner.pos - 1
		return true, nil
	}
	span.trailingComma = false
	return false, nil
}

func (scanner *jsoncScanner) parseStringArray() (*arraySpan, error) {
	if !scanner.consume('[') {
		return nil, scanner.errorf("OpenCode skills.paths must be an array of strings")
	}
	span := &arraySpan{open: scanner.pos - 1}
	scanner.skipTrivia()
	if scanner.consume(']') {
		span.close = scanner.pos - 1
		return span, nil
	}
	for {
		value, _, _, err := scanner.parseString()
		if err != nil {
			return nil, scanner.errorf("OpenCode skills.paths must contain only strings: %v", err)
		}
		span.values = append(span.values, value)
		span.hasEntries = true
		scanner.skipTrivia()
		if scanner.consume(']') {
			span.close = scanner.pos - 1
			return span, nil
		}
		if !scanner.consume(',') {
			return nil, scanner.errorf("expected comma or skills.paths close")
		}
		span.trailingComma = true
		scanner.skipTrivia()
		if scanner.consume(']') {
			span.close = scanner.pos - 1
			return span, nil
		}
		span.trailingComma = false
	}
}

func (scanner *jsoncScanner) finishObjectEntry() (bool, error) {
	scanner.skipTrivia()
	if scanner.consume('}') {
		return true, nil
	}
	if !scanner.consume(',') {
		return false, scanner.errorf("expected comma or object close")
	}
	scanner.skipTrivia()
	if scanner.consume('}') {
		return true, nil
	}
	return false, nil
}

func (scanner *jsoncScanner) parseArray() error {
	if err := scanner.enterContainer(); err != nil {
		return err
	}
	defer scanner.leaveContainer()
	if !scanner.consume('[') {
		return scanner.errorf("expected array")
	}
	scanner.skipTrivia()
	if scanner.consume(']') {
		return nil
	}
	for {
		if err := scanner.parseValue(); err != nil {
			return err
		}
		scanner.skipTrivia()
		if scanner.consume(']') {
			return nil
		}
		if !scanner.consume(',') {
			return scanner.errorf("expected comma or array close")
		}
		scanner.skipTrivia()
		if scanner.consume(']') {
			return nil
		}
	}
}

func (scanner *jsoncScanner) enterContainer() error {
	scanner.depth++
	if scanner.depth > 256 {
		scanner.depth--
		return scanner.errorf("JSONC nesting exceeds 256 levels")
	}
	return nil
}

func (scanner *jsoncScanner) leaveContainer() {
	scanner.depth--
}
func (scanner *jsoncScanner) parseValue() error {
	if scanner.pos >= len(scanner.data) {
		return scanner.errorf("expected JSONC value")
	}
	switch scanner.data[scanner.pos] {
	case '"':
		_, _, _, err := scanner.parseString()
		return err
	case '{':
		return scanner.parseObject(false)
	case '[':
		return scanner.parseArray()
	case 't':
		return scanner.parseLiteral("true")
	case 'f':
		return scanner.parseLiteral("false")
	case 'n':
		return scanner.parseLiteral("null")
	default:
		return scanner.parseNumber()
	}
}

func (scanner *jsoncScanner) parseLiteral(literal string) error {
	if !bytes.HasPrefix(scanner.data[scanner.pos:], []byte(literal)) {
		return scanner.errorf("invalid JSONC literal")
	}
	scanner.pos += len(literal)
	return nil
}

func (scanner *jsoncScanner) parseNumber() error {
	start := scanner.pos
	if scanner.consume('-') {
		if scanner.pos >= len(scanner.data) {
			return scanner.errorf("invalid JSONC number")
		}
	}
	if scanner.consume('0') {
		if scanner.pos < len(scanner.data) && scanner.data[scanner.pos] >= '0' && scanner.data[scanner.pos] <= '9' {
			return scanner.errorf("invalid JSONC number")
		}
	} else if scanner.consumeDigits() == 0 {
		return scanner.errorf("invalid JSONC number")
	}
	if scanner.consume('.') {
		if scanner.consumeDigits() == 0 {
			return scanner.errorf("invalid JSONC fraction")
		}
	}
	if scanner.pos < len(scanner.data) && (scanner.data[scanner.pos] == 'e' || scanner.data[scanner.pos] == 'E') {
		scanner.pos++
		if scanner.pos < len(scanner.data) && (scanner.data[scanner.pos] == '+' || scanner.data[scanner.pos] == '-') {
			scanner.pos++
		}
		if scanner.consumeDigits() == 0 {
			return scanner.errorf("invalid JSONC exponent")
		}
	}
	if scanner.pos == start {
		return scanner.errorf("invalid JSONC value")
	}
	return nil
}

func (scanner *jsoncScanner) consumeDigits() int {
	start := scanner.pos
	for scanner.pos < len(scanner.data) && scanner.data[scanner.pos] >= '0' && scanner.data[scanner.pos] <= '9' {
		scanner.pos++
	}
	return scanner.pos - start
}

func (scanner *jsoncScanner) parseString() (string, int, int, error) {
	start := scanner.pos
	if !scanner.consume('"') {
		return "", 0, 0, scanner.errorf("expected quoted string")
	}
	var value strings.Builder
	for scanner.pos < len(scanner.data) {
		character := scanner.data[scanner.pos]
		scanner.pos++
		switch character {
		case '"':
			return value.String(), start, scanner.pos, nil
		case '\\':
			if err := scanner.parseEscape(&value); err != nil {
				return "", 0, 0, err
			}
		default:
			if character < 0x20 {
				return "", 0, 0, scanner.errorf("unescaped control character in JSONC string")
			}
			if character < utf8.RuneSelf {
				value.WriteByte(character)
				continue
			}
			scanner.pos--
			runeValue, size := utf8.DecodeRune(scanner.data[scanner.pos:])
			if runeValue == utf8.RuneError && size == 1 {
				return "", 0, 0, scanner.errorf("invalid UTF-8 in JSONC string")
			}
			value.WriteRune(runeValue)
			scanner.pos += size
		}
	}
	return "", 0, 0, scanner.errorf("unterminated JSONC string")
}

func (scanner *jsoncScanner) parseEscape(value *strings.Builder) error {
	if scanner.pos >= len(scanner.data) {
		return scanner.errorf("unterminated JSONC escape")
	}
	escape := scanner.data[scanner.pos]
	scanner.pos++
	switch escape {
	case '"', '\\', '/':
		value.WriteByte(escape)
	case 'b':
		value.WriteByte('\b')
	case 'f':
		value.WriteByte('\f')
	case 'n':
		value.WriteByte('\n')
	case 'r':
		value.WriteByte('\r')
	case 't':
		value.WriteByte('\t')
	case 'u':
		return scanner.parseUnicodeEscape(value)
	default:
		return scanner.errorf("unsupported JSONC escape")
	}
	return nil
}

func (scanner *jsoncScanner) parseUnicodeEscape(value *strings.Builder) error {
	code, err := scanner.readHex4()
	if err != nil {
		return err
	}
	if code >= 0xd800 && code <= 0xdbff {
		if scanner.pos+2 > len(scanner.data) || scanner.data[scanner.pos] != '\\' || scanner.data[scanner.pos+1] != 'u' {
			return scanner.errorf("high surrogate is not paired")
		}
		scanner.pos += 2
		low, err := scanner.readHex4()
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return scanner.errorf("invalid low surrogate")
		}
		value.WriteRune(rune(0x10000 + ((code - 0xd800) << 10) + (low - 0xdc00)))
		return nil
	}
	if code >= 0xdc00 && code <= 0xdfff {
		return scanner.errorf("unpaired low surrogate")
	}
	value.WriteRune(rune(code))
	return nil
}

func (scanner *jsoncScanner) readHex4() (int, error) {
	if scanner.pos+4 > len(scanner.data) {
		return 0, scanner.errorf("short Unicode escape")
	}
	value := 0
	for index := 0; index < 4; index++ {
		digit, ok := hexValue(scanner.data[scanner.pos+index])
		if !ok {
			return 0, scanner.errorf("invalid Unicode escape")
		}
		value = value*16 + digit
	}
	scanner.pos += 4
	return value, nil
}

func hexValue(value byte) (int, bool) {
	switch {
	case value >= '0' && value <= '9':
		return int(value - '0'), true
	case value >= 'a' && value <= 'f':
		return int(value-'a') + 10, true
	case value >= 'A' && value <= 'F':
		return int(value-'A') + 10, true
	default:
		return 0, false
	}
}

func (scanner *jsoncScanner) skipTrivia() {
	for scanner.pos < len(scanner.data) {
		switch scanner.data[scanner.pos] {
		case ' ', '\t', '\r', '\n':
			scanner.pos++
		case '/':
			if scanner.pos+1 >= len(scanner.data) || (scanner.data[scanner.pos+1] != '/' && scanner.data[scanner.pos+1] != '*') {
				return
			}
			if scanner.data[scanner.pos+1] == '/' {
				scanner.pos += 2
				for scanner.pos < len(scanner.data) && scanner.data[scanner.pos] != '\r' && scanner.data[scanner.pos] != '\n' {
					scanner.pos++
				}
				continue
			}
			scanner.pos += 2
			for scanner.pos+1 < len(scanner.data) && (scanner.data[scanner.pos] != '*' || scanner.data[scanner.pos+1] != '/') {
				scanner.pos++
			}
			if scanner.pos+1 < len(scanner.data) {
				scanner.pos += 2
			} else {
				scanner.triviaErr = fmt.Errorf("invalid OpenCode JSONC at byte %d: unterminated block comment", scanner.pos)
				return
			}
		default:
			return
		}
	}
}

func (scanner *jsoncScanner) consume(value byte) bool {
	if scanner.pos >= len(scanner.data) || scanner.data[scanner.pos] != value {
		return false
	}
	scanner.pos++
	return true
}

func (scanner *jsoncScanner) errorf(format string, args ...any) error {
	if scanner.triviaErr != nil {
		return scanner.triviaErr
	}
	return fmt.Errorf("invalid OpenCode JSONC at byte %d: %s", scanner.pos, fmt.Sprintf(format, args...))
}

func replaceJSONCSpan(data []byte, start, end int, replacement string) []byte {
	content := make([]byte, 0, len(data)-end+start+len(replacement))
	content = append(content, data[:start]...)
	content = append(content, replacement...)
	content = append(content, data[end:]...)
	return content
}

func insertModel(data []byte, scanner *jsoncScanner, literal string) []byte {
	field := `"model":` + literal
	if scanner.firstKey >= 0 {
		return insertAt(data, scanner.rootOpen+1, field+",")
	}
	return insertAt(data, scanner.rootOpen+1, field)
}

func insertAt(data []byte, offset int, text string) []byte {
	content := make([]byte, 0, len(data)+len(text))
	content = append(content, data[:offset]...)
	content = append(content, text...)
	content = append(content, data[offset:]...)
	return content
}
