package profilemango

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestResolveGlobalInstructionForms parses the scalar and list forms, resolves them
// through extends (a child entry replaces the parent's wholesale), and checks the
// resolved JSON: a single resource stays a string, a longer list is an array.
func TestResolveGlobalInstructionForms(t *testing.T) {
	t.Parallel()
	base := "route: primary\nglobalInstructions:\n  oh-my-pi:\n    AGENTS.md: [shared/core.md, omp/agents.md]\n    RULES.md: omp/rules.md\n"
	cases := map[string]struct {
		child    string
		want     map[string]map[string]ResourceList
		wantJSON string
	}{
		"list and scalar inherit": {child: "extends: base\n",
			want:     map[string]map[string]ResourceList{"oh-my-pi": {"AGENTS.md": {"shared/core.md", "omp/agents.md"}, "RULES.md": {"omp/rules.md"}}},
			wantJSON: `{"oh-my-pi":{"AGENTS.md":["shared/core.md","omp/agents.md"],"RULES.md":"omp/rules.md"}}`},
		"one-item list marshals like the scalar": {child: "extends: base\nglobalInstructions:\n  oh-my-pi:\n    AGENTS.md: [shared/core.md]\n",
			want:     map[string]map[string]ResourceList{"oh-my-pi": {"AGENTS.md": {"shared/core.md"}}},
			wantJSON: `{"oh-my-pi":{"AGENTS.md":"shared/core.md"}}`},
		"child scalar replaces parent list": {child: "extends: base\nglobalInstructions:\n  oh-my-pi:\n    AGENTS.md: child.md\n",
			want:     map[string]map[string]ResourceList{"oh-my-pi": {"AGENTS.md": {"child.md"}}},
			wantJSON: `{"oh-my-pi":{"AGENTS.md":"child.md"}}`},
		"block sequence list": {child: "extends: base\nglobalInstructions:\n  codex:\n    AGENTS.md:\n      - a.md\n      - b.md\n",
			want:     map[string]map[string]ResourceList{"codex": {"AGENTS.md": {"a.md", "b.md"}}, "oh-my-pi": {"AGENTS.md": {"shared/core.md", "omp/agents.md"}, "RULES.md": {"omp/rules.md"}}},
			wantJSON: `{"codex":{"AGENTS.md":["a.md","b.md"]},"oh-my-pi":{"AGENTS.md":["shared/core.md","omp/agents.md"],"RULES.md":"omp/rules.md"}}`},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			profiles := map[string]PolicyProfile{"base": mustParse(t, base), "child": mustParse(t, test.child)}
			resolved, diagnostics := Resolve(profiles, "child")
			if diagnostics.HasErrors() {
				t.Fatal(diagnostics)
			}
			if !reflect.DeepEqual(resolved.GlobalInstructions, test.want) {
				t.Fatalf("globalInstructions %#v, want %#v", resolved.GlobalInstructions, test.want)
			}
			data, err := json.Marshal(resolved.GlobalInstructions)
			if err != nil || string(data) != test.wantJSON {
				t.Fatalf("json %s (%v), want %s", data, err, test.wantJSON)
			}
		})
	}
}

// TestParseGlobalInstructionListRejectsBadFragments fails closed on an empty list and
// on null, empty, non-string, or escaping items, naming the file and item index.
func TestParseGlobalInstructionListRejectsBadFragments(t *testing.T) {
	t.Parallel()
	prefix := validProfileYAML() + "globalInstructions:\n  codex:\n    AGENTS.md: "
	cases := map[string]struct {
		value string
		code  string
		path  string
	}{
		"empty list":      {value: "[]", code: "profile.global_instructions_empty", path: "globalInstructions.codex.AGENTS.md"},
		"null item":       {value: "[a.md, null]", code: "yaml.null", path: "globalInstructions.codex.AGENTS.md[1]"},
		"empty item":      {value: "[a.md, '']", code: "profile.global_instructions_path_invalid", path: "globalInstructions.codex.AGENTS.md[1]"},
		"blank item":      {value: "['  ', a.md]", code: "profile.global_instructions_path_invalid", path: "globalInstructions.codex.AGENTS.md[0]"},
		"escaping item":   {value: "[a.md, ../secret.md]", code: "profile.global_instructions_path_invalid", path: "globalInstructions.codex.AGENTS.md[1]"},
		"absolute item":   {value: "[/etc/passwd, a.md]", code: "profile.global_instructions_path_invalid", path: "globalInstructions.codex.AGENTS.md[0]"},
		"map item":        {value: "[a.md, {path: b.md}]", code: "yaml.shape", path: "globalInstructions.codex.AGENTS.md[1]"},
		"null scalar":     {value: "null", code: "profile.global_instructions_empty", path: "globalInstructions.codex.AGENTS.md"},
		"duplicate files": {value: "a.md\n    AGENTS.md: [b.md]", code: "yaml.duplicate_key", path: "globalInstructions.codex.AGENTS.md"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diagnostics := ParseProfile([]byte(prefix + test.value + "\n"))
			for _, diagnostic := range diagnostics.Errors() {
				if diagnostic.Code == test.code && diagnostic.Path == test.path {
					return
				}
			}
			t.Fatalf("want %s at %s, got %#v", test.code, test.path, diagnostics)
		})
	}
}
