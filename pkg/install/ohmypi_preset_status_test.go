package install

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOhMyPiStatusRecognizesPresetSwitch(t *testing.T) {
	cases := map[string]struct {
		edit  string
		match bool
	}{"exact": {match: true}, "partial": {edit: "remove"}, "user edit": {edit: "edit"}, "extra role": {edit: "extra"}, "cap edit": {edit: "cap"}, "thinking edit": {edit: "thinking"}, "preset edit": {edit: "preset"}}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			request, root := ohMyPiTestRequest(t)
			request.Override = false
			writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings+"    subagentMaxEffort: high\n  alternate:\n    provider: openai\n    model: gpt-6\n    effort: low\n    subagentMaxEffort: low\n")
			applySwitchTestPlan(t, request)
			config := request.Targets[0].ConfigPath
			data, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			var raw map[string]any
			if err := yaml.Unmarshal(data, &raw); err != nil {
				t.Fatal(err)
			}
			presets := raw["modelPresets"].(map[string]any)
			preset := presets["mango-alternate"].(map[string]any)
			roles := map[string]any{}
			for role, selector := range preset["modelRoles"].(map[string]any) {
				roles[role] = selector
			}
			raw["modelRoles"] = roles
			raw["defaultThinkingLevel"] = preset["defaultThinkingLevel"]
			switch tc.edit {
			case "remove":
				delete(roles, "task")
			case "edit":
				roles["task"] = "user/other:low"
			case "extra":
				roles["advisor"] = "user/other:low"
			case "cap":
				raw["task"].(map[string]any)["maxEffort"] = "max"
			case "thinking":
				raw["defaultThinkingLevel"] = "high"
			case "preset":
				preset["defaultThinkingLevel"] = "high"
			}
			data, err = yaml.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			writeInstallTestFile(t, config, string(data))
			report, err := InspectStatus(StatusRequest{ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot, BindingsPath: request.BindingsPath, Registry: request.Registry, Targets: request.Targets, GlobalSkillsRoot: root + "/skills"})
			if err != nil {
				t.Fatal(err)
			}
			target := report.Targets[0]
			if tc.match {
				encoded, _ := json.Marshal(target)
				if !strings.Contains(string(encoded), `"modelPreset":{"name":"mango-alternate","route":"alternate"`) {
					t.Fatalf("preset missing: %s", encoded)
				}
				if target.ModelPreset.TaskMaxEffort != "high" || target.ModelPreset.RouteSubagentMaxEffort != "low" || !target.ModelPreset.MaxEffortDiffers {
					t.Fatalf("cap did not remain applied: %#v", target.ModelPreset)
				}
				if len(target.Drift) != 0 || target.Source != SourceCurrent {
					t.Fatalf("preset is generic drift: %#v", target)
				}
				plan, err := BuildPlan(request)
				if err != nil || plan.Status != StatusReady {
					t.Fatalf("use over preset: %v %s", err, plan.Status)
				}
				if !strings.Contains(plan.Diagnostics.Error(), "mango-alternate") {
					t.Fatal("plan does not name overwritten preset")
				}
			} else if len(target.Drift) == 0 {
				t.Fatalf("edit not flagged: %#v", target)
			}
			if tc.edit == "cap" {
				plan, err := BuildPlan(request)
				if err != nil || plan.Targets[0].Status != StatusConflict {
					t.Fatalf("preset allowed an edited cap: %v %s", err, plan.Status)
				}
			}
		})
	}
}
