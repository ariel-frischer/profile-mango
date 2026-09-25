package claudecode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ModelPatch is the result of changing one top-level Claude Code settings string.
type ModelPatch struct {
	Content  []byte
	Before   string
	After    string
	Found    bool
	Inserted bool
}

// PatchModel maps a complete route to the only qualified Claude Code setting.
func PatchModel(source []byte, route profilemango.RouteBinding) (ModelPatch, error) {
	model, err := validateInstallRoute(route)
	if err != nil {
		return ModelPatch{}, err
	}
	return PatchModelJSON(source, model)
}

// PatchModelJSON changes only the top-level JSON model string.
func PatchModelJSON(source []byte, model string) (ModelPatch, error) {
	if err := validateModelValue(model); err != nil {
		return ModelPatch{}, err
	}
	if len(source) == 0 {
		content := []byte("{\n  \"model\": " + strconv.Quote(model) + "\n}\n")
		return ModelPatch{Content: content, After: model, Inserted: true}, nil
	}
	scan, err := inspectSettings(source)
	if err != nil {
		return ModelPatch{}, err
	}
	if scan.model != nil {
		if scan.model.value == model {
			return ModelPatch{Content: append([]byte(nil), source...), Before: model, After: model, Found: true}, nil
		}
		content := replaceJSONSpan(source, scan.model.start, scan.model.end, strconv.Quote(model))
		return ModelPatch{Content: content, Before: scan.model.value, After: model, Found: true}, nil
	}
	content := insertTopLevelKey(source, scan, "model", strconv.Quote(model))
	return ModelPatch{Content: content, After: model, Inserted: true}, nil
}

type stringSpan struct {
	start int
	end   int
	value string
}

type settingsScan struct {
	rootOpen int
	hasKeys  bool
	model    *stringSpan
	effort   *stringSpan
}

type settingsScanner struct {
	data []byte
	dec  *json.Decoder
	scan settingsScan
}

func inspectSettings(data []byte) (settingsScan, error) {
	if err := validateSettingsBytes(data); err != nil {
		return settingsScan{}, err
	}
	rootOpen := skipJSONSpace(data, 0)
	if rootOpen >= len(data) || data[rootOpen] != '{' {
		return settingsScan{}, fmt.Errorf("claude code settings must be a top-level JSON object")
	}
	scanner := settingsScanner{data: data, dec: json.NewDecoder(bytes.NewReader(data))}
	scanner.dec.UseNumber()
	token, err := scanner.dec.Token()
	if err != nil || token != json.Delim('{') {
		return settingsScan{}, fmt.Errorf("claude code settings must be a top-level JSON object")
	}
	scanner.scan.rootOpen = rootOpen
	if err := scanner.object(true); err != nil {
		return settingsScan{}, err
	}
	if _, err := scanner.dec.Token(); err != io.EOF {
		if err == nil {
			return settingsScan{}, fmt.Errorf("trailing content after Claude Code settings object")
		}
		return settingsScan{}, fmt.Errorf("read Claude Code settings tail: %w", err)
	}
	return scanner.scan, nil
}

func validateSettingsBytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("claude code settings are empty")
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return fmt.Errorf("claude code settings must be UTF-8 without a byte-order mark")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("claude code settings are not valid UTF-8")
	}
	if !json.Valid(data) {
		return fmt.Errorf("claude code settings are not valid JSON")
	}
	return nil
}

func (scanner *settingsScanner) object(top bool) error {
	keys := make(map[string]struct{})
	for scanner.dec.More() {
		key, err := scanner.objectKey(keys)
		if err != nil {
			return err
		}
		if top && !scanner.scan.hasKeys {
			scanner.scan.hasKeys = true
		}
		if top && (key == "model" || key == effortKey) {
			span, err := scanner.stringValue(key)
			if err != nil {
				return err
			}
			if key == "model" {
				scanner.scan.model = span
			} else {
				scanner.scan.effort = span
			}
			continue
		}
		if err := scanner.value(); err != nil {
			return err
		}
	}
	token, err := scanner.dec.Token()
	if err != nil || token != json.Delim('}') {
		return fmt.Errorf("claude code settings object is not closed")
	}
	return nil
}

func (scanner *settingsScanner) objectKey(keys map[string]struct{}) (string, error) {
	token, err := scanner.dec.Token()
	if err != nil {
		return "", fmt.Errorf("read Claude Code settings key: %w", err)
	}
	key, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("claude code settings object key is not a string")
	}
	if _, found := keys[key]; found {
		return "", fmt.Errorf("duplicate Claude Code settings key %q", key)
	}
	keys[key] = struct{}{}
	return key, nil
}

