package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// allSkipSandbox gives the sandbox HOME only a codex config folder, with no agent commands on PATH.
func allSkipSandbox(t *testing.T) (installOptions, string) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	doctorFakeBinaries(t, nil)
	withAgentDetection(t)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	profiles, bindings := writeCodexInstallInputs(t, root)
	return installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, all: true}, home
}

func runInstallResult(t *testing.T, options installOptions) (string, error) {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	err := runInstall(cmd, "route-only", options)
	return output.String(), err
}

func TestInstallAllSkipsAgentsThatAreNotInstalled(t *testing.T) {
	options, home := allSkipSandbox(t)
	output := runInstallForTest(t, options)
	if !strings.Contains(output, " (ready)\n") || !strings.Contains(output, "  codex@0.154.0: ready | destination:") {
		t.Fatalf("installed codex must plan ready:\n%s", output)
	}
	for _, name := range []string{"hermes", "oh-my-pi", "openclaw", "opencode", "pi"} {
		if !strings.Contains(output, name+"@") || !strings.Contains(output, ": skipped (not installed: no ") {
			t.Fatalf("%s not listed as skipped:\n%s", name, output)
		}
	}
	if strings.Contains(output, "lstat") || !strings.Contains(output, "--all --apply --yes --expect-plan") {
		t.Fatalf("skipped targets must not leak raw errors and the plan stays applicable:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(home, ".hermes")); !os.IsNotExist(err) {
		t.Fatalf("planning created a skipped agent folder: %v", err)
	}
	options.jsonOutput = true
	data := runInstallForTest(t, options)
	if !strings.Contains(data, `"status": "skipped"`) || strings.Contains(data, home) {
		t.Fatalf("JSON plan must list skipped targets without absolute paths:\n%s", data)
	}
}

func TestInstallAllWithNoAgentsPointsToDoctor(t *testing.T) {
	options, home := allSkipSandbox(t)
	if err := os.Remove(filepath.Join(home, ".codex")); err != nil {
		t.Fatal(err)
	}
	output, err := runInstallResult(t, options)
	if err == nil || !strings.Contains(err.Error(), "no supported agents found; run mango doctor") {
		t.Fatalf("all-skipped install error = %v\n%s", err, output)
	}
	if !strings.Contains(output, "(blocked)\n") {
		t.Fatalf("all-skipped plan must be blocked:\n%s", output)
	}
}

func TestInstallTargetMissingFolderNamesFix(t *testing.T) {
	options, home := allSkipSandbox(t)
	options.all, options.targets = false, []string{"hermes"}
	output, err := runInstallResult(t, options)
	want := "hermes config folder " + filepath.Join(home, ".hermes") + " not found — is hermes installed? Pass --config hermes=<path> to choose a file."
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "lstat") {
		t.Fatalf("explicit missing folder: err=%v\n%s", err, output)
	}
	if strings.Contains(output, "hermes@0.21.3: skipped") {
		t.Fatalf("an explicit target is never skipped:\n%s", output)
	}
}
