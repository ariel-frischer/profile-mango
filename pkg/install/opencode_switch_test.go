package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// seedLegacyOpenCodeSkill recreates what the one-skill OpenCode install of earlier
// versions left behind: a SKILL.md beside the config, a skills.paths entry for the
// config directory, and manifest ownership of both, the entry marked when marked is set.
func seedLegacyOpenCodeSkill(t *testing.T, request Request, root string, marked bool) (string, string) {
	t.Helper()
	config := request.Targets[0].ConfigPath
	legacy := openCodeLegacySkillPath(config)
	pathsJSON, err := json.Marshal(filepath.Dir(legacy))
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, `{ "skills": { "paths": ["existing" /* keep */, `+string(pathsJSON)+`] } }`)
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), switchTestProfile(false))
	applySwitchTestPlan(t, request)
	writeInstallTestFile(t, legacy, openCodeTestSkill)
	manifest := readSwitchManifest(t, config)
	for index := range manifest.Files {
		if manifest.Files[index].Path == config && marked {
			manifest.Files[index].Fields = append(manifest.Files[index].Fields, openCodeSkillsPathOwnership)
		}
	}
	manifest.Files = append(manifest.Files, ManifestFile{Path: legacy, SHA256: installfs.Hash([]byte(openCodeTestSkill))})
	writeSwitchManifest(t, config+".profile-mango.manifest.json", manifest)
	return config, legacy
}

func TestOpenCodeLegacySkillMigration(t *testing.T) {
	tests := map[string]struct {
		skill       bool
		marked      bool
		wantPath    bool
		wantWarning bool
	}{
		"marked entry, skill profile":   {skill: true, marked: true},
		"marked entry, no skills":       {marked: true},
		"unmarked entry is preserved":   {skill: true, wantPath: true, wantWarning: true},
		"unmarked entry without skills": {wantPath: true, wantWarning: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root := openCodeTestRequest(t)
			request.Default = true
			config, legacy := seedLegacyOpenCodeSkill(t, request, root, test.marked)
			writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), switchTestProfile(test.skill))
			writeInstallTestFile(t, filepath.Join(root, "skills", "profile-mango-synthetic", "SKILL.md"), openCodeTestSkill)
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status != StatusReady || hasSwitchDiagnostic(plan.Targets[0].Diagnostics, "opencode.install.skills_path_preserved") != test.wantWarning {
				t.Fatalf("migration plan = %s %#v", plan.Status, plan.Targets[0].Diagnostics)
			}
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			assertLegacySkillReleased(t, config, legacy, test.wantPath)
			installed := filepath.Join(filepath.Dir(config), "skills", "profile-mango-synthetic", "SKILL.md")
			if _, err := os.Stat(installed); (err == nil) != test.skill {
				t.Fatalf("skill folder installed = %v, want %v", err == nil, test.skill)
			}
		})
	}
}

// assertLegacySkillReleased checks the legacy SKILL.md and its ownership are gone and
// the skills.paths entry for the config directory remains only when wantPath is set.
func assertLegacySkillReleased(t *testing.T, config, legacy string, wantPath bool) {
	t.Helper()
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy SKILL.md remains: %v", err)
	}
	configData, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	pathsJSON, _ := json.Marshal(filepath.Dir(legacy))
	if !strings.Contains(string(configData), `"existing" /* keep */`) || strings.Contains(string(configData), string(pathsJSON)) != wantPath {
		t.Fatalf("skills.paths after migration = %s", configData)
	}
	for _, file := range readSwitchManifest(t, config).Files {
		if file.Path == legacy || (file.Path == config && containsSwitchValue(file.Fields, openCodeSkillsPathOwnership)) {
			t.Fatalf("legacy skill ownership remains: %#v", file)
		}
	}
}

func switchTestProfile(skill bool) string {
	profile := "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: primary\n"
	if skill {
		profile += "  skills:\n    - skills/profile-mango-synthetic/SKILL.md\n"
	}
	return profile
}

func applySwitchTestPlan(t *testing.T, request Request) {
	t.Helper()
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady && plan.Status != StatusNoop {
		t.Fatalf("switch plan blocked: %#v", plan.Diagnostics)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeOverrideEditedOwnedAgent(t *testing.T) {
	request, _ := openCodeTestRequest(t)
	request.Override = false
	applySwitchTestPlan(t, request)
	agent := filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "agents", "route-only.md")
	original, err := os.ReadFile(agent)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(original), "gpt-5.6", "gpt-6.1-sol", 1)
	if edited == string(original) {
		t.Fatal("fixture model was not edited")
	}
	writeInstallTestFile(t, agent, edited)
	blocked, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Status != StatusBlocked || blocked.Targets[0].Status != StatusConflict {
		t.Fatalf("edited agent without override = %s/%s", blocked.Status, blocked.Targets[0].Status)
	}
	request.Override = true
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady {
		t.Fatalf("override plan = %s: %#v", plan.Status, plan.Targets[0].Diagnostics)
	}
	if file := piFilePlan(plan.Targets[0], filepath.Base(agent)); file.Action != ActionOverride {
		t.Fatalf("override file = %#v", file)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	assertInstallTestFile(t, agent, string(original))
	assertInstallTestFile(t, installfs.BackupPath(agent, plan.PlanID), edited)
}

func TestOpenCodeSwitchRejectsEditedLegacySkill(t *testing.T) {
	request, root := openCodeTestRequest(t)
	request.Default = true
	_, legacy := seedLegacyOpenCodeSkill(t, request, root, true)
	writeInstallTestFile(t, legacy, "edited by user")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || plan.Targets[0].Status != StatusConflict {
		t.Fatalf("edited omission status = %s/%s, want blocked/conflict", plan.Status, plan.Targets[0].Status)
	}
	assertInstallTestFile(t, legacy, "edited by user")
}

func TestOpenCodeSwitchRejectsMismatchedOwnedSkillHash(t *testing.T) {
	request, root := openCodeTestRequest(t)
	request.Default = true
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, switchTestProfile(false))
	config := request.Targets[0].ConfigPath
	skill := filepath.Join(filepath.Dir(config), "SKILL.md")
	writeInstallTestFile(t, skill, "unowned content")
	manifest := Manifest{APIVersion: ManifestAPIVersion, Kind: ManifestKind, Owner: "profile-mango", Generation: 1, Profile: "route-only", Target: request.Targets[0].Target, Files: []ManifestFile{{Path: skill, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}
	writeSwitchManifest(t, config+".profile-mango.manifest.json", manifest)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || plan.Targets[0].Status != StatusConflict {
		t.Fatalf("mismatched owned omission status = %s/%s, want blocked/conflict", plan.Status, plan.Targets[0].Status)
	}
}

func TestOpenCodeSwitchCleansMissingLegacySkillManifest(t *testing.T) {
	request, root := openCodeTestRequest(t)
	request.Default = true
	config, legacy := seedLegacyOpenCodeSkill(t, request, root, true)
	if err := os.Remove(legacy); err != nil {
		t.Fatal(err)
	}
	applySwitchTestPlan(t, request)
	assertLegacySkillReleased(t, config, legacy, false)
}

func readSwitchManifest(t *testing.T, config string) Manifest {
	t.Helper()
	data, err := os.ReadFile(config + ".profile-mango.manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeSwitchManifest(t *testing.T, path string, manifest Manifest) {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, path, string(data)+"\n")
}

func containsSwitchValue(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func hasSwitchDiagnostic(diagnostics []profilemango.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
