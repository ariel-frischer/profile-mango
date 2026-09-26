package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/spf13/cobra"
)

const baseOpenAIBindings = "routes:\n  main:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-6-sol\n    effort: high\n"

func TestInstallAllUsesPerTargetRoutes(t *testing.T) {
	cases := map[string]struct {
		bindings    string
		claudeModel string
		claudeState string
		claudeWhy   string
	}{
		"claude-code override": {bindings: baseOpenAIBindings + "    targets:\n      claude-code:\n        provider: anthropic\n        model: claude-fable-5-1\n", claudeModel: "claude-fable-5-1", claudeState: install.StatusReady},
		"flat openai route":    {bindings: baseOpenAIBindings, claudeState: install.StatusBlocked, claudeWhy: "requires provider anthropic"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			plan := planAllWithBindings(t, test.bindings)
			claude, codex := targetPlanNamed(t, plan, "claude-code"), targetPlanNamed(t, plan, "codex")
			if claude.Status != test.claudeState {
				t.Fatalf("claude-code status %s: %#v", claude.Status, claude)
			}
			if !strings.Contains(claude.Reason, test.claudeWhy) {
				t.Fatalf("claude-code reason %q", claude.Reason)
			}
			if test.claudeModel != "" && !planHasField(claude, "profile.model", test.claudeModel) {
				t.Fatalf("claude-code fields %#v", claude.Fields)
			}
			if codex.Status != install.StatusReady || !planHasField(codex, "default.config.model_provider", "openai") || !planHasField(codex, "default.config.model", "gpt-6-sol") {
				t.Fatalf("codex plan %#v", codex)
			}
		})
	}
}

func planAllWithBindings(t *testing.T, bindingsYAML string) install.Plan {
	t.Helper()
	root := t.TempDir()
	profiles := filepath.Join(root, "profiles")
	if err := os.MkdirAll(filepath.Join(profiles, "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profiles, "default", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: default\nspec:\n  routeRef: main\n")
	bindings := filepath.Join(root, "bindings.yaml")
	writeFile(t, bindings, bindingsYAML)
	if err := os.MkdirAll(filepath.Join(root, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	configs := make([]string, 0, 8)
	for _, target := range install.DefaultRegistry().Targets() {
		configs = append(configs, target.Name+"="+filepath.Join(root, "target", target.Name+".config"))
	}
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	err := runInstall(cmd, "default", installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, all: true, configs: configs, jsonOutput: true})
	var plan install.Plan
	if decodeErr := json.Unmarshal(output.Bytes(), &plan); decodeErr != nil {
		t.Fatalf("decode plan: %v (install error %v)\n%s", decodeErr, err, output.String())
	}
	return plan
}

func targetPlanNamed(t *testing.T, plan install.Plan, name string) install.TargetPlan {
	t.Helper()
	for _, target := range plan.Targets {
		if target.Target.Name == name {
			return target
		}
	}
	t.Fatalf("plan has no %s target", name)
	return install.TargetPlan{}
}

func planHasField(target install.TargetPlan, path, value string) bool {
	for _, field := range target.Fields {
		if field.Path == path && field.After == value {
			return true
		}
	}
	return false
}
