package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestInstallHermesNamedProfileAtDefaultHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", "")
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	before := "# keep\nunknown: true\n"
	writeFile(t, config, before)
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"hermes"}}
	output := runInstallForTest(t, options)
	for _, want := range []string{"    destination: " + config + "\n", "    use it: hermes -p route-only\n", "config.yaml: create\n"} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan omitted %q:\n%s", want, output)
		}
	}
	planID := regexp.MustCompile(`(?m)^plan ([0-9a-f]{64}) \(ready\)$`).FindStringSubmatch(output)
	if planID == nil {
		t.Fatalf("plan not ready:\n%s", output)
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	runInstallForTest(t, options)
	if data, err := os.ReadFile(config); err != nil || string(data) != before {
		t.Fatalf("named install changed the default config = %q, err=%v", data, err)
	}
	profile := filepath.Join(home, ".hermes", "profiles", "route-only", "config.yaml")
	if data, err := os.ReadFile(profile); err != nil || !strings.Contains(string(data), `provider: "openai"`) {
		t.Fatalf("installed profile = %q, err=%v", data, err)
	}
}

func TestInstallHermesDefaultFlagAlsoPatchesMainConfig(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", "")
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	before := "# keep\nunknown: true\n"
	writeFile(t, config, before)
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"hermes"}, makeDefault: true}
	output := runInstallForTest(t, options)
	planID := regexp.MustCompile(`(?m)^plan ([0-9a-f]{64}) \(ready\)$`).FindStringSubmatch(output)
	if planID == nil {
		t.Fatalf("plan not ready:\n%s", output)
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	runInstallForTest(t, options)
	data, err := os.ReadFile(config)
	if err != nil || !strings.Contains(string(data), `provider: "openai"`) || !strings.Contains(string(data), "unknown: true") {
		t.Fatalf("--default did not patch the main config = %q, err=%v", data, err)
	}
}
