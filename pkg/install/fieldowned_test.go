package install

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestFieldOwnedConfigConflictsOnlyOnOwnedValueEdits installs an Oh My Pi config,
// edits it, changes the profile's route, and plans again without --override: edits
// outside the owned fields update in place, an edited owned value still conflicts.
func TestFieldOwnedConfigConflictsOnlyOnOwnedValueEdits(t *testing.T) {
	cases := map[string]struct {
		edit       func(t *testing.T, config string)
		wantStatus string
		wantAction string
	}{
		"re-serialized config with unrelated keys updates": {
			edit: func(t *testing.T, config string) {
				replaceInstallTestFile(t, config, `default: "openai/gpt-5.6:high"`, "default: openai/gpt-5.6:high")
				appendInstallTestFile(t, config, "theme: dark\n")
			},
			wantStatus: StatusReady,
			wantAction: ActionUpdate,
		},
		"hand-edited owned value conflicts": {
			edit: func(t *testing.T, config string) {
				replaceInstallTestFile(t, config, `default: "openai/gpt-5.6:high"`, "default: other/model:low")
			},
			wantStatus: StatusConflict,
			wantAction: ActionUpdate,
		},
		"removed owned value conflicts": {
			edit: func(t *testing.T, config string) {
				replaceInstallTestFile(t, config, `  default: "openai/gpt-5.6:high"`+"\n", "  reviewer: other/model\n")
			},
			wantStatus: StatusConflict,
			wantAction: ActionUpdate,
		},
		"manifest without written values keeps the whole-file hash": {
			edit: func(t *testing.T, config string) {
				stripWrittenMarkers(t, config+".profile-mango.manifest.json")
				replaceInstallTestFile(t, config, `default: "openai/gpt-5.6:high"`, "default: openai/gpt-5.6:high")
			},
			wantStatus: StatusConflict,
			wantAction: ActionUpdate,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			request, config := installedOhMyPiConfig(t)
			tc.edit(t, config)
			writeInstallTestFile(t, request.BindingsPath, strings.Replace(ohMyPiTestBindings, "effort: high", "effort: low", 1))
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			file := claudeCodeFile(plan.Targets[0], "config.yml")
			if plan.Targets[0].Status != tc.wantStatus || file.Action != tc.wantAction {
				t.Fatalf("status = %s, config action = %s, diagnostics = %v", plan.Targets[0].Status, file.Action, plan.Targets[0].Diagnostics)
			}
			if tc.wantStatus == StatusReady {
				assertFieldOwnedApply(t, request, plan, config)
			}
		})
	}
}

// assertFieldOwnedApply applies plan, checks unrelated edits survived beside the new
// owned value, and checks the recorded written values let the next plan settle.
func assertFieldOwnedApply(t *testing.T, request Request, plan Plan, config string) {
	t.Helper()
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "theme: dark\n") || !strings.Contains(string(data), "openai/gpt-5.6:low") {
		t.Fatalf("config after apply:\n%s", data)
	}
	appendInstallTestFile(t, config, "editor: vim\n")
	writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings)
	again, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if again.Targets[0].Status != StatusReady {
		t.Fatalf("second change after re-edit = %s, diagnostics = %v", again.Targets[0].Status, again.Targets[0].Diagnostics)
	}
}

const ohMyPiTestBindings = `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`

// installedOhMyPiConfig applies a fresh --default Oh My Pi install into a synthetic
// home and returns a request planned without --override.
func installedOhMyPiConfig(t *testing.T) (Request, string) {
	t.Helper()
	request, _ := ohMyPiTestRequest(t)
	config := request.Targets[0].ConfigPath
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	request.Override = false
	return request, config
}

// stripWrittenMarkers rewrites a manifest as an earlier version recorded it, without
// written-value markers.
func stripWrittenMarkers(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	stripped := 0
	for index, file := range manifest.Files {
		kept := file.Fields[:0]
		for _, field := range file.Fields {
			if strings.HasPrefix(field, writtenSHA256Prefix) {
				stripped++
				continue
			}
			kept = append(kept, field)
		}
		manifest.Files[index].Fields = kept
	}
	if stripped == 0 {
		t.Fatalf("manifest has no written-value markers:\n%s", data)
	}
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, path, string(data)+"\n")
}

func replaceInstallTestFile(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q:\n%s", path, old, data)
	}
	writeInstallTestFile(t, path, strings.Replace(string(data), old, replacement, 1))
}

func appendInstallTestFile(t *testing.T, path, suffix string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, path, string(data)+suffix)
}
