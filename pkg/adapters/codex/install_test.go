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

func TestPatchConfigInsertsMissingAfterReplacedRootValue(t *testing.T) {
	source := "model = \"old/model\"\n"
	want := "model = \"gpt-5.6\"\nmodel_provider = \"openai\"\nmodel_reasoning_effort = \"high\"\n"
	patch, err := PatchConfig([]byte(source), installRoute())
	if err != nil || string(patch.Content) != want {
		t.Fatalf("patch = %q, err=%v, want %q", patch.Content, err, want)
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
			want:   "invalid codex TOML",
		},
		"non-string model": {
			source: "model = 42\n",
			route:  installRoute(),
			want:   "must be a string",
		},
		"unterminated string": {
			source: "model = \"unterminated\n",
			route:  installRoute(),
			want:   "invalid codex TOML",
		},
		"unterminated multiline string": {
			source: "instructions = \"\"\"never closed\"\"",
			route:  installRoute(),
			want:   "invalid codex TOML",
		},
		"invalid table": {
			source: "[features\n",
			route:  installRoute(),
			want:   "invalid codex TOML",
		},
		"empty table": {
			source: "[]\n",
			route:  installRoute(),
			want:   "invalid codex TOML",
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
		"unqualified effort": {
			source: "",
			route:  profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "max"},
			want:   `source-qualified reasoning efforts none, minimal, low, medium, high, xhigh, not "max"`,
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

func TestPatchConfigRejectsMalformedUnmanagedTOML(t *testing.T) {
	tests := map[string]string{
		"invalid bare token":   "unmanaged = not_a_toml_value\n",
		"invalid date":         "unmanaged = 1979-13-40\n",
		"malformed array":      "unmanaged = [1,,2]\n",
		"duplicate inline key": "unmanaged = { key = true, key = false }\n",
		"redefined table":      "[features]\napps = false\n[features]\nother = true\n",
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchConfig([]byte(source), installRoute())
			if err == nil || !strings.Contains(err.Error(), "invalid codex TOML") {
				t.Fatalf("malformed TOML result = %#v, %v; want syntax rejection", patch, err)
			}
			if len(patch.Content) > 0 || len(patch.Fields) > 0 {
				t.Fatalf("malformed TOML produced a patch: %#v", patch)
			}
		})
	}
}

func TestPatchConfigRedactsInvalidTOMLDiagnostics(t *testing.T) {
	source := "private_key = credential_canary_12345\n"
	_, err := PatchConfig([]byte(source), installRoute())
	if err == nil || !strings.Contains(err.Error(), "invalid codex TOML at line 1") {
		t.Fatalf("invalid TOML diagnostic = %v", err)
	}
	for _, sensitive := range []string{"private_key", "cre", "credential_canary_12345"} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("invalid TOML diagnostic exposed target-owned input: %v", err)
		}
	}
}

func TestPatchConfigRedactsAmbiguousTableName(t *testing.T) {
	source := "[\"private.credential_canary\"]\nkey = true\n"
	_, err := PatchConfig([]byte(source), installRoute())
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous table diagnostic = %v", err)
	}
	if strings.Contains(err.Error(), "credential_canary") {
		t.Fatalf("ambiguous table diagnostic exposed target-owned name: %v", err)
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
		"legacy profile selector": {
			source: "profile = \"work\"\n",
			want:   "legacy profile = setting",
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

func TestPatchConfigPreservesLegacyProfileTables(t *testing.T) {
	sources := []string{
		"[profiles.work]\nmodel = \"profile-model\"\n",
		"profiles.work = { model = \"profile-model\" }\n",
	}
	for _, source := range sources {
		patch, err := PatchConfig([]byte(source), installRoute())
		if err != nil {
			t.Fatalf("PatchConfig(%q) = %v", source, err)
		}
		if !strings.Contains(string(patch.Content), source) {
			t.Fatalf("legacy profile state was not preserved: %q", patch.Content)
		}
	}
}

func installRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
