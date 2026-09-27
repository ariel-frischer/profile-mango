package profilemango

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

// RouteRefFields are the route fields a {{route.…}} placeholder may name.
var RouteRefFields = []string{"provider", "model", "effort", "transport", "authentication"}

const (
	routeRefOpen  = "{{route."
	routeRefClose = "}}"
)

// RenderRouteRefs replaces each {{route.<name>.<field>}} and
// {{route.<name>.roles.<role>.<field>}} placeholder with that field of the named route
// as resolved for target by Bindings.RouteFor. Only tokens starting {{route. are read;
// all other text is literal. A placeholder that does not resolve fails the render.
func RenderRouteRefs(content []byte, bindings Bindings, target string) ([]byte, error) {
	if !bytes.Contains(content, []byte(routeRefOpen)) {
		return content, nil
	}
	var rendered bytes.Buffer
	offset := 0
	for {
		start, end := nextRouteRef(content, offset)
		if start < 0 {
			rendered.Write(content[offset:])
			return rendered.Bytes(), nil
		}
		rendered.Write(content[offset:start])
		line := 1 + bytes.Count(content[:start], []byte("\n"))
		if end < 0 {
			return nil, fmt.Errorf("line %d: %s placeholder is not closed with %s on the same line", line, routeRefOpen, routeRefClose)
		}
		token := string(content[start:end])
		value, err := bindings.routeRefValue(token[len(routeRefOpen):len(token)-len(routeRefClose)], target)
		if err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", line, token, err)
		}
		rendered.WriteString(value)
		offset = end
	}
}

// WithoutRouteRefs drops every closed {{route.…}} placeholder from line, so a search
// for literal values ignores the text inside them.
func WithoutRouteRefs(line string) string {
	var kept strings.Builder
	offset := 0
	for {
		start, end := nextRouteRef([]byte(line), offset)
		if start < 0 || end < 0 {
			kept.WriteString(line[offset:])
			return kept.String()
		}
		kept.WriteString(line[offset:start])
		offset = end
	}
}

// nextRouteRef returns the byte range of the next placeholder at or after offset:
// start is -1 when none remains, end is -1 when it is not closed on its line.
func nextRouteRef(content []byte, offset int) (int, int) {
	index := bytes.Index(content[offset:], []byte(routeRefOpen))
	if index < 0 {
		return -1, -1
	}
	start := offset + index
	body := content[start+len(routeRefOpen):]
	if newline := bytes.IndexByte(body, '\n'); newline >= 0 {
		body = body[:newline]
	}
	closing := bytes.Index(body, []byte(routeRefClose))
	if closing < 0 {
		return start, -1
	}
	return start, start + len(routeRefOpen) + closing + len(routeRefClose)
}

// routeRefValue resolves "<name>.<field>" or "<name>.roles.<role>.<field>" for target.
func (bindings Bindings) routeRefValue(reference, target string) (string, error) {
	parts := strings.Split(reference, ".")
	if len(parts) != 2 && (len(parts) != 4 || parts[1] != "roles") {
		return "", fmt.Errorf("expected {{route.<name>.<field>}} or {{route.<name>.roles.<role>.<field>}}")
	}
	field := parts[len(parts)-1]
	if !slices.Contains(RouteRefFields, field) {
		return "", fmt.Errorf("unknown route field %q; expected one of %s", field, strings.Join(RouteRefFields, ", "))
	}
	route, found := bindings.RouteFor(parts[0], target)
	if !found {
		return "", fmt.Errorf("route %q is not in the bindings file; known routes: %s", parts[0], strings.Join(sortedKeys(bindings.Routes), ", "))
	}
	if len(parts) == 2 {
		return routeFieldValue(route, field), nil
	}
	role, found := route.Roles[parts[2]]
	if !found {
		return "", fmt.Errorf("route %q has no role %q; its roles: %s", parts[0], parts[2], strings.Join(sortedKeys(route.Roles), ", "))
	}
	value := map[string]string{"provider": role.Provider, "model": role.Model, "effort": role.Effort}[field]
	if value == "" {
		return "", fmt.Errorf("route %q role %q sets no %s; roles set provider, model, and optionally effort", parts[0], parts[2], field)
	}
	return value, nil
}

func routeFieldValue(route RouteBinding, field string) string {
	return map[string]string{
		"provider": route.Provider, "model": route.Model, "effort": route.Effort,
		"transport": route.Transport, "authentication": route.Authentication,
	}[field]
}