// stringValue reads the root string member key and records its byte span.
func (scanner *settingsScanner) stringValue(key string) (*stringSpan, error) {
	start, err := scanner.valueStart()
	if err != nil {
		return nil, err
	}
	token, err := scanner.dec.Token()
	if err != nil {
		return nil, fmt.Errorf("read Claude Code %s: %w", key, err)
	}
	value, ok := token.(string)
	if !ok {
		return nil, fmt.Errorf("claude code settings %s must be a string", key)
	}
	return &stringSpan{start: start, end: int(scanner.dec.InputOffset()), value: value}, nil
}

func (scanner *settingsScanner) valueStart() (int, error) {
	pos := skipJSONSpace(scanner.data, int(scanner.dec.InputOffset()))
	if pos >= len(scanner.data) || scanner.data[pos] != ':' {
		return 0, fmt.Errorf("claude code settings key is missing a colon")
	}
	pos = skipJSONSpace(scanner.data, pos+1)
	if pos >= len(scanner.data) {
		return 0, fmt.Errorf("claude code settings value is missing")
	}
	return pos, nil
}

func (scanner *settingsScanner) value() error {
	token, err := scanner.dec.Token()
	if err != nil {
		return fmt.Errorf("read Claude Code settings value: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		return scanner.object(false)
	case '[':
		return scanner.array()
	default:
		return fmt.Errorf("unexpected Claude Code JSON delimiter %q", delim)
	}
}

func (scanner *settingsScanner) array() error {
	for scanner.dec.More() {
		if err := scanner.value(); err != nil {
			return err
		}
	}
	token, err := scanner.dec.Token()
	if err != nil || token != json.Delim(']') {
		return fmt.Errorf("claude code settings array is not closed")
	}
	return nil
}

func validateInstallRoute(route profilemango.RouteBinding) (string, error) {
	if route.Provider != "anthropic" {
		return "", fmt.Errorf("claude code model install requires provider anthropic")
	}
	if route.Transport != "native" {
		return "", fmt.Errorf("claude code model install requires native transport")
	}
	if route.Authentication == "" {
		return "", fmt.Errorf("claude code model install requires an authentication mode")
	}
	if !validEffort(route.Effort) {
		return "", fmt.Errorf("claude code model install rejects unsupported effort %q", route.Effort)
	}
	if err := validateModelValue(route.Model); err != nil {
		return "", fmt.Errorf("claude code model install: %w", err)
	}
	return route.Model, nil
}

func validateModelValue(value string) error {
	if value == "" || strings.TrimSpace(value) != value {
		return fmt.Errorf("claude code model must be a non-empty string without surrounding whitespace")
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("claude code model is not valid UTF-8")
	}
	for _, character := range value {
		if character < 0x20 {
			return fmt.Errorf("claude code model contains an unsafe control character")
		}
	}
	return nil
}

func skipJSONSpace(data []byte, pos int) int {
	for pos < len(data) {
		switch data[pos] {
		case ' ', '\t', '\r', '\n':
			pos++
		default:
			return pos
		}
	}
	return pos
}

func replaceJSONSpan(data []byte, start, end int, replacement string) []byte {
	content := make([]byte, 0, len(data)-end+start+len(replacement))
	content = append(content, data[:start]...)
	content = append(content, replacement...)
	content = append(content, data[end:]...)
	return content
}

// insertTopLevelKey adds key as the first root member without reformatting
// the rest of the file. In a multi-line object the member goes on its own line
// with the first existing member's indentation and line ending; an empty
// object gains one indented line; a one-line object stays on one line.
func insertTopLevelKey(data []byte, scan settingsScan, key, literal string) []byte {
	open := scan.rootOpen + 1
	first := skipJSONSpace(data, open)
	gap := data[open:first]
	newline := "\n"
	if bytes.Contains(gap, []byte("\r\n")) {
		newline = "\r\n"
	}
	member := strconv.Quote(key) + ": " + literal
	if !scan.hasKeys {
		return replaceJSONSpan(data, open, first, newline+"  "+member+newline)
	}
	lineBreak := bytes.LastIndexByte(gap, '\n')
	if lineBreak < 0 {
		return replaceJSONSpan(data, open, open, strconv.Quote(key)+":"+literal+",")
	}
	indent := string(gap[lineBreak+1:])
	return replaceJSONSpan(data, first, first, member+","+newline+indent)
}
