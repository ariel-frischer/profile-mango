package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRouteSetListsStaleLiteralReferences pins that route set, dry run included,
// names file:line of the profile.yaml and resources of profiles using the route that
// still spell the old model, ignoring placeholders, other routes' profiles, and
// longer names that merely start with it.
func TestRouteSetListsStaleLiteralReferences(t *testing.T) {
	cases := map[string]struct {
		args      []string
		wantOut   []string
		wantNoOut []string
	}{
		"base model change lists every literal use": {
			args: []string{"route", "set", "sol", "--model", "gpt-7-sol", "--dry-run"},
			wantOut: []string{
				"Still names gpt-6-sol:\n  instructions/daily.md:2\n  instructions/daily.md:4\n  profiles/daily/profile.yaml:2\n  skills/review/SKILL.md:1\n",
				"Dry run: nothing was written.",
			},
			wantNoOut: []string{"instructions/daily.md:3", "instructions/spare.md"},
		},
		"target override change names the agent's old model": {
			args:    []string{"route", "set", "sol", "--target", "claude-code", "--model", "claude-opus-6"},
			wantOut: []string{"Still names claude-opus-5-5:\n  instructions/daily.md:5\n"},
		},
		"effort change lists nothing": {
			args:      []string{"route", "set", "sol", "--effort", "low"},
			wantNoOut: []string{"Still names"},
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			home, _ := routeTestHome(t)
			writeFile(t, filepath.Join(home, "profiles", "daily", "profile.yaml"), "route: sol\ndescription: runs gpt-6-sol\ninstructions: {append: [instructions/daily.md]}\nskills: [skills/review/SKILL.md]\n")
			writeFile(t, filepath.Join(home, "profiles", "other", "profile.yaml"), "route: spare\ninstructions: {append: [instructions/spare.md]}\n")
			writeFile(t, filepath.Join(home, "instructions", "daily.md"), "# Daily\nUse gpt-6-sol.\nUse {{route.sol.model}}; gpt-6-sol-mini is different.\nfallback: openai/gpt-6-sol\nclaude-opus-5-5 on Claude\n")
			writeFile(t, filepath.Join(home, "skills", "review", "SKILL.md"), "Review with gpt-6-sol\n")
			writeFile(t, filepath.Join(home, "instructions", "spare.md"), "gpt-6-sol\n")
			out, err := executeCommandResult(t, append(test.args, "--home", home)...)
			if err != nil {
				t.Fatalf("route set: %v\n%s", err, out)
			}
			for _, want := range test.wantOut {
				if !strings.Contains(out, want) {
					t.Errorf("output lacks %q:\n%s", want, out)
				}
			}
			for _, unwanted := range test.wantNoOut {
				if strings.Contains(out, unwanted) {
					t.Errorf("output has %q:\n%s", unwanted, out)
				}
			}
		})
	}
}
