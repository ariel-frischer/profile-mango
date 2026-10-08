package profilemango

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// plainScalar matches values safe to write unquoted in block and flow maps.
var plainScalar = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./@+-]*$`)

func (doc *bindingsDocument) replaceScalar(node *yaml.Node, value string) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: expected a single value", node.Line)
	}
	if node.Value == value {
		return nil
	}
	line := []rune(doc.lines[node.Line-1])
	start := node.Column - 1
	end, err := valueEnd(line, node)
	if err != nil {
		return err
	}
	doc.lines[node.Line-1] = string(line[:start]) + formatScalar(value, node.Style) + string(line[end:])
	return nil
}

func (doc *bindingsDocument) insertPair(mapping *yaml.Node, entry yamlEntry) error {
	if mapping.Style&yaml.FlowStyle != 0 {
		return doc.insertFlowPair(mapping, entry)
	}
	if len(mapping.Content) == 0 {
		return fmt.Errorf("line %d: cannot add to an empty map", mapping.Line)
	}
	indent := strings.Repeat(" ", mapping.Content[0].Column-1)
	after := lastLine(mapping)
	if entry.children == nil {
		after = lastScalarLine(mapping, after)
	}
	doc.lines = slices.Insert(doc.lines, after+1, doc.blockLines(entry, indent)...)
	return nil
}

// lastScalarLine keeps a new field beside the existing single-value fields,
// ahead of any targets or roles map that follows them.
func lastScalarLine(mapping *yaml.Node, fallback int) int {
	for index := len(mapping.Content) - 1; index > 0; index -= 2 {
		if mapping.Content[index].Kind == yaml.ScalarNode {
			return lastLine(mapping.Content[index])
		}
	}
	return fallback
}

func (doc *bindingsDocument) blockLines(entry yamlEntry, indent string) []string {
	if entry.children == nil {
		return []string{indent + entry.key + ": " + formatScalar(entry.value, 0) + doc.lineEnd}
	}
	lines := []string{indent + entry.key + ":" + doc.lineEnd}
	childIndent := indent + strings.Repeat(" ", doc.step)
	for _, child := range entry.children {
		lines = append(lines, doc.blockLines(child, childIndent)...)
	}
	return lines
}

func (doc *bindingsDocument) insertFlowPair(mapping *yaml.Node, entry yamlEntry) error {
	line := []rune(doc.lines[mapping.Line-1])
	position := mapping.Column
	text := flowText(entry)
	if len(mapping.Content) > 0 {
		end, err := valueEnd(line, mapping.Content[len(mapping.Content)-1])
		if err != nil {
			return err
		}
		position, text = end, ", "+text
	}
	doc.lines[mapping.Line-1] = string(line[:position]) + text + string(line[position:])
	return nil
}

func flowText(entry yamlEntry) string {
	if entry.children == nil {
		return entry.key + ": " + formatScalar(entry.value, 0)
	}
	parts := make([]string, 0, len(entry.children))
	for _, child := range entry.children {
		parts = append(parts, flowText(child))
	}
	return entry.key + ": {" + strings.Join(parts, ", ") + "}"
}

// removePair deletes the pair whose key is mapping.Content[index]. Removing a
// block map entry also removes the comment written directly above its key.
func (doc *bindingsDocument) removePair(mapping *yaml.Node, index int) error {
	key, value := mapping.Content[index], mapping.Content[index+1]
	if mapping.Style&yaml.FlowStyle != 0 {
		return doc.removeFlowPair(mapping, index)
	}
	start := key.Line - 1
	if value.Kind == yaml.MappingNode {
		start = doc.headCommentStart(start, key.HeadComment)
	}
	doc.lines = slices.Delete(doc.lines, start, lastLine(value)+1)
	return nil
}

func (doc *bindingsDocument) headCommentStart(start int, comment string) int {
	if comment == "" {
		return start
	}
	commentLines := strings.Split(comment, "\n")
	first := start - len(commentLines)
	if first < 0 {
		return start
	}
	for offset, text := range commentLines {
		if strings.TrimSpace(doc.lines[first+offset]) != strings.TrimSpace(text) {
			return start
		}
	}
	return first
}

func (doc *bindingsDocument) removeFlowPair(mapping *yaml.Node, index int) error {
	line := []rune(doc.lines[mapping.Line-1])
	start := mapping.Content[index].Column - 1
	end, err := valueEnd(line, mapping.Content[index+1])
	if err != nil {
		return err
	}
	switch {
	case index+2 < len(mapping.Content):
		end = mapping.Content[index+2].Column - 1
	case index > 0:
		if start, err = valueEnd(line, mapping.Content[index-1]); err != nil {
			return err
		}
	}
	doc.lines[mapping.Line-1] = string(line[:start]) + string(line[end:])
	return nil
}

// valueEnd is the rune offset just past a single-line scalar or {...} map.
func valueEnd(line []rune, node *yaml.Node) (int, error) {
	start := node.Column - 1
	end := -1
	switch {
	case node.Kind == yaml.MappingNode && node.Style&yaml.FlowStyle != 0:
		end = flowMapEnd(line, start)
	case node.Kind != yaml.ScalarNode:
	case node.Style&yaml.DoubleQuotedStyle != 0:
		end = quotedEnd(line, start, '"')
	case node.Style&yaml.SingleQuotedStyle != 0:
		end = quotedEnd(line, start, '\'')
	case node.Style == 0 && start+len([]rune(node.Value)) <= len(line) && string(line[start:start+len([]rune(node.Value))]) == node.Value:
		end = start + len([]rune(node.Value))
	}
	if end < 0 {
		return 0, fmt.Errorf("line %d: cannot locate the value to edit; edit it by hand", node.Line)
	}
	return end, nil
}

func quotedEnd(line []rune, start int, quote rune) int {
	if start >= len(line) || line[start] != quote {
		return -1
	}
	for index := start + 1; index < len(line); index++ {
		switch {
		case quote == '"' && line[index] == '\\':
			index++
		case line[index] == quote && quote == '\'' && index+1 < len(line) && line[index+1] == '\'':
			index++
		case line[index] == quote:
			return index + 1
		}
	}
	return -1
}

func flowMapEnd(line []rune, start int) int {
	if start >= len(line) || line[start] != '{' {
		return -1
	}
	depth := 0
	for index := start; index < len(line); index++ {
		switch line[index] {
		case '"', '\'':
			end := quotedEnd(line, index, line[index])
			if end < 0 {
				return -1
			}
			index = end - 1
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return index + 1
			}
		}
	}
	return -1
}

// formatScalar writes value in the style the replaced value used, falling back
// to double quotes when an unquoted value would read as another type.
func formatScalar(value string, style yaml.Style) string {
	switch {
	case style&yaml.SingleQuotedStyle != 0:
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	case style&yaml.DoubleQuotedStyle != 0 || !plainString(value):
		return strconv.Quote(value)
	}
	return value
}

func plainString(value string) bool {
	if !plainScalar.MatchString(value) {
		return false
	}
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(value), &node); err != nil || len(node.Content) != 1 {
		return false
	}
	return node.Content[0].Tag == "!!str"
}
