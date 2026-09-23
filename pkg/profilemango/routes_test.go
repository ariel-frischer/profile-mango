package profilemango

import (
	"reflect"
	"testing"
)

const flatBindingsYAML = `routes:
  main:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`

const targetedBindingsYAML = flatBindingsYAML + `    targets:
      claude-code:
        provider: anthropic
        model: claude-sonnet-5
`

func TestParseBindingsTargets(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml string
		code string
		path string
	}{
		"flat":           {yaml: flatBindingsYAML},
		"targeted":       {yaml: targetedBindingsYAML},
		"unknown target": {yaml: flatBindingsYAML + "    targets:\n      claude:\n        provider: anthropic\n", code: "binding.target_unknown", path: "routes.main.targets.claude"},
		"empty override": {yaml: flatBindingsYAML + "    targets:\n      codex: {}\n", code: "binding.target_override_empty", path: "routes.main.targets.codex"},
		"unknown field":  {yaml: flatBindingsYAML + "    targets:\n      codex:\n        token: secret\n", code: "yaml.strict", path: "document"},
		"null override":  {yaml: flatBindingsYAML + "    targets:\n      codex:\n", code: "yaml.null", path: "routes.main.targets.codex"},
		"empty targets":  {yaml: flatBindingsYAML + "    targets: {}\n", code: "binding.targets_empty", path: "routes.main.targets"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diagnostics := ParseBindings([]byte(test.yaml))
			if test.code == "" {
				if diagnostics.HasErrors() {
					t.Fatalf("unexpected diagnostics: %v", diagnostics)
				}
				return
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == test.code && diagnostic.Path == test.path {
					return
				}
			}
			t.Fatalf("missing %s at %s in %v", test.code, test.path, diagnostics)
		})
	}
}

func TestRouteForMergesTargetOverride(t *testing.T) {
	t.Parallel()
	bindings, diagnostics := ParseBindings([]byte(targetedBindingsYAML))
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	base := RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
	cases := map[string]struct {
		route  string
		target string
		want   RouteBinding
		found  bool
	}{
		"override":      {route: "main", target: "claude-code", want: RouteBinding{Provider: "anthropic", Transport: "native", Authentication: "oauth", Model: "claude-sonnet-5", Effort: "high"}, found: true},
		"base":          {route: "main", target: "codex", want: base, found: true},
		"empty target":  {route: "main", target: "", want: base, found: true},
		"missing route": {route: "absent", target: "codex"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, found := bindings.RouteFor(test.route, test.target)
			if found != test.found || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("RouteFor = %#v, %v; want %#v, %v", got, found, test.want, test.found)
			}
		})
	}
}

func TestFlatRouteForIsUnchanged(t *testing.T) {
	t.Parallel()
	bindings, diagnostics := ParseBindings([]byte(flatBindingsYAML))
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	for _, target := range RouteTargets {
		got, _ := bindings.RouteFor("main", target)
		if !reflect.DeepEqual(got, bindings.Routes["main"]) {
			t.Fatalf("%s: flat route changed to %#v", target, got)
		}
	}
}

func TestBuildPlanUsesTargetRoute(t *testing.T) {
	t.Parallel()
	bindings, diagnostics := ParseBindings([]byte(targetedBindingsYAML))
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	profile := ResolvedProfile{Metadata: Metadata{Name: "demo"}, RouteRef: "main"}
	cases := map[string]struct {
		target   string
		provider string
	}{
		"claude-code": {target: "claude-code", provider: "anthropic"},
		"codex":       {target: "codex", provider: "openai"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan, err := BuildPlan(profile, test.target, bindings, nil)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Route.Provider != test.provider || plan.Route.Targets != nil {
				t.Fatalf("route %#v", plan.Route)
			}
		})
	}
}
