package ohmypi

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gopkg.in/yaml.v3"
)

func TestPatchConfigTable(t *testing.T) {
	tests := map[string]struct {
		source string
		route  profilemango.RouteBinding
		want   string
		roles  []RoleChange
	}{
		"missing config": {
			want:  "modelRoles:\n  default: \"openai/gpt-5.6:high\"\n",
			roles: []RoleChange{{Role: "default", After: "openai/gpt-5.6:high"}},
		},
		"replace while preserving comments, unrelated keys, and thinking default": {
			source: "# keep\r\nmodelRoles:\r\n  reviewer: 'other/model'\r\n  default: 'old/model' # owned\r\ndefaultThinkingLevel: low # keep\r\nunknown:\r\n  token: SYNTHETIC\r\n",
			want:   "# keep\r\nmodelRoles:\r\n  reviewer: 'other/model'\r\n  default: \"openai/gpt-5.6:high\" # owned\r\ndefaultThinkingLevel: low # keep\r\nunknown:\r\n  token: SYNTHETIC\r\n",
			roles:  []RoleChange{{Role: "default", Before: "old/model", After: "openai/gpt-5.6:high"}},
		},
		"insert missing fields into existing block": {
			source: "modelRoles:\n  reviewer: other/model\nunknown: true\n",
			want:   "modelRoles:\n  reviewer: other/model\n  default: \"openai/gpt-5.6:high\"\nunknown: true\n",
			roles:  []RoleChange{{Role: "default", After: "openai/gpt-5.6:high"}},
		},
		"roles into missing config": {
			route: roleRoute(),
			want:  "modelRoles:\n  default: \"openai/gpt-5.6:high\"\n  plan: \"anthropic/claude-opus-5-5:high\"\n  smol: \"opencode-go/gpt-6-luna:high\"\n  tiny: \"opencode-go/glm-5.3-flash\"\n",
			roles: []RoleChange{
				{Role: "default", After: "openai/gpt-5.6:high"},
				{Role: "plan", After: "anthropic/claude-opus-5-5:high"},
				{Role: "smol", After: "opencode-go/gpt-6-luna:high"},
				{Role: "tiny", After: "opencode-go/glm-5.3-flash"},
			},
		},
		"roles replace present and insert absent with existing indentation": {
			route:  roleRoute(),
			source: "modelRoles:\n    smol: old/fast:low # mine\n    custom: keep/me\nother: 1\n",
			want:   "modelRoles:\n    smol: \"opencode-go/gpt-6-luna:high\" # mine\n    custom: keep/me\n    default: \"openai/gpt-5.6:high\"\n    plan: \"anthropic/claude-opus-5-5:high\"\n    tiny: \"opencode-go/glm-5.3-flash\"\nother: 1\n",
			roles: []RoleChange{
				{Role: "default", After: "openai/gpt-5.6:high"},
				{Role: "plan", After: "anthropic/claude-opus-5-5:high"},
				{Role: "smol", Before: "old/fast:low", After: "opencode-go/gpt-6-luna:high"},
				{Role: "tiny", After: "opencode-go/glm-5.3-flash"},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			route := test.route
			if route.Provider == "" {
				route = installRoute()
			}
			patch, err := PatchConfig([]byte(test.source), route)
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want {
				t.Fatalf("content = %q, want %q", patch.Content, test.want)
			}
			if !reflect.DeepEqual(patch.Roles, test.roles) {
				t.Fatalf("roles = %#v, want %#v", patch.Roles, test.roles)
			}
			assertParsedRoles(t, patch.Content, test.roles)
			second, err := PatchConfig([]byte(test.source), route)
			if err != nil || !bytes.Equal(patch.Content, second.Content) {
				t.Fatalf("patch is not deterministic: %v", err)
			}
		})
	}
}

func assertParsedRoles(t *testing.T, content []byte, roles []RoleChange) {
	t.Helper()
	var parsed struct {
		ModelRoles map[string]string `yaml:"modelRoles"`
	}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("patched YAML does not parse: %v", err)
	}
	for _, role := range roles {
		if parsed.ModelRoles[role.Role] != role.After {
			t.Fatalf("modelRoles.%s = %q, want %q", role.Role, parsed.ModelRoles[role.Role], role.After)
		}
	}
}

