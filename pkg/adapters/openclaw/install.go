package openclaw

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ConfigPatch changes only the qualified OpenClaw model and thinking fields.
type ConfigPatch struct {
	Content        []byte
	BeforeModel    string
	AfterModel     string
	BeforeThinking string
	AfterThinking  string
}

// PatchConfig preserves unrelated OpenClaw JSON5 bytes while changing the
// default model and thinking level fields.
func PatchConfig(data []byte, route profilemango.RouteBinding) (ConfigPatch, error) {
	model, err := installModel(route)
	if err != nil {
		return ConfigPatch{}, err
	}
	if len(data) == 0 {
		return newConfigPatch(model, route.Effort), nil
	}
	scanner, err := newJSON5Scanner(data)
	if err != nil {
		return ConfigPatch{}, err
	}
	return patchObjects(data, scanner.root, model, route.Effort)
}

func installModel(route profilemango.RouteBinding) (string, error) {
	if route.Transport != "native" {
		return "", fmt.Errorf("OpenClaw install requires native transport")
	}
	if route.Authentication == "" {
		return "", fmt.Errorf("OpenClaw install requires an authentication mode")
	}
	if route.Provider == "" || route.Model == "" {
		return "", fmt.Errorf("OpenClaw install requires provider and model")
	}
	if strings.Contains(route.Provider, "/") || unsafeRoutePart(route.Provider) || unsafeRoutePart(route.Model) {
		return "", fmt.Errorf("OpenClaw provider/model contains unsupported characters")
	}
	if !validInstallThinkingLevel(route.Effort) {
		return "", fmt.Errorf("OpenClaw install rejects unsupported thinking level %q", route.Effort)
	}
	return route.Provider + "/" + route.Model, nil
}

func validInstallThinkingLevel(value string) bool {
	switch value {
	case "off", "minimal", "low", "medium", "high", "xhigh", "adaptive", "max", "ultra":
		return true
	default:
		return false
	}
}

func unsafeRoutePart(value string) bool {
	return strings.IndexFunc(value, func(character rune) bool {
		return character == 0 || unicodeSpaceOrControl(character)
	}) >= 0
}

func unicodeSpaceOrControl(value rune) bool {
	return unicode.IsSpace(value) || unicode.IsControl(value)
}

func newConfigPatch(model, thinking string) ConfigPatch {
	content := []byte("{agents:{defaults:{model:{primary:" + strconv.Quote(model) + "},thinkingDefault:" + strconv.Quote(thinking) + "}}}\n")
	return ConfigPatch{Content: content, AfterModel: model, AfterThinking: thinking}
}

type valueKind uint8

const (
	valueScalar valueKind = iota
	valueString
	valueObject
	valueArray
)

type json5Value struct {
	start       int
	end         int
	kind        valueKind
	stringValue string
	object      *json5Object
}

type json5Object struct {
	open     int
	close    int
	firstKey int
	entries  map[string]json5Value
}

type json5Scanner struct {
	data      []byte
	pos       int
	depth     int
	triviaErr error
	root      *json5Object
}

func newJSON5Scanner(data []byte) (*json5Scanner, error) {
	if err := validateJSON5Bytes(data); err != nil {
		return nil, err
	}
	scanner := &json5Scanner{data: data}
	scanner.skipTrivia()
	if scanner.triviaErr != nil {
		return nil, scanner.triviaErr
	}
	if scanner.pos >= len(data) || data[scanner.pos] != '{' {
		return nil, scanner.errorf("config must be a top-level object")
	}
	root, err := scanner.parseObject()
	if err != nil {
		return nil, err
	}
	scanner.root = root
	scanner.skipTrivia()
	if scanner.triviaErr != nil {
		return nil, scanner.triviaErr
	}
	if scanner.pos != len(data) {
		return nil, scanner.errorf("trailing content after top-level object")
	}
	return scanner, nil
}

func validateJSON5Bytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("OpenClaw config is empty")
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return fmt.Errorf("OpenClaw config must be UTF-8 without a byte-order mark")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("OpenClaw config is not valid UTF-8")
	}
	return nil
}

