package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCodeSkillOverrideProtectsExistingResource(t *testing.T) {
	tests := map[string]struct {
		existingSkill bool
		status        string
	}{
		"existing config and new skill":     {status: StatusReady},
		"existing config and unowned skill": {existingSkill: true, status: StatusBlocked},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root := openCodeTestRequest(t)
			addOpenCodeTestSkill(t, root)
			request.Override = true
			config := request.Targets[0].ConfigPath
			before := "{\"model\":\"openai/old\",\"unknown\":true}\n"
			writeInstallTestFile(t, config, before)
			skill := filepath.Join(filepath.Dir(config), "SKILL.md")
			if test.existingSkill {
				writeInstallTestFile(t, skill, "user-owned skill")
			}
			plan, err := BuildPlan(request)
			if err != nil || plan.Status != test.status {
				t.Fatalf("status=%s want=%s err=%v diagnostics=%v", plan.Status, test.status, err, plan.Targets[0].Diagnostics)
			}
			assertInstallTestFile(t, config, before)
			if test.existingSkill {
				assertInstallTestFile(t, skill, "user-owned skill")
				return
			}
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			assertInstallTestFile(t, skill, openCodeTestSkill)
		})
	}
}

func addOpenCodeTestSkill(t *testing.T, root string) {
	t.Helper()
	profile := "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: primary\n  skills:\n    - skills/research/SKILL.md\n"
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), profile)
	resource := filepath.Join(root, "skills", "research", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(resource), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, resource, openCodeTestSkill)
}

func TestOpenCodeSkillOverrideProtectsEditedOwnedResource(t *testing.T) {
	request, root := openCodeTestRequest(t)
	addOpenCodeTestSkill(t, root)
	request.Override = true
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "SKILL.md")
	writeInstallTestFile(t, skill, "externally edited skill")
	plan, err = BuildPlan(request)
	if err != nil || plan.Status != StatusBlocked {
		t.Fatalf("edited skill not protected: %s %v", plan.Status, err)
	}
	assertInstallTestFile(t, skill, "externally edited skill")
}
