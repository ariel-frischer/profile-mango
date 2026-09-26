package codex

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestRenderRoleFile(t *testing.T) {
	tests := map[string]struct {
		role RoleFile
		err  string
	}{
		"multi-line prompt with quotes round-trips": {role: RoleFile{Name: "worker", Description: "Implements", Instructions: "Line \"one\"\n\\ two '''\n\"\"\" three", Model: "gpt-5.6", Effort: "high"}},
		"unbound role inherits":                     {role: RoleFile{Name: "research", Description: "Scouts", Instructions: "Scouts"}},
		"blank prompt is rejected":                  {role: RoleFile{Name: "worker", Description: "Implements", Instructions: " \n"}, err: "requires name, description, and developer_instructions"},
		"unsupported effort is rejected":            {role: RoleFile{Name: "worker", Description: "Implements", Instructions: "x", Model: "gpt-5.6", Effort: "max"}, err: "model_reasoning_effort is installed only as"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			content, err := RenderRoleFile(test.role)
			if test.err != "" {
				if err == nil || !strings.Contains(err.Error(), test.err) {
					t.Fatalf("err = %v, want %q", err, test.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]string
			if _, err := toml.Decode(string(content), &decoded); err != nil {
				t.Fatalf("decode %s: %v", content, err)
			}
			if decoded["developer_instructions"] != test.role.Instructions || decoded["model"] != test.role.Model || decoded["model_reasoning_effort"] != test.role.Effort {
				t.Fatalf("decoded = %#v", decoded)
			}
			if _, hasProvider := decoded["model_provider"]; hasProvider != (test.role.Model != "") {
				t.Fatalf("model_provider present = %v for model %q", hasProvider, test.role.Model)
			}
		})
	}
}
