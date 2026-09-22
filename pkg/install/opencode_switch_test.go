package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestOpenCodeSwitchRemovesCleanOwnedLegacySkill(t *testing.T) {
	request, root := openCodeTestRequest(t)
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, switchTestProfile(true))
	writeInstallTestFile(t, filepath.Join(root, "SKILL.md"), openCodeTestSkill)
	applySwitchTestPlan(t, request)
	installed := filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "SKILL.md")
	assertInstallTestFile(t, installed, openCodeTestSkill)
	writeInstallTestFile(t, profile, switchTestProfile(false))
	applySwitchTestPlan(t, request)
	if _, err := os.Stat(installed); !os.IsNotExist(err) {
		t.Fatalf("omitted clean owned skill was not removed: %v", err)
	}
}

func switchTestProfile(skill bool) string {
	profile := "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: primary\n"
	if skill {
		profile += "  skills:\n    - SKILL.md\n"
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

func TestOpenCodeSwitchRemovesManagedSkillPathAndManifestOwnership(t *testing.T) {
	request, root := openCodeTestRequest(t)
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, switchTestProfile(true))
	writeInstallTestFile(t, filepath.Join(root, "SKILL.md"), openCodeTestSkill)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, `{ "skills": { "paths": ["existing" /* keep */] } }`)
	applySwitchTestPlan(t, request)
	writeInstallTestFile(t, profile, switchTestProfile(false))
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady {
		t.Fatalf("cleanup plan status = %s, diagnostics=%v", plan.Status, plan.Targets[0].Diagnostics)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	configData, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	configText := string(configData)
	if !strings.Contains(configText, `"existing" /* keep */`) || strings.Contains(configText, skillConfigDir(config)) {
		t.Fatalf("managed path cleanup changed unrelated config: %s", configData)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(config), "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("managed skill remains after cleanup: %v", err)
	}
	manifest := readSwitchManifest(t, config)
	for _, file := range manifest.Files {
		if file.Path == filepath.Join(filepath.Dir(config), "SKILL.md") {
			t.Fatal("removed skill remains in ownership manifest")
		}
		if file.Path == config && containsSwitchValue(file.Fields, openCodeSkillsPathOwnership) {
			t.Fatal("removed skills path remains in ownership manifest")
		}
	}
}

func TestOpenCodeSwitchPreservesLegacyUnmarkedSkillsPath(t *testing.T) {
	request, root := openCodeTestRequest(t)
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, switchTestProfile(true))
	writeInstallTestFile(t, filepath.Join(root, "SKILL.md"), openCodeTestSkill)
	applySwitchTestPlan(t, request)
	config := request.Targets[0].ConfigPath
	manifestPath := config + ".profile-mango.manifest.json"
	manifest := readSwitchManifest(t, config)
	for index := range manifest.Files {
		if manifest.Files[index].Path == config {
			manifest.Files[index].Fields = removeSwitchValue(manifest.Files[index].Fields, openCodeSkillsPathOwnership)
		}
	}
	writeSwitchManifest(t, manifestPath, manifest)
	writeInstallTestFile(t, profile, switchTestProfile(false))
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || !hasSwitchDiagnostic(plan.Targets[0].Diagnostics, "opencode.install.skills_path_preserved") {
		t.Fatalf("legacy cleanup plan = %#v", plan.Targets[0])
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	configData, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configData), skillConfigDir(config)) {
		t.Fatalf("legacy unmarked skills path was removed: %s", configData)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(config), "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("owned legacy skill was not removed: %v", err)
	}
}

func TestOpenCodeSwitchRejectsEditedOwnedSkill(t *testing.T) {
	request, root := openCodeTestRequest(t)
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, switchTestProfile(true))
	writeInstallTestFile(t, filepath.Join(root, "SKILL.md"), openCodeTestSkill)
	applySwitchTestPlan(t, request)
	config := request.Targets[0].ConfigPath
	skill := filepath.Join(filepath.Dir(config), "SKILL.md")
	writeInstallTestFile(t, skill, "edited by user")
	writeInstallTestFile(t, profile, switchTestProfile(false))
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || plan.Targets[0].Status != StatusConflict {
		t.Fatalf("edited omission status = %s/%s, want blocked/conflict", plan.Status, plan.Targets[0].Status)
	}
	if _, err := os.Stat(skill); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeSwitchRejectsMismatchedOwnedSkillHash(t *testing.T) {
	request, root := openCodeTestRequest(t)
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

func TestOpenCodeSwitchCleansMissingOwnedSkillManifest(t *testing.T) {
	request, root := openCodeTestRequest(t)
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, switchTestProfile(true))
	writeInstallTestFile(t, filepath.Join(root, "SKILL.md"), openCodeTestSkill)
	applySwitchTestPlan(t, request)
	config := request.Targets[0].ConfigPath
	if err := os.Remove(filepath.Join(filepath.Dir(config), "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, profile, switchTestProfile(false))
	applySwitchTestPlan(t, request)
	manifest := readSwitchManifest(t, config)
	for _, file := range manifest.Files {
		if file.Path == filepath.Join(filepath.Dir(config), "SKILL.md") {
			t.Fatal("missing skill was retained in ownership manifest")
		}
	}
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

func removeSwitchValue(values []string, value string) []string {
	result := make([]string, 0, len(values))
	for _, candidate := range values {
		if candidate != value {
			result = append(result, candidate)
		}
	}
	return result
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
