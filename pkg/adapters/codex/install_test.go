package codex

import (
	"bytes"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchConfigPreservesUnmanagedTOML(t *testing.T) {
	source := "# keep\nmodel_provider = \"old-provider\"\nunknown = { nested = true }\nmodel = 'old/model'\nmodel_reasoning_effort = \"low\" # keep comment\n[features]\napps = false\n"
	want := "# keep\nmodel_provider = \"openai\"\nunknown = { nested = true }\nmodel = \"gpt-5.6\"\nmodel_reasoning_effort = \"high\" # keep comment\n[features]\napps = false\n"
	patch, err := PatchConfig([]byte(source), installRoute())
	if err != nil {
		t.Fatal(err)
	}
	if string(patch.Content) != want {
		t.Fatalf("patched config = %q, want %q", patch.Content, want)
	}
	if len(patch.Fields) != 3 || patch.Fields[1].Before != "old/model" {
		t.Fatalf("fields = %#v", patch.Fields)
	}
	second, err := PatchConfig(patch.Content, installRoute())
	if err != nil || !bytes.Equal(second.Content, patch.Content) {
		t.Fatalf("repatch changed content: err=%v content=%q", err, second.Content)
	}
}

func TestPatchConfigInsertsRootFieldsBeforeTables(t *testing.T) {
	source := "# keep\r\n[features]\r\napps = false\r\n"
	want := "# keep\r\nmodel_provider = \"openai\"\r\nmodel = \"gpt-5.6\"\r\nmodel_reasoning_effort = \"high\"\r\n[features]\r\napps = false\r\n"
	patch, err := PatchConfig([]byte(source), installRoute())
	if err != nil {
		t.Fatal(err)
	}
	if string(patch.Content) != want {
		t.Fatalf("patched config = %q, want %q", patch.Content, want)
	}
}

func TestPatchConfigPreservesUnknownMultilineValue(t *testing.T) {
	source := "instructions = \"\"\"first\nsecond\"\"\"\n[features]\napps = false\n"
	patch, err := PatchConfig([]byte(source), installRoute())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(patch.Content), "instructions = \"\"\"first\nsecond\"\"\"\n") {
		t.Fatalf("multiline value was not preserved: %q", patch.Content)
	}
	if !strings.Contains(string(patch.Content), "model_reasoning_effort = \"high\"\n[features]") {
		t.Fatalf("missing fields were inserted in the wrong section: %q", patch.Content)
	}
}

func TestPatchConfigRejectsAmbiguousOrUnsupportedInput(t *testing.T) {
	tests := map[string]struct {
		source string
		route  profilemango.RouteBinding
		want   string
	}{
		"duplicate model": {
			source: "model = \"one\"\nmodel = \"two\"\n",
			route:  installRoute(),
			want:   "duplicate",
		},
		"non-string model": {
			source: "model = 42\n",
			route:  installRoute(),
			want:   "must be a string",
		},
		"unterminated string": {
			source: "model = \"unterminated\n",
			route:  installRoute(),
			want:   "unterminated",
		},
		"invalid table": {
			source: "[features\n",
			route:  installRoute(),
			want:   "table header",
		},
		"empty table": {
			source: "[]\n",
			route:  installRoute(),
			want:   "table header",
		},
		"wrong transport": {
			source: "",
			route:  profilemango.RouteBinding{Provider: "openai", Transport: "proxy", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"},
			want:   "native transport",
		},
		"unsupported provider": {
			source: "",
			route:  profilemango.RouteBinding{Provider: "sentinel", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"},
			want:   "built-in openai",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchConfig([]byte(test.source), test.route)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPatchConfigDoesNotPatchNestedKeys(t *testing.T) {
	source := "[model_providers.openai]\nmodel = \"nested\"\n"
	patch, err := PatchConfig([]byte(source), installRoute())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patch.Content), "model = \"nested\"\n") || !strings.Contains(string(patch.Content), "model_provider = \"openai\"\n") {
		t.Fatalf("nested state was not preserved or root fields missing: %q", patch.Content)
	}
}

func installRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
