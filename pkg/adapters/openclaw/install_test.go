package openclaw

import (
	"bytes"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchConfigPreservesOpenClawJSON5State(t *testing.T) {
	route := installTestRoute()
	source := []byte("{\r\n  // preserve this comment\r\n  agents: {\r\n    defaults: {\r\n      model: { primary: 'old/model', fallbacks: ['keep/model'], unknown: { token: 'SYNTHETIC' }, },\r\n      thinkingDefault: 'low',\r\n      unknownDefaults: true,\r\n    },\r\n    entries: { main: {}, },\r\n  },\r\n  unrelated: { secret: 'SYNTHETIC' },\r\n}\r\n")
	patch, err := PatchConfig(source, route)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\r\n  // preserve this comment\r\n  agents: {\r\n    defaults: {\r\n      model: { primary: \"openai/gpt-5.6\", fallbacks: ['keep/model'], unknown: { token: 'SYNTHETIC' }, },\r\n      thinkingDefault: \"high\",\r\n      unknownDefaults: true,\r\n    },\r\n    entries: { main: {}, },\r\n  },\r\n  unrelated: { secret: 'SYNTHETIC' },\r\n}\r\n"
	if string(patch.Content) != want {
		t.Fatalf("patched content = %q, want %q", patch.Content, want)
	}
	if patch.BeforeModel != "old/model" || patch.BeforeThinking != "low" {
		t.Fatalf("patch before values = %#v", patch)
	}
	if !strings.Contains(string(patch.Content), "SYNTHETIC") || !strings.Contains(string(patch.Content), "unknownDefaults") {
		t.Fatal("patch removed unrelated or secret-shaped synthetic state")
	}
}

func TestPatchConfigInsertionAndIdempotence(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"missing config": {
			want: "{agents:{defaults:{model:{primary:\"openai/gpt-5.6\"},thinkingDefault:\"high\"}}}\n",
		},
		"missing target tree": {
			source: "{\n  unrelated: true,\n}\n",
			want:   "{agents:{defaults:{model:{primary:\"openai/gpt-5.6\"},thinkingDefault:\"high\"}},\n  unrelated: true,\n}\n",
		},
		"existing defaults": {
			source: "{agents:{entries:{main:{}},defaults:{unknown:true}}}",
			want:   "{agents:{entries:{main:{}},defaults:{model:{primary:\"openai/gpt-5.6\"},thinkingDefault:\"high\",unknown:true}}}",
		},
		"existing model string": {
			source: "{agents:{defaults:{model:'old/model',thinkingDefault:'low'},entries:{main:{}}}}",
			want:   "{agents:{defaults:{model:{primary:\"openai/gpt-5.6\"},thinkingDefault:\"high\"},entries:{main:{}}}}",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			first, err := PatchConfig([]byte(test.source), installTestRoute())
			if err != nil {
				t.Fatal(err)
			}
			if string(first.Content) != test.want {
				t.Fatalf("content = %q, want %q", first.Content, test.want)
			}
			second, err := PatchConfig(first.Content, installTestRoute())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first.Content, second.Content) {
				t.Fatalf("reapply changed content:\n%s\n---\n%s", first.Content, second.Content)
			}
		})
	}
}

func TestPatchConfigRejectsAmbiguousState(t *testing.T) {
	tests := map[string]struct {
		source string
		route  profilemango.RouteBinding
		want   string
	}{
		"duplicate nested key":          {source: `{agents:{defaults:{model:{primary:"a",primary:"b"},thinkingDefault:"high"},entries:{main:{}}}}`, want: "duplicate"},
		"duplicate unrelated key":       {source: `{unknown:1,unknown:2}`, want: "duplicate"},
		"non-string primary":            {source: `{agents:{defaults:{model:{primary:false},thinkingDefault:"high"},entries:{main:{}}}}`, want: "primary must be a string"},
		"non-string thinking":           {source: `{agents:{defaults:{model:{primary:"a"},thinkingDefault:false},entries:{main:{}}}}`, want: "thinkingDefault must be a string"},
		"unsupported existing thinking": {source: `{agents:{defaults:{model:{primary:"a"},thinkingDefault:"turbo"},entries:{main:{}}}}`, want: "unsupported value"},
		"wrong agents type":             {source: `{agents:[],unknown:true}`, want: "agents must be an object"},
		"trailing content":              {source: `{}` + ` {}`, want: "trailing content"},
		"unterminated comment":          {source: `{/*`, want: "unterminated block comment"},
		"byte order mark":               {source: "\ufeff{}", want: "byte-order mark"},
		"wrong transport":               {source: `{}`, route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "proxy", Authentication: "oauth", Effort: "high"}, want: "native transport"},
		"unsupported thinking":          {source: `{}`, route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "native", Authentication: "oauth", Effort: "turbo"}, want: "unsupported thinking level"},
		"missing authentication":        {source: `{}`, route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "native", Effort: "high"}, want: "authentication"},
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

func TestPatchConfigRejectsUnsafeRouteParts(t *testing.T) {
	tests := map[string]string{
		"provider slash": "open/ai",
		"model newline":  "gpt\n5.6",
		"model control":  "gpt\x005.6",
	}
	for name, model := range tests {
		t.Run(name, func(t *testing.T) {
			route := installTestRoute()
			route.Model = model
			if strings.Contains(name, "provider") {
				route.Provider = model
				route.Model = "gpt-5.6"
			}
			if _, err := PatchConfig(nil, route); err == nil {
				t.Fatal("unsafe route part was accepted")
			}
		})
	}
}

func installTestRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func TestPatchConfigRejectsAmbiguousEscapes(t *testing.T) {
	tests := map[string]struct{ source string }{
		"hex duplicate":         {`{"\xE9":1,"é":2}`},
		"unsupported surrogate": {`{"\uD800":1}`},
		"decimal escape":        {`{"value":"\01"}`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := PatchConfig([]byte(test.source), installTestRoute()); err == nil {
				t.Fatal("ambiguous or invalid escape was accepted")
			}
		})
	}
}
