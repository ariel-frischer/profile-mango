package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

// withRouteEffort rewrites the synthetic codex bindings to request effort.
func withRouteEffort(t *testing.T, bindings, effort string) {
	t.Helper()
	data, err := os.ReadFile(bindings)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, bindings, strings.Replace(string(data), "effort: high", "effort: "+effort, 1))
}

func applyPlannedInstall(t *testing.T, options installOptions) string {
	t.Helper()
	output := runInstallForTest(t, options)
	planID := regexp.MustCompile(`(?m)^plan ([0-9a-f]{64}) \(ready\)$`).FindStringSubmatch(output)
	if planID == nil {
		t.Fatalf("plan not ready:\n%s", output)
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	runInstallForTest(t, options)
	return output
}

func TestInstallCodexAppliesMediumEffortInSandbox(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	profiles, bindings := writeCodexInstallInputs(t, root)
	withRouteEffort(t, bindings, "medium")
	config := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, config, "# keep\n")
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"codex"}, makeDefault: true}
	plan := applyPlannedInstall(t, options)
	if !strings.Contains(plan, "effort medium") || strings.Contains(plan, "NOT APPLIED") {
		t.Fatalf("plan must show medium effort as applied:\n%s", plan)
	}
	want := "model_reasoning_effort = \"medium\"\n"
	for _, path := range []string{config, filepath.Join(filepath.Dir(config), "route-only.config.toml")} {
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), want) {
			t.Fatalf("%s = %q, err=%v; want %q", path, data, err, want)
		}
	}
}

// effortTargetSandbox plans one of claude-code or opencode at explicit synthetic
// config paths with the route effort; claude-code gets an Anthropic override.
func effortTargetSandbox(t *testing.T, target, effort string) (installOptions, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	profiles, bindings := writeCodexInstallInputs(t, root)
	writeFile(t, bindings, "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: "+effort+"\n    targets:\n      claude-code:\n        provider: anthropic\n        model: claude-sonnet-4-5\n")
	dir := filepath.Join(root, "target")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	config, installed := filepath.Join(dir, "settings.json"), filepath.Join(dir, "profiles", "route-only.json")
	if target == "opencode" {
		config, installed = filepath.Join(dir, "opencode.json"), filepath.Join(dir, "agents", "route-only.md")
	}
	return installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{target}, configs: []string{target + "=" + config}}, installed
}

func TestInstallWritesSupportedEffort(t *testing.T) {
	tests := map[string]struct {
		target string
		want   string
	}{
		"claude-code effortLevel": {target: "claude-code", want: `"effortLevel": "medium"`},
		"opencode agent variant":  {target: "opencode", want: "variant: \"medium\"\n"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options, installed := effortTargetSandbox(t, test.target, "medium")
			plan := applyPlannedInstall(t, options)
			if !strings.Contains(plan, "effort medium") || strings.Contains(plan, "NOT APPLIED") {
				t.Fatalf("plan must show medium effort as applied:\n%s", plan)
			}
			data, err := os.ReadFile(installed)
			if err != nil || !strings.Contains(string(data), test.want) {
				t.Fatalf("installed %s = %q, err=%v; want %q", installed, data, err, test.want)
			}
		})
	}
}

func TestInstallUnsupportedEffortIsExplicitlyNotApplied(t *testing.T) {
	tests := map[string]struct {
		target, effort, reason, absent string
	}{
		"claude-code max": {target: "claude-code", effort: "max", reason: "Claude Code 2.1.278 settings effortLevel accepts only low, medium, high, xhigh", absent: "effortLevel"},
		"opencode ultra":  {target: "opencode", effort: "ultra", reason: "OpenCode 1.18.31 built-in model variants are only none, minimal, low, medium, high, xhigh, max", absent: "variant"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options, installed := effortTargetSandbox(t, test.target, test.effort)
			human := applyPlannedInstall(t, options)
			if !strings.Contains(human, "    effort "+test.effort+": NOT APPLIED ("+test.reason+")\n") || !strings.Contains(human, "effort "+test.effort+" NOT APPLIED") {
				t.Fatalf("human plan lacks the explicit not-applied effort line:\n%s", human)
			}
			if data, err := os.ReadFile(installed); err != nil || strings.Contains(string(data), test.absent) {
				t.Fatalf("installed %s = %q, err=%v; must not carry %s", installed, data, err, test.absent)
			}
			options.jsonOutput = true
			var plan install.Plan
			if err := json.Unmarshal([]byte(runInstallForTest(t, options)), &plan); err != nil {
				t.Fatal(err)
			}
			want := install.SkippedRequirement{Requirement: install.RequirementEffort, Value: test.effort, Reason: test.reason}
			if skipped := plan.Targets[0].SkippedRequirements; len(skipped) != 1 || skipped[0] != want {
				t.Fatalf("JSON skippedRequirements = %#v, want %#v", skipped, want)
			}
			options.jsonOutput, options.strict = false, true
			if output, err := runInstallResult(t, options); err == nil || !strings.Contains(output, "effort "+test.effort+" is not applied") {
				t.Fatalf("strict plan must block on the unapplied effort: err=%v\n%s", err, output)
			}
		})
	}
}
