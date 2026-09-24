package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type installedConfig struct {
	target, path, before string
	installed            []byte
}

// installClaudeAndCodexAtDefault installs both targets in one transaction at their default paths under a per-test HOME.
func installClaudeAndCodexAtDefault(t *testing.T) []installedConfig {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	configs := []installedConfig{
		{target: "claude-code", path: filepath.Join(home, ".claude", "settings.json"), before: "{\n  \"theme\": \"dark\"\n}\n"},
		{target: "codex", path: filepath.Join(home, ".codex", "config.toml"), before: "# keep\nunknown = true\n"},
	}
	for _, config := range configs {
		if err := os.MkdirAll(filepath.Dir(config.path), 0o700); err != nil {
			t.Fatal(err)
		}
		writeFile(t, config.path, config.before)
	}
	profiles := filepath.Join(root, "profiles")
	if err := os.MkdirAll(filepath.Join(profiles, "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profiles, "default", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: default\nspec:\n  routeRef: main\n")
	bindings := filepath.Join(root, "bindings.yaml")
	writeFile(t, bindings, baseOpenAIBindings+"    targets:\n      claude-code:\n        provider: anthropic\n        model: claude-fable-5-1\n")
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"claude-code", "codex"}, override: true, makeDefault: true}
	planID := regexp.MustCompile(`plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(runInstallDefaultForTest(t, options))
	if planID == nil {
		t.Fatal("multi-target install plan not ready")
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	runInstallDefaultForTest(t, options)
	for index := range configs {
		data, err := os.ReadFile(configs[index].path)
		if err != nil || string(data) == configs[index].before {
			t.Fatalf("%s not installed: %q, err=%v", configs[index].target, data, err)
		}
		configs[index].installed = data
	}
	return configs
}

func runInstallDefaultForTest(t *testing.T, options installOptions) string {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := runInstall(cmd, "default", options); err != nil {
		t.Fatalf("install: %v\n%s", err, output.String())
	}
	return output.String()
}

func undoTargetForTest(t *testing.T, target string) {
	t.Helper()
	preview, err := runUndoForTest(t, restoreOptions{target: target})
	if err != nil {
		t.Fatalf("preview undo %s: %v\n%s", target, err, preview)
	}
	planID := regexp.MustCompile(`undo plan ([0-9a-f]{64}) \(ready\), original install [0-9a-f]{64}`).FindStringSubmatch(preview)
	if planID == nil {
		t.Fatalf("undo %s preview lacks plan:\n%s", target, preview)
	}
	if out, err := runUndoForTest(t, restoreOptions{target: target, apply: true, yes: true, expectPlan: planID[1]}); err != nil {
		t.Fatalf("apply undo %s: %v\n%s", target, err, out)
	}
}

func TestUndoEachTargetOfMultiTargetInstall(t *testing.T) {
	orders := map[string][]int{"codex first": {1, 0}, "claude-code first": {0, 1}}
	for name, order := range orders {
		t.Run(name, func(t *testing.T) {
			configs := installClaudeAndCodexAtDefault(t)
			for step, index := range order {
				undoTargetForTest(t, configs[index].target)
				if data, err := os.ReadFile(configs[index].path); err != nil || string(data) != configs[index].before {
					t.Fatalf("undone %s config = %q, err=%v", configs[index].target, data, err)
				}
				if step == 0 {
					other := configs[order[1]]
					if data, err := os.ReadFile(other.path); err != nil || string(data) != string(other.installed) {
						t.Fatalf("undoing %s changed %s: %q, err=%v", configs[index].target, other.target, data, err)
					}
					if _, err := os.Stat(other.path + ".profile-mango.manifest.json"); err != nil {
						t.Fatalf("undoing %s removed %s manifest: %v", configs[index].target, other.target, err)
					}
				}
			}
		})
	}
}

func TestUndoMultiTargetByOriginalPlanAndRejectsForgedReference(t *testing.T) {
	configs := installClaudeAndCodexAtDefault(t)
	codex := configs[1]
	refs, err := filepath.Glob(codex.path + ".profile-mango.journal-ref.*.json")
	if err != nil || len(refs) != 1 {
		t.Fatalf("codex journal references = %v, err=%v", refs, err)
	}
	preview, err := runUndoForTest(t, restoreOptions{target: "codex"})
	original := regexp.MustCompile(`original install ([0-9a-f]{64})`).FindStringSubmatch(preview)
	if err != nil || original == nil {
		t.Fatalf("preview: %v\n%s", err, preview)
	}
	if out, err := runUndoForTest(t, restoreOptions{target: "codex", originalPlan: original[1]}); err != nil || !strings.Contains(out, "original install "+original[1]) {
		t.Fatalf("named undo preview: %v\n%s", err, out)
	}
	data, err := os.ReadFile(refs[0])
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, refs[0], strings.Replace(string(data), filepath.Join(".claude", "settings.json"), filepath.Join(".codex", "other.toml"), 1))
	if _, err := runUndoForTest(t, restoreOptions{target: "codex"}); err == nil {
		t.Fatal("undo accepted a reference to a journal that does not exist")
	}
}
