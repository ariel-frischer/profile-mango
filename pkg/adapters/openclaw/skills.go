package openclaw

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
)

// agentIDPattern is the agents.entries key pattern of the pinned config schema
// (src/config/zod-schema.agents.ts), lowercase only so one id names one key.
var agentIDPattern = regexp.MustCompile(`^[a-z0-9_][a-z0-9_-]{0,63}$`)

// ValidAgentID reports whether id can name an agents.entries key.
func ValidAgentID(id string) bool { return agentIDPattern.MatchString(id) }

// SkillsValue renders sorted skill names as the compact JSON array written to
// agents.entries.<id>.skills.
func SkillsValue(names []string) string {
	sorted := append(make([]string, 0, len(names)), names...)
	slices.Sort(sorted)
	data, _ := json.Marshal(sorted)
	return string(data)
}

// AgentSkills reads agents.entries.<agent>.skills. found reports whether the agent entry
// exists; value is the allowlist as a compact JSON array, or "" when the key is absent.
func AgentSkills(data []byte, agent string) (value string, found bool, err error) {
	entry, err := agentEntry(data, agent)
	if err != nil || entry == nil {
		return "", false, err
	}
	current, set := entry.entries["skills"]
	if !set {
		return "", true, nil
	}
	names, err := stringArray(data, current)
	if err != nil {
		return "", true, fmt.Errorf("openclaw agents.entries.%s.skills: %w", agent, err)
	}
	encoded, _ := json.Marshal(names)
	return string(encoded), true, nil
}

// SetAgentSkills sets agents.entries.<agent>.skills to value, a JSON array of strings, or
// removes the key when value is "", preserving every other byte. The entry must exist:
// adding an agent would change the configured roster.
func SetAgentSkills(data []byte, agent, value string) ([]byte, error) {
	entry, err := agentEntry(data, agent)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("openclaw agents.entries.%s does not exist", agent)
	}
	builder := newPatchBuilder(data)
	current, set := entry.entries["skills"]
	switch {
	case value == "" && set:
		start, end := entrySpan(data, current)
		builder.replace(start, end, "")
	case value == "":
	case set:
		builder.replace(current.start, current.end, value)
	default:
		builder.insert(entry, "skills:"+value)
	}
	return builder.build(), nil
}

// agentEntry returns the agents.entries.<agent> object, or nil when any level is absent.
func agentEntry(data []byte, agent string) (*json5Object, error) {
	if len(data) == 0 {
		return nil, nil
	}
	scanner, err := newJSON5Scanner(data)
	if err != nil {
		return nil, err
	}
	object := scanner.root
	for _, key := range []string{"agents", "entries", agent} {
		value, found := object.entries[key]
		if !found {
			return nil, nil
		}
		if value.kind != valueObject {
			return nil, fmt.Errorf("openclaw %s must be an object", objectPath(key, agent))
		}
		object = value.object
	}
	return object, nil
}

func objectPath(key, agent string) string {
	switch key {
	case "agents":
		return "agents"
	case "entries":
		return "agents.entries"
	}
	return "agents.entries." + agent
}

// stringArray decodes an already validated JSON5 array whose items must be strings.
func stringArray(data []byte, value json5Value) ([]string, error) {
	if value.kind != valueArray {
		return nil, fmt.Errorf("must be an array of strings")
	}
	scanner := &json5Scanner{data: data, pos: value.start + 1}
	names := []string{}
	for {
		scanner.skipTrivia()
		if scanner.consume(']') {
			return names, nil
		}
		item, err := scanner.parseValue()
		if err != nil {
			return nil, err
		}
		if item.kind != valueString {
			return nil, fmt.Errorf("must be an array of strings")
		}
		names = append(names, item.stringValue)
		scanner.skipTrivia()
		scanner.consume(',')
	}
}

// entrySpan is the byte range removing one object entry: its key through its value and
// the comma after it, else the comma before it, so the object stays valid.
func entrySpan(data []byte, value json5Value) (int, int) {
	scanner := &json5Scanner{data: data, pos: value.end}
	scanner.skipTrivia()
	if scanner.consume(',') {
		return value.keyStart, scanner.pos
	}
	start := value.keyStart
	for start > 0 && isJSONSpace(data[start-1]) {
		start--
	}
	if start > 0 && data[start-1] == ',' {
		return start - 1, value.end
	}
	return value.keyStart, value.end
}

func isJSONSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}
