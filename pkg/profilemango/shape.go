package profilemango

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// fieldShape is the YAML node kind a flat profile field requires and a
// human-readable description of it, used in shape diagnostics.
type fieldShape struct {
	kind     yaml.Kind
	expected string
}

// profileShapes maps a flat profile path to its required shape. A trailing
// ".*" entry covers every mapping value and "[]" covers every list item.
var profileShapes = map[string]fieldShape{
	"name":                  {yaml.ScalarNode, "a string"},
	"description":           {yaml.ScalarNode, "a string"},
	"extends":               {yaml.ScalarNode, "a profile name string"},
	"route":                 {yaml.ScalarNode, "a route name string"},
	"labels":                {yaml.MappingNode, "a map of string keys to string values (e.g. labels: {team: core})"},
	"labels.*":              {yaml.ScalarNode, "a string value (e.g. labels: {team: core})"},
	"permissions":           {yaml.MappingNode, "a map (e.g. permissions: {mode: read-only})"},
	"permissions.*":         {yaml.ScalarNode, "a string"},
	"tools":                 {yaml.MappingNode, "a map of allow/deny lists (e.g. tools: {allow: [read]})"},
	"tools.allow":           {yaml.SequenceNode, "a list of tool names (e.g. allow: [read, edit])"},
	"tools.deny":            {yaml.SequenceNode, "a list of tool names (e.g. deny: [shell])"},
	"tools.allow[]":         {yaml.ScalarNode, "a tool name string"},
	"tools.deny[]":          {yaml.ScalarNode, "a tool name string"},
	"instructions":          {yaml.MappingNode, "a map (e.g. instructions: {append: [instructions/base.md]})"},
	"instructions.append":   {yaml.SequenceNode, "a list of instruction file paths"},
	"instructions.append[]": {yaml.ScalarNode, "a file path string"},
	"skills":                {yaml.SequenceNode, "a list of skill paths (e.g. skills: [skills/review/SKILL.md])"},
	"skills[]":              {yaml.ScalarNode, "a skill path string"},
}

// validateProfileShape reports every known flat-profile field whose YAML node
// kind differs from its required shape, naming the field path and the
// expected shape. It returns false when any mismatch was reported so callers
// skip the generic decoder error for the same problem.
func validateProfileShape(document *yaml.Node, diagnostics *Diagnostics) bool {
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return true
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return true
	}
	before := len(diagnostics.Errors())
	checkShapeChildren(root, "", diagnostics)
	return len(diagnostics.Errors()) == before
}

func checkShapeChildren(node *yaml.Node, path string, diagnostics *Diagnostics) {
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			checkShape(item, path+"[]", path+"[]", diagnostics)
		}
		return
	}
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		childPath := strings.TrimPrefix(path+"."+key, ".")
		checkShape(value, childPath, strings.TrimPrefix(path+".*", "."), diagnostics)
	}
}

func checkShape(node *yaml.Node, path, wildcard string, diagnostics *Diagnostics) {
	shape, known := profileShapes[path]
	if !known {
		shape, known = profileShapes[wildcard]
	}
	if !known || node.Tag == "!!null" {
		return
	}
	if node.Kind != shape.kind {
		message := "expected " + shape.expected + ", got " + describeNodeKind(node.Kind)
		diagnostics.Add(SeverityError, "yaml.shape", path, message, node.Line, node.Column)
		return
	}
	checkShapeChildren(node, path, diagnostics)
}

func describeNodeKind(kind yaml.Kind) string {
	switch kind {
	case yaml.MappingNode:
		return "a map"
	case yaml.SequenceNode:
		return "a list"
	case yaml.AliasNode:
		return "an alias"
	default:
		return "a scalar value"
	}
}