func (scanner *json5Scanner) parseObject() (*json5Object, error) {
	if err := scanner.enterContainer(); err != nil {
		return nil, err
	}
	defer scanner.leaveContainer()
	open := scanner.pos
	if !scanner.consume('{') {
		return nil, scanner.errorf("expected object")
	}
	object := &json5Object{open: open, firstKey: -1, entries: map[string]json5Value{}}
	scanner.skipTrivia()
	if scanner.consume('}') {
		object.close = scanner.pos - 1
		return object, nil
	}
	for {
		if err := scanner.parseObjectEntry(object); err != nil {
			return nil, err
		}
		scanner.skipTrivia()
		if scanner.consume('}') {
			object.close = scanner.pos - 1
			return object, nil
		}
		if !scanner.consume(',') {
			return nil, scanner.errorf("expected comma or object close")
		}
		scanner.skipTrivia()
		if scanner.consume('}') {
			object.close = scanner.pos - 1
			return object, nil
		}
	}
}

func (scanner *json5Scanner) parseObjectEntry(object *json5Object) error {
	keyStart := scanner.pos
	key, err := scanner.parseKey()
	if err != nil {
		return err
	}
	if _, found := object.entries[key]; found {
		return scanner.errorf("duplicate object key %q", key)
	}
	scanner.skipTrivia()
	if !scanner.consume(':') {
		return scanner.errorf("expected colon after object key")
	}
	scanner.skipTrivia()
	value, err := scanner.parseValue()
	if err != nil {
		return err
	}
	object.entries[key] = value
	if object.firstKey < 0 {
		object.firstKey = keyStart
	}
	return nil
}

func (scanner *json5Scanner) parseKey() (string, error) {
	if scanner.pos >= len(scanner.data) {
		return "", scanner.errorf("expected object key")
	}
	if scanner.data[scanner.pos] == '\'' || scanner.data[scanner.pos] == '"' {
		value, _, _, err := scanner.parseString()
		return value, err
	}
	start := scanner.pos
	if !isIdentifierStart(scanner.data[scanner.pos]) {
		return "", scanner.errorf("object key must be quoted or an identifier")
	}
	scanner.pos++
	for scanner.pos < len(scanner.data) && isIdentifierPart(scanner.data[scanner.pos]) {
		scanner.pos++
	}
	return string(scanner.data[start:scanner.pos]), nil
}

func (scanner *json5Scanner) parseValue() (json5Value, error) {
	start := scanner.pos
	if scanner.pos >= len(scanner.data) {
		return json5Value{}, scanner.errorf("expected value")
	}
	switch scanner.data[scanner.pos] {
	case '\'', '"':
		value, _, end, err := scanner.parseString()
		return json5Value{start: start, end: end, kind: valueString, stringValue: value}, err
	case '{':
		object, err := scanner.parseObject()
		return json5Value{start: start, end: scanner.pos, kind: valueObject, object: object}, err
	case '[':
		if err := scanner.parseArray(); err != nil {
			return json5Value{}, err
		}
		return json5Value{start: start, end: scanner.pos, kind: valueArray}, nil
	default:
		if err := scanner.parseScalar(); err != nil {
			return json5Value{}, err
		}
		return json5Value{start: start, end: scanner.pos, kind: valueScalar}, nil
	}
}

