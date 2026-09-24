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
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
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
	for _, want := range []string{"    destination: " + config + "\n", "    use it: codex --profile route-only\n", "    route-only.config.toml: create\n"} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan omitted %q:\n%s", want, output)
		}
	}
	planID := regexp.MustCompile(`(?m)^plan ([0-9a-f]{64}) \(ready\)$`).FindStringSubmatch(output)
	if planID == nil {
		t.Fatalf("plan not ready:\n%s", output)
	}
	want := "  mango install route-only --profiles " + profiles + " --resource-root " + root + " --bindings " + bindings + " --target codex --apply --yes --expect-plan " + planID[1] + "\n"
	if !strings.HasSuffix(output, want) {
		t.Fatalf("plan does not end with the apply command %q:\n%s", want, output)
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	runInstallForTest(t, options)
	if data, err := os.ReadFile(config); err != nil || string(data) != before {
		t.Fatalf("named install changed config.toml = %q, err=%v", data, err)
	}
	profile := filepath.Join(filepath.Dir(config), "route-only.config.toml")
	if data, err := os.ReadFile(profile); err != nil || !strings.Contains(string(data), `model = "gpt-5.6"`) {
		t.Fatalf("installed profile = %q, err=%v", data, err)
	}
}

func TestInstallDefaultFlagAlsoAdoptsConfig(t *testing.T) {
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
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"codex"}, makeDefault: true}
	output := runInstallForTest(t, options)
	if !strings.Contains(output, "also the default") || !strings.Contains(output, "    config.toml: adopt\n") || !strings.Contains(output, "backed up") {
		t.Fatalf("plan omitted the default or adoption backup:\n%s", output)
	}
	planID := regexp.MustCompile(`(?m)^plan ([0-9a-f]{64}) \(ready\)$`).FindStringSubmatch(output)
	if planID == nil || !strings.Contains(output, " --target codex --default --apply --yes --expect-plan "+planID[1]+"\n") {
		t.Fatalf("apply command lost --default:\n%s", output)
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
	if !strings.Contains(output, "    destination: "+explicit+"\n") || strings.Contains(output, home) {
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

func TestWriteInstallModeExplainsUseAndFallback(t *testing.T) {
	tests := map[string]struct {
		mode *install.InstallMode
		want string
	}{
		"named":         {&install.InstallMode{Mode: install.InstallModeNamedProfile, ProfileName: "coding", UseCommand: "codex --profile coding"}, "    use it: codex --profile coding\n"},
		"named default": {&install.InstallMode{Mode: install.InstallModeNamedProfile, UseCommand: "codex --profile coding", SetsDefault: true}, "    use it: codex --profile coding\n    also the default: the agent uses this profile when none is chosen\n"},
		"no profiles":   {&install.InstallMode{Mode: install.InstallModeDefaultConfig}, "    note: this agent has no profiles; installed as its default settings\n"},
		"named agent":   {nil, ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			if err := writeInstallMode(&output, test.mode); err != nil || output.String() != test.want {
				t.Fatalf("output = %q, err=%v, want %q", output.String(), err, test.want)
			}
		})
	}
}
