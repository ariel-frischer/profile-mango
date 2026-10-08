package profilemango

import (
	"strings"
	"testing"
)

const editFixture = `# Route identity only.
routes:
  # Daily driver.
  sol:
    provider: openai
    model: gpt-6-sol
    effort: high # keep
    # Each listed agent gets the route above with these fields replaced.
    targets:
      claude-code:          # Claude only
        provider: anthropic
        model: claude-opus-5-5
  opus55:
    provider: anthropic
    model: "claude-opus-5-5"
    effort: medium
    subagentMaxEffort: high
    roles:
      worker: {provider: anthropic, model: claude-opus-5-5, effort: medium}
      planner: {effort: low, provider: anthropic, model: claude-opus-5-5}
      tiny: {provider: opencode-go, model: gpt-6-luna}
  plain:
    provider: openai
    model: gpt-6-sol
    effort: low
# trailing note
`

// targetRoleFixture has a base role and a target override without roles.
const targetRoleFixture = `routes:
  tr:
    provider: openai-codex
    model: gpt-6-sol
    effort: high
    roles:
      research:
        provider: openai-codex # ChatGPT OAuth on Oh My Pi
        model: gpt-6-luna
    targets:
      codex:
        provider: openai
`

func replaced(old, updated string) string {
	if !strings.Contains(editFixture, old) {
		panic("fixture lacks " + old)
	}
	return strings.Replace(editFixture, old, updated, 1)
}