func (scanner *json5Scanner) parseArray() error {
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
		if _, err := scanner.parseValue(); err != nil {
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

func (scanner *json5Scanner) parseScalar() error {
	start := scanner.pos
	if scanner.data[scanner.pos] == '+' || scanner.data[scanner.pos] == '-' {
		scanner.pos++
		if scanner.pos >= len(scanner.data) {
			return scanner.errorf("invalid scalar")
		}
	}
	if bytes.HasPrefix(scanner.data[scanner.pos:], []byte("Infinity")) || bytes.HasPrefix(scanner.data[scanner.pos:], []byte("NaN")) {
		scanner.pos += lenScalarName(scanner.data[scanner.pos:])
		return scanner.finishScalar(start)
	}
	if bytes.HasPrefix(scanner.data[scanner.pos:], []byte("true")) {
		scanner.pos += len("true")
		return scanner.finishScalar(start)
	}
	if bytes.HasPrefix(scanner.data[scanner.pos:], []byte("false")) {
		scanner.pos += len("false")
		return scanner.finishScalar(start)
	}
	if bytes.HasPrefix(scanner.data[scanner.pos:], []byte("null")) {
		scanner.pos += len("null")
		return scanner.finishScalar(start)
	}
	scanner.pos = start
	return scanner.parseNumber()
}

func lenScalarName(data []byte) int {
	if bytes.HasPrefix(data, []byte("Infinity")) {
		return len("Infinity")
	}
	return len("NaN")
}

func (scanner *json5Scanner) finishScalar(start int) error {
	if scanner.pos == start {
		return scanner.errorf("invalid scalar")
	}
	if scanner.pos < len(scanner.data) && !isValueDelimiter(scanner.data[scanner.pos]) {
		return scanner.errorf("invalid scalar")
	}
	return nil
}

func (scanner *json5Scanner) parseNumber() error {
	start := scanner.pos
	if scanner.consume('+') || scanner.consume('-') {
		if scanner.pos >= len(scanner.data) {
			return scanner.errorf("invalid number")
		}
	}
	if scanner.pos+2 <= len(scanner.data) && scanner.data[scanner.pos] == '0' && (scanner.data[scanner.pos+1] == 'x' || scanner.data[scanner.pos+1] == 'X') {
		scanner.pos += 2
		if scanner.consumeHexDigits() == 0 {
			return scanner.errorf("invalid hexadecimal number")
		}
		return scanner.finishNumber(start)
	}
	before := scanner.consumeDigits()
	after := 0
	if scanner.consume('.') {
		after = scanner.consumeDigits()
	}
	if before == 0 && after == 0 {
		return scanner.errorf("invalid number")
	}
	if scanner.pos < len(scanner.data) && (scanner.data[scanner.pos] == 'e' || scanner.data[scanner.pos] == 'E') {
		scanner.pos++
		if scanner.pos < len(scanner.data) && (scanner.data[scanner.pos] == '+' || scanner.data[scanner.pos] == '-') {
			scanner.pos++
		}
		if scanner.consumeDigits() == 0 {
			return scanner.errorf("invalid number exponent")
		}
	}
	return scanner.finishNumber(start)
}

func (scanner *json5Scanner) finishNumber(start int) error {
	if scanner.pos == start || (scanner.pos < len(scanner.data) && !isValueDelimiter(scanner.data[scanner.pos])) {
		return scanner.errorf("invalid number")
	}
	return nil
}

func (scanner *json5Scanner) consumeDigits() int {
	start := scanner.pos
	for scanner.pos < len(scanner.data) && scanner.data[scanner.pos] >= '0' && scanner.data[scanner.pos] <= '9' {
		scanner.pos++
	}
	return scanner.pos - start
}

func (scanner *json5Scanner) consumeHexDigits() int {
	start := scanner.pos
	for scanner.pos < len(scanner.data) && isHex(scanner.data[scanner.pos]) {
		scanner.pos++
	}
	return scanner.pos - start
}

func (scanner *json5Scanner) parseString() (string, int, int, error) {
	start := scanner.pos
	quote := scanner.data[scanner.pos]
	scanner.pos++
	var value strings.Builder
	for scanner.pos < len(scanner.data) {
		character := scanner.data[scanner.pos]
		scanner.pos++
		if character == quote {
			return value.String(), start, scanner.pos, nil
		}
		if character == '\\' {
			if err := scanner.parseEscape(&value); err != nil {
				return "", 0, 0, err
			}
			continue
		}
		if character < 0x20 || character == '\r' || character == '\n' {
			return "", 0, 0, scanner.errorf("unescaped control character in string")
		}
		if character < utf8.RuneSelf {
			value.WriteByte(character)
			continue
		}
		scanner.pos--
		runeValue, size := utf8.DecodeRune(scanner.data[scanner.pos:])
		if runeValue == utf8.RuneError && size == 1 {
			return "", 0, 0, scanner.errorf("invalid UTF-8 in string")
		}
		value.WriteRune(runeValue)
		scanner.pos += size
	}
	return "", 0, 0, scanner.errorf("unterminated string")
}

func (scanner *json5Scanner) parseEscape(value *strings.Builder) error {
	if scanner.pos >= len(scanner.data) {
		return scanner.errorf("unterminated escape")
	}
	escape := scanner.data[scanner.pos]
	scanner.pos++
	switch escape {
	case '\'', '"', '\\', '/':
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
	case 'v':
		value.WriteByte('\v')
	case '0':
		value.WriteByte(0)
	case 'x':
		code, err := scanner.readHex(2)
		if err != nil {
			return err
		}
		value.WriteByte(byte(code))
	case 'u':
		code, err := scanner.readHex(4)
		if err != nil {
			return err
		}
		value.WriteRune(rune(code))
	case '\n':
	case '\r':
		if scanner.pos < len(scanner.data) && scanner.data[scanner.pos] == '\n' {
			scanner.pos++
		}
	default:
		return scanner.errorf("unsupported string escape")
	}
	return nil
}

func (scanner *json5Scanner) readHex(count int) (int, error) {
	if scanner.pos+count > len(scanner.data) {
		return 0, scanner.errorf("short hexadecimal escape")
	}
	value := 0
	for index := 0; index < count; index++ {
		digit, ok := hexValue(scanner.data[scanner.pos+index])
		if !ok {
			return 0, scanner.errorf("invalid hexadecimal escape")
		}
		value = value*16 + digit
	}
	scanner.pos += count
	return value, nil
}

func (scanner *json5Scanner) skipTrivia() {
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
			if scanner.pos+1 >= len(scanner.data) {
				scanner.triviaErr = scanner.errorf("unterminated block comment")
				return
			}
			scanner.pos += 2
		default:
			return
		}
	}
}

