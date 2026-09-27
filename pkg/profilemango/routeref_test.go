package profilemango

import (
	"strings"
	"testing"
)

const routeRefBindingsYAML = targetedBindingsYAML + `    roles:
      tiny: {provider: openai, model: gpt-5.6-mini}
`

func TestRenderRouteRefs(t *testing.T) {
	t.Parallel()
	bindings, diagnostics := ParseBindings([]byte(routeRefBindingsYAML))
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics.Error())
	}
	cases := map[string]struct {
		content, target, want, wantErr string
	}{
		"base route for an agent without override": {content: "Use {{route.main.model}} via {{route.main.provider}}/{{route.main.transport}}.", target: "codex", want: "Use gpt-5.6 via openai/native."},
		"target override wins":                     {content: "Use {{route.main.provider}}/{{route.main.model}} at {{route.main.effort}}", target: "claude-code", want: "Use anthropic/claude-sonnet-5 at high"},
		"role field":                               {content: "tiny: {{route.main.roles.tiny.model}}", target: "codex", want: "tiny: gpt-5.6-mini"},
		"other braces stay literal":                {content: "{{ .Model }} {{routes.main}} {route.main.model}", target: "codex", want: "{{ .Model }} {{routes.main}} {route.main.model}"},
		"unknown route":                            {content: "a\n{{route.nope.model}}", wantErr: `line 2: {{route.nope.model}}: route "nope" is not in the bindings file; known routes: main`},
		"unknown field":                            {content: "{{route.main.colour}}", wantErr: `unknown route field "colour"`},
		"unknown role":                             {content: "{{route.main.roles.worker.model}}", wantErr: `route "main" has no role "worker"; its roles: tiny`},
		"unset role effort":                        {content: "{{route.main.roles.tiny.effort}}", wantErr: `role "tiny" sets no effort`},
		"malformed path":                           {content: "{{route.main.targets.codex.model}}", wantErr: "expected {{route.<name>.<field>}}"},
		"unclosed on its line":                     {content: "{{route.main.model\n}}", wantErr: "line 1: {{route. placeholder is not closed"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := RenderRouteRefs([]byte(test.content), bindings, test.target)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil || string(got) != test.want {
				t.Fatalf("render = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestWithoutRouteRefs(t *testing.T) {
	t.Parallel()
	cases := map[string]struct{ line, want string }{
		"drops closed placeholders": {line: "model {{route.main.model}} or gpt-5.6", want: "model  or gpt-5.6"},
		"keeps an unclosed one":     {line: "{{route.main.model gpt-5.6", want: "{{route.main.model gpt-5.6"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if got := WithoutRouteRefs(test.line); got != test.want {
				t.Fatalf("WithoutRouteRefs(%q) = %q, want %q", test.line, got, test.want)
			}
		})
	}
}
