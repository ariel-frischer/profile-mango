package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

func TestInstallPlansAndAppliesAgainstDefaultConfigPath(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	before := "# keep\nunknown = true\n"
	writeFile(t, config, before)
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"codex"}}
	output := runInstallForTest(t, options)
	if !strings.Contains(output, "    config: "+config+" (default)\n") || !strings.Contains(output, "config.toml: adopt\n") || !strings.Contains(output, "backed up") {
		t.Fatalf("plan omitted default config path or adoption backup:\n%s", output)
	}
	planID := regexp.MustCompile(`(?m)^plan ([0-9a-f]{64}) \(ready\)$`).FindStringSubmatch(output)
	if planID == nil {
		t.Fatalf("plan not ready:\n%s", output)
	}
	want := "  profile-mango install route-only --profiles " + profiles + " --resource-root " + root + " --bindings " + bindings + " --target codex --apply --yes --expect-plan " + planID[1] + "\n"
	if !strings.HasSuffix(output, want) {
		t.Fatalf("plan does not end with the apply command %q:\n%s", want, output)
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	runInstallForTest(t, options)
	data, err := os.ReadFile(config)
	if err != nil || !strings.HasPrefix(string(data), before) || !strings.Contains(string(data), `model = "gpt-5.6"`) {
		t.Fatalf("installed config = %q, err=%v", data, err)
	}
	if backup, err := os.ReadFile(installfs.BackupPath(config, planID[1])); err != nil || string(backup) != before {
		t.Fatalf("adoption backup = %q, err=%v", backup, err)
	}
}

func TestInstallExplicitConfigOverridesDefault(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	profiles, bindings := writeCodexInstallInputs(t, root)
	explicit := filepath.Join(root, "explicit", "config.toml")
	if err := os.MkdirAll(filepath.Dir(explicit), 0o755); err != nil {
		t.Fatal(err)
	}
	output := runInstallForTest(t, installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, configs: []string{"codex=" + explicit}})
	if !strings.Contains(output, "    config: "+explicit+" (explicit)\n") || strings.Contains(output, home) {
		t.Fatalf("explicit config not used:\n%s", output)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("explicit plan touched default home: %v", err)
	}
}

func runInstallForTest(t *testing.T, options installOptions) string {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := runInstall(cmd, "route-only", options); err != nil {
		t.Fatalf("install: %v\n%s", err, output.String())
	}
	return output.String()
}
