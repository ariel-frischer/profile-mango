package hermes

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchConfigTable(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
		before map[string]string
	}{
		"missing config": {
			want:   "model:\n  provider: \"openai\"\n  default: \"gpt-5.6\"\nagent:\n  reasoning_effort: \"high\"\n",
			before: map[string]string{},
		},
		"lossless replacements": {
			source: "# keep header\nmodel:\n  # keep model\n  provider: old-provider # keep provider\n  default: old-model\n  extra: keep\nagent:\n  reasoning_effort: low # keep effort\nunknown:\n  keep: true\n",
			want:   "# keep header\nmodel:\n  # keep model\n  provider: \"openai\" # keep provider\n  default: \"gpt-5.6\"\n  extra: keep\nagent:\n  reasoning_effort: \"high\" # keep effort\nunknown:\n  keep: true\n",
			before: map[string]string{
				"model.provider":         "old-provider",
				"model.default":          "old-model",
				"agent.reasoning_effort": "low",
			},
		},
		"insert missing fields": {
			source: "model:\n  extra: keep\nagent:\n  other: keep\n",
			want:   "model:\n  provider: \"openai\"\n  default: \"gpt-5.6\"\n  extra: keep\nagent:\n  reasoning_effort: \"high\"\n  other: keep\n",
			before: map[string]string{},
		},
		"scalar model shorthand": {
			source: "model: old-model # preserve\nagent:\n  reasoning_effort: false\n",
			want:   "model: {provider: \"openai\", default: \"gpt-5.6\"} # preserve\nagent:\n  reasoning_effort: \"high\"\n",
			before: map[string]string{
				"model.default":          "old-model",
				"agent.reasoning_effort": "false",
			},
		},
		"crlf preservation": {
			source: "model:\r\n  provider: old\r\nagent:\r\n  reasoning_effort: low\r\n",
			want:   "model:\r\n  provider: \"openai\"\r\n  default: \"gpt-5.6\"\r\nagent:\r\n  reasoning_effort: \"high\"\r\n",
			before: map[string]string{
				"model.provider":         "old",
				"agent.reasoning_effort": "low",
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchConfig([]byte(test.source), installTestRoute())
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want {
				t.Fatalf("content = %q, want %q", patch.Content, test.want)
			}
			if !sameStringMap(patch.Before, test.before) {
				t.Fatalf("before = %#v, want %#v", patch.Before, test.before)
			}
			second, err := PatchConfig([]byte(test.source), installTestRoute())
			if err != nil || !bytes.Equal(patch.Content, second.Content) {
				t.Fatalf("patch is not deterministic: %v", err)
			}
		})
	}
}

func TestPatchConfigRejectsAmbiguousOrUnsupportedYAML(t *testing.T) {
	tests := map[string]struct {
		source string
		route  profilemango.RouteBinding
		want   string
	}{
		"duplicate nested key":    {source: "model:\n  provider: one\n  provider: two\n", want: "duplicate"},
		"duplicate top-level key": {source: "model: old\nmodel: newer\n", want: "duplicate"},
		"null value":              {source: "model:\n  provider: null\n", want: "null"},
		"flow model":              {source: "model: {provider: old, default: old}\n", want: "flow"},
		"alias":                   {source: "defaults: &defaults\n  keep: true\nmodel:\n  <<: *defaults\n", want: "alias"},
		"multiple documents":      {source: "model: old\n---\nmodel: newer\n", want: "multiple YAML documents"},
		"invalid effort":          {source: "model: old\n", route: profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "turbo"}, want: "unsupported effort"},
		"wrong transport":         {source: "model: old\n", route: profilemango.RouteBinding{Provider: "openai", Transport: "proxy", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}, want: "native transport"},
		"missing authentication":  {source: "model: old\n", route: profilemango.RouteBinding{Provider: "openai", Transport: "native", Model: "gpt-5.6", Effort: "high"}, want: "authentication"},
		"unsafe model":            {source: "model: old\n", route: profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt\n5.6", Effort: "high"}, want: "unsafe"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			route := test.route
			if route.Provider == "" {
				route = installTestRoute()
			}
			_, err := PatchConfig([]byte(test.source), route)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPatchConfigDoesNotCopyAuthentication(t *testing.T) {
	route := installTestRoute()
	route.Authentication = "SYNTHETIC-SECRET-SENTINEL"
	patch, err := PatchConfig(nil, route)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(patch.Content), route.Authentication) || strings.Contains(string(patch.Content), "secret") {
		t.Fatalf("authentication leaked into config: %s", patch.Content)
	}
}

func sameStringMap(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for key, value := range want {
		if got[key] != value {
			return false
		}
	}
	return true
}

func installTestRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
