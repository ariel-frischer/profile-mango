package integration_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func testInstalledExampleProfiles(t *testing.T, binary, repoRoot string, env []string) {
	t.Helper()
	root := filepath.Join(repoRoot, "examples")
	profiles := filepath.Join(root, "profiles")
	bindings := filepath.Join(root, "bindings", "local.example.yaml")
	for _, profile := range []string{"coding", "review", "docs-research"} {
		t.Run("example-"+profile, func(t *testing.T) {
			result := runCommand(binary, repoRoot, env, validateArgs(filepath.Join(profiles, profile, "profile.yaml"), bindings)...)
			if result.err != nil {
				t.Fatalf("example validation failed: %v\n%s", result.err, result.stderr)
			}
			assertExampleRenders(t, binary, repoRoot, env, root, profiles, bindings, profile)
		})
	}
	assertExampleInstallGate(t, binary, repoRoot, env, root, profiles, bindings)
}

func assertExampleRenders(t *testing.T, binary, repoRoot string, env []string, root, profiles, bindings, profile string) {
	t.Helper()
	for target, version := range publicTargetVersions() {
		out := filepath.Join(t.TempDir(), target)
		result := runCommand(binary, repoRoot, env, "render", profile, "--profiles", profiles, "--resource-root", root, "--bindings", bindings, "--target", target, "--target-version", version, "--out", out, "--preview", "--json")
		if result.err == nil {
			t.Fatalf("%s preview unexpectedly applicable", target)
		}
		var report struct {
			Profile    string `json:"profile"`
			Target     string `json:"target"`
			Applicable bool   `json:"applicable"`
			Artifacts  []any  `json:"artifacts"`
		}
		if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
			t.Fatalf("%s report: %v\n%s", target, err, result.stdout)
		}
		if report.Profile != profile || report.Target != target || report.Applicable || len(report.Artifacts) == 0 {
			t.Fatalf("unexpected %s report: %#v", target, report)
		}
	}
}

func assertExampleInstallGate(t *testing.T, binary, repoRoot string, env []string, root, profiles, bindings string) {
	t.Helper()
	result := runCommand(binary, repoRoot, env, "install", "coding", "--profiles", profiles, "--resource-root", root, "--bindings", bindings, "--all", "--json")
	if result.err == nil {
		t.Fatal("production --all install unexpectedly succeeded")
	}
	var plan struct {
		Status  string `json:"status"`
		Targets []any  `json:"targets"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &plan); err != nil {
		t.Fatalf("install plan: %v\n%s", err, result.stdout)
	}
	if plan.Status != "blocked" || len(plan.Targets) != len(publicTargetVersions()) {
		t.Fatalf("unexpected production install gate: %#v", plan)
	}
}

func publicTargetVersions() map[string]string {
	return map[string]string{
		"claude-code": "2.1.278",
		"codex":       "0.154.0",
		"hermes":      "0.21.3",
		"oh-my-pi":    "18.2.6",
		"openclaw":    "2026.9.5",
		"opencode":    "1.18.31",
		"pi":          "0.86.1",
	}
}
