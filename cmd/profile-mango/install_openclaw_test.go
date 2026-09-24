package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestInstallOpenClawNamedProfileAtDefaultHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("OPENCLAW_CONFIG_PATH", "")
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(home, ".openclaw", "openclaw.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	before := "// keep\n{gateway:{port:1234}}\n"
	writeFile(t, config, before)
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, targets: []string{"openclaw"}}
	output := runInstallForTest(t, options)
	for _, want := range []string{"    destination: " + filepath.Join(home, ".openclaw-route-only", "openclaw.json") + "\n", "    use it: openclaw --profile route-only\n", "    openclaw.json: create\n"} {
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
	profile := filepath.Join(home, ".openclaw-route-only", "openclaw.json")
	if data, err := os.ReadFile(profile); err != nil || !strings.Contains(string(data), `primary:"openai/gpt-5.6"`) {
		t.Fatalf("installed profile = %q, err=%v", data, err)
	}
}

func TestInstallOpenClawExplicitConfigMustBeDerivable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	profiles, bindings := writeCodexInstallInputs(t, root)
	explicit := filepath.Join(root, "explicit", "openclaw.json")
	options := installOptions{profiles: profiles, resourceRoot: root, bindings: bindings, configs: []string{"openclaw=" + explicit}}
	output, err := runInstallResult(t, options)
	if err == nil || !strings.Contains(output, "<home>/.openclaw/openclaw.json") {
		t.Fatalf("underivable explicit config was not blocked: err=%v\n%s", err, output)
	}
	canonical := filepath.Join(root, "alt", ".openclaw", "openclaw.json")
	options.configs = []string{"openclaw=" + canonical}
	output = runInstallForTest(t, options)
	if !strings.Contains(output, "    destination: "+filepath.Join(root, "alt", ".openclaw-route-only", "openclaw.json")+"\n") || !strings.Contains(output, "use it: openclaw --profile route-only") {
		t.Fatalf("canonical explicit config not planned:\n%s", output)
	}
}
