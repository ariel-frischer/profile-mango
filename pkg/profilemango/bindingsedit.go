package profilemango

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// RouteEdit is one in-place change to an existing route in a bindings file.
// Target or Role, at most one, selects the edited entry; neither selects the
// base route. Set and Unset use YAML field names such as effort and
// subagentMaxEffort.
type RouteEdit struct {
	Route  string
	Target string
	Role   string
	Set    map[string]string
	Unset  []string
}

// Fields lists the fields the edit's scope accepts, in the order they are applied.
func (edit RouteEdit) Fields() []string {
	if edit.Target == "" && edit.Role == "" {
		return []string{"provider", "model", "effort", "subagentMaxEffort"}
	}
	return []string{"provider", "model", "effort"}
}

// EditBindings applies edit to the bindings bytes by changing only the
// affected lines, so comments, key order, and unrelated bytes survive. Unset
// drops a target or role entry, and then its targets or roles map, once it is
// empty. The result must pass ParseBindings; an unchanged result returns data.
func EditBindings(data []byte, edit RouteEdit) ([]byte, error) {
	if err := edit.check(); err != nil {
		return nil, err
	}
	result := data
	for _, field := range edit.Fields() {
		value, set := edit.Set[field]
		if !set && !slices.Contains(edit.Unset, field) {
			continue
		}
		var err error
		result, err = editDocument(result, edit.Route, func(doc *bindingsDocument, route *yaml.Node) error {
			if set {
				return doc.setField(route, edit.scopePath(), field, value)
			}
			return doc.unsetField(route, edit.scopePath(), field)
		})
		if err != nil {
			return nil, err
		}
	}
	if bytes.Equal(result, data) {
		return data, nil
	}
	if _, diagnostics := ParseBindings(result); diagnostics.HasErrors() {
		return nil, fmt.Errorf("the edited bindings would be invalid, so nothing was written:\n%w", diagnostics.Err())
	}
	return result, nil
}

// RouteSource returns the route's entry exactly as written in the bindings file.
func RouteSource(data []byte, name string) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return "", fmt.Errorf("parse bindings: %w", err)
	}
	key, route, err := findRoute(&root, name)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	end := min(lastLine(route)+1, len(lines))
	return strings.Join(lines[key.Line-1:end], "\n") + "\n", nil
}

func (edit RouteEdit) check() error {
	if edit.Target != "" && edit.Role != "" {
		return errors.New("choose a target or a role, not both")
	}
	if edit.Target != "" && !knownRouteTarget(edit.Target) {
		return fmt.Errorf("unknown target %q; expected one of %s", edit.Target, strings.Join(RouteTargets, ", "))
	}
	if edit.Role != "" && !PortableRole(edit.Role) {
		return errors.New(unknownRoleMessage(edit.Role))
	}
	if len(edit.Set)+len(edit.Unset) == 0 {
		return errors.New("nothing to change")
	}
	for _, field := range slices.Sorted(maps.Keys(edit.Set)) {
		if err := edit.checkField(field); err != nil {
			return err
		}
		if err := checkRouteValue(field, edit.Set[field]); err != nil {
			return err
		}
	}
	for _, field := range edit.Unset {
		if err := edit.checkField(field); err != nil {
			return err
		}
	}
	return nil
}

func (edit RouteEdit) checkField(field string) error {
	if slices.Contains(edit.Fields(), field) {
		return nil
	}
	if field == "subagentMaxEffort" {
		return errors.New("subagentMaxEffort applies to the whole route, not to one target or role")
	}
	return fmt.Errorf("unknown route field %q; expected one of %s", field, strings.Join(edit.Fields(), ", "))
}

func checkRouteValue(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s needs a value; use route unset to remove it", field)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return fmt.Errorf("%s must be a single line of text", field)
	}
	return nil
}

func (edit RouteEdit) scopePath() []string {
	switch {
	case edit.Target != "":
		return []string{"targets", edit.Target}
	case edit.Role != "":
		return []string{"roles", edit.Role}
	}
	return nil
}

// bindingsDocument is the bindings file as lines plus the layout facts needed
// to add lines that match it.
type bindingsDocument struct {
	lines   []string
	lineEnd string
	step    int
}

