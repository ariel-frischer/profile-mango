package integration_test

import (
	"encoding/json"
	"os"
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
		if result.err != nil {
			t.Fatalf("%s staged preview must exit 0: %v\n%s", target, result.err, result.stderr)
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

// assertExampleInstallGate installs the coding example with --all into a
// synthetic home holding only Claude Code and Codex: the supported subset plans
// ready with every skipped requirement listed, and --strict still blocks.
func assertExampleInstallGate(t *testing.T, binary, repoRoot string, env []string, root, profiles, bindings string) {
	t.Helper()
	home := t.TempDir()
	for _, folder := range []string{".claude", ".codex"} {
		if err := os.MkdirAll(filepath.Join(home, folder), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	env = replaceEnvironment(withoutAgentCommands(env, root), []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config")})
	args := []string{"install", "coding", "--profiles", profiles, "--resource-root", root, "--bindings", bindings, "--all", "--json"}
	plan := runExampleInstall(t, binary, repoRoot, env, args, true)
	if plan.Status != "ready" || len(plan.Targets) != len(publicTargetVersions()) {
		t.Fatalf("unexpected example install plan: %#v", plan)
	}
	for _, target := range plan.Targets {
		installed := target.Target.Name == "claude-code" || target.Target.Name == "codex"
		// permissions, tools, instructions, skills, role-definitions (named-only
		// install), roles (bound tiny with no profile role), subagentMaxEffort.
		if installed && (target.Status != "ready" || len(target.SkippedRequirements) != 7) {
			t.Fatalf("%s must plan ready with seven skipped requirements: %#v", target.Target.Name, target)
		}
		if !installed && target.Status != "skipped" {
			t.Fatalf("%s must be skipped as not installed: %#v", target.Target.Name, target)
		}
	}
	strict := runExampleInstall(t, binary, repoRoot, env, append(args, "--strict"), false)
	if strict.Status != "blocked" {
		t.Fatalf("--strict example install must block: %#v", strict)
	}
}

type examplePlan struct {
	Status  string `json:"status"`
	Targets []struct {
		Target struct {
			Name string `json:"name"`
		} `json:"target"`
		Status              string `json:"status"`
		SkippedRequirements []any  `json:"skippedRequirements"`
	} `json:"targets"`
}

func runExampleInstall(t *testing.T, binary, repoRoot string, env, args []string, wantSuccess bool) examplePlan {
	t.Helper()
	result := runCommand(binary, repoRoot, env, args...)
	if (result.err == nil) != wantSuccess {
		t.Fatalf("install %v: err=%v\n%s\n%s", args, result.err, result.stdout, result.stderr)
	}
	var plan examplePlan
	if err := json.Unmarshal([]byte(result.stdout), &plan); err != nil {
		t.Fatalf("install plan: %v\n%s", err, result.stdout)
	}
	return plan
}

func publicTargetVersions() map[string]string {
	return map[string]string{
		"claude-code": "2.1.278",
		"codex":       "0.157.1",
		"hermes":      "0.21.3",
		"oh-my-pi":    "18.3.2",
		"openclaw":    "2026.9.5",
		"opencode":    "1.18.31",
		"pi":          "0.87.1",
	}
}
