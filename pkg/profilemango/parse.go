package profilemango

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var profileNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// ParseProfile strictly decodes one profile document and validates its shape.
func ParseProfile(data []byte) (PolicyProfile, Diagnostics) {
	var node yaml.Node
	var diagnostics Diagnostics
	if err := yaml.Unmarshal(data, &node); err != nil {
		diagnostics.Add(SeverityError, "yaml.invalid", "document", err.Error(), 0, 0)
		return PolicyProfile{}, diagnostics
	}
	inspectNode(&node, "", &diagnostics)

	var profile PolicyProfile
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&profile); err != nil {
		diagnostics.Add(SeverityError, "yaml.strict", "document", err.Error(), 0, 0)
		return PolicyProfile{}, diagnostics.Sorted()
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		diagnostics.Add(SeverityError, "yaml.multiple_documents", "document", "exactly one YAML document is allowed", 0, 0)
	}
	validateProfile(profile, &diagnostics)
	return profile, diagnostics.Sorted()
}

// ParseBindings strictly decodes local route bindings. It never resolves credentials.
func ParseBindings(data []byte) (Bindings, Diagnostics) {
	var node yaml.Node
	var diagnostics Diagnostics
	if err := yaml.Unmarshal(data, &node); err != nil {
		diagnostics.Add(SeverityError, "yaml.invalid", "document", err.Error(), 0, 0)
		return Bindings{}, diagnostics
	}
	inspectNode(&node, "", &diagnostics)

	var bindings Bindings
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&bindings); err != nil {
		diagnostics.Add(SeverityError, "yaml.strict", "document", err.Error(), 0, 0)
		return Bindings{}, diagnostics.Sorted()
	}
	if len(bindings.Routes) == 0 {
		diagnostics.Add(SeverityError, "binding.routes_required", "routes", "at least one route is required", 0, 0)
	}
	for name, route := range bindings.Routes {
		path := "routes." + name
		if !profileNamePattern.MatchString(name) {
			diagnostics.Add(SeverityError, "binding.name_invalid", path, "route name must be lowercase kebab-case", 0, 0)
		}
		if route.Provider == "" || route.Transport == "" || route.Authentication == "" || route.Model == "" || route.Effort == "" {
			diagnostics.Add(SeverityError, "binding.route_incomplete", path, "provider, transport, authentication, model, and effort are required", 0, 0)
		}
		validateRouteTargets(path, route, &diagnostics)
	}
	return bindings, diagnostics.Sorted()
}

func validateProfile(profile PolicyProfile, diagnostics *Diagnostics) {
	if profile.APIVersion != APIVersion {
		diagnostics.Add(SeverityError, "profile.api_version", "apiVersion", fmt.Sprintf("expected %q", APIVersion), 0, 0)
	}
	if profile.Kind != KindPolicyProfile {
		diagnostics.Add(SeverityError, "profile.kind", "kind", fmt.Sprintf("expected %q", KindPolicyProfile), 0, 0)
	}
	if !profileNamePattern.MatchString(profile.Metadata.Name) {
		diagnostics.Add(SeverityError, "profile.name_invalid", "metadata.name", "name must be lowercase kebab-case", 0, 0)
	}
	validatePermission(profile.Spec.Permissions, diagnostics)
	validateRules(profile.Spec.Tools, diagnostics)
}

func validatePermission(policy *PermissionPolicy, diagnostics *Diagnostics) {
	if policy == nil {
		return
	}
	validateEnum(policy.Mode, "spec.permissions.mode", []string{"read-only", "workspace-write", "unrestricted"}, diagnostics)
	validateEnum(policy.Network, "spec.permissions.network", []string{"allow", "deny", "unmanaged"}, diagnostics)
	validateEnum(policy.Shell, "spec.permissions.shell", []string{"allow", "deny", "unmanaged"}, diagnostics)
}

func validateEnum(value *string, path string, allowed []string, diagnostics *Diagnostics) {
	if value == nil {
		return
	}
	for _, candidate := range allowed {
		if *value == candidate {
			return
		}
	}
	diagnostics.Add(SeverityError, "profile.enum", path, "unsupported value", 0, 0)
}

func validateRules(rules *AccessRules, diagnostics *Diagnostics) {
	if rules == nil || rules.Allow == nil || rules.Deny == nil {
		return
	}
	denied := make(map[string]struct{}, len(*rules.Deny))
	for _, item := range *rules.Deny {
		denied[item] = struct{}{}
	}
	for _, item := range *rules.Allow {
		if _, found := denied[item]; found {
			diagnostics.Add(SeverityWarning, "policy.deny_wins", "spec.tools", fmt.Sprintf("%q appears in both allow and deny; deny wins", item), 0, 0)
		}
	}
}

func inspectNode(node *yaml.Node, path string, diagnostics *Diagnostics) {
	if node == nil {
		return
	}
	if node.Tag == "!!null" {
		at := path
		if at == "" {
			at = "document"
		}
		diagnostics.Add(SeverityError, "yaml.null", at, "null values are not allowed", node.Line, node.Column)
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]struct{}{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			childPath := key.Value
			if path != "" {
				childPath = path + "." + key.Value
			}
			if _, found := seen[key.Value]; found {
				diagnostics.Add(SeverityError, "yaml.duplicate_key", childPath, "duplicate mapping key", key.Line, key.Column)
			}
			seen[key.Value] = struct{}{}
			inspectNode(value, childPath, diagnostics)
		}
		return
	}
	for index, child := range node.Content {
		childPath := path
		if node.Kind == yaml.SequenceNode {
			childPath = fmt.Sprintf("%s[%d]", strings.TrimPrefix(path, "."), index)
		}
		inspectNode(child, childPath, diagnostics)
	}
}
