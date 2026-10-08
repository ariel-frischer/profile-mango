package install

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOhMyPiPresetsCarryUnmanagedRoles(t *testing.T) {
	for _, global := range []bool{true, false} {
		t.Run(map[bool]string{true: "default", false: "overlay"}[global], func(t *testing.T) {
			request, _ := ohMyPiTestRequest(t)
			request.Default = global
			request.Override = false
			config := request.Targets[0].ConfigPath
			for _, selector := range []string{"openai-codex/gpt-6.1-sol:medium", "openai-codex/gpt-6.1-sol:high"} {
				data, err := os.ReadFile(config)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				var raw map[string]any
				if len(data) > 0 {
					if err := yaml.Unmarshal(data, &raw); err != nil {
						t.Fatal(err)
					}
				}
				if raw == nil {
					raw = map[string]any{}
				}
				roles, ok := raw["modelRoles"].(map[string]any)
				if !ok {
					roles = map[string]any{}
				}
				roles["code"], roles["web"] = selector, "web/exa"
				raw["modelRoles"] = roles
				data, err = yaml.Marshal(raw)
				if err != nil {
					t.Fatal(err)
				}
				writeInstallTestFile(t, config, string(data))
				plan, err := BuildPlan(request)
				if err != nil || plan.Status != StatusReady {
					t.Fatalf("plan: %v %#v", err, plan)
				}
				applySwitchTestPlan(t, request)
				destination := config
				if !global {
					destination = filepath.Join(filepath.Dir(config), "profiles", request.ProfileName+".yml")
				}
				data, err = os.ReadFile(destination)
				if err != nil {
					t.Fatal(err)
				}
				if err := yaml.Unmarshal(data, &raw); err != nil {
					t.Fatal(err)
				}
				presets := raw["modelPresets"].(map[string]any)
				for name, value := range presets {
					roles := value.(map[string]any)["modelRoles"].(map[string]any)
					if roles["code"] != selector || roles["web"] != "web/exa" {
						t.Fatalf("%s carried roles = %#v", name, roles)
					}
				}
			}
		})
	}
}

const ohMyPiCustomBindings = "routes:\n  primary:\n    provider: openai\n    model: main\n    effort: high\n    roles:\n      code: {provider: openai, model: coding, effort: medium}\n      review: {provider: openai, model: reviewing}\n"

func TestOhMyPiCustomBindingsOwnAndReleaseSlots(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "modelRoles:\n  code: original/code\n  web: web/exa\n")
	writeInstallTestFile(t, request.BindingsPath, ohMyPiCustomBindings)
	applySwitchTestPlan(t, request)
	raw := readOhMyPiTestConfig(t, config)
	roles := raw["modelRoles"].(map[string]any)
	if roles["code"] != "openai/coding:medium" || roles["review"] != "openai/reviewing" || roles["web"] != "web/exa" {
		t.Fatalf("custom slots = %#v", roles)
	}
	preset := raw["modelPresets"].(map[string]any)["mango-primary"].(map[string]any)["modelRoles"].(map[string]any)
	if preset["code"] != roles["code"] || preset["review"] != roles["review"] || preset["web"] != roles["web"] {
		t.Fatalf("preset = %#v", preset)
	}
	entry, _ := manifestEntry(readSwitchManifest(t, config), config)
	for _, role := range []string{"code", "review"} {
		if !slices.Contains(entry.Fields, "config.modelRoles."+role) {
			t.Fatalf("custom role %s not owned: %#v", role, entry)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(config), "agents", role+".md")); !os.IsNotExist(err) {
			t.Fatalf("binding-only role %s wrote an agent file: %v", role, err)
		}
	}
	writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings)
	applySwitchTestPlan(t, request)
	raw = readOhMyPiTestConfig(t, config)
	roles = raw["modelRoles"].(map[string]any)
	if _, found := roles["review"]; found || roles["code"] != "original/code" || roles["web"] != "web/exa" {
		t.Fatalf("released slots = %#v", roles)
	}
	preset = raw["modelPresets"].(map[string]any)["mango-primary"].(map[string]any)["modelRoles"].(map[string]any)
	if _, found := preset["review"]; found || preset["code"] != "original/code" {
		t.Fatalf("released slots carried into preset: %#v", preset)
	}
}

func readOhMyPiTestConfig(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestOhMyPiPresetOnlyCustomSlotRelease(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	request.Override = false
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "modelRoles:\n  code: original/code\n  web: web/exa\n")
	writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings+"  alternate:\n    provider: openai\n    model: alternate\n    effort: low\n    roles:\n      code: {provider: openai, model: coding}\n      review: {provider: openai, model: reviewing}\n")
	applySwitchTestPlan(t, request)
	raw := readOhMyPiTestConfig(t, config)
	preset := raw["modelPresets"].(map[string]any)["mango-alternate"].(map[string]any)
	raw["modelRoles"], raw["defaultThinkingLevel"] = preset["modelRoles"], preset["defaultThinkingLevel"]
	data, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, string(data))
	entry, _ := manifestEntry(readSwitchManifest(t, config), config)
	if status := matchingOhMyPiPreset(data, entry); status == nil || status.Route != "alternate" {
		t.Fatalf("custom preset not recognized: %#v", status)
	}
	// Remove the switched route from bindings too: persisted ownership must suffice.
	writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings)
	applySwitchTestPlan(t, request)
	roles := readOhMyPiTestConfig(t, config)["modelRoles"].(map[string]any)
	if _, found := roles["review"]; found || roles["code"] != "original/code" || roles["web"] != "web/exa" {
		t.Fatalf("preset-only slots not released: %#v", roles)
	}
}

func TestOhMyPiCustomBindingOverlay(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	request.Default, request.Strict = false, true
	writeInstallTestFile(t, request.BindingsPath, ohMyPiCustomBindings)
	applySwitchTestPlan(t, request)
	config := request.Targets[0].ConfigPath
	path := filepath.Join(filepath.Dir(config), "profiles", request.ProfileName+".yml")
	raw := readOhMyPiTestConfig(t, path)
	roles := raw["modelRoles"].(map[string]any)
	if roles["code"] != "openai/coding:medium" || roles["review"] != "openai/reviewing" {
		t.Fatalf("overlay roles = %#v", roles)
	}
	preset := raw["modelPresets"].(map[string]any)["mango-primary"].(map[string]any)["modelRoles"].(map[string]any)
	if preset["code"] != roles["code"] || preset["review"] != roles["review"] {
		t.Fatalf("overlay preset = %#v", preset)
	}
}
