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

// installCodexAtDefault installs Codex at its default path under a per-test HOME and returns that config.
func installCodexAtDefault(t *testing.T, before string) string {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, config, before)
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"codex"}, override: true, makeDefault: true}
	planID := regexp.MustCompile(`plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(runInstallForTest(t, options))
	if planID == nil {
		t.Fatal("install plan not ready")
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	runInstallForTest(t, options)
	return config
}

func runUndoForTest(t *testing.T, options restoreOptions) (string, error) {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	err := runRestore(cmd, options)
	return output.String(), err
}

func TestUndoBareTargetDefaultPathPreviewThenApply(t *testing.T) {
	before := "# keep\nunknown = true\n"
	config := installCodexAtDefault(t, before)
	preview, err := runUndoForTest(t, restoreOptions{target: "codex"})
	if err != nil {
		t.Fatalf("preview: %v\n%s", err, preview)
	}
	planID := regexp.MustCompile(`undo plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(preview)
	if planID == nil || !strings.Contains(preview, "update "+config) || !strings.Contains(preview, `-model = "gpt-5.6"`) {
		t.Fatalf("preview lacks plan, default config, or diff:\n%s", preview)
	}
	assertFileContains(t, config, `model = "gpt-5.6"`)
	if out, err := runUndoForTest(t, restoreOptions{target: "codex", apply: true, yes: true, expectPlan: planID[1]}); err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	if data, err := os.ReadFile(config); err != nil || string(data) != before {
		t.Fatalf("undone config = %q, err=%v", data, err)
	}
}

func TestUndoRefusesDriftAndShowsDiffUnlessOverride(t *testing.T) {
	before := "# keep\n"
	config := installCodexAtDefault(t, before)
	installed, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, config, string(installed)+"user_edit = true\n")
	_, err = runUndoForTest(t, restoreOptions{target: "codex", config: config})
	if err == nil || !strings.Contains(err.Error(), "--override") || !strings.Contains(err.Error(), "-user_edit = true") {
		t.Fatalf("drift error = %v", err)
	}
	preview, err := runUndoForTest(t, restoreOptions{target: "codex", config: config, override: true})
	planID := regexp.MustCompile(`undo plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(preview)
	if err != nil || planID == nil {
		t.Fatalf("override preview: %v\n%s", err, preview)
	}
	if out, err := runUndoForTest(t, restoreOptions{target: "codex", config: config, override: true, apply: true, yes: true, expectPlan: planID[1]}); err != nil {
		t.Fatalf("override apply: %v\n%s", err, out)
	}
	if data, err := os.ReadFile(config); err != nil || string(data) != before {
		t.Fatalf("undone config = %q, err=%v", data, err)
	}
}

func TestUndoCommandKeepsRestoreAliasAndRequiresTarget(t *testing.T) {
	found, _, err := rootCmd.Find([]string{"restore"})
	if err != nil || found.Name() != "undo" {
		t.Fatalf("restore alias resolves to %v, err=%v", found, err)
	}
	if _, err := runUndoForTest(t, restoreOptions{}); err == nil || !strings.Contains(err.Error(), "--target") {
		t.Fatalf("missing target error = %v", err)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), want) {
		t.Fatalf("%s = %q, err=%v; want %q", path, data, err, want)
	}
}
