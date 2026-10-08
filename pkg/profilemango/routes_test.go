package profilemango

import (
	"reflect"
	"strings"
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

const rolesBindingsYAML = flatBindingsYAML + `    roles:
      research:
        provider: opencode-go
        model: gpt-6-luna
        effort: high
      tiny:
        provider: opencode-go
        model: glm-5.3-flash
`

func roleBindingsYAML(role string) string {
	return flatBindingsYAML + "    roles:\n      " + role + ":\n        provider: openai\n        model: m\n"
}

func TestParseBindingsRoles(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml    string
		code    string
		path    string
		message string
	}{
		"valid roles":           {yaml: rolesBindingsYAML},
		"every portable role":   {yaml: roleBindingsYAML("worker") + "      planner: {provider: a, model: m}\n      research: {provider: a, model: m}\n      tiny: {provider: a, model: m}\n"},
		"empty roles":           {yaml: flatBindingsYAML + "    roles: {}\n", code: "binding.roles_empty", path: "routes.main.roles"},
		"null roles":            {yaml: flatBindingsYAML + "    roles:\n", code: "yaml.null", path: "routes.main.roles"},
		"null role":             {yaml: flatBindingsYAML + "    roles:\n      worker:\n", code: "yaml.null", path: "routes.main.roles.worker"},
		"empty role":            {yaml: flatBindingsYAML + "    roles:\n      worker: {}\n", code: "binding.role_incomplete", path: "routes.main.roles.worker"},
		"missing model":         {yaml: flatBindingsYAML + "    roles:\n      worker:\n        provider: openai\n", code: "binding.role_incomplete", path: "routes.main.roles.worker"},
		"unknown role key":      {yaml: roleBindingsYAML("worker") + "        transport: native\n", code: "yaml.strict", path: "document"},
		"unknown role name":     {yaml: roleBindingsYAML("reviewer"), code: "binding.role_unknown", path: "routes.main.roles.reviewer", message: "expected one of worker, planner, research, tiny"},
		"invalid role name":     {yaml: roleBindingsYAML("Fast_Helper"), code: "binding.role_unknown", path: "routes.main.roles.Fast_Helper"},
		"reserved default":      {yaml: roleBindingsYAML("default"), code: "binding.role_unknown", path: "routes.main.roles.default", message: "remove roles.default"},
		"old task slot":         {yaml: roleBindingsYAML("task"), code: "binding.role_unknown", path: "routes.main.roles.task", message: `use the portable role "worker"`},
		"old plan slot":         {yaml: roleBindingsYAML("plan"), code: "binding.role_unknown", path: "routes.main.roles.plan", message: `use the portable role "planner"`},
		"old slow slot":         {yaml: roleBindingsYAML("slow"), code: "binding.role_unknown", path: "routes.main.roles.slow", message: `use the portable role "planner"`},
		"old smol slot":         {yaml: roleBindingsYAML("smol"), code: "binding.role_unknown", path: "routes.main.roles.smol", message: `use the portable role "research"`},
		"old commit slot":       {yaml: roleBindingsYAML("commit"), code: "binding.role_unknown", path: "routes.main.roles.commit", message: `use the portable role "tiny"`},
		"old advisor slot":      {yaml: roleBindingsYAML("advisor"), code: "binding.role_unknown", path: "routes.main.roles.advisor", message: "no portable role"},
		"old vision slot":       {yaml: roleBindingsYAML("vision"), code: "binding.role_unknown", path: "routes.main.roles.vision", message: "no portable role"},
		"duplicate role":        {yaml: flatBindingsYAML + "    roles:\n      worker:\n        provider: a\n        model: m\n      worker:\n        provider: b\n        model: m\n", code: "yaml.duplicate_key", path: "routes.main.roles.worker"},
		"null role effort":      {yaml: roleBindingsYAML("worker") + "        effort:\n", code: "yaml.null", path: "routes.main.roles.worker.effort"},
		"subagent max effort":   {yaml: flatBindingsYAML + "    subagentMaxEffort: high\n"},
		"null subagent effort":  {yaml: flatBindingsYAML + "    subagentMaxEffort:\n", code: "yaml.null", path: "routes.main.subagentMaxEffort"},
		"unknown subagent cap":  {yaml: flatBindingsYAML + "    subagentMaxEffort: ultra\n", code: "binding.subagent_max_effort_invalid", path: "routes.main.subagentMaxEffort", message: "expected one of minimal, low, medium, high, xhigh, max"},
		"override subagent cap": {yaml: flatBindingsYAML + "    targets:\n      oh-my-pi:\n        subagentMaxEffort: high\n", code: "yaml.strict", path: "document"},
		"target role override":  {yaml: rolesBindingsYAML + "    targets:\n      codex:\n        roles:\n          research:\n            provider: openai\n"},
		"target role unbound":   {yaml: rolesBindingsYAML + "    targets:\n      codex:\n        roles:\n          worker:\n            provider: openai\n", code: "binding.target_role_unbound", path: "routes.main.targets.codex.roles.worker", message: "add roles.worker first"},
		"target role unknown":   {yaml: rolesBindingsYAML + "    targets:\n      codex:\n        roles:\n          smol:\n            provider: openai\n", code: "binding.role_unknown", path: "routes.main.targets.codex.roles.smol", message: `use the portable role "research"`},
		"target role empty":     {yaml: rolesBindingsYAML + "    targets:\n      codex:\n        roles:\n          research: {}\n", code: "binding.target_role_override_empty", path: "routes.main.targets.codex.roles.research"},
		"target roles empty":    {yaml: rolesBindingsYAML + "    targets:\n      codex:\n        roles: {}\n", code: "binding.roles_empty", path: "routes.main.targets.codex.roles"},
		"target role field":     {yaml: rolesBindingsYAML + "    targets:\n      codex:\n        roles:\n          research:\n            transport: native\n", code: "yaml.strict", path: "document"},
		"target role no base":   {yaml: flatBindingsYAML + "    targets:\n      codex:\n        roles:\n          research:\n            model: m\n", code: "binding.target_role_unbound", path: "routes.main.targets.codex.roles.research"},
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
				if diagnostic.Code == test.code && diagnostic.Path == test.path && diagnostic.Severity == SeverityError && strings.Contains(diagnostic.Message, test.message) {
					return
				}
			}
			t.Fatalf("missing %s at %s containing %q in %v", test.code, test.path, test.message, diagnostics)
		})
	}
}

