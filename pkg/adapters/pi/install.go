package pi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const InstallAdapterVersion = "profilemango.dev/pi-install/v1alpha1"

var settingsOrder = []string{"defaultProvider", "defaultModel", "defaultThinkingLevel"}

// SettingChange describes one source-grounded Pi setting effect.
type SettingChange struct {
	Path   string
	Before string
	After  string
}

// SettingsPatch changes only the documented top-level Pi settings fields.
type SettingsPatch struct {
	Content []byte
	Fields  []SettingChange
}

type jsonSpan struct {
	start int
	end   int
}

type settingsObject struct {
	rootOpen int
	offset   int
	fields   map[string]jsonSpan
	entries  int
}

type jsonContainer struct {
	kind      byte
	expectKey bool
	keys      map[string]struct{}
}

type replacement struct {
	start int
	end   int
	text  string
}

// PatchSettings validates a Pi settings object and replaces only route fields.
// The exists flag distinguishes a missing file from an existing empty file.
func PatchSettings(data []byte, exists bool, route profilemango.RouteBinding) (SettingsPatch, error) {
	if err := ValidateInstallRoute(route); err != nil {
		return SettingsPatch{}, err
	}
	desired := routeSettings(route)
	if !exists {
		return newSettingsPatch(desired)
	}
	object, err := parseSettingsObject(data)
	if err != nil {
		return SettingsPatch{}, err
	}
	return patchExistingSettings(data, object, desired)
}

// ValidateInstallRoute checks the route subset represented by Pi settings.
func ValidateInstallRoute(route profilemango.RouteBinding) error {
	if route.Transport != "native" {
		return fmt.Errorf("pi settings install requires native transport")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "provider", value: route.Provider},
		{name: "model", value: route.Model},
		{name: "effort", value: route.Effort},
	} {
		if field.value == "" {
			return fmt.Errorf("pi settings install requires route %s", field.name)
		}
		if strings.TrimSpace(field.value) != field.value || !utf8.ValidString(field.value) || hasControl(field.value) {
			return fmt.Errorf("pi route %s contains unsupported characters", field.name)
		}
	}
	if route.Authentication == "" {
		return fmt.Errorf("pi settings install requires an authentication mode")
	}
	if !validThinkingLevel(route.Effort) {
		return fmt.Errorf("pi settings install rejects unsupported thinking level %q", route.Effort)
	}
	return nil
}

func hasControl(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}

func routeSettings(route profilemango.RouteBinding) map[string]string {
	return map[string]string{
		"defaultProvider":      route.Provider,
		"defaultModel":         route.Model,
		"defaultThinkingLevel": route.Effort,
	}
}

func newSettingsPatch(desired map[string]string) (SettingsPatch, error) {
	settings := struct {
		DefaultProvider      string `json:"defaultProvider"`
		DefaultModel         string `json:"defaultModel"`
		DefaultThinkingLevel string `json:"defaultThinkingLevel"`
	}{
		DefaultProvider:      desired["defaultProvider"],
		DefaultModel:         desired["defaultModel"],
		DefaultThinkingLevel: desired["defaultThinkingLevel"],
	}
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return SettingsPatch{}, fmt.Errorf("encode Pi settings: %w", err)
	}
	fields := make([]SettingChange, 0, len(settingsOrder))
	for _, name := range settingsOrder {
		fields = append(fields, SettingChange{Path: name, After: desired[name]})
	}
	return SettingsPatch{Content: append(content, '\n'), Fields: fields}, nil
}

func patchExistingSettings(data []byte, object settingsObject, desired map[string]string) (SettingsPatch, error) {
	changes := make([]SettingChange, 0, len(settingsOrder))
	replacements := make([]replacement, 0, len(settingsOrder))
	missing := make([]string, 0, len(settingsOrder))
	for _, name := range settingsOrder {
		span, found := object.fields[name]
		if !found {
			missing = append(missing, name)
			changes = append(changes, SettingChange{Path: name, After: desired[name]})
			continue
		}
		before, err := settingString(data, span, name)
		if err != nil {
			return SettingsPatch{}, err
		}
		changes = append(changes, SettingChange{Path: name, Before: before, After: desired[name]})
		if before != desired[name] {
			replacements = append(replacements, replacement{start: span.start, end: span.end, text: strconv.Quote(desired[name])})
		}
	}
	if len(missing) > 0 {
		text := make([]string, 0, len(missing))
		for _, name := range missing {
			text = append(text, strconv.Quote(name)+":"+strconv.Quote(desired[name]))
		}
		insert := strings.Join(text, ",")
		if object.entries > 0 {
			insert += ","
		}
		replacements = append(replacements, replacement{start: object.rootOpen + 1, end: object.rootOpen + 1, text: insert})
	}
	return SettingsPatch{Content: applyReplacements(data, replacements), Fields: changes}, nil
}

func settingString(data []byte, span jsonSpan, name string) (string, error) {
	if span.start >= span.end || data[span.start] != '"' {
		return "", fmt.Errorf("pi setting %q must be a string", name)
	}
	var value string
	if err := json.Unmarshal(data[span.start:span.end], &value); err != nil {
		return "", fmt.Errorf("decode Pi setting %q: %w", name, err)
	}
	return value, nil
}