func (scanner *json5Scanner) enterContainer() error {
	scanner.depth++
	if scanner.depth > 256 {
		scanner.depth--
		return scanner.errorf("nesting exceeds 256 levels")
	}
	return nil
}

func (scanner *json5Scanner) leaveContainer() { scanner.depth-- }

func (scanner *json5Scanner) consume(value byte) bool {
	if scanner.pos >= len(scanner.data) || scanner.data[scanner.pos] != value {
		return false
	}
	scanner.pos++
	return true
}

func (scanner *json5Scanner) errorf(format string, args ...any) error {
	if scanner.triviaErr != nil {
		return scanner.triviaErr
	}
	return fmt.Errorf("invalid OpenClaw JSON5 at byte %d: %s", scanner.pos, fmt.Sprintf(format, args...))
}

type spanEdit struct {
	start       int
	end         int
	replacement string
}

type insertPlan struct {
	fields   []string
	existing bool
}

type patchBuilder struct {
	data         []byte
	replacements []spanEdit
	inserts      map[int]*insertPlan
}

func newPatchBuilder(data []byte) *patchBuilder {
	return &patchBuilder{data: data, inserts: map[int]*insertPlan{}}
}

func (builder *patchBuilder) replace(start, end int, replacement string) {
	builder.replacements = append(builder.replacements, spanEdit{start: start, end: end, replacement: replacement})
}

func (builder *patchBuilder) insert(object *json5Object, field string) {
	plan := builder.inserts[object.open]
	if plan == nil {
		plan = &insertPlan{existing: len(object.entries) > 0}
		builder.inserts[object.open] = plan
	}
	plan.fields = append(plan.fields, field)
}

func (builder *patchBuilder) build() []byte {
	edits := append([]spanEdit(nil), builder.replacements...)
	for offset, plan := range builder.inserts {
		edits = append(edits, spanEdit{start: offset + 1, end: offset + 1, replacement: renderInsert(plan)})
	}
	sortSpanEdits(edits)
	result := append([]byte(nil), builder.data...)
	for index := len(edits) - 1; index >= 0; index-- {
		edit := edits[index]
		result = replaceSpan(result, edit.start, edit.end, edit.replacement)
	}
	return result
}

func renderInsert(plan *insertPlan) string {
	text := strings.Join(plan.fields, ",")
	if plan.existing {
		text += ","
	}
	return text
}

func sortSpanEdits(edits []spanEdit) {
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].start == edits[j].start {
			return edits[i].end > edits[j].end
		}
		return edits[i].start < edits[j].start
	})
}