func TestEditBindingsChangesOnlyTheEditedLines(t *testing.T) {
	tests := map[string]struct {
		edit RouteEdit
		want string
		from string
	}{
		"base field keeps its comment": {
			edit: RouteEdit{Route: "sol", Set: map[string]string{"effort": "medium"}},
			want: replaced("effort: high # keep", "effort: medium # keep"),
		},
		"quoted value keeps its quotes": {
			edit: RouteEdit{Route: "opus55", Set: map[string]string{"model": "claude-sonnet-5"}},
			want: replaced(`model: "claude-opus-5-5"`, `model: "claude-sonnet-5"`),
		},
		"value that would read as a number is quoted": {
			edit: RouteEdit{Route: "plain", Set: map[string]string{"model": "1.5"}},
			want: replaced("    model: gpt-6-sol\n    effort: low", "    model: \"1.5\"\n    effort: low"),
		},
		"new base field sits beside the other fields": {
			edit: RouteEdit{Route: "sol", Set: map[string]string{"subagentMaxEffort": "high"}},
			want: replaced("effort: high # keep\n", "effort: high # keep\n    subagentMaxEffort: high\n"),
		},
		"existing target field": {
			edit: RouteEdit{Route: "sol", Target: "claude-code", Set: map[string]string{"model": "claude-sonnet-5"}},
			want: replaced("model: claude-opus-5-5\n  opus55", "model: claude-sonnet-5\n  opus55"),
		},
		"new target entry under existing targets": {
			edit: RouteEdit{Route: "sol", Target: "oh-my-pi", Set: map[string]string{"effort": "medium", "model": "gpt-6"}},
			want: replaced("model: claude-opus-5-5\n  opus55", "model: claude-opus-5-5\n      oh-my-pi:\n        model: gpt-6\n        effort: medium\n  opus55"),
		},
		"new targets map": {
			edit: RouteEdit{Route: "plain", Target: "codex", Set: map[string]string{"model": "gpt-x"}},
			want: replaced("effort: low\n", "effort: low\n    targets:\n      codex:\n        model: gpt-x\n"),
		},
		"flow role field": {
			edit: RouteEdit{Route: "opus55", Role: "worker", Set: map[string]string{"effort": "high"}},
			want: replaced("claude-opus-5-5, effort: medium}", "claude-opus-5-5, effort: high}"),
		},
		"field added to a flow role": {
			edit: RouteEdit{Route: "opus55", Role: "tiny", Set: map[string]string{"effort": "low"}},
			want: replaced("model: gpt-6-luna}", "model: gpt-6-luna, effort: low}"),
		},
		"new role entry": {
			edit: RouteEdit{Route: "opus55", Role: "research", Set: map[string]string{"provider": "opencode-go", "model": "gpt-6-luna"}},
			want: replaced("model: gpt-6-luna}\n", "model: gpt-6-luna}\n      research:\n        provider: opencode-go\n        model: gpt-6-luna\n"),
		},
		"new roles map": {
			edit: RouteEdit{Route: "plain", Role: "planner", Set: map[string]string{"provider": "openai", "model": "gpt-6"}},
			want: replaced("effort: low\n", "effort: low\n    roles:\n      planner:\n        provider: openai\n        model: gpt-6\n"),
		},
		"new target role entry": {
			edit: RouteEdit{Route: "opus55", Target: "codex", Role: "tiny", Set: map[string]string{"provider": "openai"}},
			want: replaced("model: gpt-6-luna}\n", "model: gpt-6-luna}\n    targets:\n      codex:\n        roles:\n          tiny:\n            provider: openai\n"),
		},
		"new role under an existing target": {
			edit: RouteEdit{Route: "tr", Target: "codex", Role: "research", Set: map[string]string{"effort": "low"}},
			want: targetRoleFixture + "        roles:\n          research:\n            effort: low\n",
		},
		"unset last target role field prunes roles but keeps the target": {
			edit: RouteEdit{Route: "tr", Target: "codex", Role: "research", Unset: []string{"provider"}},
			want: targetRoleFixture,
			from: targetRoleFixture + "        roles:\n          research:\n            provider: openai\n",
		},
		"unset target field": {
			edit: RouteEdit{Route: "sol", Target: "claude-code", Unset: []string{"model"}},
			want: replaced("        provider: anthropic\n        model: claude-opus-5-5\n", "        provider: anthropic\n"),
		},
		"unset last target fields prunes the entry, map, and its comment": {
			edit: RouteEdit{Route: "sol", Target: "claude-code", Unset: []string{"provider", "model"}},
			want: replaced("    # Each listed agent gets the route above with these fields replaced.\n    targets:\n      claude-code:          # Claude only\n        provider: anthropic\n        model: claude-opus-5-5\n", ""),
		},
		"unset last flow field": {
			edit: RouteEdit{Route: "opus55", Role: "worker", Unset: []string{"effort"}},
			want: replaced("claude-opus-5-5, effort: medium}", "claude-opus-5-5}"),
		},
		"unset first flow field": {
			edit: RouteEdit{Route: "opus55", Role: "planner", Unset: []string{"effort"}},
			want: replaced("{effort: low, provider", "{provider"),
		},
		"unset whole role keeps the other roles": {
			edit: RouteEdit{Route: "opus55", Role: "tiny", Unset: []string{"provider", "model", "effort"}},
			want: replaced("      tiny: {provider: opencode-go, model: gpt-6-luna}\n", ""),
		},
		"unset base field": {
			edit: RouteEdit{Route: "opus55", Unset: []string{"subagentMaxEffort"}},
			want: replaced("    subagentMaxEffort: high\n", ""),
		},
		"same value is no change": {
			edit: RouteEdit{Route: "sol", Set: map[string]string{"effort": "high"}},
			want: editFixture,
		},
		"unset of an absent field is no change": {
			edit: RouteEdit{Route: "sol", Target: "codex", Unset: []string{"effort"}},
			want: editFixture,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			from := test.from
			if from == "" {
				from = editFixture
				if test.edit.Route == "tr" {
					from = targetRoleFixture
				}
			}
			got, err := EditBindings([]byte(from), test.edit)
			if err != nil {
				t.Fatalf("EditBindings: %v", err)
			}
			if string(got) != test.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
}

func TestEditBindingsRejectsInvalidEdits(t *testing.T) {
	tests := map[string]struct {
		edit RouteEdit
		want string
	}{
		"invalid subagent effort":    {edit: RouteEdit{Route: "sol", Set: map[string]string{"subagentMaxEffort": "ultra"}}, want: "binding.subagent_max_effort_invalid"},
		"removing a required field":  {edit: RouteEdit{Route: "sol", Unset: []string{"provider"}}, want: "binding.route_incomplete"},
		"role without a model":       {edit: RouteEdit{Route: "opus55", Role: "research", Set: map[string]string{"provider": "openai"}}, want: "binding.role_incomplete"},
		"unknown route":              {edit: RouteEdit{Route: "missing", Set: map[string]string{"effort": "low"}}, want: `route "missing" is not in the bindings file`},
		"unknown target":             {edit: RouteEdit{Route: "sol", Target: "vim", Set: map[string]string{"effort": "low"}}, want: `unknown target "vim"`},
		"invalid role":               {edit: RouteEdit{Route: "sol", Role: "Bad_Name", Set: map[string]string{"model": "x"}}, want: `unknown role "Bad_Name"`},
		"target role the base lacks": {edit: RouteEdit{Route: "sol", Target: "codex", Role: "tiny", Set: map[string]string{"model": "x"}}, want: "binding.target_role_unbound"},
		"subagent cap per target":    {edit: RouteEdit{Route: "sol", Target: "codex", Set: map[string]string{"subagentMaxEffort": "high"}}, want: "applies to the whole route"},
		"empty value":                {edit: RouteEdit{Route: "sol", Set: map[string]string{"model": " "}}, want: "use route unset"},
		"multi-line value":           {edit: RouteEdit{Route: "sol", Set: map[string]string{"model": "a\nb"}}, want: "single line"},
		"nothing to change":          {edit: RouteEdit{Route: "sol"}, want: "nothing to change"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EditBindings([]byte(editFixture), test.edit)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("err = %v, want %q (output %q)", err, test.want, got)
			}
		})
	}
}