func TestRouteForKeepsRolesAcrossTargetOverrides(t *testing.T) {
	t.Parallel()
	bindings, diagnostics := ParseBindings([]byte(rolesBindingsYAML + "    subagentMaxEffort: high\n    targets:\n      oh-my-pi:\n        model: gpt-5.7\n"))
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	route, _ := bindings.RouteFor("main", "oh-my-pi")
	want := map[string]RoleRoute{
		"research": {Provider: "opencode-go", Model: "gpt-6-luna", Effort: "high"},
		"tiny":     {Provider: "opencode-go", Model: "glm-5.3-flash"},
	}
	if route.Model != "gpt-5.7" || route.SubagentMaxEffort != "high" || !reflect.DeepEqual(route.Roles, want) {
		t.Fatalf("route = %#v", route)
	}
}

func TestRouteForMergesTargetRoleOverrides(t *testing.T) {
	t.Parallel()
	bindings, diagnostics := ParseBindings([]byte(rolesBindingsYAML + "    targets:\n      codex:\n        roles:\n          research:\n            provider: openai\n          tiny:\n            model: glm-5.4\n            effort: low\n"))
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	base := map[string]RoleRoute{
		"research": {Provider: "opencode-go", Model: "gpt-6-luna", Effort: "high"},
		"tiny":     {Provider: "opencode-go", Model: "glm-5.3-flash"},
	}
	cases := map[string]struct {
		target string
		want   map[string]RoleRoute
	}{
		"overridden target": {target: "codex", want: map[string]RoleRoute{
			"research": {Provider: "openai", Model: "gpt-6-luna", Effort: "high"},
			"tiny":     {Provider: "opencode-go", Model: "glm-5.4", Effort: "low"},
		}},
		"other target": {target: "oh-my-pi", want: base},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			route, _ := bindings.RouteFor("main", test.target)
			if !reflect.DeepEqual(route.Roles, test.want) {
				t.Fatalf("roles = %#v; want %#v", route.Roles, test.want)
			}
		})
	}
	bindings.RouteFor("main", "codex")
	if !reflect.DeepEqual(bindings.Routes["main"].Roles, base) {
		t.Fatalf("RouteFor mutated the base roles: %#v", bindings.Routes["main"].Roles)
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

func TestParseBindingsDefaultsTransportAndAuthentication(t *testing.T) {
	t.Parallel()
	minimal := "routes:\n  main:\n    provider: openai\n    model: gpt-5.6\n    effort: high\n"
	cases := map[string]struct {
		yaml   string
		target string
		want   RouteBinding
	}{
		"minimal base": {yaml: minimal, target: "codex", want: RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}},
		"minimal override": {yaml: minimal + "    targets:\n      claude-code:\n        provider: anthropic\n        model: claude-sonnet-5\n", target: "claude-code",
			want: RouteBinding{Provider: "anthropic", Transport: "native", Authentication: "oauth", Model: "claude-sonnet-5", Effort: "high"}},
		"explicit kept": {yaml: "routes:\n  main:\n    provider: openai\n    transport: api\n    authentication: api-key\n    model: gpt-5.6\n    effort: high\n", target: "codex",
			want: RouteBinding{Provider: "openai", Transport: "api", Authentication: "api-key", Model: "gpt-5.6", Effort: "high"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			bindings, diagnostics := ParseBindings([]byte(test.yaml))
			if diagnostics.HasErrors() {
				t.Fatal(diagnostics)
			}
			got, _ := bindings.RouteFor("main", test.target)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("RouteFor = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseBindingsStillRequiresProviderModelEffort(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml string
	}{
		"provider": {yaml: "routes:\n  main:\n    model: gpt-5.6\n    effort: high\n"},
		"model":    {yaml: "routes:\n  main:\n    provider: openai\n    effort: high\n"},
		"effort":   {yaml: "routes:\n  main:\n    provider: openai\n    model: gpt-5.6\n"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diagnostics := ParseBindings([]byte(test.yaml))
			if !hasCode(diagnostics, "binding.route_incomplete") {
				t.Fatalf("missing binding.route_incomplete in %v", diagnostics)
			}
		})
	}
}
