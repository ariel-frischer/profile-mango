package profilemango

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProfileRoles(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml    string
		code    string
		path    string
		message string
	}{
		"every portable role":   {yaml: "roles:\n  worker: {description: Implements}\n  planner: {description: Plans}\n  research: {description: Scouts, instructions: roles/research.md}\n  tiny: {description: Commits}\n"},
		"empty roles":           {yaml: "roles: {}\n", code: "profile.roles_empty", path: "roles"},
		"null roles":            {yaml: "roles:\n", code: "yaml.null", path: "roles"},
		"roles list":            {yaml: "roles: [worker]\n", code: "yaml.shape", path: "roles", message: "expected a map of portable role names"},
		"role scalar":           {yaml: "roles:\n  worker: implements\n", code: "yaml.shape", path: "roles.worker"},
		"null role":             {yaml: "roles:\n  worker:\n", code: "yaml.null", path: "roles.worker"},
		"description map":       {yaml: "roles:\n  worker:\n    description: {text: x}\n", code: "yaml.shape", path: "roles.worker.description"},
		"missing description":   {yaml: "roles:\n  worker:\n    instructions: roles/worker.md\n", code: "profile.role_description_required", path: "roles.worker.description"},
		"blank description":     {yaml: "roles:\n  worker:\n    description: '  '\n", code: "profile.role_description_required", path: "roles.worker.description"},
		"null description":      {yaml: "roles:\n  worker:\n    description:\n", code: "yaml.null", path: "roles.worker.description"},
		"unknown key":           {yaml: "roles:\n  worker:\n    description: x\n    model: gpt\n", code: "yaml.strict", path: "document"},
		"unknown role":          {yaml: "roles:\n  reviewer: {description: x}\n", code: "profile.role_unknown", path: "roles.reviewer", message: "expected one of worker, planner, research, tiny"},
		"native slot name":      {yaml: "roles:\n  smol: {description: x}\n", code: "profile.role_unknown", path: "roles.smol", message: `use the portable role "research"`},
		"empty instructions":    {yaml: "roles:\n  worker: {description: x, instructions: ''}\n", code: "profile.role_instructions_path_invalid", path: "roles.worker.instructions"},
		"absolute instructions": {yaml: "roles:\n  worker: {description: x, instructions: /etc/passwd}\n", code: "profile.role_instructions_path_invalid", path: "roles.worker.instructions"},
		"escaping instructions": {yaml: "roles:\n  worker: {description: x, instructions: ../outside.md}\n", code: "profile.role_instructions_path_invalid", path: "roles.worker.instructions"},
		"null instructions":     {yaml: "roles:\n  worker:\n    description: x\n    instructions:\n", code: "yaml.null", path: "roles.worker.instructions"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diagnostics := ParseProfile([]byte("route: main\n" + test.yaml))
			if test.code == "" {
				if diagnostics.HasErrors() {
					t.Fatalf("unexpected diagnostics: %v", diagnostics)
				}
				return
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == test.code && diagnostic.Path == test.path && diagnostic.Severity == SeverityError && strings.Contains(diagnostic.Message, test.message) {
					return
				}
			}
			t.Fatalf("missing %s at %s containing %q in %v", test.code, test.path, test.message, diagnostics)
		})
	}
}

// TestResolveRoleDefinitionsGolden checks that a child replaces the parent's
// definition of a role it redeclares and inherits the others.
func TestResolveRoleDefinitionsGolden(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"base":  "route: main\nroles:\n  worker:\n    description: Implements changes\n    instructions: roles/worker.md\n  research:\n    description: Explores read-only\n",
		"child": "extends: base\nroles:\n  worker:\n    description: Implements with tests first\n  tiny:\n    description: Writes commit messages\n    instructions: roles/tiny.md\n",
	}
	profiles := map[string]PolicyProfile{}
	for name, source := range sources {
		profile, diagnostics := ParseProfileAt([]byte(source), name)
		if len(diagnostics) > 0 {
			t.Fatal(diagnostics)
		}
		profiles[name] = profile
	}
	resolved, diagnostics := Resolve(profiles, "child")
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	data, err := CanonicalJSON(resolved)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "golden", "roles.resolved.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(golden) {
		t.Fatalf("golden mismatch\n%s", data)
	}
}
