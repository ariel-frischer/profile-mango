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

func TestPatchConfigRejectsProviderShadowState(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"table": {
			source: "[model_providers.openai]\nmodel = \"nested\"\n",
			want:   "provider override state",
		},
		"quoted": {
			source: "[model_providers.\"openai\"]\nmodel = \"nested\"\n",
			want:   "ambiguous",
		},
		"dotted": {
			source: "model_providers.openai.name = \"shadow\"\n",
			want:   "provider override state",
		},
		"provider namespace": {
			source: "[model_providers]\nopenai = { name = \"shadow\" }\n",
			want:   "provider override state",
		},
		"quoted components": {
			source: "[\"model_providers\".\"openai\"]\nmodel = \"nested\"\n",
			want:   "ambiguous",
		},
		"spaced components": {
			source: "model_providers . openai . name = \"shadow\"\n",
			want:   "ambiguous",
		},
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchConfig([]byte(source.source), installRoute())
			if err == nil || !strings.Contains(err.Error(), source.want) {
				t.Fatalf("provider shadow result = %v, want %q", err, source.want)
			}
		})
	}
}

func TestPatchConfigPreservesUnrelatedProviderState(t *testing.T) {
	source := "[model_providers.other]\nname = \"unrelated\"\n"
	patch, err := PatchConfig([]byte(source), installRoute())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patch.Content), "name = \"unrelated\"\n") || !strings.Contains(string(patch.Content), "model_provider = \"openai\"\n") {
		t.Fatalf("unrelated provider state was not preserved or root fields missing: %q", patch.Content)
	}
}

func TestPatchConfigRejectsProfileAndProviderPrecedenceSurfaces(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"active profile": {
			source: "profile = \"work\"\n",
			want:   "profile selection or definitions",
		},
		"profile definitions": {
			source: "[profiles.work]\nmodel = \"profile-model\"\n",
			want:   "profile selection or definitions",
		},
		"dotted profile definition": {
			source: "profiles.work = { model = \"profile-model\" }\n",
			want:   "profile selection or definitions",
		},
		"quoted profile definition": {
			source: "[\"profiles\".foo]\nmodel = \"profile-model\"\n",
			want:   "ambiguous",
		},
		"spaced profile definition": {
			source: "profiles . work = { model = \"profile-model\" }\n",
			want:   "ambiguous",
		},
		"inline provider map": {
			source: "model_providers = { openai = { name = \"shadow\" } }\n",
			want:   "provider override state",
		},
		"built-in endpoint override": {
			source: "openai_base_url = \"https://shadow.invalid/v1\"\n",
			want:   "provider override state",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchConfig([]byte(test.source), installRoute())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func installRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