func applyReplacements(data []byte, replacements []replacement) []byte {
	if len(replacements) == 0 {
		return append([]byte(nil), data...)
	}
	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start > replacements[j].start })
	result := append([]byte(nil), data...)
	for _, change := range replacements {
		result = append(result[:change.start], append([]byte(change.text), result[change.end:]...)...)
	}
	return result
}

func parseSettingsObject(data []byte) (settingsObject, error) {
	offset := 0
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		offset = 3
		data = data[offset:]
	}
	if len(data) == 0 || !utf8.Valid(data) || !json.Valid(data) {
		return settingsObject{}, fmt.Errorf("pi settings must be valid UTF-8 JSON")
	}
	if err := validateUniqueJSONKeys(data); err != nil {
		return settingsObject{}, err
	}
	position := skipJSONSpace(data, 0)
	if position >= len(data) || data[position] != '{' {
		return settingsObject{}, fmt.Errorf("pi settings must be a top-level JSON object")
	}
	object := settingsObject{rootOpen: position + offset, offset: offset, fields: make(map[string]jsonSpan)}
	position++
	position = skipJSONSpace(data, position)
	if position < len(data) && data[position] == '}' {
		return object, nil
	}
	return parseSettingsEntries(data, position, object)
}

func validateUniqueJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	stack := make([]jsonContainer, 0, 8)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("scan Pi settings JSON: %w", err)
		}
		if delimiter, ok := token.(json.Delim); ok && (delimiter == '}' || delimiter == ']') {
			if len(stack) == 0 || (delimiter == '}' && stack[len(stack)-1].kind != '{') || (delimiter == ']' && stack[len(stack)-1].kind != '[') {
				return fmt.Errorf("scan Pi settings JSON: mismatched container")
			}
			stack = stack[:len(stack)-1]
			continue
		}
		if len(stack) > 0 && stack[len(stack)-1].kind == '{' {
			object := &stack[len(stack)-1]
			if object.expectKey {
				key, ok := token.(string)
				if !ok {
					return fmt.Errorf("scan Pi settings JSON: object key is not a string")
				}
				if _, found := object.keys[key]; found {
					return fmt.Errorf("pi settings contain duplicate key %q", key)
				}
				object.keys[key] = struct{}{}
				object.expectKey = false
				continue
			}
			object.expectKey = true
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{':
				stack = append(stack, jsonContainer{kind: '{', expectKey: true, keys: make(map[string]struct{})})
			case '[':
				stack = append(stack, jsonContainer{kind: '['})
			}
		}
	}
	return nil
}

func parseSettingsEntries(data []byte, position int, object settingsObject) (settingsObject, error) {
	for {
		next, err := parseSettingsEntry(data, position, &object)
		if err != nil {
			return settingsObject{}, err
		}
		position = skipJSONSpace(data, next)
		if position < len(data) && data[position] == '}' {
			return object, nil
		}
		if position >= len(data) || data[position] != ',' {
			return settingsObject{}, fmt.Errorf("pi settings expected comma or object close")
		}
		position = skipJSONSpace(data, position+1)
	}
}

func parseSettingsEntry(data []byte, position int, object *settingsObject) (int, error) {
	key, next, err := parseJSONString(data, position)
	if err != nil {
		return 0, err
	}
	if _, found := object.fields[key]; found {
		return 0, fmt.Errorf("pi settings contain duplicate key %q", key)
	}
	position = skipJSONSpace(data, next)
	if position >= len(data) || data[position] != ':' {
		return 0, fmt.Errorf("pi settings missing colon after %q", key)
	}
	position = skipJSONSpace(data, position+1)
	span, next, err := parseJSONValue(data, position)
	if err != nil {
		return 0, err
	}
	object.fields[key] = jsonSpan{start: span.start + object.offset, end: span.end + object.offset}
	object.entries++
	return next, nil
}

func parseJSONString(data []byte, start int) (string, int, error) {
	if start >= len(data) || data[start] != '"' {
		return "", 0, fmt.Errorf("pi settings object key must be a quoted string")
	}
	for position := start + 1; position < len(data); position++ {
		switch data[position] {
		case '\\':
			position++
		case '"':
			var value string
			if err := json.Unmarshal(data[start:position+1], &value); err != nil {
				return "", 0, fmt.Errorf("decode Pi settings key: %w", err)
			}
			return value, position + 1, nil
		}
	}
	return "", 0, fmt.Errorf("pi settings contain an unterminated string")
}

func parseJSONValue(data []byte, start int) (jsonSpan, int, error) {
	decoder := json.NewDecoder(bytes.NewReader(data[start:]))
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return jsonSpan{}, 0, fmt.Errorf("decode Pi settings value: %w", err)
	}
	end := start + int(decoder.InputOffset())
	if end <= start {
		return jsonSpan{}, 0, fmt.Errorf("pi settings value is empty")
	}
	return jsonSpan{start: start, end: end}, end, nil
}

func skipJSONSpace(data []byte, position int) int {
	for position < len(data) {
		switch data[position] {
		case ' ', '\t', '\r', '\n':
			position++
		default:
			return position
		}
	}
	return position
}
