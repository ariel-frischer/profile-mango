package install

import (
	"os"
	"path/filepath"
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