func TestPatchConfigRejectsAmbiguousInput(t *testing.T) {
	tests := map[string]struct {
		source string
		route  profilemango.RouteBinding
		want   string
	}{
		"duplicate key": {
			source: "modelRoles:\n  default: old/model\n  default: other/model\n",
			want:   "duplicate key",
		},
		"flow root": {
			source: "{modelRoles: {default: old/model}, defaultThinkingLevel: low}\n",
			want:   "root must be a block mapping",
		},
		"flow model roles": {
			source: "modelRoles: {default: old/model}\ndefaultThinkingLevel: low\n",
			want:   "modelRoles must be a block mapping",
		},
		"non-string model": {
			source: "modelRoles:\n  default: false\n",
			want:   "target field must be a string scalar",
		},
		"block scalar role": {
			source: "modelRoles:\n  default: |\n    old/model\n",
			want:   "block scalar",
		},
		"multiline model": {
			source: "modelRoles:\n  default: old/model\n    continuation\n",
			want:   "multiline scalar",
		},
		"anchor": {
			source: "modelRoles:\n  default: &role old/model\n",
			want:   "anchors",
		},
		"alias": {
			source: "base: &role old/model\nmodelRoles:\n  default: *role\n",
			want:   "anchors",
		},
		"merge key": {
			source: "modelRoles:\n  <<: {}\n",
			want:   "merge keys",
		},
		"wrong transport": {
			route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "proxy", Authentication: "oauth", Effort: "high"},
			want:  "native transport",
		},
		"unsupported effort": {
			route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "native", Authentication: "oauth", Effort: "ultra"},
			want:  "unsupported thinking level",
		},
		"unknown role": {
			route: withRole("researcher", profilemango.RoleRoute{Provider: "openai", Model: "m"}),
			want:  `rejects role "researcher"`,
		},
		"unsupported role effort": {
			route: withRole("smol", profilemango.RoleRoute{Provider: "openai", Model: "m", Effort: "ultra"}),
			want:  `role "smol" rejects unsupported thinking level`,
		},
		"role provider with slash": {
			route: withRole("smol", profilemango.RoleRoute{Provider: "open/ai", Model: "m"}),
			want:  "unsupported characters",
		},
		"role model with thinking-like suffix and no effort": {
			route: withRole("smol", profilemango.RoleRoute{Provider: "zai", Model: "glm-4.7:max"}),
			want:  "set effort explicitly",
		},
		"non-string existing role": {
			route:  roleRoute(),
			source: "modelRoles:\n  smol: 3\n",
			want:   "patch modelRoles.smol",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			route := test.route
			if route.Provider == "" {
				route = installRoute()
			}
			_, err := PatchConfig([]byte(test.source), route)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestPatchConfigInsertsBeforeDocumentEnd(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"plain marker": {
			source: "unknown: true\n...\n",
			want:   "unknown: true\nmodelRoles:\n  default: \"openai/gpt-5.6:high\"\n...\n",
		},
		"commented marker": {
			source: "unknown: true\n...\t# end\n",
			want:   "unknown: true\nmodelRoles:\n  default: \"openai/gpt-5.6:high\"\n...\t# end\n",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchConfig([]byte(test.source), installRoute())
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want {
				t.Fatalf("content = %q, want %q", patch.Content, test.want)
			}
		})
	}
}

func TestScalarSpanUsesRuneColumns(t *testing.T) {
	document, err := parseConfig([]byte("unknown: éold\n"))
	if err != nil {
		t.Fatal(err)
	}
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Line: 1, Column: 10}
	_, _, raw, err := scalarSpan(document, node)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "éold" {
		t.Fatalf("raw = %q, want %q", raw, "éold")
	}
}

func installRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func roleRoute() profilemango.RouteBinding {
	route := installRoute()
	route.Roles = map[string]profilemango.RoleRoute{
		"plan": {Provider: "anthropic", Model: "claude-opus-5-5", Effort: "high"},
		"smol": {Provider: "opencode-go", Model: "gpt-6-luna", Effort: "high"},
		"tiny": {Provider: "opencode-go", Model: "glm-5.3-flash"},
	}
	return route
}

func withRole(name string, role profilemango.RoleRoute) profilemango.RouteBinding {
	route := installRoute()
	route.Roles = map[string]profilemango.RoleRoute{name: role}
	return route
}

func TestSplitRoleSelector(t *testing.T) {
	tests := map[string]struct{ selector, model, effort string }{
		"suffix":       {selector: "anthropic/claude-opus-5-5:medium", model: "anthropic/claude-opus-5-5", effort: "medium"},
		"bare":         {selector: "openai/gpt-5.6", model: "openai/gpt-5.6"},
		"colon in id":  {selector: "bedrock/model-v2:1", model: "bedrock/model-v2:1"},
		"literal+suff": {selector: "zai/glm-4.7:max:high", model: "zai/glm-4.7:max", effort: "high"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			model, effort := SplitRoleSelector(test.selector)
			if model != test.model || effort != test.effort {
				t.Fatalf("split = %q, %q", model, effort)
			}
		})
	}
}
