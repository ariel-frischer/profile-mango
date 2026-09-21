package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

func TestInstallOptionValidationTable(t *testing.T) {
	tests := map[string]struct {
		options installOptions
		wantErr string
	}{
		"requires-selection":   {options: installOptions{}, wantErr: "at least one"},
		"all-conflicts-target": {options: installOptions{all: true, targets: []string{"codex@0.154.0"}}, wantErr: "mutually exclusive"},
		"yes-needs-apply":      {options: installOptions{targets: []string{"codex@0.154.0"}, yes: true, expectPlan: "abc"}, wantErr: "--yes requires --apply"},
		"yes-needs-hash":       {options: installOptions{targets: []string{"codex@0.154.0"}, apply: true, yes: true}, wantErr: "--yes requires --expect-plan"},
		"json-apply-bound":     {options: installOptions{targets: []string{"codex@0.154.0"}, apply: true, jsonOutput: true}, wantErr: "JSON apply requires"},
		"plan-valid":           {options: installOptions{targets: []string{"codex@0.154.0"}}, wantErr: ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateInstallOptions(test.options)
			if test.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestInstallTargetAndConfigPathParsing(t *testing.T) {
	requests, err := installTargets(installOptions{
		targets: []string{"codex@0.154.0", "opencode@1.18.31"},
		configs: []string{"opencode=/synthetic/opencode.json", "codex=/synthetic/config.toml"},
	}, install.DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0].Target.Name != "codex" || requests[1].ConfigPath != "/synthetic/opencode.json" {
		t.Fatalf("requests = %#v", requests)
	}
	if _, err := installTargets(installOptions{targets: []string{"codex"}}, install.DefaultRegistry()); err == nil {
		t.Fatal("target without exact version was accepted")
	}
}

func TestInstallProductionPlanBlockedWithoutTargetRead(t *testing.T) {
	root := t.TempDir()
	profiles := filepath.Join(root, "profiles")
	if err := os.MkdirAll(filepath.Join(profiles, "route-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profiles, "route-only", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: route\n")
	bindings := filepath.Join(root, "bindings.yaml")
	writeFile(t, bindings, "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
	targetPath := filepath.Join(root, "must-not-be-read", "config.toml")
	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := runInstall(cmd, "route-only", installOptions{
		profiles: profiles, resourceRoot: root, bindings: bindings,
		targets: []string{"codex@0.154.0"}, configs: []string{"codex=" + targetPath},
		jsonOutput: true,
	})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"status": "blocked"`) || !strings.Contains(stdout.String(), "production target installation is blocked") {
		t.Fatalf("unexpected plan: %s", stdout.String())
	}
	if _, statErr := os.Stat(filepath.Dir(targetPath)); !os.IsNotExist(statErr) {
		t.Fatalf("blocked production plan touched target path: %v", statErr)
	}
}

func TestInstallAllIncludesOpenCodeAndExcludesExperimentalJcode(t *testing.T) {
	registry := install.DefaultRegistry()
	requests := registry.Targets()
	joined := make([]string, 0, len(requests))
	for _, target := range requests {
		joined = append(joined, target.String())
	}
	text := strings.Join(joined, ",")
	if !strings.Contains(text, "opencode@1.18.31") || strings.Contains(text, "ariel-jcode") {
		t.Fatalf("production targets = %s", text)
	}
}

func TestInstallConsentRequiresTerminalOrHash(t *testing.T) {
	if terminalInput(strings.NewReader("y\n")) {
		t.Fatal("redirected input was accepted as terminal consent")
	}
	options := installOptions{targets: []string{"codex@0.154.0"}, apply: true, yes: true, expectPlan: "plan"}
	if err := validateInstallOptions(options); err != nil {
		t.Fatal(err)
	}
	if got := options.expectPlanOrPlanID(install.Plan{PlanID: "other"}); got != "plan" {
		t.Fatalf("expected plan = %q", got)
	}
}