func replaceSpan(data []byte, start, end int, replacement string) []byte {
	result := make([]byte, 0, len(data)-end+start+len(replacement))
	result = append(result, data[:start]...)
	result = append(result, replacement...)
	result = append(result, data[end:]...)
	return result
}

func patchObjects(data []byte, root *json5Object, model, thinking string) (ConfigPatch, error) {
	builder := newPatchBuilder(data)
	agents, found := root.entries["agents"]
	if !found {
		builder.insert(root, `agents:{defaults:{model:{primary:`+strconv.Quote(model)+`},thinkingDefault:`+strconv.Quote(thinking)+`}}`)
		return ConfigPatch{Content: builder.build(), AfterModel: model, AfterThinking: thinking}, nil
	}
	if agents.kind != valueObject {
		return ConfigPatch{}, fmt.Errorf("OpenClaw agents must be an object")
	}
	defaults, found := agents.object.entries["defaults"]
	if !found {
		builder.insert(agents.object, `defaults:{model:{primary:`+strconv.Quote(model)+`},thinkingDefault:`+strconv.Quote(thinking)+`}`)
		return ConfigPatch{Content: builder.build(), AfterModel: model, AfterThinking: thinking}, nil
	}
	if defaults.kind != valueObject {
		return ConfigPatch{}, fmt.Errorf("OpenClaw agents.defaults must be an object")
	}
	return patchDefaults(defaults.object, model, thinking, builder)
}

func patchDefaults(defaults *json5Object, model, thinking string, builder *patchBuilder) (ConfigPatch, error) {
	modelBefore, err := patchModel(defaults, model, builder)
	if err != nil {
		return ConfigPatch{}, err
	}
	thinkingBefore, err := patchThinking(defaults, thinking, builder)
	if err != nil {
		return ConfigPatch{}, err
	}
	return ConfigPatch{Content: builder.build(), BeforeModel: modelBefore, AfterModel: model, BeforeThinking: thinkingBefore, AfterThinking: thinking}, nil
}

func patchModel(defaults *json5Object, model string, builder *patchBuilder) (string, error) {
	value, found := defaults.entries["model"]
	if !found {
		builder.insert(defaults, `model:{primary:`+strconv.Quote(model)+`}`)
		return "", nil
	}
	if value.kind == valueString {
		builder.replace(value.start, value.end, `{primary:`+strconv.Quote(model)+`}`)
		return value.stringValue, nil
	}
	if value.kind != valueObject {
		return "", fmt.Errorf("OpenClaw agents.defaults.model must be an object or string")
	}
	primary, found := value.object.entries["primary"]
	if !found {
		builder.insert(value.object, `primary:`+strconv.Quote(model))
		return "", nil
	}
	if primary.kind != valueString {
		return "", fmt.Errorf("OpenClaw agents.defaults.model.primary must be a string")
	}
	if primary.stringValue != model {
		builder.replace(primary.start, primary.end, strconv.Quote(model))
	}
	return primary.stringValue, nil
}

func patchThinking(defaults *json5Object, thinking string, builder *patchBuilder) (string, error) {
	value, found := defaults.entries["thinkingDefault"]
	if !found {
		builder.insert(defaults, `thinkingDefault:`+strconv.Quote(thinking))
		return "", nil
	}
	if value.kind != valueString {
		return "", fmt.Errorf("OpenClaw agents.defaults.thinkingDefault must be a string")
	}
	if !validInstallThinkingLevel(value.stringValue) {
		return "", fmt.Errorf("OpenClaw agents.defaults.thinkingDefault has unsupported value %q", value.stringValue)
	}
	if value.stringValue != thinking {
		builder.replace(value.start, value.end, strconv.Quote(thinking))
	}
	return value.stringValue, nil
}

func isIdentifierStart(value byte) bool {
	return value == '_' || value == '$' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isIdentifierPart(value byte) bool {
	return isIdentifierStart(value) || value >= '0' && value <= '9'
}

func isValueDelimiter(value byte) bool {
	return value == ',' || value == ']' || value == '}' || value == '/' || value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func isHex(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
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
