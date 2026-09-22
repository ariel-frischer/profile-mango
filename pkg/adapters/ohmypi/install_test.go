package ohmypi

import (
	"bytes"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchConfigTable(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
		model  string
		level  string
	}{
		"missing config": {
			want: "modelRoles:\n  default: \"openai/gpt-5.6\"\ndefaultThinkingLevel: \"high\"\n",
		},
		"replace while preserving comments and unrelated keys": {
			source: "# keep\r\nmodelRoles:\r\n  reviewer: 'other/model'\r\n  default: 'old/model' # owned\r\ndefaultThinkingLevel: low # keep\r\nunknown:\r\n  token: SYNTHETIC\r\n",
			want:   "# keep\r\nmodelRoles:\r\n  reviewer: 'other/model'\r\n  default: \"openai/gpt-5.6\" # owned\r\ndefaultThinkingLevel: \"high\" # keep\r\nunknown:\r\n  token: SYNTHETIC\r\n",
			model:  "old/model",
			level:  "low",
		},
		"insert missing fields into existing block": {
			source: "modelRoles:\n  reviewer: other/model\nunknown: true\n",
			want:   "modelRoles:\n  reviewer: other/model\n  default: \"openai/gpt-5.6\"\nunknown: true\ndefaultThinkingLevel: \"high\"\n",
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
			if patch.BeforeModel != test.model || patch.BeforeThinkingLevel != test.level {
				t.Fatalf("before values = %#v", patch)
			}
			if patch.AfterModel != "openai/gpt-5.6" || patch.AfterThinkingLevel != "high" {
				t.Fatalf("after values = %#v", patch)
			}
			second, err := PatchConfig([]byte(test.source), installRoute())
			if err != nil || !bytes.Equal(patch.Content, second.Content) {
				t.Fatalf("patch is not deterministic: %v", err)
			}
		})
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
		"block scalar thinking level": {
			source: "defaultThinkingLevel: |\n  high\n",
			want:   "block scalar",
		},
		"wrong transport": {
			route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "proxy", Authentication: "oauth", Effort: "high"},
			want:  "native transport",
		},
		"unsupported effort": {
			route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "native", Authentication: "oauth", Effort: "ultra"},
			want:  "unsupported thinking level",
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

func installRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
