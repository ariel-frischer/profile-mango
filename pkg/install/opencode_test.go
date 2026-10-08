package install

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const openCodeTestSkill = `---
name: profile-mango-synthetic
description: Use for synthetic OpenCode qualification tests.
---

Synthetic skill body.
`

// openCodeSkillProfile is a route-only profile listing the named skill folders.
func openCodeSkillProfile(names ...string) string {
	profile := "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: primary\n  skills:\n"
	for _, name := range names {
		profile += "    - skills/" + name + "/SKILL.md\n"
	}
	return profile
}

// writeOpenCodeSkills writes the route-only profile and one SKILL.md per named folder,
// with the frontmatter name matching the folder.
func writeOpenCodeSkills(t *testing.T, root string, names ...string) {
	t.Helper()
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), openCodeSkillProfile(names...))
	for _, name := range names {
		writeInstallTestFile(t, filepath.Join(root, "skills", name, "SKILL.md"), strings.Replace(openCodeTestSkill, "profile-mango-synthetic", name, 1))
	}
}

func TestOpenCodeSkillInstallAppliesAndReapplies(t *testing.T) {
	request, root := openCodeTestRequest(t)
	request.Default = true
	writeOpenCodeSkills(t, root, "research", "review")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || !reflect.DeepEqual(plan.Targets[0].Skills, []string{"research", "review"}) {
		t.Fatalf("plan = %#v", plan)
	}
	data, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), root) || strings.Contains(string(data), "config.skills.paths") {
		t.Fatalf("plan leaked target path or wrote skills.paths: %s", data)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	config := request.Targets[0].ConfigPath
	configData, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(configData), "skills") || !strings.Contains(string(configData), `"model": "openai/gpt-5.6"`) {
		t.Fatalf("config = %s", configData)
	}
	for _, name := range []string{"research", "review"} {
		assertInstallTestFile(t, filepath.Join(filepath.Dir(config), "skills", name, "SKILL.md"), strings.Replace(openCodeTestSkill, "profile-mango-synthetic", name, 1))
	}
	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop {
		t.Fatalf("reapply = %#v", reapply)
	}
}

func TestOpenCodeSkillInstallRejectsStaleConfigBeforeResourceWrite(t *testing.T) {
	request, root := openCodeTestRequest(t)
	request.Default = true
	writeOpenCodeSkills(t, root, "research")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, request.Targets[0].ConfigPath, `{ "model": "third-party/edit" }`)
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err == nil {
		t.Fatal("stale skill plan applied")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "skills", "research")); !os.IsNotExist(err) {
		t.Fatalf("stale apply wrote skill folder: %v", err)
	}
}

func TestOpenCodeSkillInstallKeepsUnsupportedRequirementsBlocked(t *testing.T) {
	tests := map[string]struct {
		profile string
		skill   string
		want    string
	}{
		"skill without frontmatter name": {want: "sets name", profile: openCodeSkillProfile("research"), skill: "---\ndescription: Use for research.\n---\nBody.\n"},
		"permissions": {want: "unqualified", profile: `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  permissions:
    mode: read-only
`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root := openCodeTestRequest(t)
			request.Strict, request.Default = true, true
			writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), test.profile)
			if test.skill != "" {
				writeInstallTestFile(t, filepath.Join(root, "skills", "research", "SKILL.md"), test.skill)
			}
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, test.want) {
				t.Fatalf("plan = %#v", plan)
			}
		})
	}
}

func TestOpenCodeNamedInstallSkipsSkills(t *testing.T) {
	request, root := openCodeTestRequest(t)
	writeOpenCodeSkills(t, root, "one", "two")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	want := SkippedRequirement{Requirement: RequirementSkills, Count: 2, Reason: skillsNamedOnlyReason}
	target := plan.Targets[0]
	if plan.Status != StatusReady || len(target.SkippedRequirements) != 1 || target.SkippedRequirements[0] != want || len(target.Skills) != 0 {
		t.Fatalf("plan = %#v", plan)
	}
}