func TestEditBindingsSetThenUnsetRestoresBytes(t *testing.T) {
	tests := map[string]struct {
		data string
		set  RouteEdit
	}{
		"new target":          {data: editFixture, set: RouteEdit{Route: "sol", Target: "oh-my-pi", Set: map[string]string{"effort": "medium"}}},
		"new targets map":     {data: editFixture, set: RouteEdit{Route: "plain", Target: "codex", Set: map[string]string{"model": "x"}}},
		"new target role":     {data: editFixture, set: RouteEdit{Route: "opus55", Target: "codex", Role: "tiny", Set: map[string]string{"provider": "openai", "effort": "low"}}},
		"new flow role field": {data: editFixture, set: RouteEdit{Route: "opus55", Role: "tiny", Set: map[string]string{"effort": "low"}}},
		"CRLF new target":     {data: strings.ReplaceAll(editFixture, "\n", "\r\n"), set: RouteEdit{Route: "sol", Target: "pi", Set: map[string]string{"model": "x"}}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			edited, err := EditBindings([]byte(test.data), test.set)
			if err != nil || string(edited) == test.data {
				t.Fatalf("set: err=%v changed=%t", err, string(edited) != test.data)
			}
			unset := RouteEdit{Route: test.set.Route, Target: test.set.Target, Role: test.set.Role}
			for field := range test.set.Set {
				unset.Unset = append(unset.Unset, field)
			}
			restored, err := EditBindings(edited, unset)
			if err != nil || string(restored) != test.data {
				t.Fatalf("unset: err=%v\n%q", err, restored)
			}
		})
	}
}

func TestRouteSourceReturnsTheEntryAsWritten(t *testing.T) {
	got, err := RouteSource([]byte(editFixture), "plain")
	if err != nil || got != "  plain:\n    provider: openai\n    model: gpt-6-sol\n    effort: low\n" {
		t.Fatalf("RouteSource = %q, %v", got, err)
	}
}
