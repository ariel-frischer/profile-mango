package profilemango

import (
	"reflect"
	"testing"
)

// TestResolveAgentFiles covers inheritance (a child target entry replaces the parent's,
// an empty map clears it) and the fail-closed collision with a resolved role's file.
func TestResolveAgentFiles(t *testing.T) {
	t.Parallel()
	base := "route: primary\nagentFiles:\n  oh-my-pi:\n    scout.md: agents/scout.md\n  codex:\n    reviewer.toml: agents/reviewer.toml\n"
	cases := map[string]struct {
		child string
		want  map[string]map[string]string
		code  string
	}{
		"child inherits parent files": {child: "extends: base\n",
			want: map[string]map[string]string{"oh-my-pi": {"scout.md": "agents/scout.md"}, "codex": {"reviewer.toml": "agents/reviewer.toml"}}},
		"child target entry replaces parent's": {child: "extends: base\nagentFiles:\n  oh-my-pi:\n    plan.md: agents/plan.md\n",
			want: map[string]map[string]string{"oh-my-pi": {"plan.md": "agents/plan.md"}, "codex": {"reviewer.toml": "agents/reviewer.toml"}}},
		"empty map clears the target": {child: "extends: base\nagentFiles:\n  oh-my-pi: {}\n",
			want: map[string]map[string]string{"codex": {"reviewer.toml": "agents/reviewer.toml"}}},
		"file named like an inherited role fails": {child: "extends: base\nroles:\n  research: {description: Explores}\nagentFiles:\n  oh-my-pi:\n    research.md: agents/research.md\n",
			code: "profile.agent_files_role_conflict"},
		"codex toml named like a role fails": {child: "extends: base\nroles:\n  tiny: {description: Small edits}\nagentFiles:\n  codex:\n    tiny.toml: agents/tiny.toml\n", code: "profile.agent_files_role_conflict"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			profiles := map[string]PolicyProfile{"base": mustParse(t, base)}
			child, diagnostics := ParseProfile([]byte(test.child))
			if test.code == "" && diagnostics.HasErrors() {
				t.Fatal(diagnostics)
			}
			profiles["child"] = child
			resolved, diagnostics := Resolve(profiles, "child")
			if test.code != "" {
				if !hasCode(diagnostics, test.code) {
					t.Fatalf("expected %s, got %#v", test.code, diagnostics)
				}
				return
			}
			if diagnostics.HasErrors() {
				t.Fatal(diagnostics)
			}
			if !reflect.DeepEqual(resolved.AgentFiles, test.want) {
				t.Fatalf("agentFiles %#v, want %#v", resolved.AgentFiles, test.want)
			}
		})
	}
}
