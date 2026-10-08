package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const routeTestBindings = `# keep this comment
routes:
  sol:
    provider: openai
    model: gpt-6-sol
    effort: high # daily
    targets:
      claude-code:
        provider: anthropic
        model: claude-opus-5-5
  spare:
    provider: openai
    model: gpt-6
    effort: low
    roles:
      research:
        provider: openai-codex # ChatGPT OAuth on Oh My Pi
        model: gpt-6-luna
`

// routeTestHome writes a sandbox profile home whose bindings file has mode 0640, and
// points HOME and the agent relocation variables at an empty sandbox so route set
// reads no real ownership manifest.
func routeTestHome(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", filepath.Join(home, "user"))
	for _, variable := range []string{"CODEX_HOME", "HERMES_HOME", "PI_CODING_AGENT_DIR", "XDG_CONFIG_HOME", "OPENCLAW_CONFIG_PATH", "CLAUDE_CONFIG_DIR"} {
		t.Setenv(variable, "")
	}
	bindings := filepath.Join(home, "bindings", "local.yaml")
	for path, content := range map[string]string{
		bindings: routeTestBindings,
		filepath.Join(home, "profiles", "daily", "profile.yaml"): "route: sol\n",
		filepath.Join(home, "profiles", "child", "profile.yaml"): "extends: daily\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	return home, bindings
}

func TestRouteSetAndUnsetEditTheBindingsFile(t *testing.T) {
	tests := map[string]struct {
		args      []string
		wantFile  string
		wantOut   []string
		wantError string
	}{
		"set base field": {
			args:     []string{"route", "set", "sol", "--effort", "medium"},
			wantFile: strings.Replace(routeTestBindings, "effort: high # daily", "effort: medium # daily", 1),
			wantOut:  []string{"-    effort: high # daily", "+    effort: medium # daily", "Profiles using route sol: child, daily", "No agent records profile daily; to adopt one: mango install daily --target <target>"},
		},
		"set new target entry": {
			args:     []string{"route", "set", "sol", "--target", "oh-my-pi", "--effort", "medium"},
			wantFile: strings.Replace(routeTestBindings, "model: claude-opus-5-5\n", "model: claude-opus-5-5\n      oh-my-pi:\n        effort: medium\n", 1),
			wantOut:  []string{"+      oh-my-pi:", "+        effort: medium", "--target oh-my-pi"},
		},
		"unset prunes empty maps": {
			args:     []string{"route", "unset", "sol", "--target", "claude-code", "provider", "model"},
			wantFile: strings.Replace(routeTestBindings, "    targets:\n      claude-code:\n        provider: anthropic\n        model: claude-opus-5-5\n", "", 1),
			wantOut:  []string{"-    targets:"},
		},
		"route without profiles": {
			args:     []string{"route", "set", "spare", "--model", "gpt-7"},
			wantFile: strings.Replace(routeTestBindings, "model: gpt-6\n", "model: gpt-7\n", 1),
			wantOut:  []string{"No profile uses route spare yet"},
		},
		"dry run writes nothing": {
			args:     []string{"route", "set", "sol", "--effort", "low", "--dry-run"},
			wantFile: routeTestBindings,
			wantOut:  []string{"+    effort: low # daily", "Dry run: nothing was written."},
		},
		"no change writes nothing": {
			args:     []string{"route", "set", "sol", "--effort", "high"},
			wantFile: routeTestBindings,
			wantOut:  []string{"No change:"},
		},
		"invalid effort": {
			args:      []string{"route", "set", "sol", "--subagent-max-effort", "ultra"},
			wantFile:  routeTestBindings,
			wantError: "binding.subagent_max_effort_invalid",
		},
		"unknown route":  {args: []string{"route", "set", "nope", "--effort", "low"}, wantFile: routeTestBindings, wantError: `route "nope" is not in the bindings file`},
		"unknown target": {args: []string{"route", "set", "sol", "--target", "vim", "--effort", "low"}, wantFile: routeTestBindings, wantError: `unknown target "vim"`},
		"set target role": {
			args:     []string{"route", "set", "spare", "--target", "codex", "--role", "research", "--provider", "openai"},
			wantFile: routeTestBindings + "    targets:\n      codex:\n        roles:\n          research:\n            provider: openai\n",
			wantOut:  []string{"+    targets:", "+          research:", "+            provider: openai", "No profile uses route spare yet"},
		},
		"set custom role": {
			args:     []string{"route", "set", "sol", "--role", "code", "--provider", "openai", "--model", "gpt-6"},
			wantFile: strings.Replace(routeTestBindings, "  spare:\n", "    roles:\n      code:\n        provider: openai\n        model: gpt-6\n  spare:\n", 1),
			wantOut:  []string{"+      code:"},
		},
		"invalid role": {args: []string{"route", "set", "sol", "--role", "Bad_Name", "--model", "x"}, wantFile: routeTestBindings, wantError: `unknown role "Bad_Name"`},
		"reserved role": {args: []string{"route", "set", "sol", "--role", "default", "--model", "x"}, wantFile: routeTestBindings, wantError: "remove roles.default"},
		"target role the base lacks": {args: []string{"route", "set", "sol", "--target", "codex", "--role", "tiny", "--model", "x"}, wantFile: routeTestBindings, wantError: "binding.target_role_unbound"},
		"no field flags":             {args: []string{"route", "set", "sol", "--target", "codex"}, wantFile: routeTestBindings, wantError: "pass at least one of"},
		"unknown unset field":        {args: []string{"route", "unset", "sol", "colour"}, wantFile: routeTestBindings, wantError: `unknown route field "colour"`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			home, bindings := routeTestHome(t)
			out, err := executeCommandResult(t, append(test.args, "--home", home)...)
			if test.wantError == "" && err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("error = %v, want %q", err, test.wantError)
			}
			for _, want := range test.wantOut {
				if !strings.Contains(out, want) {
					t.Errorf("output lacks %q:\n%s", want, out)
				}
			}
			data, readErr := os.ReadFile(bindings)
			info, statErr := os.Stat(bindings)
			if readErr != nil || statErr != nil || string(data) != test.wantFile || info.Mode().Perm() != 0o640 {
				t.Fatalf("bindings (mode %v, errors %v %v):\n%s\nwant:\n%s", info.Mode(), readErr, statErr, data, test.wantFile)
			}
		})
	}
}