func editDocument(data []byte, name string, change func(*bindingsDocument, *yaml.Node) error) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse bindings: %w", err)
	}
	key, route, err := findRoute(&root, name)
	if err != nil {
		return nil, err
	}
	if err := checkEditable(route); err != nil {
		return nil, fmt.Errorf("routes.%s %w; edit it by hand", name, err)
	}
	doc := &bindingsDocument{lines: strings.Split(string(data), "\n"), step: 2}
	if bytes.Contains(data, []byte("\r\n")) {
		doc.lineEnd = "\r"
	}
	if len(route.Content) > 0 && route.Content[0].Column > key.Column {
		doc.step = route.Content[0].Column - key.Column
	}
	if err := change(doc, route); err != nil {
		return nil, err
	}
	return []byte(strings.Join(doc.lines, "\n")), nil
}

func findRoute(root *yaml.Node, name string) (*yaml.Node, *yaml.Node, error) {
	if root.Kind != yaml.DocumentNode || len(root.Content) != 1 {
		return nil, nil, errors.New("bindings must be one YAML document")
	}
	routes := mappingValue(root.Content[0], "routes")
	if routes == nil || routes.Kind != yaml.MappingNode {
		return nil, nil, errors.New("bindings have no routes map")
	}
	index := pairIndex(routes, name)
	if index < 0 || routes.Content[index+1].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("route %q is not in the bindings file; add new routes by hand", name)
	}
	return routes.Content[index], routes.Content[index+1], nil
}

// checkEditable rejects layouts that line edits cannot change safely:
// aliases, multi-line scalars, and flow collections spread over lines.
func checkEditable(node *yaml.Node) error {
	switch {
	case node.Kind == yaml.AliasNode:
		return errors.New("uses a YAML alias")
	case node.Kind == yaml.ScalarNode && (node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 || strings.Contains(node.Value, "\n")):
		return errors.New("has a multi-line value")
	case node.Style&yaml.FlowStyle != 0 && lastLine(node) != node.Line-1:
		return errors.New("has a {...} map spread over several lines")
	}
	for _, child := range node.Content {
		if err := checkEditable(child); err != nil {
			return err
		}
	}
	return nil
}

func pairIndex(mapping *yaml.Node, key string) int {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return -1
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return index
		}
	}
	return -1
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if index := pairIndex(mapping, key); index >= 0 {
		return mapping.Content[index+1]
	}
	return nil
}

// lastLine is the zero-based index of the last line a node occupies.
func lastLine(node *yaml.Node) int {
	last := node.Line - 1
	for _, child := range node.Content {
		last = max(last, lastLine(child))
	}
	return last
}

func (doc *bindingsDocument) setField(route *yaml.Node, path []string, field, value string) error {
	current := route
	for index, key := range path {
		next := mappingValue(current, key)
		if next == nil {
			return doc.insertPair(current, nestedEntry(path[index:], field, value))
		}
		if next.Kind != yaml.MappingNode {
			return fmt.Errorf("%s is not a map", strings.Join(path[:index+1], "."))
		}
		current = next
	}
	existing := mappingValue(current, field)
	if existing == nil {
		return doc.insertPair(current, yamlEntry{key: field, value: value})
	}
	return doc.replaceScalar(existing, value)
}

// unsetField removes the field, or the nearest enclosing target, role, or
// targets/roles map that holds nothing else.
func (doc *bindingsDocument) unsetField(route *yaml.Node, path []string, field string) error {
	chain := []*yaml.Node{route}
	for _, key := range path {
		next := mappingValue(chain[len(chain)-1], key)
		if next == nil || next.Kind != yaml.MappingNode {
			return nil
		}
		chain = append(chain, next)
	}
	level := len(chain) - 1
	index := pairIndex(chain[level], field)
	if index < 0 {
		return nil
	}
	for level > 0 && len(chain[level].Content) == 2 {
		level--
		index = pairIndex(chain[level], path[level])
	}
	return doc.removePair(chain[level], index)
}

// yamlEntry is a key with either a scalar value or child entries.
type yamlEntry struct {
	key      string
	value    string
	children []yamlEntry
}

func nestedEntry(path []string, field, value string) yamlEntry {
	entry := yamlEntry{key: field, value: value}
	for index := len(path) - 1; index >= 0; index-- {
		entry = yamlEntry{key: path[index], children: []yamlEntry{entry}}
	}
	return entry
}
