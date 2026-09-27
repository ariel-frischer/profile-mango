package profilemango

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// legacyProfile is the deprecated apiVersion/kind/metadata/spec wrapper.
type legacyProfile struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   Metadata   `yaml:"metadata"`
	Spec       legacySpec `yaml:"spec"`
}

type legacySpec struct {
	Extends      string            `yaml:"extends,omitempty"`
	RouteRef     string            `yaml:"routeRef"`
	Permissions  *PermissionPolicy `yaml:"permissions,omitempty"`
	Tools        *AccessRules      `yaml:"tools,omitempty"`
	Instructions InstructionsSpec  `yaml:"instructions,omitempty"`
	Skills       *[]string         `yaml:"skills,omitempty"`
}

var legacyWrapperKeys = map[string]struct{}{"apiVersion": {}, "kind": {}, "metadata": {}, "spec": {}}

// isLegacyProfile reports whether the document uses any wrapper key. Mixed
// documents take the legacy path so strict decoding rejects the flat keys.
func isLegacyProfile(document *yaml.Node) bool {
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return false
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(root.Content); i += 2 {
		if _, found := legacyWrapperKeys[root.Content[i].Value]; found {
			return true
		}
	}
	return false
}

func decodeLegacyProfile(data []byte, diagnostics *Diagnostics) (PolicyProfile, bool) {
	var legacy legacyProfile
	if !decodeStrictDocument(data, &legacy, diagnostics) {
		return PolicyProfile{}, false
	}
	diagnostics.Add(SeverityWarning, "profile.legacy_format", "document", "the apiVersion/kind/metadata/spec wrapper is deprecated; move metadata and spec fields to the top level and rename routeRef to route", 0, 0)
	if legacy.APIVersion != APIVersion {
		diagnostics.Add(SeverityError, "profile.api_version", "apiVersion", fmt.Sprintf("expected %q", APIVersion), 0, 0)
	}
	if legacy.Kind != KindPolicyProfile {
		diagnostics.Add(SeverityError, "profile.kind", "kind", fmt.Sprintf("expected %q", KindPolicyProfile), 0, 0)
	}
	return legacy.flat(), true
}

func (legacy legacyProfile) flat() PolicyProfile {
	return PolicyProfile{
		Name:         legacy.Metadata.Name,
		Description:  legacy.Metadata.Description,
		Labels:       legacy.Metadata.Labels,
		Extends:      legacy.Spec.Extends,
		Route:        legacy.Spec.RouteRef,
		Permissions:  legacy.Spec.Permissions,
		Tools:        legacy.Spec.Tools,
		Instructions: legacy.Spec.Instructions,
		Skills:       skillRefsFromPaths(legacy.Spec.Skills),
	}
}