func TestRouteUnsetTargetRolePrunesEmptyMaps(t *testing.T) {
	home, bindings := routeTestHome(t)
	if out, err := executeCommandResult(t, "route", "set", "spare", "--target", "codex", "--role", "research", "--provider", "openai", "--home", home); err != nil {
		t.Fatalf("route set: %v\n%s", err, out)
	}
	out, err := executeCommandResult(t, "route", "unset", "spare", "--target", "codex", "--role", "research", "provider", "--home", home)
	if err != nil || !strings.Contains(out, "-    targets:") || !strings.Contains(out, "-            provider: openai") {
		t.Fatalf("route unset: %v\n%s", err, out)
	}
	if data, err := os.ReadFile(bindings); err != nil || string(data) != routeTestBindings {
		t.Fatalf("bindings after unset (%v):\n%s", err, data)
	}
}

func TestRouteShowResolvesTheTargetRoute(t *testing.T) {
	home, _ := routeTestHome(t)
	out, err := executeCommandResult(t, "route", "show", "sol", "--target", "claude-code", "--home", home)
	if err != nil {
		t.Fatalf("route show: %v\n%s", err, out)
	}
	for _, want := range []string{"    effort: high # daily", "What claude-code gets:", "  provider: anthropic\n", "  model: claude-opus-5-5\n", "  effort: high\n", "Used by profiles: child, daily"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	out, err = executeCommandResult(t, "route", "show", "sol", "--target", "codex", "--json", "--home", home)
	var shown routeShowJSON
	if err != nil || json.Unmarshal([]byte(out), &shown) != nil || shown.Effective == nil || shown.Effective.Provider != "openai" {
		t.Fatalf("route show --json: %v\n%s", err, out)
	}
}

func TestRouteListNamesProfilesPerRoute(t *testing.T) {
	home, _ := routeTestHome(t)
	out, err := executeCommandResult(t, "route", "list", "--home", home)
	if err != nil {
		t.Fatalf("route list: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 || !strings.Contains(lines[1], "sol") || !strings.HasSuffix(lines[1], "child, daily") || !strings.HasSuffix(lines[2], "-") {
		t.Fatalf("route list:\n%s", out)
	}
}

func TestRouteEffortWarning(t *testing.T) {
	tests := map[string]struct {
		set  map[string]string
		warn bool
	}{
		"unknown effort warns":  {set: map[string]string{"effort": "turbo"}, warn: true},
		"agent effort is quiet": {set: map[string]string{"effort": "ultra"}},
		"no effort is quiet":    {set: map[string]string{"model": "gpt-6-sol"}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var output strings.Builder
			if err := writeEffortWarning(&output, profilemango.RouteEdit{Route: "sol", Set: test.set}); err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(output.String(), "warning:"); got != test.warn {
				t.Fatalf("warning = %q, want warn %v", output.String(), test.warn)
			}
		})
	}
}

// TestRouteSetHintReappliesOnlyAgentsOnItsProfiles pins that the route set hint
// re-applies the route's profile only to agents whose manifest records it: Codex runs
// daily as its default, Oh My Pi runs other, and following the hint leaves Oh My Pi alone.
func TestRouteSetHintReappliesOnlyAgentsOnItsProfiles(t *testing.T) {
	home, bindings := routeTestHome(t)
	writeFile(t, filepath.Join(home, "profiles", "other", "profile.yaml"), "route: spare\n")
	for _, dir := range []string{".codex", filepath.Join(".omp", "agent")} {
		if err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	options := installOptions{profiles: filepath.Join(home, "profiles"), resourceRoot: home, bindings: bindings}
	daily, other := options, options
	daily.targets, daily.makeDefault = []string{"codex"}, true
	other.targets = []string{"oh-my-pi"}
	useTestHome{}.planThenApply(t, "daily", daily)
	useTestHome{}.planThenApply(t, "other", other)

	out, err := executeCommandResult(t, "route", "set", "sol", "--effort", "medium", "--home", home)
	if err != nil {
		t.Fatalf("route set: %v\n%s", err, out)
	}
	for _, want := range []string{"apply with: mango install daily --target codex --default\n", "No agent records profile child; to adopt one: mango install child --target <target>\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("hint lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "oh-my-pi") || strings.Contains(out, "mango use") {
		t.Fatalf("hint names an agent on another profile or mango use:\n%s", out)
	}

	hint := strings.Fields(strings.TrimPrefix(regexp.MustCompile(`apply with: mango (.*)`).FindString(out), "apply with: mango "))
	plan, err := executeCommandResult(t, append(hint, "--home", home)...)
	planID := regexp.MustCompile(`plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(plan)
	if err != nil || planID == nil {
		t.Fatalf("hint %q plan: %v\n%s", hint, err, plan)
	}
	if applied, err := executeCommandResult(t, append(hint, "--home", home, "--apply", "--yes", "--expect-plan", planID[1])...); err != nil {
		t.Fatalf("hint apply: %v\n%s", err, applied)
	}
	report := statusForTest(t, options)
	if codex, omp := targetStatus(t, report, "codex"), targetStatus(t, report, "oh-my-pi"); codex.Profile != "daily" || codex.Source != install.SourceCurrent || omp.Profile != "other" {
		t.Fatalf("after the hint: codex %s (%s), oh-my-pi %s", codex.Profile, codex.Source, omp.Profile)
	}
}
